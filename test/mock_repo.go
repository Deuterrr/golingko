package test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golingko/internal/api"
	"golingko/internal/config"
	"golingko/internal/transit"
	"golingko/pkg/logger"
	"golingko/pkg/slug"
)

type txSession struct {
	mu          sync.Mutex
	lockedCards map[string]*sync.Mutex
}

func (s *txSession) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.lockedCards {
		m.Unlock()
	}
	s.lockedCards = nil
}

func (s *txSession) lockCard(number string, lock *sync.Mutex) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.lockedCards[number]; !ok {
		lock.Lock()
		s.lockedCards[number] = lock
	}
}

// PartitionedMemoryRepository provides an in-memory transit repository with per-card row-level locking.
type PartitionedMemoryRepository struct {
	sessionsMu sync.RWMutex
	sessions   map[*sql.Tx]*txSession

	cardsMu       sync.RWMutex
	cardsByID     map[string]*transit.CardRecord
	cardsByNumber map[string]*transit.CardRecord

	cardLocksMu sync.Mutex
	cardLocks   map[string]*sync.Mutex

	stationsMu sync.RWMutex
	stations   map[string]*transit.StationRecord

	routesMu      sync.RWMutex
	routes        map[string]*transit.RouteRecord
	routeStations map[string][]transit.RouteStopRecord // key: routeID

	journeysMu      sync.RWMutex
	journeysByID    map[string]*transit.JourneyRecord
	activeByCardID  map[string]*transit.JourneyRecord
	completedByCard map[string][]*transit.JourneyRecord

	ledgerMu     sync.RWMutex
	ledgerByCard map[string][]transit.LedgerRecord
	allLedger    []transit.LedgerRecord

	idGen int64
}

