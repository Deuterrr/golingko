package transit

import (
	"context"
	"errors"
	"math"
	"strings"
)

// Public transit operator mode in the integrated JakLingko network.
// It determines the service vehicle type and governs the applicable tariff scheme.
type TransitMode string

const (
	// Jakarta Mass Rapid Transit heavy rail service operating along high-density corridors.
	// Fares are calculated using a base platform fee combined with distance and zone increments.
	ModeMRT TransitMode = "mrt"
	// Light Rail Transit system providing urban feeder connections across municipal zones.
	// Fares use flat base rates combined with fixed distance step increments.
	ModeLRT TransitMode = "lrt"
	// TransJakarta Bus Rapid Transit corridor and feeder bus network.
	// Fares operate on a flat corridor rate across the metropolitan transit system.
	ModeTJ TransitMode = "transjakarta"
	// First and last mile neighborhood feeder minivan network.
	// Operates under a fully subsidized zero-tariff fare for registered transit commuters.
	ModeMikrotrans TransitMode = "mikrotrans"
)

// Physical station or multimodal interchange terminal.
type Station struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"` // Canonical URL-friendly slug identifier.

	// Transit operator mode operating this stop.
	Mode TransitMode `json:"mode"`

	// Geographical distance index along transit lines used for zone tariffs.
	ZoneIndex int `json:"zone_index"`

	City        string   `json:"city"`
	District    string   `json:"district"`
	Subdistrict string   `json:"subdistrict"`
	Latitude    *float64 `json:"latitude,omitempty"`  // WGS84 latitude coordinate.
	Longitude   *float64 `json:"longitude,omitempty"` // WGS84 longitude coordinate.

	IsActive bool `json:"is_active"`
}

// Filter criteria for querying the transit station registry.
type StationFilter struct {
	Mode     *TransitMode
	City     *string
	District *string
	RouteID  *string
}

var (
	ErrStationNotFound = errors.New("transit station not found")
	ErrStationInactive = errors.New("transit station is inactive")
)

type StationService interface {
	// Retrieves available transit stations across the network matching optional query filters.
	// It assists commuters in route planning and helps turnstiles verify line stops.
	ListStations(ctx context.Context, filter StationFilter) ([]Station, error)

	// Looks up details for a specific transit stop by its unique terminal code.
	// It verifies station operational readiness and platform availability for passenger entry.
	GetStation(ctx context.Context, id string) (*Station, error)
}

type stationService struct {
	repo TransitRepository
}

func NewStationService(repo TransitRepository) StationService {
	return &stationService{repo: repo}
}

func (s *stationService) ListStations(ctx context.Context, filter StationFilter) ([]Station, error) {
	var repoFilter StationFilterRecord
	if filter.Mode != nil && *filter.Mode != "" {
		str := strings.ToLower(strings.TrimSpace(string(*filter.Mode)))
		repoFilter.Mode = &str
	}
	if filter.City != nil && *filter.City != "" {
		str := strings.ToLower(strings.TrimSpace(*filter.City))
		repoFilter.City = &str
	}
	if filter.District != nil && *filter.District != "" {
		str := strings.ToLower(strings.TrimSpace(*filter.District))
		repoFilter.District = &str
	}
	if filter.RouteID != nil && *filter.RouteID != "" {
		str := strings.TrimSpace(*filter.RouteID)
		repoFilter.RouteID = &str
	}

	records, err := s.repo.ListStations(ctx, nil, repoFilter)
	if err != nil {
		return nil, err
	}

	stations := make([]Station, len(records))
	for i, r := range records {
		stations[i] = Station{
			ID:          r.ID,
			Name:        r.Name,
			Slug:        r.Slug,
			Mode:        TransitMode(r.Mode),
			ZoneIndex:   r.ZoneIndex,
			City:        r.City,
			District:    r.District,
			Subdistrict: r.Subdistrict,
			Latitude:    r.Latitude,
			Longitude:   r.Longitude,
			IsActive:    r.IsActive,
		}
	}
	return stations, nil
}

func (s *stationService) GetStation(ctx context.Context, id string) (*Station, error) {
	record, err := s.repo.GetStationByID(ctx, nil, id)
	if err != nil {
		return nil, err
	}
	if !record.IsActive {
		return nil, ErrStationInactive
	}

	return &Station{
		ID:          record.ID,
		Name:        record.Name,
		Slug:        record.Slug,
		Mode:        TransitMode(record.Mode),
		ZoneIndex:   record.ZoneIndex,
		City:        record.City,
		District:    record.District,
		Subdistrict: record.Subdistrict,
		Latitude:    record.Latitude,
		Longitude:   record.Longitude,
		IsActive:    record.IsActive,
	}, nil
}

// Estimates the corridor travel distance in kilometers between two transit stations.
// It calculates the distance from zone index deltas, applying a baseline for same-zone trips.
func EstimateDistanceKm(origin, destination *Station) float64 {
	zoneDelta := math.Abs(float64(origin.ZoneIndex - destination.ZoneIndex))
	if zoneDelta == 0 {
		return 3.0
	}
	return zoneDelta * 5.0
}
