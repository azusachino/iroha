package parsers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGPXFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.gpx")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="iroha-test">
  <trk>
    <name>Morning Run</name>
    <trkseg>
      <trkpt lat="35.0" lon="139.0">
        <ele>12.5</ele>
        <time>2026-07-07T00:00:00Z</time>
      </trkpt>
      <trkpt lat="35.1" lon="139.1">
        <ele>13.5</ele>
        <time>2026-07-07T00:05:00Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	activities, err := ParseGPXFile(path, GPXOptions{
		Title:      "fallback",
		ExternalID: "hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}

	activity := activities[0]
	if activity.Title != "Morning Run" {
		t.Fatalf("unexpected title: %s", activity.Title)
	}
	if activity.ExternalID != "hash" {
		t.Fatalf("unexpected external id: %s", activity.ExternalID)
	}
	if len(activity.RoutePoints) != 2 {
		t.Fatalf("expected 2 route points, got %d", len(activity.RoutePoints))
	}
	if activity.RoutePoints[0].ElevationM == nil || *activity.RoutePoints[0].ElevationM != 12.5 {
		t.Fatalf("unexpected first elevation: %#v", activity.RoutePoints[0].ElevationM)
	}
}