// NewPartitionedMemoryRepository constructs a PartitionedMemoryRepository populated with initial transit stations.
func NewPartitionedMemoryRepository() *PartitionedMemoryRepository {
	repo := &PartitionedMemoryRepository{
		sessions:        make(map[*sql.Tx]*txSession),
		cardsByID:       make(map[string]*transit.CardRecord),
		cardsByNumber:   make(map[string]*transit.CardRecord),
		cardLocks:       make(map[string]*sync.Mutex),
		stations:        make(map[string]*transit.StationRecord),
		routes:          make(map[string]*transit.RouteRecord),
		routeStations:   make(map[string][]transit.RouteStopRecord),
		journeysByID:    make(map[string]*transit.JourneyRecord),
		activeByCardID:  make(map[string]*transit.JourneyRecord),
		completedByCard: make(map[string][]*transit.JourneyRecord),
		ledgerByCard:    make(map[string][]transit.LedgerRecord),
		allLedger:       make([]transit.LedgerRecord, 0),
	}

	latMRT_BHI, lngMRT_BHI := -6.1925, 106.8236
	latMRT_DKA, lngMRT_DKA := -6.2008, 106.8228
	latMRT_BLM, lngMRT_BLM := -6.2444, 106.7981
	latMRT_BLB, lngMRT_BLB := -6.2892, 106.7744

	latTJ_KOTA, lngTJ_KOTA := -6.1376, 106.8146
	latTJ_HRMN, lngTJ_HRMN := -6.1661, 106.8202
	latTJ_MNAS, lngTJ_MNAS := -6.1754, 106.8272
	latTJ_TOSR, lngTJ_TOSR := -6.2005, 106.8230
	latTJ_RAGN, lngTJ_RAGN := -6.3117, 106.8206

	latLRT_DKA, lngLRT_DKA := -6.2091, 106.8285
	latLRT_CWG, lngLRT_CWG := -6.2475, 106.8647
	latLRT_JAT, lngLRT_JAT := -6.2625, 107.0175
	latLRT_HAR, lngLRT_HAR := -6.3705, 106.8970

	initialStations := []transit.StationRecord{
		{ID: "MRT-BHI", Name: "Bundaran HI", Slug: slug.Generate("Bundaran HI"), Mode: "mrt", ZoneIndex: 1, City: "jakarta pusat", District: "menteng", Subdistrict: "gondangdia", Latitude: &latMRT_BHI, Longitude: &lngMRT_BHI, IsActive: true},
		{ID: "MRT-DKA", Name: "Dukuh Atas BNI", Slug: slug.Generate("Dukuh Atas BNI"), Mode: "mrt", ZoneIndex: 1, City: "jakarta pusat", District: "tanah abang", Subdistrict: "kebon melati", Latitude: &latMRT_DKA, Longitude: &lngMRT_DKA, IsActive: true},
		{ID: "MRT-BLM", Name: "Blok M BCA", Slug: slug.Generate("Blok M BCA"), Mode: "mrt", ZoneIndex: 2, City: "jakarta selatan", District: "kebayoran baru", Subdistrict: "melawai", Latitude: &latMRT_BLM, Longitude: &lngMRT_BLM, IsActive: true},
		{ID: "MRT-BLB", Name: "Lebak Bulus Grab", Slug: slug.Generate("Lebak Bulus Grab"), Mode: "mrt", ZoneIndex: 3, City: "jakarta selatan", District: "cilandak", Subdistrict: "lebak bulus", Latitude: &latMRT_BLB, Longitude: &lngMRT_BLB, IsActive: true},

		{ID: "TJ-KOTA", Name: "Kota", Slug: slug.Generate("Kota"), Mode: "transjakarta", ZoneIndex: 1, City: "jakarta barat", District: "taman sari", Subdistrict: "pinangsia", Latitude: &latTJ_KOTA, Longitude: &lngTJ_KOTA, IsActive: true},
		{ID: "TJ-HRMN", Name: "Harmoni", Slug: slug.Generate("Harmoni"), Mode: "transjakarta", ZoneIndex: 1, City: "jakarta pusat", District: "gambir", Subdistrict: "kebon kelapa", Latitude: &latTJ_HRMN, Longitude: &lngTJ_HRMN, IsActive: true},
		{ID: "TJ-MNAS", Name: "Monas", Slug: slug.Generate("Monas"), Mode: "transjakarta", ZoneIndex: 1, City: "jakarta pusat", District: "gambir", Subdistrict: "gambir", Latitude: &latTJ_MNAS, Longitude: &lngTJ_MNAS, IsActive: true},
		{ID: "TJ-TOSR", Name: "Tosari ICBC", Slug: slug.Generate("Tosari ICBC"), Mode: "transjakarta", ZoneIndex: 1, City: "jakarta pusat", District: "menteng", Subdistrict: "gondangdia", Latitude: &latTJ_TOSR, Longitude: &lngTJ_TOSR, IsActive: true},
		{ID: "TJ-RAGN", Name: "Ragunan", Slug: slug.Generate("Ragunan"), Mode: "transjakarta", ZoneIndex: 2, City: "jakarta selatan", District: "pasar minggu", Subdistrict: "ragunan", Latitude: &latTJ_RAGN, Longitude: &lngTJ_RAGN, IsActive: true},

		{ID: "LRT-DKA", Name: "Dukuh Atas LRT", Slug: slug.Generate("Dukuh Atas LRT"), Mode: "lrt", ZoneIndex: 1, City: "jakarta selatan", District: "setiabudi", Subdistrict: "setiabudi", Latitude: &latLRT_DKA, Longitude: &lngLRT_DKA, IsActive: true},
		{ID: "LRT-CWG", Name: "Cawang", Slug: slug.Generate("Cawang"), Mode: "lrt", ZoneIndex: 2, City: "jakarta timur", District: "jatinegara", Subdistrict: "cawang", Latitude: &latLRT_CWG, Longitude: &lngLRT_CWG, IsActive: true},
		{ID: "LRT-JAT", Name: "Jatimulya", Slug: slug.Generate("Jatimulya"), Mode: "lrt", ZoneIndex: 4, City: "kota bekasi", District: "bekasi timur", Subdistrict: "jatimulya", Latitude: &latLRT_JAT, Longitude: &lngLRT_JAT, IsActive: true},
		{ID: "LRT-HAR", Name: "Harjamukti", Slug: slug.Generate("Harjamukti"), Mode: "lrt", ZoneIndex: 4, City: "kota depok", District: "cimanggis", Subdistrict: "harjamukti", Latitude: &latLRT_HAR, Longitude: &lngLRT_HAR, IsActive: true},
	}
	for _, s := range initialStations {
		copied := s
		repo.stations[s.ID] = &copied
	}

	initialRoutes := []transit.RouteRecord{
		{ID: "MRT-M1", Code: "M1", Name: "MRT North-South Line (Lebak Bulus - Bundaran HI)", Slug: slug.Generate("MRT North-South Line (Lebak Bulus - Bundaran HI)"), Mode: "mrt", ColorHex: "#1e439b", IsActive: true, CreatedAt: time.Now().UTC()},
		{ID: "TJ-1", Code: "1", Name: "Koridor 1 (Blok M - Kota)", Slug: slug.Generate("Koridor 1 (Blok M - Kota)"), Mode: "transjakarta", ColorHex: "#e30613", IsActive: true, CreatedAt: time.Now().UTC()},
		{ID: "TJ-6", Code: "6", Name: "Koridor 6 (Ragunan - Galunggung)", Slug: slug.Generate("Koridor 6 (Ragunan - Galunggung)"), Mode: "transjakarta", ColorHex: "#008040", IsActive: true, CreatedAt: time.Now().UTC()},
		{ID: "LRT-BK", Code: "BK", Name: "LRT Bekasi Line (Dukuh Atas - Jatimulya)", Slug: slug.Generate("LRT Bekasi Line (Dukuh Atas - Jatimulya)"), Mode: "lrt", ColorHex: "#d4145a", IsActive: true, CreatedAt: time.Now().UTC()},
		{ID: "LRT-CB", Code: "CB", Name: "LRT Cibubur Line (Dukuh Atas - Harjamukti)", Slug: slug.Generate("LRT Cibubur Line (Dukuh Atas - Harjamukti)"), Mode: "lrt", ColorHex: "#f39200", IsActive: true, CreatedAt: time.Now().UTC()},
	}
	for _, r := range initialRoutes {
		copied := r
		repo.routes[r.ID] = &copied
	}

	repo.routeStations["MRT-M1"] = []transit.RouteStopRecord{
		{RouteID: "MRT-M1", StationID: "MRT-BLB", StationName: "Lebak Bulus Grab", StationSlug: slug.Generate("Lebak Bulus Grab"), Mode: "mrt", City: "jakarta selatan", District: "cilandak", Subdistrict: "lebak bulus", StopSequence: 1, DistanceFromOriginKm: 0.0, Direction: "bidirectional"},
		{RouteID: "MRT-M1", StationID: "MRT-BLM", StationName: "Blok M BCA", StationSlug: slug.Generate("Blok M BCA"), Mode: "mrt", City: "jakarta selatan", District: "kebayoran baru", Subdistrict: "melawai", StopSequence: 2, DistanceFromOriginKm: 6.0, Direction: "bidirectional"},
		{RouteID: "MRT-M1", StationID: "MRT-DKA", StationName: "Dukuh Atas BNI", StationSlug: slug.Generate("Dukuh Atas BNI"), Mode: "mrt", City: "jakarta pusat", District: "tanah abang", Subdistrict: "kebon melati", StopSequence: 3, DistanceFromOriginKm: 11.2, Direction: "bidirectional"},
		{RouteID: "MRT-M1", StationID: "MRT-BHI", StationName: "Bundaran HI", StationSlug: slug.Generate("Bundaran HI"), Mode: "mrt", City: "jakarta pusat", District: "menteng", Subdistrict: "gondangdia", StopSequence: 4, DistanceFromOriginKm: 15.7, Direction: "bidirectional"},
	}

	repo.routeStations["TJ-1"] = []transit.RouteStopRecord{
		{RouteID: "TJ-1", StationID: "TJ-TOSR", StationName: "Tosari ICBC", StationSlug: slug.Generate("Tosari ICBC"), Mode: "transjakarta", City: "jakarta pusat", District: "menteng", Subdistrict: "gondangdia", StopSequence: 1, DistanceFromOriginKm: 7.4, Direction: "bidirectional"},
		{RouteID: "TJ-1", StationID: "TJ-MNAS", StationName: "Monas", StationSlug: slug.Generate("Monas"), Mode: "transjakarta", City: "jakarta pusat", District: "gambir", Subdistrict: "gambir", StopSequence: 2, DistanceFromOriginKm: 10.5, Direction: "bidirectional"},
		{RouteID: "TJ-1", StationID: "TJ-HRMN", StationName: "Harmoni", StationSlug: slug.Generate("Harmoni"), Mode: "transjakarta", City: "jakarta pusat", District: "gambir", Subdistrict: "kebon kelapa", StopSequence: 3, DistanceFromOriginKm: 12.0, Direction: "bidirectional"},
		{RouteID: "TJ-1", StationID: "TJ-KOTA", StationName: "Kota", StationSlug: slug.Generate("Kota"), Mode: "transjakarta", City: "jakarta barat", District: "taman sari", Subdistrict: "pinangsia", StopSequence: 4, DistanceFromOriginKm: 15.6, Direction: "bidirectional"},
	}

	repo.routeStations["TJ-6"] = []transit.RouteStopRecord{
		{RouteID: "TJ-6", StationID: "TJ-RAGN", StationName: "Ragunan", StationSlug: slug.Generate("Ragunan"), Mode: "transjakarta", City: "jakarta selatan", District: "pasar minggu", Subdistrict: "ragunan", StopSequence: 1, DistanceFromOriginKm: 0.0, Direction: "bidirectional"},
		{RouteID: "TJ-6", StationID: "TJ-TOSR", StationName: "Tosari ICBC", StationSlug: slug.Generate("Tosari ICBC"), Mode: "transjakarta", City: "jakarta pusat", District: "menteng", Subdistrict: "gondangdia", StopSequence: 2, DistanceFromOriginKm: 13.5, Direction: "bidirectional"},
	}

	repo.routeStations["LRT-BK"] = []transit.RouteStopRecord{
		{RouteID: "LRT-BK", StationID: "LRT-DKA", StationName: "Dukuh Atas LRT", StationSlug: slug.Generate("Dukuh Atas LRT"), Mode: "lrt", City: "jakarta selatan", District: "setiabudi", Subdistrict: "setiabudi", StopSequence: 1, DistanceFromOriginKm: 0.0, Direction: "bidirectional"},
		{RouteID: "LRT-BK", StationID: "LRT-CWG", StationName: "Cawang", StationSlug: slug.Generate("Cawang"), Mode: "lrt", City: "jakarta timur", District: "jatinegara", Subdistrict: "cawang", StopSequence: 2, DistanceFromOriginKm: 8.5, Direction: "bidirectional"},
		{RouteID: "LRT-BK", StationID: "LRT-JAT", StationName: "Jatimulya", StationSlug: slug.Generate("Jatimulya"), Mode: "lrt", City: "kota bekasi", District: "bekasi timur", Subdistrict: "jatimulya", StopSequence: 3, DistanceFromOriginKm: 27.3, Direction: "bidirectional"},
	}

	repo.routeStations["LRT-CB"] = []transit.RouteStopRecord{
		{RouteID: "LRT-CB", StationID: "LRT-DKA", StationName: "Dukuh Atas LRT", StationSlug: slug.Generate("Dukuh Atas LRT"), Mode: "lrt", City: "jakarta selatan", District: "setiabudi", Subdistrict: "setiabudi", StopSequence: 1, DistanceFromOriginKm: 0.0, Direction: "bidirectional"},
		{RouteID: "LRT-CB", StationID: "LRT-CWG", StationName: "Cawang", StationSlug: slug.Generate("Cawang"), Mode: "lrt", City: "jakarta timur", District: "jatinegara", Subdistrict: "cawang", StopSequence: 2, DistanceFromOriginKm: 8.5, Direction: "bidirectional"},
		{RouteID: "LRT-CB", StationID: "LRT-HAR", StationName: "Harjamukti", StationSlug: slug.Generate("Harjamukti"), Mode: "lrt", City: "kota depok", District: "cimanggis", Subdistrict: "harjamukti", StopSequence: 3, DistanceFromOriginKm: 24.3, Direction: "bidirectional"},
	}

	return repo
}

