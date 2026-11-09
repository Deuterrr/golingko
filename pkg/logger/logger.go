// Package logger provides structured execution telemetry and pluggable logging backends.
package logger

import (
	"fmt"
	"io"
	"log/slog"
)

// Logging backend implementation.
type Driver string

const (
	DriverSlog    Driver = "slog"
	DriverZap     Driver = "zap"
	DriverZerolog Driver = "zerolog"
	DriverNoop    Driver = "noop"
)

// Specifies the threshold for emitting log records.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Output layout for structured log records.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Operational parameters for instantiating the structured logging engine.
type Config struct {
	// Logging backend implementation.
	Driver Driver
	// Specifies the threshold for emitting log records.
	Level Level
	// Output layout for structured log records.
	Format Format
	Output io.Writer
}

func New(cfg Config) (*slog.Logger, func() error, error) {
	switch cfg.Driver {
	case DriverSlog, "":
		handler, cleanup, err := newSlogHandler(cfg)
		if err != nil {
			return nil, nil, err
		}
		return slog.New(handler), cleanup, nil
	case DriverZap:
		handler, cleanup, err := newZapHandler(cfg)
		if err != nil {
			return nil, nil, err
		}
		return slog.New(handler), cleanup, nil
	case DriverZerolog:
		handler, cleanup, err := newZerologHandler(cfg)
		if err != nil {
			return nil, nil, err
		}
		return slog.New(handler), cleanup, nil
	case DriverNoop:
		return NewNoopLogger(), func() error { return nil }, nil
	default:
		return nil, nil, fmt.Errorf("unsupported logging driver: %q (accepted: slog, zap, zerolog, noop)", cfg.Driver)
	}
}
