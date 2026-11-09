package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type zerologHandler struct {
	logger zerolog.Logger
	level  slog.Level
	attrs  []slog.Attr
	group  string
}

func newZerologHandler(cfg Config) (slog.Handler, func() error, error) {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	zLevel := toZerologLevel(cfg.Level)
	sLevel := toSlogLevel(cfg.Level)

	var zLogger zerolog.Logger
	if cfg.Format == FormatText {
		zLogger = zerolog.New(zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}).
			Level(zLevel).
			With().
			Timestamp().
			Logger()
	} else {
		zLogger = zerolog.New(out).
			Level(zLevel).
			With().
			Timestamp().
			Logger()
	}

	h := &zerologHandler{
		logger: zLogger,
		level:  sLevel,
	}

	cleanup := func() error {
		if closer, ok := out.(io.Closer); ok && out != os.Stdout && out != os.Stderr {
			return closer.Close()
		}
		return nil
	}

	return h, cleanup, nil
}

func (h *zerologHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *zerologHandler) Handle(_ context.Context, r slog.Record) error {
	var event *zerolog.Event
	switch {
	case r.Level < slog.LevelInfo:
		event = h.logger.Debug()
	case r.Level < slog.LevelWarn:
		event = h.logger.Info()
	case r.Level < slog.LevelError:
		event = h.logger.Warn()
	default:
		event = h.logger.Error()
	}

	if !event.Enabled() {
		return nil
	}

	if !r.Time.IsZero() {
		event.Time(zerolog.TimestampFieldName, r.Time)
	}

	for _, a := range h.attrs {
		appendZerologAttr(event, h.group, a)
	}

	r.Attrs(func(a slog.Attr) bool {
		appendZerologAttr(event, h.group, a)
		return true
	})

	event.Msg(r.Message)
	return nil
}

func (h *zerologHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cloned := *h
	cloned.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &cloned
}

func (h *zerologHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	cloned := *h
	if cloned.group == "" {
		cloned.group = name
	} else {
		cloned.group = cloned.group + "." + name
	}
	return &cloned
}

func appendZerologAttr(e *zerolog.Event, group string, a slog.Attr) {
	key := a.Key
	if group != "" {
		key = group + "." + key
	}
	val := a.Value.Resolve()
	switch val.Kind() {
	case slog.KindString:
		e.Str(key, val.String())
	case slog.KindInt64:
		e.Int64(key, val.Int64())
	case slog.KindUint64:
		e.Uint64(key, val.Uint64())
	case slog.KindFloat64:
		e.Float64(key, val.Float64())
	case slog.KindBool:
		e.Bool(key, val.Bool())
	case slog.KindDuration:
		e.Dur(key, val.Duration())
	case slog.KindTime:
		e.Time(key, val.Time())
	default:
		e.Any(key, val.Any())
	}
}

func toZerologLevel(lvl Level) zerolog.Level {
	switch lvl {
	case LevelDebug:
		return zerolog.DebugLevel
	case LevelWarn:
		return zerolog.WarnLevel
	case LevelError:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
