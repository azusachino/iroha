//go:build integration

package adminexport

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/testdb"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestIntegrationPrivateExports(t *testing.T) {
	db, err := gorm.Open(postgres.Open(testdb.DSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	rawID, activityID := uuid.New(), uuid.New()
	if err := db.Create(&models.RawFile{ID: rawID, SHA256: rawID.String(), OriginalFilename: "synthetic.gpx", StoragePath: "synthetic", SourceKind: "gpx", UploadedVia: "test", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DELETE FROM tb_raw_files WHERE id = ?", rawID) })
	if err := db.Create(&models.Activity{ID: activityID, SportType: "run", Title: "Run <&> private", StartedAt: now, SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Exec("DELETE FROM tb_activities WHERE id = ?", activityID) })
	for seq := range 2 {
		if err := db.Exec(`INSERT INTO tb_activity_route_points (activity_id, seq, lat, lon, elevation_m, ts, geom)
			VALUES (?, ?, ?, ?, ?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)`, activityID, seq, 35.123456789, 139.987654321, 12.3456789, now, 139.987654321, 35.123456789).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { db.Exec("DELETE FROM tb_activity_route_points WHERE activity_id = ?", activityID) })

	for domain := range domainTables {
		var out bytes.Buffer
		if err := NDJSON(context.Background(), db, domain, &out); err != nil {
			t.Fatalf("%s: %v", domain, err)
		}
		decoder := json.NewDecoder(&out)
		foundActivity, foundPoints := false, 0
		for {
			var envelope record
			if err := decoder.Decode(&envelope); errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				t.Fatal(err)
			}
			if envelope.Version != exportVersion || envelope.Domain != domain {
				t.Fatalf("wrong envelope: %+v", envelope)
			}
			var row map[string]any
			if err := json.Unmarshal(envelope.Record, &row); err != nil {
				t.Fatal(err)
			}
			if _, exists := row["geom"]; exists {
				t.Fatal("exported duplicate geography column")
			}
			if envelope.Table == "tb_activities" && row["id"] == activityID.String() {
				foundActivity = true
				if row["first_raw_file_id"] != rawID.String() || row["ended_at"] != nil || row["title"] != "Run <&> private" {
					t.Fatalf("changed canonical values: %+v", row)
				}
			}
			if envelope.Table == "tb_activity_route_points" && row["activity_id"] == activityID.String() {
				foundPoints++
				if row["lat"] != 35.123456789 || row["lon"] != 139.987654321 {
					t.Fatalf("changed coordinates: %+v", row)
				}
			}
		}
		if domain == "activities" && (!foundActivity || foundPoints != 2) {
			t.Fatalf("lost fixture: activity=%v points=%d", foundActivity, foundPoints)
		}
	}

	var out bytes.Buffer
	id := ids.Encode(ids.ActivityPrefix, activityID)
	if err := GPX(context.Background(), db, id, &out); err != nil {
		t.Fatal(err)
	}
	var document struct {
		XMLName xml.Name
		Version string `xml:"version,attr"`
		Track   struct {
			Name   string       `xml:"name"`
			Type   string       `xml:"type"`
			Points []trackPoint `xml:"trkseg>trkpt"`
		} `xml:"trk"`
	}
	if err := xml.Unmarshal(out.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.XMLName.Space != gpxNamespace || document.Version != gpxVersion || document.Track.Name != "Run <&> private" || document.Track.Type != "run" || len(document.Track.Points) != 2 {
		t.Fatalf("GPX round trip: %+v", document)
	}
	for _, point := range document.Track.Points {
		if point.Latitude != 35.123456789 || point.Longitude != 139.987654321 || point.Time != "2026-01-01T00:00:00Z" || point.Elevation == nil || *point.Elevation != 12.3456789 {
			t.Fatalf("GPX point changed: %+v", point)
		}
	}
	if err := GPX(context.Background(), db, ids.Encode(ids.ActivityPrefix, uuid.New()), io.Discard); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing activity: %v", err)
	}
	for _, export := range []func(context.Context, io.Writer) error{
		func(ctx context.Context, out io.Writer) error { return NDJSON(ctx, db, "activities", out) },
		func(ctx context.Context, out io.Writer) error { return GPX(ctx, db, id, out) },
	} {
		if err := export(context.Background(), brokenWriter{}); !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("writer failure lost: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := export(ctx, io.Discard); err == nil {
			t.Fatal("cancellation ignored")
		}
	}
}
