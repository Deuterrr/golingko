package logger

import (
	"io"
	"testing"
)

func BenchmarkSlog(b *testing.B) {
	l, cleanup, err := New(Config{
		Driver: DriverSlog,
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: io.Discard,
	})
	if err != nil {
		b.Fatalf("failed to init slog: %v", err)
	}
	defer cleanup()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Info("gate tap-in", "card_number", "1000123456789012", "station", "MRT-DKA", "status", 200)
	}
}

func BenchmarkZap(b *testing.B) {
	l, cleanup, err := New(Config{
		Driver: DriverZap,
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: io.Discard,
	})
	if err != nil {
		b.Fatalf("failed to init zap: %v", err)
	}
	defer cleanup()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Info("gate tap-in", "card_number", "1000123456789012", "station", "MRT-DKA", "status", 200)
	}
}

func BenchmarkZerolog(b *testing.B) {
	l, cleanup, err := New(Config{
		Driver: DriverZerolog,
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: io.Discard,
	})
	if err != nil {
		b.Fatalf("failed to init zerolog: %v", err)
	}
	defer cleanup()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Info("gate tap-in", "card_number", "1000123456789012", "station", "MRT-DKA", "status", 200)
	}
}

func BenchmarkNoop(b *testing.B) {
	l := NewNoopLogger()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Info("gate tap-in", "card_number", "1000123456789012", "station", "MRT-DKA", "status", 200)
	}
}
