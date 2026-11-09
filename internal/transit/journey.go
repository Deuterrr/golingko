package transit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"
)

// Lifecycle progression of an active or concluded transit commute.
type JourneyStatus string

const (
	// Commuter checked in via turnstile entry and is currently in transit.
	// Tap-out at an exit gate is pending to conclude the journey and calculate the fare.
	StatusInProgress JourneyStatus = "in_progress"
	// Commuter successfully passed through the destination turnstile.
	// The journey is concluded and the final travel tariff has been settled from the card.
	StatusCompleted JourneyStatus = "completed"
	// Overdue journey that exceeded the maximum permitted transit duration without an exit tap.
	// A penalty fee has been assessed to preserve station turnstile and trip integrity.
	StatusPenaltySettled JourneyStatus = "penalty_settled"
	// Journey flagged as timed out before penalty deduction processing begins.
	// It marks an unclosed commute leg awaiting background sweeper reconciliation.
	StatusExpired JourneyStatus = "expired"
)

// Commute leg between transit network turnstiles.
type Journey struct {
	ID             string     `json:"id"`
	CardID         string     `json:"card_id"`
	EntryStationID string     `json:"entry_station_id"`
	ExitStationID  *string    `json:"exit_station_id,omitempty"` // Destination station; nil while in_progress.
	RouteID        *string    `json:"route_id,omitempty"`        // Associated route line or vehicle run.
	EntryTime      time.Time  `json:"entry_time"`
	ExitTime       *time.Time `json:"exit_time,omitempty"`

	// Tariff in IDR (Rp) assessed for this trip leg (0 until settled at tap-out).
	FareAmount int64 `json:"fare_amount"`

	// Active or final settlement lifecycle status.
	Status JourneyStatus `json:"status"`

	CreatedAt time.Time `json:"created_at"`
}

type TapOutResult struct {
	Journey *Journey   `json:"journey"`
	Fare    FareResult `json:"fare"`
	// Remaining card balance in IDR (Rp) post-deduction.
	RemainingBalance int64 `json:"remaining_balance"`
	// True if the gate turnstile unlocked; false if insufficient balance barred exit.
	GateOpened bool `json:"gate_opened"`
}

var (
	ErrInsufficientBalance        = errors.New("insufficient balance to enter transit gate")
	ErrAlreadyInJourney           = errors.New("card already has an in-progress journey")
	ErrNoActiveJourney            = errors.New("no active journey found for card")
	ErrInsufficientBalanceForExit = errors.New("insufficient balance for exit fare: top up required at gate")
)

const (
	MaxJourneyDuration             = 4 * time.Hour
	AbandonedTripPenaltyFare int64 = 10000
)

// Journey lifecycle.
type JourneyOrchestrator interface {
	// Grants a commuter entry through a station turnstile, verifying travel eligibility and starting a new journey leg.
	// It ensures the card holds the required entry balance and clears overdue trips before unlocking the gate.
	TapIn(ctx context.Context, cardNumber string, stationID string, routeID *string) (*Journey, error)

	// Concludes a commuter's journey at an exit turnstile, calculating applicable transit fares and settling payment from the card.
	// It applies integrated multi-modal discounts and authorizes passenger exit upon successful payment.
	TapOut(ctx context.Context, cardNumber string, stationID string, allowOverdraft bool) (*TapOutResult, error)
}

type journeyOrchestrator struct {
	repo          TransitRepository
	minTapBalance int64
}

func NewJourneyOrchestrator(repo TransitRepository, minTapBalance int64) JourneyOrchestrator {
	return &journeyOrchestrator{
		repo:          repo,
		minTapBalance: minTapBalance,
	}
}

