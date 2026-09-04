package publicexport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
)

// ExportOptions configures a full export run.
type ExportOptions struct {
	// Privacy omits all route traces from the public snapshot when true.
	Privacy bool
}

// ExportResult summarizes a completed export, for callers that only need to
// log or report on it rather than inspect the written files.
type ExportResult struct {
	ActivityCount int
	RouteCount    int
}

// Export builds the full sanitized public snapshot (summary/activities/
// routes/activity-details/meta) and writes it to outDir. Every file is
// written to a temporary sibling directory first and only swapped into
// outDir once the whole set has been built and validated successfully — a
// caller (a CLI, a scheduled job) never leaves outDir holding a partial
// write, and a failed run leaves whatever was previously there untouched.
func Export(ctx context.Context, activityService *activities.Service, geocodeService *geocode.Service, outDir string, opts ExportOptions) (ExportResult, error) {
	summary, err := Summary(activityService, "", "")
	if err != nil {
		return ExportResult{}, fmt.Errorf("build summary: %w", err)
	}

	activityList, err := collectAllActivities(activityService)
	if err != nil {
		return ExportResult{}, fmt.Errorf("collect activities: %w", err)
	}

	routes := RouteFeatureCollection{}
	if !opts.Privacy {
		routes, err = Routes(ctx, activityService, geocodeService, false)
		if err != nil {
			return ExportResult{}, fmt.Errorf("build routes: %w", err)
		}
	}

	activityDetails, err := ActivityDetails(activityService, activityList, !opts.Privacy)
	if err != nil {
		return ExportResult{}, fmt.Errorf("build activity details: %w", err)
	}

	// Validate before anything touches disk: a failure here must leave outDir
	// exactly as it was.
	if err := Validate(summary, activityList, routes); err != nil {
		return ExportResult{}, fmt.Errorf("validate export: %w", err)
	}
	if err := ValidateActivityDetails(activityDetails); err != nil {
		return ExportResult{}, fmt.Errorf("validate activity details: %w", err)
	}

	tmpDir := outDir + ".tmp"
	if err := os.RemoveAll(tmpDir); err != nil {
		return ExportResult{}, fmt.Errorf("clear stale temp dir: %w", err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return ExportResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }() // no-op once renamed away; cleans up on any early return

	if err := writeJSON(filepath.Join(tmpDir, "summary.json"), summary); err != nil {
		return ExportResult{}, fmt.Errorf("write summary.json: %w", err)
	}
	if err := writeJSON(filepath.Join(tmpDir, "activities.json"), activityList); err != nil {
		return ExportResult{}, fmt.Errorf("write activities.json: %w", err)
	}
	if err := writeJSON(filepath.Join(tmpDir, "routes.geojson"), routes); err != nil {
		return ExportResult{}, fmt.Errorf("write routes.geojson: %w", err)
	}
	if err := writeJSON(filepath.Join(tmpDir, "activity-details.json"), activityDetails); err != nil {
		return ExportResult{}, fmt.Errorf("write activity-details.json: %w", err)
	}
	meta := Meta{
		GeneratedAt:    time.Now().UTC(),
		RoutesIncluded: !opts.Privacy,
		ActivityCount:  len(activityDetails),
	}
	if err := writeJSON(filepath.Join(tmpDir, "meta.json"), meta); err != nil {
		return ExportResult{}, fmt.Errorf("write meta.json: %w", err)
	}

	// Swap by rename-aside rather than remove-then-rename: outDir is moved
	// out of the way first and only removed once tmpDir has successfully
	// taken its place, so a failure partway through never leaves outDir
	// missing — the previous export keeps serving under oldDir until the
	// swap actually succeeds.
	oldDir := outDir + ".old"
	if err := os.RemoveAll(oldDir); err != nil {
		return ExportResult{}, fmt.Errorf("clear stale old dir: %w", err)
	}
	if err := os.Rename(outDir, oldDir); err != nil && !os.IsNotExist(err) {
		return ExportResult{}, fmt.Errorf("move previous export dir aside: %w", err)
	}
	if err := os.Rename(tmpDir, outDir); err != nil {
		_ = os.Rename(oldDir, outDir) // best-effort restore of the previous export
		return ExportResult{}, fmt.Errorf("swap export dir into place: %w", err)
	}
	if err := os.RemoveAll(oldDir); err != nil {
		return ExportResult{}, fmt.Errorf("remove previous export dir: %w", err)
	}

	return ExportResult{ActivityCount: len(activityList), RouteCount: len(routes.Features)}, nil
}

// collectAllActivities pages through every activity via Activities,
// following its encoded cursor, to build the full sanitized dataset for a
// static snapshot. A static export has no notion of "next page" for a caller
// to request later, so this drives pagination to completion rather than
// returning one page at a time like the (now-removed) live handler did.
func collectAllActivities(svc *activities.Service) ([]Activity, error) {
	const pageLimit = 100

	all := []Activity{}
	filters := activities.ListFilters{Limit: pageLimit}
	for {
		page, err := Activities(svc, filters)
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
