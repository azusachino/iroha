//go:build integration

package imports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	"github.com/azusachino/iroha/apps/iroha-providers/drifter"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const drifterIntegrationBookID = "integration-drifter-klara"

// drifterIntegrationSnapshot is one page of the Drifter reading snapshot shape
// documented in docs/plans/2026-11-11-drifter-reading-gateway-integration.md:
// a finished WeRead edition with author and translator credits, an ISBN, and a
// confirmed Hardcover edition mapping.
const drifterIntegrationSnapshot = `{
  "version": "1.0",
  "generated_at": "2026-11-11T10:00:00Z",
  "source_account": "default",
  "has_more": false,
  "books": [
    {
      "source_provider": "weread",
      "source_book_id": "` + drifterIntegrationBookID + `",
      "title": "克拉拉与太阳",
      "author": "石黑一雄",
      "translator": "赵松",
      "publisher": "上海译文出版社",
      "isbn": "9787532786331",
      "status": "completed",
      "progress_percent": 100,
      "last_read_at": "2024-03-04T00:20:00+08:00",
      "finished_at": "2024-03-04T00:30:00+08:00",
      "is_custom": false,
      "matched_target": {"target_provider": "hardcover", "work_id": 9001, "edition_id": 424242}
    }
  ]
}`

// TestIntegrationDrifterImportPersistsCreatorsAndFinishedEvent drives the real
// import pipeline for a Drifter snapshot page: raw evidence -> queued import
// job -> drifter adapter -> canonical media rows. It locks the ADR-0011
// amendment's creator persistence and honest time semantics.
func TestIntegrationDrifterImportPersistsCreatorsAndFinishedEvent(t *testing.T) {
	db := openImportsIntegrationDB(t)
	now := time.Now().UTC()
	body := []byte(drifterIntegrationSnapshot)

	rawID := uuid.New()
	hash := sha256.Sum256(body)
	storagePath := filepath.Join(t.TempDir(), "drifter_reading_snapshot.json")
	if err := os.WriteFile(storagePath, body, 0o600); err != nil {
		t.Fatalf("write drifter snapshot: %v", err)
	}
	if err := db.Create(&models.RawFile{
		ID: rawID, SHA256: hex.EncodeToString(hash[:]), OriginalFilename: "drifter_reading_snapshot.json",
		ContentType: "application/json", SizeBytes: int64(len(body)), StoragePath: storagePath,
		SourceKind: coreimports.KindDrifter, UploadedVia: "connector", ObservedAt: &now, CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	enqueuer := &drifterIntegrationEnqueuer{db: db}
	providers, err := provider.NewRegistry(drifter.NewAdapter())
	if err != nil {
		t.Fatalf("build provider registry: %v", err)
	}
	service := NewServiceWithRegistry(db, nil, DefaultParserVersion, enqueuer, nil, providers)
	importJob, err := service.Create(CreateInput{
		RawFileID:  ids.Encode(ids.RawFilePrefix, rawID),
		ParserKind: coreimports.KindDrifter,
	})
	if err != nil {
		t.Fatalf("create drifter import job: %v", err)
	}
	t.Cleanup(func() { cleanupDrifterIntegrationRows(t, db, importJob.ID, enqueuer.job.ID, rawID) })

	ctx := jobs.WithClaim(context.Background(), jobs.Claim{JobID: enqueuer.job.ID, WorkerID: "drifter-integration", Attempt: 1})
	if err := service.ProcessContext(ctx, importJob.ID); err != nil {
		t.Fatalf("process drifter import: %v", err)
	}

	var itemRef models.MediaExternalRef
	if err := db.Where("provider = ? and external_id = ?", "weread", drifterIntegrationBookID).First(&itemRef).Error; err != nil {
		t.Fatalf("load weread external ref: %v", err)
	}
	var item models.MediaItem
	if err := db.First(&item, "id = ?", itemRef.ScopeID).Error; err != nil {
		t.Fatalf("load media item: %v", err)
	}
	if item.MediaType != "book" || item.Title != "克拉拉与太阳" {
		t.Fatalf("media item = %q/%q, want book/克拉拉与太阳", item.MediaType, item.Title)
	}

	assertDrifterCreators(t, db, item.ID)
	assertDrifterFinishEvent(t, db, item.ID)
	assertDrifterProgress(t, db, item.ID)
}

func assertDrifterCreators(t *testing.T, db *gorm.DB, itemID uuid.UUID) {
	t.Helper()
	type creatorRoleRow struct {
		Name     string
		Role     string
		Provider string
	}
	var roles []creatorRoleRow
	if err := db.Table("tb_media_creator_roles AS r").
		Select("c.name, r.role, r.provider").
		Joins("JOIN tb_media_creators AS c ON c.id = r.creator_id").
		Where("r.scope_type = ? AND r.scope_id = ?", "item", itemID).
		Scan(&roles).Error; err != nil {
		t.Fatalf("load creator roles: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("creator roles = %#v, want author and translator", roles)
	}
	got := make(map[string]string, len(roles))
	for _, role := range roles {
		got[role.Role] = role.Name
		if role.Provider != "weread" {
			t.Errorf("creator role %q provider = %q, want weread", role.Role, role.Provider)
		}
	}
	if got["author"] != "石黑一雄" || got["translator"] != "赵松" {
		t.Fatalf("creator roles = %#v, want author 石黑一雄 and translator 赵松", got)
	}
}

func assertDrifterFinishEvent(t *testing.T, db *gorm.DB, itemID uuid.UUID) {
	t.Helper()
	var event models.MediaConsumptionEvent
	if err := db.Where("media_item_id = ? AND event_type = ?", itemID, "finished").First(&event).Error; err != nil {
		t.Fatalf("load finished event: %v", err)
	}
	want := time.Date(2024, time.March, 3, 16, 30, 0, 0, time.UTC)
	if !event.EventAt.Equal(want) {
		t.Fatalf("finished event at = %v, want %v", event.EventAt, want)
	}
	if event.SourceKind != coreimports.KindDrifter || event.SourceEventID != "weread-finish-"+drifterIntegrationBookID {
		t.Fatalf("finished event provenance = %q/%q", event.SourceKind, event.SourceEventID)
	}
}

func assertDrifterProgress(t *testing.T, db *gorm.DB, itemID uuid.UUID) {
	t.Helper()
	var progress models.MediaProgress
	if err := db.First(&progress, "media_item_id = ?", itemID).Error; err != nil {
		t.Fatalf("load media progress: %v", err)
	}
	if progress.Status != "completed" {
		t.Fatalf("progress status = %q, want completed", progress.Status)
	}
	if progress.ProgressPercent == nil || *progress.ProgressPercent != 100 {
		t.Fatalf("progress percent = %v, want 100", progress.ProgressPercent)
	}
}

func cleanupDrifterIntegrationRows(t *testing.T, db *gorm.DB, importJobID, queueJobID, rawFileID uuid.UUID) {
	t.Helper()
	var ref models.MediaExternalRef
	itemID := uuid.Nil
	if err := db.Where("provider = ? and external_id = ?", "weread", drifterIntegrationBookID).First(&ref).Error; err == nil {
		itemID = ref.ScopeID
	}
	var workID *uuid.UUID
	if itemID != uuid.Nil {
		var item models.MediaItem
		if err := db.First(&item, "id = ?", itemID).Error; err == nil {
			workID = item.WorkID
		}
	}
	statements := []cleanupStatement{
		{"delete from tb_media_creator_roles where scope_type = 'item' and scope_id = ?", []any{itemID}},
		{"delete from tb_media_consumption_events where media_item_id = ?", []any{itemID}},
		{"delete from tb_media_state_history where media_item_id = ?", []any{itemID}},
		{"delete from tb_media_progress where media_item_id = ?", []any{itemID}},
		{"delete from tb_media_titles where scope_type = 'item' and scope_id = ?", []any{itemID}},
		{"delete from tb_media_external_refs where scope_type = 'item' and scope_id = ?", []any{itemID}},
		{"delete from tb_media_list_items where media_item_id = ?", []any{itemID}},
		{"delete from tb_media_items where id = ?", []any{itemID}},
		{"delete from tb_import_snapshots where import_job_id = ?", []any{importJobID}},
		{"delete from tb_import_jobs where id = ?", []any{importJobID}},
		{"delete from tb_jobs where id = ?", []any{queueJobID}},
		{"delete from tb_raw_files where id = ?", []any{rawFileID}},
		{"delete from tb_media_creators c where not exists (select 1 from tb_media_creator_roles r where r.creator_id = c.id)", nil},
		{"delete from tb_media_lists l where l.name = 'weread library' and not exists (select 1 from tb_media_list_items li where li.list_id = l.id)", nil},
	}
	if workID != nil {
		statements = append(statements, cleanupStatement{"delete from tb_media_works where id = ?", []any{*workID}})
	}
	for _, statement := range statements {
		if err := db.Exec(statement.query, statement.args...).Error; err != nil {
			t.Errorf("cleanup %q: %v", statement.query, err)
		}
	}
}

type cleanupStatement struct {
	query string
	args  []any
}

// drifterIntegrationEnqueuer creates the queue job service.Create would
// enqueue, already claimed the way the worker's claim context expects. Keeping
// the claim real means ProcessContext runs the same protected transaction an
// actual worker does instead of a test-only bypass.
type drifterIntegrationEnqueuer struct {
	db  *gorm.DB
	job models.Job
}

func (e *drifterIntegrationEnqueuer) EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return models.Job{}, err
	}
	now := time.Now().UTC()
	worker := "drifter-integration"
	e.job = models.Job{
		ID: uuid.New(), Kind: kind, Status: jobs.StatusRunning, PayloadJSON: encoded,
		Attempts: 1, MaxAttempts: jobs.DefaultMaxAttempts, RunAfter: now,
		LockedBy: &worker, LockedAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	if tx == nil {
		tx = e.db
	}
	return e.job, tx.Create(&e.job).Error
}
