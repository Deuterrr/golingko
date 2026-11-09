package worker

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"golingko/internal/transit"
)

// Background lifecycle monitor that periodically resolves abandoned commuter journeys.
type JourneySweeper interface {
	// Initiates the periodic background routine scanning for overdue in-progress trips.
	// It runs on a scheduled interval to process abandoned journeys without blocking incoming requests.
	Start()

	// Halts the periodic background scanning daemon and releases timer resources.
	// It guarantees graceful shutdown when the application server terminates.
	Stop()

	// Executes a single reconciliation pass identifying and closing expired journeys.
	// It assesses penalty deductions for trips exceeding the travel horizon and returns the processed count.
	SweepOnce(ctx context.Context) (int, error)
}

type sweeper struct {
	repo          transit.TransitRepository
	interval      time.Duration
	expiryTimeout time.Duration
	logger        *slog.Logger
	stopChan      chan struct{}
}

func NewSweeper(repo transit.TransitRepository, interval, expiryTimeout time.Duration, logger *slog.Logger) JourneySweeper {
	return &sweeper{
		repo:          repo,
		interval:      interval,
		expiryTimeout: expiryTimeout,
		logger:        logger,
		stopChan:      make(chan struct{}),
	}
}

func (s *sweeper) Start() {
	if s.repo == nil {
		s.logger.Info("Sweeper skipped: repository is detached")
		return
	}

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.logger.Info("Journey sweeper started",
			"interval", s.interval.String(),
			"expiry_timeout", s.expiryTimeout.String(),
		)

		for {
			select {
			case <-s.stopChan:
				s.logger.Info("Journey sweeper stopped")
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				swept, err := s.SweepOnce(ctx)
				cancel()
				if err != nil {
					s.logger.Error("Failed to sweep orphaned journeys", "error", err)
				} else if swept > 0 {
					s.logger.Info("Swept expired orphaned journeys", "count", swept)
				}
			}
		}
	}()
}

func (s *sweeper) Stop() {
	select {
	case <-s.stopChan:
		return
	default:
		close(s.stopChan)
	}
}

func (s *sweeper) SweepOnce(ctx context.Context) (int, error) {
	if s.repo == nil {
		return 0, nil
	}

	threshold := time.Now().UTC().Add(-s.expiryTimeout)
	trips, err := s.repo.ListExpiredInProgressJourneys(ctx, nil, threshold, 50)
	if err != nil {
		return 0, err
	}

	count := 0
	const penaltyFee int64 = 10000

	for _, t := range trips {
		err := s.repo.WithTx(ctx, func(tx *sql.Tx) error {
			if err := s.repo.ExpireJourney(ctx, tx, t.JourneyID, penaltyFee); err != nil {
				return err
			}

			deduction := penaltyFee
			if t.Balance < deduction {
				deduction = t.Balance
			}
			newBalance := t.Balance - deduction

			if err := s.repo.UpdateCardBalance(ctx, tx, t.CardID, newBalance); err != nil {
				return err
			}

			journeyID := t.JourneyID
			_, err = s.repo.RecordLedgerTransaction(ctx, tx, transit.LedgerRecord{
				CardID:          t.CardID,
				JourneyID:       &journeyID,
				TransactionType: string(transit.TxTypePenaltyFee),
				Amount:          -deduction,
				BalanceAfter:    newBalance,
			})
			return err
		})

		if err == nil {
			count++
		}
	}

	return count, nil
}
