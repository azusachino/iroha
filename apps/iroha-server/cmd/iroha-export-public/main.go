// Command iroha-export-public writes a static, sanitized snapshot of
// activity data (summary, activities, routes) to disk as JSON/GeoJSON. It is
// kept for local/manual runs; the scheduled production export runs inside
// iroha-job as the projection_refresh job kind (see apps/iroha-job).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/dbconnect"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/publicexport"
	"gorm.io/gorm"
)

func main() {
	out := flag.String("out", "./dist/public-data", "output directory for summary.json/activities.json/routes.geojson")
	privacy := flag.Bool("privacy", false, "omit all route traces from the public snapshot")
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

	result, err := publicexport.Export(ctx, activityService, geocodeService, *out, publicexport.ExportOptions{Privacy: *privacy})
	if err != nil {
		logger.Error("export", "error", err)
		os.Exit(1)
	}

	logger.Info("export complete", "out", *out, "activities", result.ActivityCount, "routes", result.RouteCount)
}
