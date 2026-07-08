package activities

import "testing"

// straightLine builds a roughly north-south track of n points spaced ~stepM
// meters apart, starting near lat 35 (≈111 km per degree of latitude).
func straightLine(n int, stepM float64) [][2]float64 {
	const metersPerDegLat = 111000.0
	coords := make([][2]float64, n)
	for i := range coords {
		coords[i] = [2]float64{139.7, 35.0 + float64(i)*stepM/metersPerDegLat}
	}
	return coords
}

func TestTrimRouteEnds_DropsStartAndEnd(t *testing.T) {
	// ~1 km line, ~10 m spacing.
	coords := straightLine(100, 10)

	trimmed := trimRouteEnds(coords)
	if len(trimmed) < routeMinPoints {
		t.Fatalf("trimmed too short: %d", len(trimmed))
	}

	// The exposed endpoints must be at least ~routeTrimMeters away from the
	// real start/end — this is the whole privacy guarantee.
	const minGap = routeTrimMeters - 20 // allow one sample of slack
	if gap := haversineMeters(coords[0], trimmed[0]); gap < minGap {
		t.Errorf("trimmed start only %.0f m from real start, want >= %d", gap, minGap)
	}
	if gap := haversineMeters(coords[len(coords)-1], trimmed[len(trimmed)-1]); gap < minGap {
		t.Errorf("trimmed end only %.0f m from real end, want >= %d", gap, minGap)
	}
}

func TestTrimRouteEnds_ShortTrackDropped(t *testing.T) {
	// ~300 m total (< 2*routeTrimMeters) — nothing should survive trimming.
	coords := straightLine(30, 10)
	if trimmed := trimRouteEnds(coords); trimmed != nil {
		t.Fatalf("short track should be dropped, got %d points", len(trimmed))
	}
}

func TestHaversineMeters_KnownDistance(t *testing.T) {
	// One degree of latitude is ~111 km; assert within 1%.
	got := haversineMeters([2]float64{139.7, 35.0}, [2]float64{139.7, 36.0})
	if got < 110000 || got > 112000 {
		t.Fatalf("haversine 1deg lat = %.0f m, want ~111000", got)
	}
}
