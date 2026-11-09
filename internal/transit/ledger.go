package transit

import (
	"context"
	"time"
)

// Classifies financial events.
type TransactionType string

const (
	// Records funds added to a card balance through an authorized payment channel.
	// This increases the commuter's available balance for travel across transit operators.
	TxTypeTopUp TransactionType = "top_up"
	// Deducts the assessed journey fare from the card balance at an exit turnstile.
	// This settles travel fees for completed trip legs, including applicable transfer discounts.
	TxTypeFareDeduction TransactionType = "fare_deduction"
	// Applies a penalty deduction when a commuter exceeds the maximum permitted journey duration.
	// This resolves abandoned or expired transit trips to preserve station turnstile and trip integrity.
	TxTypePenaltyFee TransactionType = "penalty_fee"
	// Restores credit to a card balance following a billing adjustment or payment dispute.
	// This reimburses commuters for erroneous fare deductions or cancelled transit services.
	TxTypeRefund TransactionType = "refund"
)

// Transaction history for a transit card.
type LedgerTransaction struct {
	// Unique identifier.
	ID        string  `json:"id"`
	CardID    string  `json:"card_id"`              // Pointing to the tapped transit card.
	JourneyID *string `json:"journey_id,omitempty"` // Associated journey leg

	// Category of financial movement.
	TransactionType TransactionType `json:"transaction_type"`

	Amount int64 `json:"amount"` // in IDR (Rp).
	// Verifiable snapshot of the card balance immediately following this mutation.
	BalanceAfter int64 `json:"balance_after"`

	// Optional unique client token.
	IdempotencyKey *string `json:"idempotency_key,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// Financial audit and transaction history tracking
type LedgerService interface {
	// Retrieves the transaction statement history for a commuter's transit card.
	// It presents a chronological summary of balance top-ups, trip fares, penalties, and adjustments for account auditing.
	ListTransactions(ctx context.Context, cardID string, limit int) ([]LedgerTransaction, error)
}

type ledgerService struct {
	repo TransitRepository
}

func NewLedgerService(repo TransitRepository) LedgerService {
	return &ledgerService{repo: repo}
}

func (s *ledgerService) ListTransactions(ctx context.Context, cardID string, limit int) ([]LedgerTransaction, error) {
	records, err := s.repo.ListLedgerTransactions(ctx, nil, cardID, limit)
	if err != nil {
		return nil, err
	}

	transactions := make([]LedgerTransaction, len(records))
	for i, r := range records {
		transactions[i] = *toDomainLedger(&r)
	}
	return transactions, nil
}

func toDomainLedger(r *LedgerRecord) *LedgerTransaction {
	if r == nil {
		return nil
	}
	return &LedgerTransaction{
		ID:              r.ID,
		CardID:          r.CardID,
		JourneyID:       r.JourneyID,
		TransactionType: TransactionType(r.TransactionType),
		Amount:          r.Amount,
		BalanceAfter:    r.BalanceAfter,
		IdempotencyKey:  r.IdempotencyKey,
		CreatedAt:       r.CreatedAt,
	}
}