func (r *PartitionedMemoryRepository) nextID(prefix string) string {
	id := atomic.AddInt64(&r.idGen, 1)
	return fmt.Sprintf("%s-%d", prefix, id)
}

func (r *PartitionedMemoryRepository) getCardLock(cardNumber string) *sync.Mutex {
	r.cardLocksMu.Lock()
	defer r.cardLocksMu.Unlock()
	lock, ok := r.cardLocks[cardNumber]
	if !ok {
		lock = &sync.Mutex{}
		r.cardLocks[cardNumber] = lock
	}
	return lock
}

func (r *PartitionedMemoryRepository) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx := &sql.Tx{}
	session := &txSession{
		lockedCards: make(map[string]*sync.Mutex),
	}

	r.sessionsMu.Lock()
	r.sessions[tx] = session
	r.sessionsMu.Unlock()

	defer func() {
		session.release()
		r.sessionsMu.Lock()
		delete(r.sessions, tx)
		r.sessionsMu.Unlock()
	}()

	return fn(tx)
}

func (r *PartitionedMemoryRepository) CreateCard(ctx context.Context, exec transit.DBExecutor, cardNumber string, initialBalance int64, status string) (*transit.CardRecord, error) {
	r.cardsMu.Lock()
	defer r.cardsMu.Unlock()

	if _, exists := r.cardsByNumber[cardNumber]; exists {
		return nil, errors.New("card number already exists")
	}

	id := r.nextID("card")
	now := time.Now().UTC()
	c := &transit.CardRecord{
		ID:         id,
		CardNumber: cardNumber,
		Balance:    initialBalance,
		Status:     status,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	r.cardsByID[id] = c
	r.cardsByNumber[cardNumber] = c

	copied := *c
	return &copied, nil
}

func (r *PartitionedMemoryRepository) GetCardByNumber(ctx context.Context, exec transit.DBExecutor, number string) (*transit.CardRecord, error) {
	r.cardsMu.RLock()
	defer r.cardsMu.RUnlock()

	c, ok := r.cardsByNumber[number]
	if !ok {
		return nil, transit.ErrCardNotFound
	}
	copied := *c
	return &copied, nil
}

func (r *PartitionedMemoryRepository) GetCardForUpdate(ctx context.Context, tx *sql.Tx, number string) (*transit.CardRecord, error) {
	lock := r.getCardLock(number)

	r.sessionsMu.RLock()
	session, ok := r.sessions[tx]
	r.sessionsMu.RUnlock()

	if ok {
		session.lockCard(number, lock)
	}

	r.cardsMu.RLock()
	defer r.cardsMu.RUnlock()

	c, exists := r.cardsByNumber[number]
	if !exists {
		return nil, transit.ErrCardNotFound
	}
	copied := *c
	return &copied, nil
}

func (r *PartitionedMemoryRepository) UpdateCardBalance(ctx context.Context, tx *sql.Tx, cardID string, balance int64) error {
	if balance < 0 {
		return transit.ErrNegativeBalance
	}

	r.cardsMu.Lock()
	defer r.cardsMu.Unlock()

	c, ok := r.cardsByID[cardID]
	if !ok {
		return transit.ErrCardNotFound
	}
	c.Balance = balance
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *PartitionedMemoryRepository) UpdateCardStatus(ctx context.Context, tx *sql.Tx, cardID string, status string) error {
	r.cardsMu.Lock()
	defer r.cardsMu.Unlock()

	c, ok := r.cardsByID[cardID]
	if !ok {
		return transit.ErrCardNotFound
	}
	c.Status = status
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *PartitionedMemoryRepository) ListStations(ctx context.Context, exec transit.DBExecutor, filter transit.StationFilterRecord) ([]transit.StationRecord, error) {
	r.stationsMu.RLock()
	defer r.stationsMu.RUnlock()

	var result []transit.StationRecord
	for _, s := range r.stations {
		if filter.Mode != nil && *filter.Mode != "" && !strings.EqualFold(s.Mode, *filter.Mode) {
			continue
		}
		if filter.City != nil && *filter.City != "" && !strings.EqualFold(s.City, *filter.City) {
			continue
		}
		if filter.District != nil && *filter.District != "" && !strings.EqualFold(s.District, *filter.District) {
			continue
		}
		if filter.RouteID != nil && *filter.RouteID != "" {
			r.routesMu.RLock()
			stops := r.routeStations[*filter.RouteID]
			r.routesMu.RUnlock()
			found := false
			for _, st := range stops {
				if st.StationID == s.ID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, *s)
	}
	return result, nil
}

func (r *PartitionedMemoryRepository) GetStationByID(ctx context.Context, exec transit.DBExecutor, id string) (*transit.StationRecord, error) {
	r.stationsMu.RLock()
	defer r.stationsMu.RUnlock()

	s, ok := r.stations[id]
	if !ok {
		for _, candidate := range r.stations {
			if candidate.Slug == id {
				s = candidate
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, transit.ErrStationNotFound
	}
	copied := *s
	return &copied, nil
}

func (r *PartitionedMemoryRepository) GetStationRoutes(ctx context.Context, exec transit.DBExecutor, stationID string) ([]transit.RouteRecord, error) {
	r.routesMu.RLock()
	defer r.routesMu.RUnlock()

	var result []transit.RouteRecord
	for routeID, stops := range r.routeStations {
		for _, st := range stops {
			if st.StationID == stationID {
				if route, ok := r.routes[routeID]; ok && route.IsActive {
					result = append(result, *route)
				}
				break
			}
		}
	}
	return result, nil
}

func (r *PartitionedMemoryRepository) ListRoutes(ctx context.Context, exec transit.DBExecutor, mode *string, city *string) ([]transit.RouteRecord, error) {
	r.routesMu.RLock()
	defer r.routesMu.RUnlock()

	var result []transit.RouteRecord
	for _, route := range r.routes {
		if !route.IsActive {
			continue
		}
		if mode != nil && *mode != "" && !strings.EqualFold(route.Mode, *mode) {
			continue
		}
		if city != nil && *city != "" {
			stops := r.routeStations[route.ID]
			found := false
			for _, st := range stops {
				if strings.EqualFold(st.City, *city) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, *route)
	}
	return result, nil
}

func (r *PartitionedMemoryRepository) GetRouteByID(ctx context.Context, exec transit.DBExecutor, id string) (*transit.RouteRecord, []transit.RouteStopRecord, error) {
	r.routesMu.RLock()
	defer r.routesMu.RUnlock()

	route, ok := r.routes[id]
	if !ok {
		for _, candidate := range r.routes {
			if candidate.Slug == id {
				route = candidate
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, nil, transit.ErrRouteNotFound
	}
	stops := r.routeStations[route.ID]
	copiedStops := make([]transit.RouteStopRecord, len(stops))
	copy(copiedStops, stops)

	copiedRoute := *route
	return &copiedRoute, copiedStops, nil
}

func (r *PartitionedMemoryRepository) GetConnectingRoutes(ctx context.Context, exec transit.DBExecutor, originStationID, destStationID string) ([]transit.RouteRecord, error) {
	r.routesMu.RLock()
	defer r.routesMu.RUnlock()

	var result []transit.RouteRecord
	for routeID, stops := range r.routeStations {
		hasOrigin := false
		hasDest := false
		for _, st := range stops {
			if st.StationID == originStationID {
				hasOrigin = true
			}
			if st.StationID == destStationID {
				hasDest = true
			}
		}
		if hasOrigin && hasDest {
			if route, ok := r.routes[routeID]; ok && route.IsActive {
				result = append(result, *route)
			}
		}
	}
	return result, nil
}

func (r *PartitionedMemoryRepository) IsStationOnRoute(ctx context.Context, exec transit.DBExecutor, routeID, stationID string) (bool, error) {
	r.routesMu.RLock()
	defer r.routesMu.RUnlock()

	route, ok := r.routes[routeID]
	if !ok || !route.IsActive {
		return false, nil
	}

	stops := r.routeStations[routeID]
	for _, st := range stops {
		if st.StationID == stationID {
			return true, nil
		}
	}
	return false, nil
}

func (r *PartitionedMemoryRepository) GetActiveJourneyForUpdate(ctx context.Context, tx *sql.Tx, cardID string) (*transit.JourneyRecord, error) {
	r.journeysMu.RLock()
	defer r.journeysMu.RUnlock()

	j, ok := r.activeByCardID[cardID]
	if !ok || j.Status != string(transit.StatusInProgress) {
		return nil, transit.ErrNoActiveJourney
	}
	copied := *j
	return &copied, nil
}

func (r *PartitionedMemoryRepository) CreateJourney(ctx context.Context, tx *sql.Tx, cardID, stationID string, routeID *string) (*transit.JourneyRecord, error) {
	r.journeysMu.Lock()
	defer r.journeysMu.Unlock()

	id := r.nextID("journey")
	j := &transit.JourneyRecord{
		ID:             id,
		CardID:         cardID,
		EntryStationID: stationID,
		RouteID:        routeID,
		EntryTime:      time.Now().UTC(),
		Status:         string(transit.StatusInProgress),
		CreatedAt:      time.Now().UTC(),
	}
	r.journeysByID[id] = j
	r.activeByCardID[cardID] = j

	copied := *j
	return &copied, nil
}

func (r *PartitionedMemoryRepository) CompleteJourney(ctx context.Context, tx *sql.Tx, journeyID, exitStationID string, exitTime time.Time, fare int64) error {
	r.journeysMu.Lock()
	defer r.journeysMu.Unlock()

	j, ok := r.journeysByID[journeyID]
	if !ok {
		return errors.New("journey not found")
	}
	j.ExitStationID = &exitStationID
	j.ExitTime = &exitTime
	j.FareAmount = fare
	j.Status = string(transit.StatusCompleted)

	delete(r.activeByCardID, j.CardID)
	r.completedByCard[j.CardID] = append(r.completedByCard[j.CardID], j)
	return nil
}

func (r *PartitionedMemoryRepository) ExpireJourney(ctx context.Context, tx *sql.Tx, journeyID string, penalty int64) error {
	r.journeysMu.Lock()
	defer r.journeysMu.Unlock()

	j, ok := r.journeysByID[journeyID]
	if !ok {
		return errors.New("journey not found")
	}
	now := time.Now().UTC()
	j.ExitTime = &now
	j.FareAmount = penalty
	j.Status = string(transit.StatusExpired)

	delete(r.activeByCardID, j.CardID)
	r.completedByCard[j.CardID] = append(r.completedByCard[j.CardID], j)
	return nil
}

func (r *PartitionedMemoryRepository) ListRecentCompletedJourneys(ctx context.Context, exec transit.DBExecutor, cardID string, since time.Time) ([]transit.CompletedLegDTO, error) {
	r.journeysMu.RLock()
	defer r.journeysMu.RUnlock()

	legs, ok := r.completedByCard[cardID]
	if !ok {
		return nil, nil
	}

	var result []transit.CompletedLegDTO
	for _, leg := range legs {
		if leg.ExitTime != nil && leg.ExitTime.After(since) {
			mode := "mrt"
			if st, ok := r.stations[leg.EntryStationID]; ok {
				mode = st.Mode
			}
			result = append(result, transit.CompletedLegDTO{
				Mode:       mode,
				FareAmount: leg.FareAmount,
				ExitTime:   *leg.ExitTime,
			})
		}
	}
	return result, nil
}

func (r *PartitionedMemoryRepository) ListExpiredInProgressJourneys(ctx context.Context, exec transit.DBExecutor, threshold time.Time, limit int) ([]transit.ExpiredTripDTO, error) {
	r.journeysMu.RLock()
	defer r.journeysMu.RUnlock()

	var result []transit.ExpiredTripDTO
	count := 0
	for _, j := range r.activeByCardID {
		if j.EntryTime.Before(threshold) {
			r.cardsMu.RLock()
			card, ok := r.cardsByID[j.CardID]
			balance := int64(0)
			if ok {
				balance = card.Balance
			}
			r.cardsMu.RUnlock()

			result = append(result, transit.ExpiredTripDTO{
				JourneyID: j.ID,
				CardID:    j.CardID,
				Balance:   balance,
			})
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *PartitionedMemoryRepository) RecordLedgerTransaction(ctx context.Context, tx *sql.Tx, entry transit.LedgerRecord) (*transit.LedgerRecord, error) {
	r.ledgerMu.Lock()
	defer r.ledgerMu.Unlock()

	id := r.nextID("tx")
	rec := transit.LedgerRecord{
		ID:              id,
		CardID:          entry.CardID,
		JourneyID:       entry.JourneyID,
		TransactionType: entry.TransactionType,
		Amount:          entry.Amount,
		BalanceAfter:    entry.BalanceAfter,
		IdempotencyKey:  entry.IdempotencyKey,
		CreatedAt:       time.Now().UTC(),
	}

	r.ledgerByCard[entry.CardID] = append(r.ledgerByCard[entry.CardID], rec)
	r.allLedger = append(r.allLedger, rec)

	copied := rec
	return &copied, nil
}

func (r *PartitionedMemoryRepository) ListLedgerTransactions(ctx context.Context, exec transit.DBExecutor, cardID string, limit int) ([]transit.LedgerRecord, error) {
	r.ledgerMu.RLock()
	defer r.ledgerMu.RUnlock()

	txs, ok := r.ledgerByCard[cardID]
	if !ok {
		return nil, nil
	}

	if limit <= 0 || limit > len(txs) {
		limit = len(txs)
	}

	result := make([]transit.LedgerRecord, 0, limit)
	for i := len(txs) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, txs[i])
	}
	return result, nil
}

// SetupPartitionedApp initializes an in-memory transit HTTP handler with partitioned row-level locking
// and a high-performance discard logger to maximize test execution speed.
func SetupPartitionedApp() (http.Handler, *PartitionedMemoryRepository) {
	return SetupPartitionedAppWithLogger(logger.NewNoopLogger())
}

// SetupPartitionedAppWithLogger initializes an in-memory transit HTTP handler bound to an explicit logger
// for comparative observability profiling and performance benchmarking across logging drivers.
func SetupPartitionedAppWithLogger(appLog *slog.Logger) (http.Handler, *PartitionedMemoryRepository) {
	return SetupPartitionedAppWithEngine(api.RouterEngineNetHTTP, appLog)
}

// SetupPartitionedAppWithEngine boots an in-memory transit HTTP handler bound to an explicit router engine.
// This supports comparative benchmarking across multiplexers under identical domain constraints.
func SetupPartitionedAppWithEngine(engine api.RouterEngine, appLog *slog.Logger) (http.Handler, *PartitionedMemoryRepository) {
	repo := NewPartitionedMemoryRepository()
	cfg := config.Load()

	cardSvc := transit.NewCardService(repo)
	stationSvc := transit.NewStationService(repo)
	routeSvc := transit.NewRouteService(repo)
	journeyOrch := transit.NewJourneyOrchestrator(repo, cfg.MinTapBalance)
	ledgerSvc := transit.NewLedgerService(repo)

	handler := api.NewHandler(cardSvc, stationSvc, routeSvc, journeyOrch, ledgerSvc, nil, appLog)
	router, err := api.SetupRouterWithEngine(engine, handler, appLog)
	if err != nil {
		panic(err)
	}
	return router, repo
}
