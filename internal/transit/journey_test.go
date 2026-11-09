package transit

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

type mockRepository struct {
	card           *CardRecord
	originStation  *StationRecord
	destStation    *StationRecord
	activeJourney  *JourneyRecord
	recentLegs     []CompletedLegDTO
	createdJourney *JourneyRecord
	completedFare  int64
	ledgerTx       *LedgerRecord

	route        *RouteRecord
	routeStops   []RouteStopRecord
	stationRoute map[string][]string // routeID -> stationIDs
}

func (m *mockRepository) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return fn(nil)
}

func (m *mockRepository) CreateCard(ctx context.Context, exec DBExecutor, cardNumber string, initialBalance int64, status string) (*CardRecord, error) {
	return nil, nil
}

func (m *mockRepository) GetCardByNumber(ctx context.Context, exec DBExecutor, number string) (*CardRecord, error) {
	return m.card, nil
}

func (m *mockRepository) GetCardForUpdate(ctx context.Context, tx *sql.Tx, number string) (*CardRecord, error) {
	return m.card, nil
}

func (m *mockRepository) UpdateCardBalance(ctx context.Context, tx *sql.Tx, cardID string, balance int64) error {
	if m.card != nil {
		m.card.Balance = balance
	}
	return nil
}

func (m *mockRepository) UpdateCardStatus(ctx context.Context, tx *sql.Tx, cardID string, status string) error {
	if m.card != nil {
		m.card.Status = status
	}
	return nil
}

func (m *mockRepository) ListStations(ctx context.Context, exec DBExecutor, filter StationFilterRecord) ([]StationRecord, error) {
	return []StationRecord{*m.originStation, *m.destStation}, nil
}

func (m *mockRepository) GetStationByID(ctx context.Context, exec DBExecutor, id string) (*StationRecord, error) {
	if m.originStation != nil && m.originStation.ID == id {
		return m.originStation, nil
	}
	if m.destStation != nil && m.destStation.ID == id {
		return m.destStation, nil
	}
	return nil, ErrStationNotFound
}

func (m *mockRepository) GetStationRoutes(ctx context.Context, exec DBExecutor, stationID string) ([]RouteRecord, error) {
	if m.route != nil {
		return []RouteRecord{*m.route}, nil
	}
	return nil, nil
}

func (m *mockRepository) ListRoutes(ctx context.Context, exec DBExecutor, mode *string, city *string) ([]RouteRecord, error) {
	if m.route != nil {
		return []RouteRecord{*m.route}, nil
	}
	return nil, nil
}

func (m *mockRepository) GetRouteByID(ctx context.Context, exec DBExecutor, id string) (*RouteRecord, []RouteStopRecord, error) {
	if m.route != nil && m.route.ID == id {
		return m.route, m.routeStops, nil
	}
	return nil, nil, ErrRouteNotFound
}

func (m *mockRepository) GetConnectingRoutes(ctx context.Context, exec DBExecutor, originStationID, destStationID string) ([]RouteRecord, error) {
	if m.route != nil && m.route.IsActive {
		return []RouteRecord{*m.route}, nil
	}
	return nil, nil
}

func (m *mockRepository) IsStationOnRoute(ctx context.Context, exec DBExecutor, routeID, stationID string) (bool, error) {
	if m.stationRoute != nil {
		stations, ok := m.stationRoute[routeID]
		if ok {
			for _, s := range stations {
				if s == stationID {
					return true, nil
				}
			}
			return false, nil
		}
	}
	return true, nil
}

func (m *mockRepository) GetActiveJourneyForUpdate(ctx context.Context, tx *sql.Tx, cardID string) (*JourneyRecord, error) {
	if m.activeJourney == nil {
		return nil, ErrNoActiveJourney
	}
	return m.activeJourney, nil
}

func (m *mockRepository) CreateJourney(ctx context.Context, tx *sql.Tx, cardID, stationID string, routeID *string) (*JourneyRecord, error) {
	m.createdJourney = &JourneyRecord{
		ID:             "journey-123",
		CardID:         cardID,
		EntryStationID: stationID,
		RouteID:        routeID,
		EntryTime:      time.Now().UTC(),
		Status:         string(StatusInProgress),
		CreatedAt:      time.Now().UTC(),
	}
	return m.createdJourney, nil
}

