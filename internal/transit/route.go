package transit

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
)

var (
	ErrRouteNotFound     = errors.New("transit route not found")
	ErrRouteInactive     = errors.New("transit route is inactive")
	ErrStationNotInRoute = errors.New("station is not part of the specified transit route")
	ErrNoConnectingRoute = errors.New("no connecting transit route between origin and destination")
)

// Scheduled transit corridor or feeder line.
type Route struct {
	ID        string      `json:"id"`
	Code      string      `json:"code"` // Operational line code (e.g. M1, 1).
	Name      string      `json:"name"` // Full corridor description.
	Slug      string      `json:"slug"` // Canonical URL-friendly slug identifier.
	Mode      TransitMode `json:"mode"`
	ColorHex  string      `json:"color_hex"` // Visual line rendering color.
	IsActive  bool        `json:"is_active"`
	CreatedAt time.Time   `json:"created_at"`
}

// Ordered station stop along a transit route.
type RouteStop struct {
	RouteID              string      `json:"route_id"`
	StationID            string      `json:"station_id"`
	StationName          string      `json:"station_name"`
	StationSlug          string      `json:"station_slug"` // Canonical station slug identifier.
	Mode                 TransitMode `json:"mode"`
	City                 string      `json:"city"`
	District             string      `json:"district"`
	Subdistrict          string      `json:"subdistrict"`
	StopSequence         int         `json:"stop_sequence"`           // Order index from route origin.
	DistanceFromOriginKm float64     `json:"distance_from_origin_km"` // Cumulative corridor distance in km.
}

// Route reachability assessment result.
type RouteConnectivity struct {
	IsDirect        bool    `json:"is_direct"`
	ConnectingRoute *Route  `json:"connecting_route,omitempty"`
	HopCount        int     `json:"hop_count"`         // Station count between origin and destination.
	DistanceDeltaKm float64 `json:"distance_delta_km"` // Estimated route distance between stops.
}

// Filter parameters for route directory queries.
type RouteFilter struct {
	Mode *TransitMode
	City *string
}

// Transit line route topology and stop sequence management.
type RouteService interface {
	// Retrieves transit routes across the network matching optional operator or municipal filters.
	// It assists commuters in exploring transit corridors and helps operators review line coverage.
	ListRoutes(ctx context.Context, filter RouteFilter) ([]Route, error)

	// Looks up detailed route specifications along with its ordered sequence of transit stops.
	// It supplies line pathing data to clients and verifies corridor station connectivity.
	GetRoute(ctx context.Context, id string) (*Route, []RouteStop, error)

	// Evaluates whether a direct transit corridor connects the origin and destination stations.
	// It verifies station stop sequences and checks corridor reachability before passenger gate clearing.
	CheckConnectivity(ctx context.Context, originStationID, destStationID string) (*RouteConnectivity, error)

	// Verifies if a station stop belongs to a specific transit route.
	// It ensures onboard vehicle turnstiles only accept check-ins along registered corridors.
	ValidateStationOnRoute(ctx context.Context, routeID, stationID string) error
}

type routeService struct {
	repo TransitRepository
}

func NewRouteService(repo TransitRepository) RouteService {
	return &routeService{repo: repo}
}

func (s *routeService) ListRoutes(ctx context.Context, filter RouteFilter) ([]Route, error) {
	var modeStr *string
	if filter.Mode != nil && *filter.Mode != "" {
		str := strings.ToLower(strings.TrimSpace(string(*filter.Mode)))
		modeStr = &str
	}
	var cityStr *string
	if filter.City != nil && *filter.City != "" {
		str := strings.ToLower(strings.TrimSpace(*filter.City))
		cityStr = &str
	}

	records, err := s.repo.ListRoutes(ctx, nil, modeStr, cityStr)
	if err != nil {
		return nil, err
	}

	routes := make([]Route, len(records))
	for i, r := range records {
		routes[i] = Route{
			ID:        r.ID,
			Code:      r.Code,
			Name:      r.Name,
			Slug:      r.Slug,
			Mode:      TransitMode(r.Mode),
			ColorHex:  r.ColorHex,
			IsActive:  r.IsActive,
			CreatedAt: r.CreatedAt,
		}
	}
	return routes, nil
}

