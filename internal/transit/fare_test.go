package transit

import (
	"testing"
	"time"
)

func TestCalculateFare(t *testing.T) {
	stMRT_BHI := &Station{ID: "MRT-BHI", Name: "Bundaran HI", Mode: ModeMRT, ZoneIndex: 1, IsActive: true}
	stMRT_DKA := &Station{ID: "MRT-DKA", Name: "Dukuh Atas BNI", Mode: ModeMRT, ZoneIndex: 1, IsActive: true}
	stMRT_BLB := &Station{ID: "MRT-BLB", Name: "Lebak Bulus Grab", Mode: ModeMRT, ZoneIndex: 3, IsActive: true}
	stTJ_KOTA := &Station{ID: "TJ-KOTA", Name: "Kota", Mode: ModeTJ, ZoneIndex: 1, IsActive: true}
	stTJ_MNAS := &Station{ID: "TJ-MNAS", Name: "Monas", Mode: ModeTJ, ZoneIndex: 1, IsActive: true}
	stMikro1 := &Station{ID: "MIKRO-01", Name: "Mikrotrans 01", Mode: ModeMikrotrans, ZoneIndex: 1, IsActive: true}
	stMikro2 := &Station{ID: "MIKRO-02", Name: "Mikrotrans 02", Mode: ModeMikrotrans, ZoneIndex: 1, IsActive: true}

	t.Run("TransJakarta flat fare", func(t *testing.T) {
		res := CalculateFare(FareCalculationParams{
			Origin:      stTJ_KOTA,
			Destination: stTJ_MNAS,
			Duration:    25 * time.Minute,
		})
		if res.FareAmount != FlatFareTJ {
			t.Fatalf("expected TJ fare %d, got %d", FlatFareTJ, res.FareAmount)
		}
	})

	t.Run("Mikrotrans zero fare", func(t *testing.T) {
		res := CalculateFare(FareCalculationParams{
			Origin:      stMikro1,
			Destination: stMikro2,
			Duration:    10 * time.Minute,
		})
		if res.FareAmount != 0 {
			t.Fatalf("expected Mikrotrans fare 0, got %d", res.FareAmount)
		}
	})

	t.Run("MRT distance tariff", func(t *testing.T) {
		// Zone 1 to Zone 3: 2 zone delta => 3000 + 2000 = 5000
		res := CalculateFare(FareCalculationParams{
			Origin:      stMRT_BHI,
			Destination: stMRT_BLB,
			Duration:    30 * time.Minute,
		})
		expected := int64(5000)
		if res.FareAmount != expected {
			t.Fatalf("expected MRT fare %d, got %d", expected, res.FareAmount)
		}
	})

	t.Run("Same station turnaround within grace period (< 3m)", func(t *testing.T) {
		res := CalculateFare(FareCalculationParams{
			Origin:      stMRT_BHI,
			Destination: stMRT_BHI,
			Duration:    2 * time.Minute,
		})
		if res.FareAmount != 0 {
			t.Fatalf("expected Rp 0 for grace turnaround, got %d", res.FareAmount)
		}
	})

	t.Run("Same station turnaround after platform timeout (> 15m)", func(t *testing.T) {
		res := CalculateFare(FareCalculationParams{
			Origin:      stMRT_BHI,
			Destination: stMRT_BHI,
			Duration:    20 * time.Minute,
		})
		if res.FareAmount != PlatformAccessFee {
			t.Fatalf("expected Rp %d platform fee, got %d", PlatformAccessFee, res.FareAmount)
		}
	})

	t.Run("JakLingko multi-modal transfer cap (180m / Rp 10,000 cap)", func(t *testing.T) {
		// Passenger already paid Rp 7,000 in TransJakarta/earlier legs within 60 minutes
		res := CalculateFare(FareCalculationParams{
			Origin:             stMRT_DKA,
			Destination:        stMRT_BLB,
			Duration:           25 * time.Minute,
			PriorFaresInWindow: 7000,
			PriorModes:         []TransitMode{ModeTJ},
			WindowElapsed:      60 * time.Minute,
		})

		if !res.IsIntegrated {
			t.Errorf("expected trip to be integrated")
		}
		// Normal integrated leg fare might be e.g. 2500 + 10km*250 = 5000,
		// but remaining allowance is 10000 - 7000 = 3000!
		if res.FareAmount > 3000 {
			t.Fatalf("expected fare capped at Rp 3,000, got %d", res.FareAmount)
		}
		if !res.CapApplied {
			t.Errorf("expected CapApplied to be true")
		}
	})

	t.Run("JakLingko transfer when cap already reached", func(t *testing.T) {
		// Passenger already reached Rp 10,000 cap in previous legs within 100 minutes
		res := CalculateFare(FareCalculationParams{
			Origin:             stMRT_DKA,
			Destination:        stMRT_BHI,
			Duration:           10 * time.Minute,
			PriorFaresInWindow: 10000,
			PriorModes:         []TransitMode{ModeTJ},
			WindowElapsed:      100 * time.Minute,
		})

		if res.FareAmount != 0 {
			t.Fatalf("expected Rp 0 fare after reaching cap, got %d", res.FareAmount)
		}
		if !res.CapApplied {
			t.Errorf("expected CapApplied to be true")
		}
	})
}
