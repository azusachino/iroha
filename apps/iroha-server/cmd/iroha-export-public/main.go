// Command iroha-export-public writes a static, sanitized snapshot of
// activity data (summary, activities, routes) to disk as JSON/GeoJSON. It is
// the data source for the standalone public GitHub Pages site: a NAS-side
// cron runs this, inside the private network, then commits and pushes the
// output — there is no live public HTTP surface to serve this data instead.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/dbconnect"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/publicexport"
	"gorm.io/gorm"
)

func main() {
	out := flag.String("out", "./dist/public-data", "output directory for summary.json/activities.json/routes.geojson")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	cfg, err := config.Load("iroha.toml")
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := dbconnect.Connect(cfg.Database.URL, &gorm.Config{}, logger)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}

	activityService := activities.NewService(db)
	// No enqueuer, no cache: LookupCity (the only geocode method this export
	// calls, via refreshOnMiss=false) reads tb_geocode_cache directly and
	// touches neither.
	geocodeService := geocode.NewService(db, nil, nil)

	if err := os.MkdirAll(*out, 0o755); err != nil {
		logger.Error("create output directory", "error", err, "dir", *out)
		os.Exit(1)
	}

	summary, err := publicexport.Summary(activityService, "", "")
	if err != nil {
		logger.Error("build summary", "error", err)
		os.Exit(1)
	}

	activityList, err := collectAllActivities(activityService)
	if err != nil {
		logger.Error("collect activities", "error", err)
		os.Exit(1)
	}

	routes, err := publicexport.Routes(ctx, activityService, geocodeService, false)
	if err != nil {
		logger.Error("build routes", "error", err)
		os.Exit(1)
	}

	// Validate before anything is written: a failure here must leave no
	// partial output on disk for the cron script to diff and push.
	if err := publicexport.Validate(summary, activityList, routes); err != nil {
		logger.Error("validate export", "error", err)
		os.Exit(1)
	}

	if err := writeJSON(filepath.Join(*out, "summary.json"), summary); err != nil {
		logger.Error("write summary.json", "error", err)
		os.Exit(1)
	}
	if err := writeJSON(filepath.Join(*out, "activities.json"), activityList); err != nil {
		logger.Error("write activities.json", "error", err)
		os.Exit(1)
	}
	if err := writeJSON(filepath.Join(*out, "routes.geojson"), routes); err != nil {
		logger.Error("write routes.geojson", "error", err)
		os.Exit(1)
	}
	meta := publicexport.Meta{GeneratedAt: time.Now().UTC()}
	if err := writeJSON(filepath.Join(*out, "meta.json"), meta); err != nil {
		logger.Error("write meta.json", "error", err)
		os.Exit(1)
	}

	logger.Info("export complete", "out", *out, "activities", len(activityList), "routes", len(routes.Features))
}

// collectAllActivities pages through every activity via publicexport.Activities,
// following its encoded cursor, to build the full sanitized dataset for a
// static snapshot. A static export has no notion of "next page" for a caller
// to request later, so this drives pagination to completion rather than
// returning one page at a time like the (now-removed) live handler did.
func collectAllActivities(svc *activities.Service) ([]publicexport.Activity, error) {
	const pageLimit = 100

	all := []publicexport.Activity{}
	filters := activities.ListFilters{Limit: pageLimit}
	for {
		page, err := publicexport.Activities(svc, filters)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		if !page.HasMore || page.NextCursor == nil {
			break
		}
		cursor, err := activities.DecodeCursor(*page.NextCursor)
		if err != nil {
			return nil, err
		}
		filters.Cursor = &cursor
	}
	return all, nil
}

func writeJSON(path string, data any) error {
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}