func (m *mockRepository) CompleteJourney(ctx context.Context, tx *sql.Tx, journeyID, exitStationID string, exitTime time.Time, fare int64) error {
	m.completedFare = fare
	return nil
}

func (m *mockRepository) ExpireJourney(ctx context.Context, tx *sql.Tx, journeyID string, penalty int64) error {
	return nil
}

func (m *mockRepository) ListRecentCompletedJourneys(ctx context.Context, exec DBExecutor, cardID string, since time.Time) ([]CompletedLegDTO, error) {
	return m.recentLegs, nil
}

func (m *mockRepository) ListExpiredInProgressJourneys(ctx context.Context, exec DBExecutor, threshold time.Time, limit int) ([]ExpiredTripDTO, error) {
	return nil, nil
}

func (m *mockRepository) RecordLedgerTransaction(ctx context.Context, tx *sql.Tx, entry LedgerRecord) (*LedgerRecord, error) {
	m.ledgerTx = &entry
	return &entry, nil
}

func (m *mockRepository) ListLedgerTransactions(ctx context.Context, exec DBExecutor, cardID string, limit int) ([]LedgerRecord, error) {
	return nil, nil
}

func TestJourneyOrchestrator_TapIn(t *testing.T) {
	t.Run("Rejects entry when balance is below threshold", func(t *testing.T) {
		mock := &mockRepository{
			card: &CardRecord{
				ID:         "c1",
				CardNumber: "1000111122223333",
				Balance:    2000,
				Status:     "active",
			},
			originStation: &StationRecord{
				ID:       "MRT-DKA",
				Name:     "Dukuh Atas BNI",
				Mode:     "mrt",
				IsActive: true,
			},
		}

		orch := NewJourneyOrchestrator(mock, 5000)
		_, err := orch.TapIn(context.Background(), mock.card.CardNumber, "MRT-DKA", nil)
		if err != ErrInsufficientBalance {
			t.Fatalf("expected ErrInsufficientBalance, got %v", err)
		}
	})

	t.Run("Authorizes entry when card is active with sufficient balance", func(t *testing.T) {
		mock := &mockRepository{
			card: &CardRecord{
				ID:         "c1",
				CardNumber: "1000111122223333",
				Balance:    50000,
				Status:     "active",
			},
			originStation: &StationRecord{
				ID:       "MRT-DKA",
				Name:     "Dukuh Atas BNI",
				Mode:     "mrt",
				IsActive: true,
			},
		}

		orch := NewJourneyOrchestrator(mock, 5000)
		trip, err := orch.TapIn(context.Background(), mock.card.CardNumber, "MRT-DKA", nil)
		if err != nil {
			t.Fatalf("expected successful tap in, got error %v", err)
		}
		if trip.Status != StatusInProgress {
			t.Errorf("expected trip status IN_PROGRESS, got %s", trip.Status)
		}
	})

	t.Run("Validates route at entry and records routeID", func(t *testing.T) {
		mock := &mockRepository{
			card: &CardRecord{
				ID:         "c1",
				CardNumber: "1000111122223333",
				Balance:    50000,
				Status:     "active",
			},
			originStation: &StationRecord{
				ID:       "MRT-DKA",
				Name:     "Dukuh Atas BNI",
				Mode:     "mrt",
				IsActive: true,
			},
			route: &RouteRecord{
				ID:       "MRT-M1",
				Code:     "M1",
				IsActive: true,
			},
			stationRoute: map[string][]string{
				"MRT-M1": {"MRT-DKA", "MRT-BLB"},
			},
		}

		orch := NewJourneyOrchestrator(mock, 5000)
		routeID := "MRT-M1"
		trip, err := orch.TapIn(context.Background(), mock.card.CardNumber, "MRT-DKA", &routeID)
		if err != nil {
			t.Fatalf("expected successful tap in with route, got %v", err)
		}
		if trip.RouteID == nil || *trip.RouteID != "MRT-M1" {
			t.Errorf("expected route ID MRT-M1 recorded, got %v", trip.RouteID)
		}
	})

	t.Run("Rejects entry when station does not belong to specified route", func(t *testing.T) {
		mock := &mockRepository{
			card: &CardRecord{
				ID:         "c1",
				CardNumber: "1000111122223333",
				Balance:    50000,
				Status:     "active",
			},
			originStation: &StationRecord{
				ID:       "MRT-DKA",
				Name:     "Dukuh Atas BNI",
				Mode:     "mrt",
				IsActive: true,
			},
			route: &RouteRecord{
				ID:       "TJ-1",
				Code:     "1",
				IsActive: true,
			},
			stationRoute: map[string][]string{
				"TJ-1": {"TJ-KOTA", "TJ-MNAS"},
			},
		}

		orch := NewJourneyOrchestrator(mock, 5000)
		routeID := "TJ-1"
		_, err := orch.TapIn(context.Background(), mock.card.CardNumber, "MRT-DKA", &routeID)
		if err != ErrStationNotInRoute {
			t.Fatalf("expected ErrStationNotInRoute, got %v", err)
		}
	})
}

