package logger

import (
	"io"
	"log/slog"
	"os"
)

func newSlogHandler(cfg Config) (slog.Handler, func() error, error) {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}

	level := toSlogLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == FormatText {
		handler = slog.NewTextHandler(out, opts)
	} else {
		handler = slog.NewJSONHandler(out, opts)
	}

	cleanup := func() error {
		if closer, ok := out.(io.Closer); ok && out != os.Stdout && out != os.Stderr {
			return closer.Close()
		}
		return nil
	}

	return handler, cleanup, nil
}

func toSlogLevel(lvl Level) slog.Level {
	switch lvl {
	case LevelDebug:
		return slog.LevelDebug
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
