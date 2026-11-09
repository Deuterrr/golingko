package transit

import (
	"context"
	"testing"
)

type stationTestRepo struct {
	TransitRepository
	capturedMode *string
}

func (m *stationTestRepo) ListStations(ctx context.Context, exec DBExecutor, filter StationFilterRecord) ([]StationRecord, error) {
	m.capturedMode = filter.Mode
	return []StationRecord{
		{ID: "MRT-DKA", Name: "Dukuh Atas BNI", Slug: "dukuh-atas-bni", Mode: "mrt", ZoneIndex: 1, IsActive: true},
	}, nil
}

func TestStationService_ListStations_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{input: "mrt", expected: "mrt"},
		{input: "Mrt", expected: "mrt"},
		{input: "MRT", expected: "mrt"},
		{input: "transjakarta", expected: "transjakarta"},
		{input: "TransJakarta", expected: "transjakarta"},
		{input: "mikrotrans", expected: "mikrotrans"},
		{input: "lrt", expected: "lrt"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			repo := &stationTestRepo{}
			svc := NewStationService(repo)

			mode := TransitMode(tc.input)
			stations, err := svc.ListStations(context.Background(), StationFilter{Mode: &mode})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(stations) != 1 {
				t.Fatalf("expected 1 station, got %d", len(stations))
			}

			if repo.capturedMode == nil {
				t.Fatalf("expected capturedMode to be non-nil")
			}

			if *repo.capturedMode != tc.expected {
				t.Errorf("for input %q, expected repo mode %q, got %q", tc.input, tc.expected, *repo.capturedMode)
			}
		})
	}
}
