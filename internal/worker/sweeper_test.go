package worker

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"golingko/internal/transit"
)

type stubSweeperRepo struct {
	transit.TransitRepository
	expiredTrips   []transit.ExpiredTripDTO
	lockedCard     *transit.CardRecord
	expireErr      error
	lockedCardNum  string
	updatedBalance *int64
	ledgerRecorded []transit.LedgerRecord
}

func (r *stubSweeperRepo) ListExpiredInProgressJourneys(ctx context.Context, exec transit.DBExecutor, threshold time.Time, limit int) ([]transit.ExpiredTripDTO, error) {
	return r.expiredTrips, nil
}

func (r *stubSweeperRepo) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

func (r *stubSweeperRepo) GetCardForUpdate(ctx context.Context, tx *sql.Tx, number string) (*transit.CardRecord, error) {
	r.lockedCardNum = number
	copied := *r.lockedCard
	return &copied, nil
}

func (r *stubSweeperRepo) ExpireJourney(ctx context.Context, tx *sql.Tx, journeyID string, penalty int64) error {
	return r.expireErr
}

func (r *stubSweeperRepo) UpdateCardBalance(ctx context.Context, tx *sql.Tx, cardID string, balance int64) error {
	r.updatedBalance = &balance
	return nil
}

func (r *stubSweeperRepo) RecordLedgerTransaction(ctx context.Context, tx *sql.Tx, entry transit.LedgerRecord) (*transit.LedgerRecord, error) {
	r.ledgerRecorded = append(r.ledgerRecorded, entry)
	return &entry, nil
}

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

func TestSweeper_PreservesTopUpAndLocksCard(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := &stubSweeperRepo{
		expiredTrips: []transit.ExpiredTripDTO{
			{
				JourneyID:  "journey-1",
				CardID:     "card-1",
				CardNumber: "1000111122223333",
				Balance:    15000,
			},
		},
		lockedCard: &transit.CardRecord{
			ID:         "card-1",
			CardNumber: "1000111122223333",
			Balance:    65000,
			Status:     "active",
		},
	}

	sweeper := NewSweeper(repo, 1*time.Minute, 4*time.Hour, logger)
	swept, err := sweeper.SweepOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if swept != 1 {
		t.Fatalf("expected 1 swept trip, got %d", swept)
	}
	if repo.lockedCardNum != "1000111122223333" {
		t.Fatalf("expected GetCardForUpdate to lock card 1000111122223333, got %q", repo.lockedCardNum)
	}
	if repo.updatedBalance == nil || *repo.updatedBalance != 55000 {
		t.Fatalf("expected balance 55000 after penalty on topped-up card, got %v", repo.updatedBalance)
	}
	if len(repo.ledgerRecorded) != 1 || repo.ledgerRecorded[0].BalanceAfter != 55000 {
		t.Fatalf("expected 1 penalty ledger transaction with BalanceAfter 55000, got %+v", repo.ledgerRecorded)
	}
}

func TestSweeper_SkipsPenaltyWhenJourneyAlreadyCompleted(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := &stubSweeperRepo{
		expiredTrips: []transit.ExpiredTripDTO{
			{
				JourneyID:  "journey-2",
				CardID:     "card-2",
				CardNumber: "1000444455556666",
				Balance:    30000,
			},
		},
		lockedCard: &transit.CardRecord{
			ID:         "card-2",
			CardNumber: "1000444455556666",
			Balance:    26500,
			Status:     "active",
		},
		expireErr: transit.ErrNoActiveJourney,
	}

	sweeper := NewSweeper(repo, 1*time.Minute, 4*time.Hour, logger)
	swept, err := sweeper.SweepOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if swept != 0 {
		t.Fatalf("expected 0 swept trips when journey already completed, got %d", swept)
	}
	if repo.updatedBalance != nil {
		t.Fatalf("expected no balance deduction when ExpireJourney fails, got %d", *repo.updatedBalance)
	}
	if len(repo.ledgerRecorded) != 0 {
		t.Fatalf("expected 0 penalty ledger records, got %d", len(repo.ledgerRecorded))
	}
}
