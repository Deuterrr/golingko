package transit

import (
	"context"
	"testing"
	"time"
)

type routeMockRepo struct {
	TransitRepository
	routes        []RouteRecord
	routeStops    map[string][]RouteStopRecord
	stationRoutes map[string][]RouteRecord
}

func (m *routeMockRepo) ListRoutes(ctx context.Context, exec DBExecutor, mode *string, city *string) ([]RouteRecord, error) {
	var result []RouteRecord
	for _, r := range m.routes {
		if mode != nil && *mode != "" && r.Mode != *mode {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

func (m *routeMockRepo) GetRouteByID(ctx context.Context, exec DBExecutor, id string) (*RouteRecord, []RouteStopRecord, error) {
	for _, r := range m.routes {
		if r.ID == id || r.Slug == id {
			return &r, m.routeStops[r.ID], nil
		}
	}
	return nil, nil, ErrRouteNotFound
}

func (m *routeMockRepo) GetConnectingRoutes(ctx context.Context, exec DBExecutor, originStationID, destStationID string) ([]RouteRecord, error) {
	key := originStationID + "->" + destStationID
	return m.stationRoutes[key], nil
}

func (m *routeMockRepo) IsStationOnRoute(ctx context.Context, exec DBExecutor, routeID, stationID string) (bool, error) {
	stops := m.routeStops[routeID]
	for _, st := range stops {
		if st.StationID == stationID {
			return true, nil
		}
	}
	return false, nil
}

func TestRouteService_ListRoutes(t *testing.T) {
	mock := &routeMockRepo{
		routes: []RouteRecord{
			{ID: "MRT-M1", Code: "M1", Name: "MRT North-South", Slug: "mrt-north-south", Mode: "mrt", IsActive: true, CreatedAt: time.Now()},
			{ID: "TJ-1", Code: "1", Name: "Koridor 1", Slug: "koridor-1", Mode: "transjakarta", IsActive: true, CreatedAt: time.Now()},
		},
	}
	svc := NewRouteService(mock)

	t.Run("Returns all active routes", func(t *testing.T) {
		routes, err := svc.ListRoutes(context.Background(), RouteFilter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(routes) != 2 {
			t.Fatalf("expected 2 routes, got %d", len(routes))
		}
	})

	t.Run("Filters by mode case-insensitively", func(t *testing.T) {
		mode := ModeMRT
		routes, err := svc.ListRoutes(context.Background(), RouteFilter{Mode: &mode})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(routes) != 1 || routes[0].ID != "MRT-M1" {
			t.Fatalf("expected MRT-M1, got %+v", routes)
		}
	})
}

func TestRouteService_GetRoute(t *testing.T) {
	mock := &routeMockRepo{
		routes: []RouteRecord{
			{ID: "MRT-M1", Code: "M1", Name: "MRT North-South", Slug: "mrt-north-south", Mode: "mrt", IsActive: true, CreatedAt: time.Now()},
			{ID: "MRT-INACTIVE", Code: "INACT", Name: "Under Maintenance", Slug: "under-maintenance", Mode: "mrt", IsActive: false, CreatedAt: time.Now()},
		},
		routeStops: map[string][]RouteStopRecord{
			"MRT-M1": {
				{RouteID: "MRT-M1", StationID: "MRT-BLB", StationName: "Lebak Bulus", StationSlug: "lebak-bulus", Mode: "mrt", StopSequence: 1, DistanceFromOriginKm: 0.0},
				{RouteID: "MRT-M1", StationID: "MRT-DKA", StationName: "Dukuh Atas BNI", StationSlug: "dukuh-atas-bni", Mode: "mrt", StopSequence: 2, DistanceFromOriginKm: 11.2},
			},
		},
	}
	svc := NewRouteService(mock)

	t.Run("Retrieves route with ordered stops", func(t *testing.T) {
		route, stops, err := svc.GetRoute(context.Background(), "MRT-M1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if route.ID != "MRT-M1" {
			t.Errorf("expected route MRT-M1, got %s", route.ID)
		}
		if len(stops) != 2 {
			t.Fatalf("expected 2 stops, got %d", len(stops))
		}
		if stops[0].StationID != "MRT-BLB" || stops[1].StationID != "MRT-DKA" {
			t.Errorf("unexpected stop order: %+v", stops)
		}
	})

	t.Run("Returns ErrRouteInactive for suspended line", func(t *testing.T) {
		_, _, err := svc.GetRoute(context.Background(), "MRT-INACTIVE")
		if err != ErrRouteInactive {
			t.Errorf("expected ErrRouteInactive, got %v", err)
		}
	})

	t.Run("Returns ErrRouteNotFound for missing route", func(t *testing.T) {
		_, _, err := svc.GetRoute(context.Background(), "NONEXISTENT")
		if err != ErrRouteNotFound {
			t.Errorf("expected ErrRouteNotFound, got %v", err)
		}
	})
}

func TestRouteService_CheckConnectivity(t *testing.T) {
	mock := &routeMockRepo{
		routes: []RouteRecord{
			{ID: "MRT-M1", Code: "M1", Name: "MRT North-South", Slug: "mrt-north-south", Mode: "mrt", IsActive: true, CreatedAt: time.Now()},
		},
		routeStops: map[string][]RouteStopRecord{
			"MRT-M1": {
				{RouteID: "MRT-M1", StationID: "MRT-BLB", StopSequence: 1, DistanceFromOriginKm: 0.0},
				{RouteID: "MRT-M1", StationID: "MRT-BLM", StopSequence: 2, DistanceFromOriginKm: 6.0},
				{RouteID: "MRT-M1", StationID: "MRT-DKA", StopSequence: 3, DistanceFromOriginKm: 11.2},
			},
		},
		stationRoutes: map[string][]RouteRecord{
			"MRT-BLB->MRT-DKA": {
				{ID: "MRT-M1", Code: "M1", Name: "MRT North-South", Slug: "mrt-north-south", Mode: "mrt", IsActive: true},
			},
		},
	}
	svc := NewRouteService(mock)

	t.Run("Identifies direct route with hop count and distance delta", func(t *testing.T) {
		conn, err := svc.CheckConnectivity(context.Background(), "MRT-BLB", "MRT-DKA")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !conn.IsDirect {
			t.Fatalf("expected direct connectivity")
		}
		if conn.HopCount != 2 {
			t.Errorf("expected hop count 2, got %d", conn.HopCount)
		}
		if conn.DistanceDeltaKm != 11.2 {
			t.Errorf("expected distance delta 11.2 km, got %f", conn.DistanceDeltaKm)
		}
	})

	t.Run("Returns IsDirect false when no route connects stops", func(t *testing.T) {
		conn, err := svc.CheckConnectivity(context.Background(), "MRT-BLB", "LRT-JAT")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if conn.IsDirect {
			t.Errorf("expected disconnected result for unconnected stations")
		}
	})
}
