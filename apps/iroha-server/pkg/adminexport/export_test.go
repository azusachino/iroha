package adminexport

import (
	"context"
	"io"
	"math"
	"testing"
	"time"
)

func TestExportAllowlistRejectsOperationalAndArbitraryObjects(t *testing.T) {
	for _, domain := range []string{"tb_users", "credentials", "activities; DROP TABLE tb_users", "", "jobs"} {
		if ValidDomain(domain) {
			t.Fatalf("accepted %q", domain)
		}
		if err := NDJSON(context.Background(), nil, domain, io.Discard); err == nil {
			t.Fatalf("export accepted %q", domain)
		}
	}
	if err := GPX(context.Background(), nil, "invalid", io.Discard); err == nil {
		t.Fatal("accepted malformed activity ID")
	}
}

func TestTrackPointPreservesPrecisionAndOptionalValues(t *testing.T) {
	timestamp := time.Date(2026, time.January, 1, 9, 0, 0, 123, time.FixedZone("+09", 9*60*60))
	ele := 12.3456789
	point, err := makePoint(35.123456789, 139.987654321, &ele, &timestamp)
	if err != nil {
		t.Fatal(err)
	}
	if point.Latitude != 35.123456789 || point.Longitude != 139.987654321 || *point.Elevation != ele || point.Time != "2026-01-01T00:00:00.000000123Z" {
		t.Fatalf("point changed: %+v", point)
	}
	point, err = makePoint(0, 0, nil, nil)
	if err != nil || point.Elevation != nil || point.Time != "" {
		t.Fatalf("optional fields fabricated: %+v, %v", point, err)
	}
}

func TestTrackPointRejectsInvalidNumbers(t *testing.T) {
	for _, coords := range [][2]float64{{91, 0}, {0, 181}, {-91, 0}, {0, -181}, {math.NaN(), 0}, {0, math.Inf(1)}} {
		if _, err := makePoint(coords[0], coords[1], nil, nil); err == nil {
			t.Fatalf("accepted %v", coords)
		}
	}
	ele := math.NaN()
	if _, err := makePoint(0, 0, &ele, nil); err == nil {
		t.Fatal("accepted invalid elevation")
	}
}