func (o *journeyOrchestrator) TapIn(ctx context.Context, cardNumber string, stationID string, routeID *string) (*Journey, error) {
	var createdJourney *Journey

	err := o.repo.WithTx(ctx, func(tx *sql.Tx) error {
		cardRecord, err := o.repo.GetCardForUpdate(ctx, tx, cardNumber)
		if err != nil {
			return err
		}

		if cardRecord.Status != string(CardStatusActive) {
			return ErrCardInactive
		}

		if cardRecord.Balance < o.minTapBalance {
			return ErrInsufficientBalance
		}

		stationRecord, err := o.repo.GetStationByID(ctx, tx, stationID)
		if err != nil {
			return err
		}
		if !stationRecord.IsActive {
			return ErrStationInactive
		}

		if routeID != nil && *routeID != "" {
			routeRecord, _, err := o.repo.GetRouteByID(ctx, tx, *routeID)
			if err != nil {
				return err
			}
			if !routeRecord.IsActive {
				return ErrRouteInactive
			}
			onRoute, err := o.repo.IsStationOnRoute(ctx, tx, *routeID, stationID)
			if err != nil {
				return err
			}
			if !onRoute {
				return ErrStationNotInRoute
			}
		}

		activeJourney, err := o.repo.GetActiveJourneyForUpdate(ctx, tx, cardRecord.ID)
		if err == nil {
			if time.Since(activeJourney.EntryTime) > MaxJourneyDuration {
				if err := o.repo.ExpireJourney(ctx, tx, activeJourney.ID, AbandonedTripPenaltyFare); err != nil {
					return err
				}

				penaltyDeduction := AbandonedTripPenaltyFare
				if cardRecord.Balance < penaltyDeduction {
					penaltyDeduction = cardRecord.Balance
				}
				newBalance := cardRecord.Balance - penaltyDeduction
				if err := o.repo.UpdateCardBalance(ctx, tx, cardRecord.ID, newBalance); err != nil {
					return err
				}

				_, _ = o.repo.RecordLedgerTransaction(ctx, tx, LedgerRecord{
					CardID:          cardRecord.ID,
					JourneyID:       &activeJourney.ID,
					TransactionType: string(TxTypePenaltyFee),
					Amount:          -penaltyDeduction,
					BalanceAfter:    newBalance,
				})
			} else {
				return ErrAlreadyInJourney
			}
		} else if !errors.Is(err, ErrNoActiveJourney) {
			return err
		}

		newTrip, err := o.repo.CreateJourney(ctx, tx, cardRecord.ID, stationID, routeID)
		if err != nil {
			return err
		}

		createdJourney = toDomainJourney(newTrip)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdJourney, nil
}

func (o *journeyOrchestrator) TapOut(ctx context.Context, cardNumber string, stationID string, allowOverdraft bool) (*TapOutResult, error) {
	var result *TapOutResult

	err := o.repo.WithTx(ctx, func(tx *sql.Tx) error {
		cardRecord, err := o.repo.GetCardForUpdate(ctx, tx, cardNumber)
		if err != nil {
			return err
		}

		activeJourney, err := o.repo.GetActiveJourneyForUpdate(ctx, tx, cardRecord.ID)
		if err != nil {
			return err
		}

		originRecord, err := o.repo.GetStationByID(ctx, tx, activeJourney.EntryStationID)
		if err != nil {
			return err
		}
		destRecord, err := o.repo.GetStationByID(ctx, tx, stationID)
		if err != nil {
			return err
		}

		var routeDistKm *float64
		if activeJourney.EntryStationID != stationID {
			if activeJourney.RouteID != nil && *activeJourney.RouteID != "" {
				onRoute, err := o.repo.IsStationOnRoute(ctx, tx, *activeJourney.RouteID, stationID)
				if err != nil {
					return err
				}
				if !onRoute {
					return ErrStationNotInRoute
				}
				_, stops, err := o.repo.GetRouteByID(ctx, tx, *activeJourney.RouteID)
				if err == nil {
					var originDist, destDist float64
					for _, st := range stops {
						if st.StationID == activeJourney.EntryStationID {
							originDist = st.DistanceFromOriginKm
						}
						if st.StationID == stationID {
							destDist = st.DistanceFromOriginKm
						}
					}
					delta := math.Abs(destDist - originDist)
					if delta > 0 {
						routeDistKm = &delta
					}
				}
			} else {
				connecting, err := o.repo.GetConnectingRoutes(ctx, tx, activeJourney.EntryStationID, stationID)
				if err != nil {
					return err
				}
				if len(connecting) == 0 {
					return ErrNoConnectingRoute
				}
				_, stops, err := o.repo.GetRouteByID(ctx, tx, connecting[0].ID)
				if err == nil {
					var originDist, destDist float64
					for _, st := range stops {
						if st.StationID == activeJourney.EntryStationID {
							originDist = st.DistanceFromOriginKm
						}
						if st.StationID == stationID {
							destDist = st.DistanceFromOriginKm
						}
					}
					delta := math.Abs(destDist - originDist)
					if delta > 0 {
						routeDistKm = &delta
					}
				}
			}
		}

		now := time.Now().UTC()
		duration := now.Sub(activeJourney.EntryTime)

		windowStart := now.Add(-TransferWindowDuration)
		recentLegs, err := o.repo.ListRecentCompletedJourneys(ctx, tx, cardRecord.ID, windowStart)
		if err != nil {
			return fmt.Errorf("failed to query transfer window trips: %w", err)
		}

		var priorModes []TransitMode
		var priorFares int64
		for _, leg := range recentLegs {
			priorModes = append(priorModes, TransitMode(leg.Mode))
			priorFares += leg.FareAmount
		}

		originStation := &Station{
			ID:          originRecord.ID,
			Name:        originRecord.Name,
			Slug:        originRecord.Slug,
			Mode:        TransitMode(originRecord.Mode),
			ZoneIndex:   originRecord.ZoneIndex,
			City:        originRecord.City,
			District:    originRecord.District,
			Subdistrict: originRecord.Subdistrict,
			Latitude:    originRecord.Latitude,
			Longitude:   originRecord.Longitude,
			IsActive:    originRecord.IsActive,
		}
		destStation := &Station{
			ID:          destRecord.ID,
			Name:        destRecord.Name,
			Slug:        destRecord.Slug,
			Mode:        TransitMode(destRecord.Mode),
			ZoneIndex:   destRecord.ZoneIndex,
			City:        destRecord.City,
			District:    destRecord.District,
			Subdistrict: destRecord.Subdistrict,
			Latitude:    destRecord.Latitude,
			Longitude:   destRecord.Longitude,
			IsActive:    destRecord.IsActive,
		}

		fareCalc := CalculateFare(FareCalculationParams{
			Origin:             originStation,
			Destination:        destStation,
			Duration:           duration,
			PriorFaresInWindow: priorFares,
			PriorModes:         priorModes,
			WindowElapsed:      duration,
			RouteDistanceKm:    routeDistKm,
		})

		// Dense multi-variable context: validate balance against fare, apply
		// overdraft policy, update card status, and settle journey & ledger atomically.
		if cardRecord.Balance < fareCalc.FareAmount {
			if !allowOverdraft {
				return ErrInsufficientBalanceForExit
			}
			cardRecord.Status = string(CardStatusBlocked)
			_ = o.repo.UpdateCardStatus(ctx, tx, cardRecord.ID, cardRecord.Status)
		}

		newBalance := cardRecord.Balance - fareCalc.FareAmount
		if err := o.repo.UpdateCardBalance(ctx, tx, cardRecord.ID, newBalance); err != nil {
			return err
		}

		if err := o.repo.CompleteJourney(ctx, tx, activeJourney.ID, destStation.ID, now, fareCalc.FareAmount); err != nil {
			return err
		}

		_, err = o.repo.RecordLedgerTransaction(ctx, tx, LedgerRecord{
			CardID:          cardRecord.ID,
			JourneyID:       &activeJourney.ID,
			TransactionType: string(TxTypeFareDeduction),
			Amount:          -fareCalc.FareAmount,
			BalanceAfter:    newBalance,
		})
		if err != nil {
			return err
		}

		exitTime := now
		activeJourney.ExitStationID = &destStation.ID
		activeJourney.ExitTime = &exitTime
		activeJourney.FareAmount = fareCalc.FareAmount
		activeJourney.Status = string(StatusCompleted)

		result = &TapOutResult{
			Journey:          toDomainJourney(activeJourney),
			Fare:             fareCalc,
			RemainingBalance: newBalance,
			GateOpened:       true,
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func toDomainJourney(r *JourneyRecord) *Journey {
	if r == nil {
		return nil
	}
	return &Journey{
		ID:             r.ID,
		CardID:         r.CardID,
		EntryStationID: r.EntryStationID,
		ExitStationID:  r.ExitStationID,
		RouteID:        r.RouteID,
		EntryTime:      r.EntryTime,
		ExitTime:       r.ExitTime,
		FareAmount:     r.FareAmount,
		Status:         JourneyStatus(r.Status),
		CreatedAt:      r.CreatedAt,
	}
}
