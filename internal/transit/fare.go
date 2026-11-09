package transit

import (
	"time"
)

// Operational parameters for computing journey fares.
type FareCalculationParams struct {
	// Departure station entity including mode and geographical zone index.
	Origin *Station

	// Arrival station entity including mode and geographical zone index.
	Destination *Station

	// Elapsed time between tap-in and tap-out on this specific trip leg.
	Duration time.Duration

	// Cumulative sum of IDR fares paid across prior legs inside the active 180-minute window.
	PriorFaresInWindow int64

	// Chronological slice of transit modes used in prior legs within the transfer session.
	PriorModes []TransitMode

	// Total duration elapsed since the initial tap-in of the first trip in the transfer window.
	WindowElapsed time.Duration

	// Optional accurate route distance in km between stations.
	RouteDistanceKm *float64
}

// Tariff assessment outcome for a journey leg.
type FareResult struct {
	// Net tariff in IDR (Rp) owed for this leg after applying base rates, distance zones, or caps.
	FareAmount int64 `json:"fare_amount"`

	// True if the trip qualified for JakLingko multi-modal transfer integration.
	IsIntegrated bool `json:"is_integrated"`

	// True if the net tariff was truncated by the Rp 10,000 maximum integrated transfer ceiling.
	CapApplied bool `json:"cap_applied"`

	// Human-readable narrative detailing tariff rules, discounts, or penalty conditions applied.
	Description string `json:"description"`
}

const (
	MaxIntegratedFareCap    int64 = 10000
	TransferWindowDuration        = 180 * time.Minute
	GracePeriodDuration           = 3 * time.Minute
	PlatformTimeoutDuration       = 15 * time.Minute
	PlatformAccessFee       int64 = 3000

	FlatFareTJ     int64 = 3500
	BaseFareMRT    int64 = 3000
	PerZoneRateMRT int64 = 1000
	MaxFareMRT     int64 = 14000
	BaseFareLRT    int64 = 5000
)

// Evaluates travel parameters against integrated transit pricing rules to determine trip tariffs.
// It applies transfer discounts within eligible windows and enforces operator fare ceilings.
func CalculateFare(params FareCalculationParams) FareResult {
	if params.Origin.ID == params.Destination.ID {
		if params.Duration <= GracePeriodDuration {
			return FareResult{
				FareAmount:  0,
				Description: "accidental tap grace period: cancelled with rp 0 charge",
			}
		}
		if params.Duration > PlatformTimeoutDuration {
			return FareResult{
				FareAmount:  PlatformAccessFee,
				Description: "same-station turnaround after platform timeout: platform fee applied",
			}
		}
		return FareResult{
			FareAmount:  BaseFareMRT,
			Description: "same-station exit",
		}
	}

	distKm := EstimateDistanceKm(params.Origin, params.Destination)
	if params.RouteDistanceKm != nil && *params.RouteDistanceKm > 0 {
		distKm = *params.RouteDistanceKm
	}

	isMultiModal := false
	if len(params.PriorModes) > 0 && params.WindowElapsed <= TransferWindowDuration {
		for _, m := range params.PriorModes {
			if m != params.Origin.Mode {
				isMultiModal = true
				break
			}
		}
	}

	if isMultiModal {
		// Multi-modal transfer within 180-minute window:
		// Base Rp 2,500 + Rp 250/km, strictly capped at Rp 10,000 cumulative ceiling.
		legFare := int64(2500 + (distKm * 250))
		remainingAllowance := MaxIntegratedFareCap - params.PriorFaresInWindow

		if remainingAllowance <= 0 {
			return FareResult{
				FareAmount:   0,
				IsIntegrated: true,
				CapApplied:   true,
				Description:  "jaklingko 180m transfer window: rp 10,000 cap reached, free transfer",
			}
		}

		if legFare > remainingAllowance {
			return FareResult{
				FareAmount:   remainingAllowance,
				IsIntegrated: true,
				CapApplied:   true,
				Description:  "jaklingko integrated tariff: rp 10,000 cap enforced",
			}
		}

		return FareResult{
			FareAmount:   legFare,
			IsIntegrated: true,
			Description:  "jaklingko multi-modal integrated fare",
		}
	}

	switch params.Origin.Mode {
	case ModeTJ:
		return FareResult{
			FareAmount:  FlatFareTJ,
			Description: "transjakarta standard flat tariff",
		}
	case ModeMikrotrans:
		return FareResult{
			FareAmount:  0,
			Description: "mikrotrans zero tariff",
		}
	case ModeLRT:
		return FareResult{
			FareAmount:  BaseFareLRT,
			Description: "lrt flat tariff",
		}
	case ModeMRT:
		fallthrough
	default:
		zoneDelta := int64(params.Destination.ZoneIndex - params.Origin.ZoneIndex)
		if zoneDelta < 0 {
			zoneDelta = -zoneDelta
		}
		fare := BaseFareMRT + (zoneDelta * PerZoneRateMRT)
		if fare > MaxFareMRT {
			fare = MaxFareMRT
		}
		return FareResult{
			FareAmount:  fare,
			Description: "mrt distance-based tariff",
		}
	}
}