func TestJourneyOrchestrator_TapOut(t *testing.T) {
	t.Run("Settles journey and deducts fare correctly", func(t *testing.T) {
		mock := &mockRepository{
			card: &CardRecord{
				ID:         "c1",
				CardNumber: "1000111122223333",
				Balance:    50000,
				Status:     "active",
			},
			originStation: &StationRecord{
				ID:        "TJ-KOTA",
				Name:      "Kota",
				Mode:      "transjakarta",
				ZoneIndex: 1,
				IsActive:  true,
			},
			destStation: &StationRecord{
				ID:        "TJ-MNAS",
				Name:      "Monas",
				Mode:      "transjakarta",
				ZoneIndex: 1,
				IsActive:  true,
			},
			activeJourney: &JourneyRecord{
				ID:             "j1",
				CardID:         "c1",
				EntryStationID: "TJ-KOTA",
				EntryTime:      time.Now().UTC().Add(-20 * time.Minute),
				Status:         "in_progress",
			},
			route: &RouteRecord{
				ID:       "TJ-1",
				Code:     "1",
				Mode:     "transjakarta",
				IsActive: true,
			},
		}

		orch := NewJourneyOrchestrator(mock, 5000)
		result, err := orch.TapOut(context.Background(), mock.card.CardNumber, "TJ-MNAS", false)
		if err != nil {
			t.Fatalf("expected successful tap out, got %v", err)
		}

		if !result.GateOpened {
			t.Errorf("expected gate to open")
		}
		if result.Fare.FareAmount != FlatFareTJ {
			t.Errorf("expected fare %d, got %d", FlatFareTJ, result.Fare.FareAmount)
		}
		expectedBalance := int64(50000 - FlatFareTJ)
		if result.RemainingBalance != expectedBalance {
			t.Errorf("expected remaining balance %d, got %d", expectedBalance, result.RemainingBalance)
		}
	})

	t.Run("Rejects exit when no connecting route exists", func(t *testing.T) {
		mock := &mockRepository{
			card: &CardRecord{
				ID:         "c1",
				CardNumber: "1000111122223333",
				Balance:    50000,
				Status:     "active",
			},
			originStation: &StationRecord{
				ID:        "MRT-BLB",
				Name:      "Lebak Bulus Grab",
				Mode:      "mrt",
				ZoneIndex: 3,
				IsActive:  true,
			},
			destStation: &StationRecord{
				ID:        "LRT-JAT",
				Name:      "Jatimulya",
				Mode:      "lrt",
				ZoneIndex: 4,
				IsActive:  true,
			},
			activeJourney: &JourneyRecord{
				ID:             "j1",
				CardID:         "c1",
				EntryStationID: "MRT-BLB",
				EntryTime:      time.Now().UTC().Add(-20 * time.Minute),
				Status:         "in_progress",
			},
			route: nil, // No connecting route
		}

		orch := NewJourneyOrchestrator(mock, 5000)
		_, err := orch.TapOut(context.Background(), mock.card.CardNumber, "LRT-JAT", false)
		if err != ErrNoConnectingRoute {
			t.Fatalf("expected ErrNoConnectingRoute, got %v", err)
		}
	})
}
