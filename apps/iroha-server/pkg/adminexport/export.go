// Package adminexport streams private canonical data in open formats.
package adminexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/gorm"
)

const (
	exportVersion = 1
	gpxVersion    = "1.1"
	gpxNamespace  = "http://www.topografix.com/GPX/1/1"
)

// Only canonical domain tables are selectable; never auth or operational payloads.
var domainTables = map[string][]string{
	"activities": {"tb_activities", "tb_external_refs", "tb_activity_route_points", "tb_activity_samplings", "tb_activity_laps"},
	"sleep":      {"tb_sleep_sessions", "tb_sleep_segments"},
	"daily":      {"tb_daily_summaries", "tb_daily_metrics"},
	"media":      {"tb_media_works", "tb_media_items", "tb_media_titles", "tb_media_relations", "tb_media_external_refs", "tb_media_creators", "tb_media_creator_roles", "tb_media_consumption_events", "tb_media_progress", "tb_media_state_history", "tb_media_lists", "tb_media_list_items", "tb_media_matching_decisions"},
	"expenses":   {"tb_expenses", "tb_expense_statements", "tb_expense_statement_rows"},
	"tasks":      {"tb_tasks"},
}

// ValidDomain reports whether the requested domain has a fixed export allowlist.
func ValidDomain(domain string) bool {
	_, ok := domainTables[domain]
	return ok
}

type record struct {
	Version int             `json:"version"`
	Domain  string          `json:"domain"`
	Table   string          `json:"table"`
	Record  json.RawMessage `json:"record"`
}

// NDJSON writes one canonical record per line from a consistent domain snapshot.
func NDJSON(ctx context.Context, db *gorm.DB, domain string, out io.Writer) error {
	tables, ok := domainTables[domain]
	if !ok {
		return errors.New("unknown export domain")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		encoder := json.NewEncoder(out)
		for _, table := range tables {
			if err := writeTable(tx, encoder, domain, table); err != nil {
				return err
			}
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
}

func writeTable(db *gorm.DB, encoder *json.Encoder, domain, table string) (err error) {
	// table comes only from domainTables, never interpolated user input.
	rows, err := db.Raw("SELECT to_jsonb(t) - 'geom' FROM " + table + " AS t").Rows()
	if err != nil {
		return fmt.Errorf("export %s: %w", table, err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}
		if err := encoder.Encode(record{Version: exportVersion, Domain: domain, Table: table, Record: data}); err != nil {
			return err
		}
	}
	return rows.Err()
}

type trackPoint struct {
	Latitude  float64  `xml:"lat,attr"`
	Longitude float64  `xml:"lon,attr"`
	Elevation *float64 `xml:"ele,omitempty"`
	Time      string   `xml:"time,omitempty"`
}

// GPX streams the private canonical route, with neither trimming nor rounding.
func GPX(ctx context.Context, db *gorm.DB, activityID string, out io.Writer) error {
	id, err := ids.Decode(ids.ActivityPrefix, activityID)
	if err != nil {
		return fmt.Errorf("invalid activity ID: %w", err)
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) (err error) {
		var activity models.Activity
		if err := tx.First(&activity, "id = ?", id).Error; err != nil {
			return fmt.Errorf("find activity: %w", err)
		}
		rows, err := tx.Table("tb_activity_route_points").Select("lat, lon, elevation_m, ts").Where("activity_id = ?", id).Order("seq").Rows()
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, rows.Close()) }()
		encoder := xml.NewEncoder(out)
		root := xml.StartElement{Name: xml.Name{Local: "gpx"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmlns"}, Value: gpxNamespace},
			{Name: xml.Name{Local: "version"}, Value: gpxVersion},
			{Name: xml.Name{Local: "creator"}, Value: "iroha-admin"},
		}}
		track := xml.StartElement{Name: xml.Name{Local: "trk"}}
		segment := xml.StartElement{Name: xml.Name{Local: "trkseg"}}
		for _, start := range []xml.StartElement{root, track} {
			if err := encoder.EncodeToken(start); err != nil {
				return err
			}
		}
		if err := encoder.EncodeElement(activity.Title, xml.StartElement{Name: xml.Name{Local: "name"}}); err != nil {
			return err
		}
		if err := encoder.EncodeElement(activity.SportType, xml.StartElement{Name: xml.Name{Local: "type"}}); err != nil {
			return err
		}
		if err := encoder.EncodeToken(segment); err != nil {
			return err
		}
		for rows.Next() {
			var lat, lon, elevation *float64
			var timestamp *time.Time
			if err := rows.Scan(&lat, &lon, &elevation, &timestamp); err != nil {
				return err
			}
			if lat == nil || lon == nil {
				continue
			}
			point, err := makePoint(*lat, *lon, elevation, timestamp)
			if err != nil {
				return err
			}
			if err := encoder.EncodeElement(point, xml.StartElement{Name: xml.Name{Local: "trkpt"}}); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, end := range []xml.EndElement{segment.End(), track.End(), root.End()} {
			if err := encoder.EncodeToken(end); err != nil {
				return err
			}
		}
		return encoder.Flush()
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
}

func makePoint(lat, lon float64, elevation *float64, timestamp *time.Time) (trackPoint, error) {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 || math.IsNaN(lon) || math.IsInf(lon, 0) || lon < -180 || lon > 180 {
		return trackPoint{}, errors.New("invalid route coordinates")
	}
	if elevation != nil && (math.IsNaN(*elevation) || math.IsInf(*elevation, 0)) {
		return trackPoint{}, errors.New("invalid route elevation")
	}
	point := trackPoint{Latitude: lat, Longitude: lon, Elevation: elevation}
	if timestamp != nil {
		point.Time = timestamp.UTC().Format(time.RFC3339Nano)
	}
	return point, nil
}
