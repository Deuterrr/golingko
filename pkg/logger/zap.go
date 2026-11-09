package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapFieldsPool = sync.Pool{
	New: func() any {
		slice := make([]zapcore.Field, 0, 16)
		return &slice
	},
}

type zapHandler struct {
	core zapcore.Core
}

type zapGroupObject []slog.Attr

func (gs zapGroupObject) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, attr := range gs {
		f := convertAttrToField(attr)
		if f.Type != zapcore.SkipType {
			f.AddTo(enc)
		}
	}
	return nil
}

func newZapHandler(cfg Config) (slog.Handler, func() error, error) {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	ws := zapcore.AddSync(out)
	level := toZapLevel(cfg.Level)

	var encoder zapcore.Encoder
	if cfg.Format == FormatText {
		encCfg := zap.NewDevelopmentEncoderConfig()
		encCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000Z0700")
		encoder = zapcore.NewConsoleEncoder(encCfg)
	} else {
		encCfg := zap.NewProductionEncoderConfig()
		encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(encCfg)
	}

	core := zapcore.NewCore(encoder, ws, level)
	handler := &zapHandler{core: core}

	cleanup := func() error {
		_ = core.Sync()
		if closer, ok := out.(io.Closer); ok && out != os.Stdout && out != os.Stderr {
			return closer.Close()
		}
		return nil
	}

	return handler, cleanup, nil
}

func (h *zapHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.core.Enabled(toZapcoreLevel(level))
}

func (h *zapHandler) Handle(_ context.Context, r slog.Record) error {
	ent := zapcore.Entry{
		Level:   toZapcoreLevel(r.Level),
		Time:    r.Time,
		Message: r.Message,
	}

	ce := h.core.Check(ent, nil)
	if ce == nil {
		return nil
	}

	fieldsPtr := zapFieldsPool.Get().(*[]zapcore.Field)
	fields := (*fieldsPtr)[:0]

	r.Attrs(func(attr slog.Attr) bool {
		f := convertAttrToField(attr)
		if f.Type != zapcore.SkipType {
			fields = append(fields, f)
		}
		return true
	})

	ce.Write(fields...)

	if cap(fields) <= 128 {
		*fieldsPtr = fields[:0]
		zapFieldsPool.Put(fieldsPtr)
	}
	return nil
}

func (h *zapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	fields := make([]zapcore.Field, 0, len(attrs))
	for _, attr := range attrs {
		f := convertAttrToField(attr)
		if f.Type != zapcore.SkipType {
			fields = append(fields, f)
		}
	}
	return &zapHandler{core: h.core.With(fields)}
}

func (h *zapHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &zapHandler{core: h.core.With([]zapcore.Field{zap.Namespace(name)})}
}

func convertAttrToField(attr slog.Attr) zapcore.Field {
	if attr.Equal(slog.Attr{}) {
		return zap.Skip()
	}

	val := attr.Value.Resolve()
	switch val.Kind() {
	case slog.KindBool:
		return zap.Bool(attr.Key, val.Bool())
	case slog.KindDuration:
		return zap.Duration(attr.Key, val.Duration())
	case slog.KindFloat64:
		return zap.Float64(attr.Key, val.Float64())
	case slog.KindInt64:
		return zap.Int64(attr.Key, val.Int64())
	case slog.KindString:
		return zap.String(attr.Key, val.String())
	case slog.KindTime:
		return zap.Time(attr.Key, val.Time())
	case slog.KindUint64:
		return zap.Uint64(attr.Key, val.Uint64())
	case slog.KindGroup:
		if attr.Key == "" {
			return zap.Inline(zapGroupObject(val.Group()))
		}
		return zap.Object(attr.Key, zapGroupObject(val.Group()))
	default:
		return zap.Any(attr.Key, val.Any())
	}
}

func toZapcoreLevel(l slog.Level) zapcore.Level {
	switch {
	case l >= slog.LevelError:
		return zapcore.ErrorLevel
	case l >= slog.LevelWarn:
		return zapcore.WarnLevel
	case l >= slog.LevelInfo:
		return zapcore.InfoLevel
	default:
		return zapcore.DebugLevel
	}
}

func toZapLevel(lvl Level) zapcore.Level {
	switch lvl {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
