package activities

import (
	"testing"
	"time"
)

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

func TestRouteElevationGainMeters_SumsRealClimbsOnly(t *testing.T) {
	// Real climbs (each step > elevationNoiseFloorMeters) interleaved with
	// small pullbacks that must not subtract from the sum, and are too small
	// to be climbs themselves even where they reverse direction.
	elevations := []float64{100, 105, 104.5, 110, 109.2, 115, 120, 119.3, 125, 130, 150}
	got := routeElevationGainMeters(elevations)
	if got < 50 || got > 54 {
		t.Fatalf("elevation gain = %.1f m, want about 52 m (5+5.5+5.8+5+5.7+5+20, jitter excluded)", got)
	}
}

func TestRouteElevationGainMeters_FlatTrackIsZero(t *testing.T) {
	// Every step is within the noise floor -- a flat route must not accrue
	// gain from GPS jitter alone.
	elevations := []float64{100, 101, 100.5, 101.8, 100.2, 101.5}
	if got := routeElevationGainMeters(elevations); got != 0 {
		t.Fatalf("flat route gain = %.1f m, want 0", got)
	}
}

func TestRouteElevationGainMeters_EmptyTrack(t *testing.T) {
	if got := routeElevationGainMeters(nil); got != 0 {
		t.Fatalf("empty route elevation gain = %.1f m, want 0", got)
	}
}

func TestRouteMovingTimeSeconds_ExcludesStoppedInterval(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// 10 points ~3 m apart, 1 s apart (a running pace, well above the
	// threshold): 9 moving seconds.
	moving := straightLine(10, 3)
	points := make([]timedPoint, 0, 15)
	for i, c := range moving {
		points = append(points, timedPoint{ts: start.Add(time.Duration(i) * time.Second), lon: c[0], lat: c[1]})
	}
	// 5 more points at the same spot (stopped at a light), 1 s apart:
	// 0 m/s, must not add to moving time.
	last := moving[len(moving)-1]
	for i := 1; i <= 5; i++ {
		points = append(points, timedPoint{
			ts:  start.Add(time.Duration(len(moving)-1+i) * time.Second),
			lon: last[0], lat: last[1],
		})
	}

	got := routeMovingTimeSeconds(points)
	if got < 8 || got > 10 {
		t.Fatalf("moving time = %d s, want about 9 s (elapsed is 13 s, 4 s stopped)", got)
	}
}

func TestRouteMovingTimeSeconds_EmptyTrack(t *testing.T) {
	if got := routeMovingTimeSeconds(nil); got != 0 {
		t.Fatalf("empty route moving time = %d s, want 0", got)
	}
}

func TestPeriodSummaryRequiresHalfOpenRange(t *testing.T) {
	from := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if _, err := (&Service{}).PeriodSummary(PeriodFilters{From: from, To: from, Timezone: "UTC"}); err == nil {
		t.Fatal("PeriodSummary accepted an empty [from,to) range")
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
