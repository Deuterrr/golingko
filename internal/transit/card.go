package transit

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"golingko/pkg/randutil"
)

// Lifecycle state of a transit card account.
type CardStatus string

const (
	// Normal operational state allowing commuter travel across all transit networks.
	// Cards in this state are fully eligible for gate turnstiles and balance top-ups.
	CardStatusActive CardStatus = "active"
	// Temporarily frozen state enacted by transit operators or account holders.
	// Gate turnstile passage is blocked while account balance remains preserved.
	CardStatusSuspended CardStatus = "suspended"
	// Permanently revoked state resulting from reported theft, fraud, or card retirement.
	// Cards in this state cannot be used for transit access or balance replenishment.
	CardStatusBlocked CardStatus = "blocked"
)

// Commuter's transit smart card.
type Card struct {
	ID         string     `json:"id"`
	CardNumber string     `json:"card_number"`
	Balance    int64      `json:"balance"` // in IDR (Rp).
	Status     CardStatus `json:"status"`  // Lifecycle state (active, suspended, blocked).

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrCardNotFound       = errors.New("transit card not found")
	ErrCardInactive       = errors.New("transit card is not active")
	ErrInsufficientAmount = errors.New("amount must be greater than zero")
	ErrNegativeBalance    = errors.New("card balance cannot be negative")
)

// Transit card account lifecycles.
type CardService interface {
	// Issues a new transit smart card to a commuter, establishing their travel account with an initial balance.
	// It makes the card immediately active and ready for station turnstile check-in.
	CreateCard(ctx context.Context, cardNumber string, initialBalance int64) (*Card, error)

	// Looks up an existing transit card account by its primary account number.
	GetCard(ctx context.Context, cardNumber string) (*Card, error)

	// Adds funds to a commuter's transit card balance through an authorized payment channel.
	TopUp(ctx context.Context, cardNumber string, amount int64, idempotencyKey *string) (*Card, *LedgerTransaction, error)
}

type cardService struct {
	repo TransitRepository
}

func NewCardService(repo TransitRepository) CardService {
	return &cardService{repo: repo}
}

func (s *cardService) CreateCard(ctx context.Context, cardNumber string, initialBalance int64) (*Card, error) {
	if cardNumber == "" {
		generated, err := GenerateCardNumber()
		if err != nil {
			return nil, err
		}
		cardNumber = generated
	}

	record, err := s.repo.CreateCard(ctx, nil, cardNumber, initialBalance, string(CardStatusActive))
	if err != nil {
		return nil, err
	}

	return toDomainCard(record), nil
}

func (s *cardService) GetCard(ctx context.Context, cardNumber string) (*Card, error) {
	record, err := s.repo.GetCardByNumber(ctx, nil, cardNumber)
	if err != nil {
		return nil, err
	}
	return toDomainCard(record), nil
}

func (s *cardService) TopUp(ctx context.Context, cardNumber string, amount int64, idempotencyKey *string) (*Card, *LedgerTransaction, error) {
	if amount <= 0 {
		return nil, nil, ErrInsufficientAmount
	}

	var updatedCard *Card
	var recordedTx *LedgerTransaction

	err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
		cardRecord, err := s.repo.GetCardForUpdate(ctx, tx, cardNumber)
		if err != nil {
			return err
		}

		if cardRecord.Status != string(CardStatusActive) {
			return ErrCardInactive
		}

		newBalance := cardRecord.Balance + amount
		if err := s.repo.UpdateCardBalance(ctx, tx, cardRecord.ID, newBalance); err != nil {
			return err
		}

		ledgerEntry := LedgerRecord{
			CardID:          cardRecord.ID,
			TransactionType: string(TxTypeTopUp),
			Amount:          amount,
			BalanceAfter:    newBalance,
			IdempotencyKey:  idempotencyKey,
		}

		rec, err := s.repo.RecordLedgerTransaction(ctx, tx, ledgerEntry)
		if err != nil {
			return err
		}

		cardRecord.Balance = newBalance
		updatedCard = toDomainCard(cardRecord)
		recordedTx = toDomainLedger(rec)
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return updatedCard, recordedTx, nil
}

// Generates a unique 16-digit primary account number.
func GenerateCardNumber() (string, error) {
	const prefix = "1000"
	suffix, err := randutil.Numeric(12)
	if err != nil {
		return "", err
	}
	return prefix + suffix, nil
}

func toDomainCard(r *CardRecord) *Card {
	if r == nil {
		return nil
	}
	return &Card{
		ID:         r.ID,
		CardNumber: r.CardNumber,
		Balance:    r.Balance,
		Status:     CardStatus(r.Status),
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}
