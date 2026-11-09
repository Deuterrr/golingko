package transit

import (
	"context"
	"testing"
	"time"
)

func TestSqlRepository_NilExecutorNilDB(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	// Testing that methods receiving nil exec when db is nil return error rather than panicking
	_, err := repo.ListExpiredInProgressJourneys(ctx, nil, time.Now(), 10)
	if err == nil {
		t.Errorf("expected error when both exec and db are nil, got nil")
	}

	_, err = repo.GetCardByNumber(ctx, nil, "6032981012345678")
	if err == nil {
		t.Errorf("expected error when both exec and db are nil, got nil")
	}

	_, err = repo.ListStations(ctx, nil, StationFilterRecord{})
	if err == nil {
		t.Errorf("expected error when both exec and db are nil, got nil")
	}

	_, err = repo.ListLedgerTransactions(ctx, nil, "c0000001", 10)
	if err == nil {
		t.Errorf("expected error when both exec and db are nil, got nil")
	}
}
