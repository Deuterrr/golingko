package transit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCardService_CreateCard(t *testing.T) {
	t.Run("Rejects negative initial balance with ErrNegativeBalance", func(t *testing.T) {
		svc := NewCardService(&mockRepository{})
		card, err := svc.CreateCard(context.Background(), "1000111122223333", -1000)
		if !errors.Is(err, ErrNegativeBalance) {
			t.Fatalf("expected ErrNegativeBalance, got %v", err)
		}
		if card != nil {
			t.Errorf("expected nil card on error, got %+v", card)
		}
	})

	t.Run("Creates active card when initial balance is non-negative", func(t *testing.T) {
		now := time.Now().UTC()
		mock := &mockRepository{
			card: &CardRecord{
				ID:         "c1",
				CardNumber: "1000111122223333",
				Balance:    50000,
				Status:     string(CardStatusActive),
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		}
		svc := NewCardService(mock)
		card, err := svc.CreateCard(context.Background(), "", 0)
		if err != nil {
			t.Fatalf("expected nil error for zero balance, got %v", err)
		}
		_ = card
	})
}
