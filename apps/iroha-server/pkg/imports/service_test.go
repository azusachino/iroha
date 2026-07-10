package imports

import (
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/parsers"
	"github.com/google/uuid"
)

func TestBuildRoutePointsInsertSQL(t *testing.T) {
	activityID := uuid.New()
	elev1 := 12.5
	ts1 := time.Unix(1000, 0).UTC()
	ts2 := time.Unix(2000, 0).UTC()
	points := []parsers.RoutePoint{
		{Ts: &ts1, Lat: 37.1, Lon: -122.1, ElevationM: &elev1},
		{Ts: &ts2, Lat: 37.2, Lon: -122.2, ElevationM: nil},
	}

	sql, args := buildRoutePointsInsertSQL(activityID, points, 5)

	wantTuples := 2
	if got := strings.Count(sql, "ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)"); got != wantTuples {
		t.Fatalf("expected %d value tuples in sql, got %d: %s", wantTuples, got, sql)
	}
	if !strings.Contains(sql, "insert into tb_activity_route_points") {
		t.Fatalf("expected insert statement, got %s", sql)
	}

	// 8 args per row.
	if len(args) != len(points)*8 {
		t.Fatalf("expected %d args, got %d", len(points)*8, len(args))
	}

	// First row: activityID, seq(=startSeq+0), ts, lat, lon, elevation, lon, lat.
	row0 := args[0:8]
	if row0[0] != activityID {
		t.Errorf("row0 activity_id = %v, want %v", row0[0], activityID)
	}
	if row0[1] != 5 {
		t.Errorf("row0 seq = %v, want 5 (startSeq+0)", row0[1])
	}
	if row0[3] != 37.1 || row0[4] != -122.1 {
		t.Errorf("row0 lat/lon = %v/%v, want 37.1/-122.1", row0[3], row0[4])
	}
	if row0[5] != &elev1 {
		t.Errorf("row0 elevation_m pointer mismatch")
	}
	// ST_MakePoint args duplicate lon,lat (lon first, then lat).
	if row0[6] != -122.1 || row0[7] != 37.1 {
		t.Errorf("row0 ST_MakePoint args = %v/%v, want lon=-122.1 lat=37.1", row0[6], row0[7])
	}

	// Second row: seq should be startSeq+1 = 6.
	row1 := args[8:16]
	if row1[1] != 6 {
		t.Errorf("row1 seq = %v, want 6 (startSeq+1)", row1[1])
	}
	if row1[5] != (*float64)(nil) {
		t.Errorf("row1 elevation_m = %v, want nil", row1[5])
	}
	if row1[6] != -122.2 || row1[7] != 37.2 {
		t.Errorf("row1 ST_MakePoint args = %v/%v, want lon=-122.2 lat=37.2", row1[6], row1[7])
	}
}

func TestBuildRoutePointsInsertSQL_SeqAcrossChunks(t *testing.T) {
	activityID := uuid.New()
	points := make([]parsers.RoutePoint, 3)
	now := time.Now().UTC()
	for i := range points {
		points[i] = parsers.RoutePoint{Ts: &now, Lat: float64(i), Lon: float64(i)}
	}

	// Simulate chunk 2 of a batch starting at absolute seq 1000.
	_, args := buildRoutePointsInsertSQL(activityID, points, 1000)
	for i := 0; i < len(points); i++ {
		seq := args[i*8+1]
		if seq != 1000+i {
			t.Errorf("point %d: seq = %v, want %d", i, seq, 1000+i)
		}
	}
}

func TestBuildRoutePointsInsertSQL_Empty(t *testing.T) {
	sql, args := buildRoutePointsInsertSQL(uuid.New(), nil, 0)
	if len(args) != 0 {
		t.Fatalf("expected no args for empty points, got %d", len(args))
	}
	if strings.Contains(sql, "?") {
		t.Fatalf("expected no placeholders for empty points, got %s", sql)
	}
}
