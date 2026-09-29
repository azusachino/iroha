//go:build integration

package imports

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/google/uuid"
)

func TestIntegrationRawFilePurge(t *testing.T) {
	db := openImportsIntegrationDB(t)
	dir := t.TempDir()
	now := time.Now().UTC()
	day := 24 * time.Hour

	seed := func(name, status string, age time.Duration, withFile bool) (uuid.UUID, string) {
		id, err := ids.New()
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name)
		if withFile {
			if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if err := db.Create(&models.RawFile{ID: id, SHA256: "purge-" + id.String(), OriginalFilename: name, StoragePath: path, SourceKind: "anilist", UploadedVia: "integration", CreatedAt: now.Add(-age)}).Error; err != nil {
			t.Fatal(err)
		}
		jobID, _ := ids.New()
		if err := db.Exec(`insert into tb_import_jobs (id, raw_file_id, status, parser_kind, parser_version, created_at) values (?, ?, ?, 'anilist', 'test', ?)`, jobID, id, status, now.Add(-age)).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			db.Exec(`delete from tb_import_jobs where raw_file_id = ?`, id)
			db.Exec(`delete from tb_raw_files where id = ?`, id)
		})
		return id, path
	}

	oldDone, oldDonePath := seed("old-done", "completed", 8*day, true)
	freshDone, freshDonePath := seed("fresh-done", "completed", 6*day, true)
	oldMissing, _ := seed("old-missing", "completed", 8*day, false)
	failedYoung, failedYoungPath := seed("failed-young", "failed", 10*day, true)
	failedOld, failedOldPath := seed("failed-old", "failed", 31*day, true)
	parsing, parsingPath := seed("parsing", "parsing", 40*day, true)

	svc, err := rawfiles.NewService(db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Purge(context.Background(), 7*day, 30*day); err != nil {
		t.Fatal(err)
	}

	purged := func(id uuid.UUID) bool {
		var f models.RawFile
		if err := db.First(&f, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		return f.PurgedAt != nil
	}
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }

	for name, c := range map[string]struct {
		id     uuid.UUID
		path   string
		purged bool
	}{
		"old completed":    {oldDone, oldDonePath, true},
		"fresh completed":  {freshDone, freshDonePath, false},
		"missing file":     {oldMissing, "", true},
		"failed under 30d": {failedYoung, failedYoungPath, false},
		"failed over 30d":  {failedOld, failedOldPath, true},
		"parsing never":    {parsing, parsingPath, false},
	} {
		if purged(c.id) != c.purged {
			t.Errorf("%s: purged = %v, want %v", name, !c.purged, c.purged)
		}
		if c.path != "" && exists(c.path) == c.purged {
			t.Errorf("%s: file exists = %v, want %v", name, !c.purged, !c.purged)
		}
	}
}
