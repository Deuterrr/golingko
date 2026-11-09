package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"golingko/internal/transit"
)

func newTestRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(nil, nil, nil, nil, nil, nil, logger)
	return SetupRouter(handler, logger)
}

func TestHealthCheck(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}

	if resp.Database != "disconnected" {
		t.Errorf("expected database 'disconnected' when db is nil, got '%s'", resp.Database)
	}
}

func TestHandler_ServiceUnavailable(t *testing.T) {
	router := newTestRouter()

	t.Run("CreateCard returns 503 when service is nil", func(t *testing.T) {
		body := bytes.NewBufferString(`{"initial_balance": 50000}`)
		req := httptest.NewRequest(http.MethodPost, "/api/cards", body)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", rr.Code)
		}
	})

	t.Run("TapIn returns 503 when orchestrator is nil", func(t *testing.T) {
		body := bytes.NewBufferString(`{"card_number": "1000123456789012", "station_id": "MRT-DKA"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-in", body)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", rr.Code)
		}
	})

	t.Run("TapOut returns 503 when orchestrator is nil", func(t *testing.T) {
		body := bytes.NewBufferString(`{"card_number": "1000123456789012", "station_id": "MRT-BHI"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-out", body)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", rr.Code)
		}
	})

	t.Run("DocsRedirect returns 308 permanent redirect", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusPermanentRedirect {
			t.Errorf("expected status 308, got %d", rr.Code)
		}
	})
}

type mockStationService struct {
	transit.StationService
	capturedMode *transit.TransitMode
}

func (m *mockStationService) ListStations(ctx context.Context, filter transit.StationFilter) ([]transit.Station, error) {
	m.capturedMode = filter.Mode
	return []transit.Station{
		{ID: "MRT-DKA", Name: "Dukuh Atas BNI", Mode: transit.ModeMRT, ZoneIndex: 1, IsActive: true},
	}, nil
}

func TestHandler_ListStations_CaseInsensitive(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockStations := &mockStationService{}
	handler := NewHandler(nil, mockStations, nil, nil, nil, nil, logger)
	router := SetupRouter(handler, logger)

	cases := []struct {
		urlQuery     string
		expectedMode transit.TransitMode
	}{
		{urlQuery: "/api/stations?mode=mrt", expectedMode: transit.ModeMRT},
		{urlQuery: "/api/stations?mode=Mrt", expectedMode: transit.ModeMRT},
		{urlQuery: "/api/stations?mode=transjakarta", expectedMode: transit.ModeTJ},
		{urlQuery: "/api/stations?mode=TransJakarta", expectedMode: transit.ModeTJ},
		{urlQuery: "/api/stations?mode=mikrotrans", expectedMode: transit.ModeMikrotrans},
		{urlQuery: "/api/stations?mode=lrt", expectedMode: transit.ModeLRT},
	}

	for _, tc := range cases {
		t.Run(tc.urlQuery, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.urlQuery, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rr.Code)
			}

			if mockStations.capturedMode == nil {
				t.Fatalf("expected capturedMode to be non-nil")
			}

			if *mockStations.capturedMode != tc.expectedMode {
				t.Errorf("for query %s, expected mode %v, got %v", tc.urlQuery, tc.expectedMode, *mockStations.capturedMode)
			}
		})
	}
}

type mockRouteService struct {
	transit.RouteService
	routes []transit.Route
	stops  []transit.RouteStop
}

func (m *mockRouteService) ListRoutes(ctx context.Context, filter transit.RouteFilter) ([]transit.Route, error) {
	return m.routes, nil
}

func (m *mockRouteService) GetRoute(ctx context.Context, id string) (*transit.Route, []transit.RouteStop, error) {
	for _, r := range m.routes {
		if r.ID == id {
			return &r, m.stops, nil
		}
	}
	return nil, nil, transit.ErrRouteNotFound
}

func (m *mockRouteService) CheckConnectivity(ctx context.Context, originStationID, destStationID string) (*transit.RouteConnectivity, error) {
	return &transit.RouteConnectivity{
		IsDirect:        true,
		HopCount:        2,
		DistanceDeltaKm: 11.2,
	}, nil
}

func TestHandler_RouteEndpoints(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockRoutes := &mockRouteService{
		routes: []transit.Route{
			{ID: "MRT-M1", Code: "M1", Name: "MRT North-South", Mode: transit.ModeMRT, IsActive: true},
		},
		stops: []transit.RouteStop{
			{RouteID: "MRT-M1", StationID: "MRT-BLB", StopSequence: 1, DistanceFromOriginKm: 0.0},
			{RouteID: "MRT-M1", StationID: "MRT-DKA", StopSequence: 2, DistanceFromOriginKm: 11.2},
		},
	}
	handler := NewHandler(nil, nil, mockRoutes, nil, nil, nil, logger)
	router := SetupRouter(handler, logger)

	t.Run("GET /api/routes returns route array", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/routes", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
		var list []transit.Route
		if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
			t.Fatalf("failed to decode routes: %v", err)
		}
		if len(list) != 1 || list[0].ID != "MRT-M1" {
			t.Errorf("unexpected routes response: %+v", list)
		}
	})

	t.Run("GET /api/routes/{id} returns route and stops", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/routes/MRT-M1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
		var resp struct {
			Route transit.Route       `json:"route"`
			Stops []transit.RouteStop `json:"stops"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode route response: %v", err)
		}
		if resp.Route.ID != "MRT-M1" || len(resp.Stops) != 2 {
			t.Errorf("unexpected route response: %+v", resp)
		}
	})

	t.Run("GET /api/routes/connectivity checks reachability", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/routes/connectivity?origin=MRT-BLB&destination=MRT-DKA", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
		var conn transit.RouteConnectivity
		if err := json.Unmarshal(rr.Body.Bytes(), &conn); err != nil {
			t.Fatalf("failed to decode connectivity response: %v", err)
		}
		if !conn.IsDirect || conn.HopCount != 2 {
			t.Errorf("unexpected connectivity: %+v", conn)
		}
	})
}