func (s *routeService) GetRoute(ctx context.Context, id string) (*Route, []RouteStop, error) {
	record, stopRecords, err := s.repo.GetRouteByID(ctx, nil, id)
	if err != nil {
		return nil, nil, err
	}
	if !record.IsActive {
		return nil, nil, ErrRouteInactive
	}

	route := &Route{
		ID:        record.ID,
		Code:      record.Code,
		Name:      record.Name,
		Slug:      record.Slug,
		Mode:      TransitMode(record.Mode),
		ColorHex:  record.ColorHex,
		IsActive:  record.IsActive,
		CreatedAt: record.CreatedAt,
	}

	stops := make([]RouteStop, len(stopRecords))
	for i, st := range stopRecords {
		stops[i] = RouteStop{
			RouteID:              st.RouteID,
			StationID:            st.StationID,
			StationName:          st.StationName,
			StationSlug:          st.StationSlug,
			Mode:                 TransitMode(st.Mode),
			City:                 st.City,
			District:             st.District,
			Subdistrict:          st.Subdistrict,
			StopSequence:         st.StopSequence,
			DistanceFromOriginKm: st.DistanceFromOriginKm,
		}
	}

	return route, stops, nil
}

func (s *routeService) CheckConnectivity(ctx context.Context, originStationID, destStationID string) (*RouteConnectivity, error) {
	if originStationID == destStationID {
		return &RouteConnectivity{
			IsDirect:        true,
			HopCount:        0,
			DistanceDeltaKm: 0.0,
		}, nil
	}

	connecting, err := s.repo.GetConnectingRoutes(ctx, nil, originStationID, destStationID)
	if err != nil {
		return nil, err
	}
	if len(connecting) == 0 {
		return &RouteConnectivity{
			IsDirect: false,
		}, nil
	}

	chosen := connecting[0]
	route := &Route{
		ID:        chosen.ID,
		Code:      chosen.Code,
		Name:      chosen.Name,
		Slug:      chosen.Slug,
		Mode:      TransitMode(chosen.Mode),
		ColorHex:  chosen.ColorHex,
		IsActive:  chosen.IsActive,
		CreatedAt: chosen.CreatedAt,
	}

	_, stops, err := s.GetRoute(ctx, chosen.ID)
	if err != nil {
		return &RouteConnectivity{
			IsDirect:        true,
			ConnectingRoute: route,
		}, nil
	}

	var originSeq, destSeq int
	var originDist, destDist float64
	for _, st := range stops {
		if st.StationID == originStationID {
			originSeq = st.StopSequence
			originDist = st.DistanceFromOriginKm
		}
		if st.StationID == destStationID {
			destSeq = st.StopSequence
			destDist = st.DistanceFromOriginKm
		}
	}

	hopCount := int(math.Abs(float64(destSeq - originSeq)))
	distDelta := math.Abs(destDist - originDist)

	return &RouteConnectivity{
		IsDirect:        true,
		ConnectingRoute: route,
		HopCount:        hopCount,
		DistanceDeltaKm: distDelta,
	}, nil
}

func (s *routeService) ValidateStationOnRoute(ctx context.Context, routeID, stationID string) error {
	record, _, err := s.repo.GetRouteByID(ctx, nil, routeID)
	if err != nil {
		return err
	}
	if !record.IsActive {
		return ErrRouteInactive
	}

	onRoute, err := s.repo.IsStationOnRoute(ctx, nil, routeID, stationID)
	if err != nil {
		return err
	}
	if !onRoute {
		return ErrStationNotInRoute
	}
	return nil
}
