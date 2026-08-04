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

func TestRouteDistanceMeters_SumsTrack(t *testing.T) {
	got := routeDistanceMeters(straightLine(11, 10))
	if got < 95 || got > 105 {
		t.Fatalf("route distance = %.1f m, want about 100 m", got)
	}
}

func TestRouteDistanceMeters_EmptyTrack(t *testing.T) {
	if got := routeDistanceMeters(nil); got != 0 {
		t.Fatalf("empty route distance = %.1f m, want 0", got)
	}
}

func TestDetectPrivateZones_FindsMultipleHubs(t *testing.T) {
	home := [2]float64{139.700, 35.000}
	gym := [2]float64{139.750, 35.050} // ~6 km away
	var anchors [][2]float64
	for i := 0; i < 4; i++ {
		anchors = append(anchors, home)
	}
	for i := 0; i < 3; i++ {
		anchors = append(anchors, gym)
	}
	anchors = append(anchors, [2]float64{139.680, 34.980}) // lone outlier

	zones := detectPrivateZones(anchors)
	// All 7 clustered anchors (4 home + 3 gym) qualify; the lone outlier does not.
	if len(zones) != 7 {
		t.Fatalf("expected 7 dense anchors (home + gym), got %d", len(zones))
	}
	for _, hub := range [][2]float64{home, gym} {
		covered := false
		for _, z := range zones {
			if haversineMeters(hub, z) <= privateZoneRadiusMeters {
				covered = true
			}
		}
		if !covered {
			t.Errorf("hub %v not covered by any zone", hub)
		}
	}
}

func TestDetectPrivateZones_ScatteredNone(t *testing.T) {
	// Every anchor >1 km apart — nothing reaches privateZoneMinCluster.
	anchors := [][2]float64{
		{139.70, 35.00}, {139.72, 35.02}, {139.68, 34.98}, {139.74, 35.04},
	}
	if zones := detectPrivateZones(anchors); len(zones) != 0 {
		t.Fatalf("scattered anchors should yield no zones, got %d", len(zones))
	}
}

func TestMaskPrivateZones_SplitsAcrossZone(t *testing.T) {
	// A ~2 km line straight through a zone: masking must split it into two
	// segments, and no surviving point may fall inside the zone.
	zone := [2]float64{139.7, 35.0}
	coords := make([][2]float64, 0, 200)
	for i := -100; i < 100; i++ {
		coords = append(coords, [2]float64{139.7, 35.0 + float64(i)*10/111000.0})
	}

	segments := maskPrivateZones(coords, [][2]float64{zone})
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments split across zone, got %d", len(segments))
	}
	for _, seg := range segments {
		for _, c := range seg {
			if d := haversineMeters(c, zone); d <= privateZoneRadiusMeters {
				t.Errorf("point %.0f m from zone survived masking (<= %d)", d, privateZoneRadiusMeters)
			}
		}
	}
}
