package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDrivers_Initialization(t *testing.T) {
	drivers := []Driver{DriverSlog, DriverZap, DriverZerolog, DriverNoop}

	for _, d := range drivers {
		t.Run(string(d), func(t *testing.T) {
			var buf bytes.Buffer
			cfg := Config{
				Driver: d,
				Level:  LevelInfo,
				Format: FormatJSON,
				Output: &buf,
			}

			l, cleanup, err := New(cfg)
			if err != nil {
				t.Fatalf("failed to create logger for driver %s: %v", d, err)
			}
			defer func() {
				if err := cleanup(); err != nil {
					t.Errorf("cleanup error: %v", err)
				}
			}()

			l.Info("test message", "station_id", "MRT-DKA", "fare", 3000)

			if d == DriverNoop {
				if buf.Len() != 0 {
					t.Fatalf("expected no output for noop, got: %s", buf.String())
				}
				return
			}

			output := buf.String()
			if !strings.Contains(output, "test message") {
				t.Fatalf("expected output to contain message, got: %s", output)
			}

			var parsed map[string]any
			if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
				t.Fatalf("output is not valid JSON (%s): %v", output, err)
			}
		})
	}
}

func TestDrivers_LevelFiltering(t *testing.T) {
	drivers := []Driver{DriverSlog, DriverZap, DriverZerolog}

	for _, d := range drivers {
		t.Run(string(d), func(t *testing.T) {
			var buf bytes.Buffer
			cfg := Config{
				Driver: d,
				Level:  LevelWarn,
				Format: FormatJSON,
				Output: &buf,
			}

			l, cleanup, err := New(cfg)
			if err != nil {
				t.Fatalf("failed to create logger: %v", err)
			}
			defer cleanup()

			l.Info("this should be ignored")
			if buf.Len() != 0 {
				t.Fatalf("expected no output for info level, got: %s", buf.String())
			}

			l.Warn("warning logged")
			if !strings.Contains(buf.String(), "warning logged") {
				t.Fatalf("expected warning message, got: %s", buf.String())
			}
		})
	}
}

func TestDrivers_InvalidDriver(t *testing.T) {
	cfg := Config{
		Driver: "unknown_logger",
	}
	_, _, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for invalid driver, got nil")
	}
}
