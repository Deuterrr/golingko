package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestSweeper_DetachedDB(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sweeper := NewSweeper(nil, 1*time.Minute, 4*time.Hour, logger)

	sweeper.Start()
	sweeper.Stop()

	count, err := sweeper.SweepOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error on detached DB, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 swept trips on detached DB, got %d", count)
	}
}
