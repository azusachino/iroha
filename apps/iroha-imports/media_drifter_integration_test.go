//go:build integration

package imports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	connectorregistry "github.com/azusachino/iroha/apps/iroha-providers/connectors"
	"github.com/azusachino/iroha/apps/iroha-providers/drifter"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const drifterIntegrationBookID = "integration-drifter-klara"

// drifterIntegrationWorker is the worker identity the integration enqueuer
// claims jobs under.
const drifterIntegrationWorker = "drifter-integration"

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
      "reading_time_seconds": 18240,
      "last_read_at": "2024-03-04T00:20:00+08:00",
      "finished_at": "2024-03-04T00:30:00+08:00",
      "is_custom": false,
      "matched_target": {"target_provider": "hardcover", "work_id": 9001, "edition_id": 424242, "original_title": "Klara and the Sun"}
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
	if progress.ReadingTimeSeconds == nil || *progress.ReadingTimeSeconds != 18240 {
		t.Fatalf("reading time seconds = %v, want 18240", progress.ReadingTimeSeconds)
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
		statements = append(statements,
			cleanupStatement{"delete from tb_media_external_refs where scope_type = 'work' and scope_id = ?", []any{*workID}},
			cleanupStatement{"delete from tb_media_works where id = ?", []any{*workID}},
		)
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
	db       *gorm.DB
	job      models.Job
	enqueued []drifterIntegrationQueued
	drained  int
}

// drifterIntegrationQueued pairs a queued parse job with the import it will
// process, so a test can drain a sync run's imports the way the worker would.
type drifterIntegrationQueued struct {
	importJobID uuid.UUID
	queueJob    models.Job
}

func (e *drifterIntegrationEnqueuer) EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return models.Job{}, err
	}
	now := time.Now().UTC()
	worker := drifterIntegrationWorker
	e.job = models.Job{
		ID: uuid.New(), Kind: kind, Status: jobs.StatusRunning, PayloadJSON: encoded,
		Attempts: 1, MaxAttempts: jobs.DefaultMaxAttempts, RunAfter: now,
		LockedBy: &worker, LockedAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	if tx == nil {
		tx = e.db
	}
	if err := tx.Create(&e.job).Error; err != nil {
		return models.Job{}, err
	}
	e.enqueued = append(e.enqueued, drifterIntegrationQueued{importJobID: importJobIDFromPayload(payload), queueJob: e.job})
	return e.job, nil
}

// drain returns the parse jobs enqueued since the previous drain.
func (e *drifterIntegrationEnqueuer) drain() []drifterIntegrationQueued {
	pending := e.enqueued[e.drained:]
	e.drained = len(e.enqueued)
	return pending
}

func importJobIDFromPayload(payload any) uuid.UUID {
	object, ok := payload.(map[string]any)
	if !ok {
		return uuid.Nil
	}
	value, ok := object["import_job_id"].(string)
	if !ok {
		return uuid.Nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// TestIntegrationDrifterSyncFollowerStateMachine drives the real sync runner
// against a stand-in drifter-server. It locks the replication state machine:
// a first run bootstraps from a full snapshot and stores its watermark, the
// next run asks only for changes since that watermark, and a watermark the
// gateway no longer serves falls back to a full snapshot inside the same run.
func TestIntegrationDrifterSyncFollowerStateMachine(t *testing.T) {
	db := openImportsIntegrationDB(t)
	// The watermark is read from the latest receipt of the source kind, and
	// the run lock is per connector id, so a row left by an interrupted run
	// would steer this test. The integration database is disposable.
	for _, statement := range []string{
		"delete from tb_source_receipts where source_kind = '" + coreimports.KindDrifter + "'",
		"delete from tb_source_instances where provider = '" + coreimports.KindDrifter + "'",
		"delete from tb_media_sync_state where connector_id = '" + drifter.ProviderID + "'",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("reset drifter sync state (%s): %v", statement, err)
		}
	}

	feed := &drifterSyncFeed{}
	server := httptest.NewServer(feed)
	t.Cleanup(server.Close)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	providers, err := provider.NewRegistry(drifter.NewAdapter())
	if err != nil {
		t.Fatalf("build provider registry: %v", err)
	}
	enqueuer := &drifterIntegrationEnqueuer{db: db}
	service := NewServiceWithRegistry(db, logger, DefaultParserVersion, enqueuer, nil, providers)
	store, err := rawfiles.NewService(db, t.TempDir())
	if err != nil {
		t.Fatalf("create raw file service: %v", err)
	}
	connectors, err := connectorregistry.New(drifter.NewConnector(server.URL, "sync-token"))
	if err != nil {
		t.Fatalf("build connector registry: %v", err)
	}
	runner := NewSyncRunner(db, connectors, store, service)
	t.Cleanup(func() { cleanupDrifterSyncRows(t, db, enqueuer) })

	// 1. Bootstrap: no stored watermark, so the run reads the whole shelf and
	//    its receipt becomes the resume point.
	feed.serveSnapshot(drifterSyncPage("2026-11-11T10:00:00Z", "completed", 100, "2024-03-04T00:20:00+08:00", "2024-03-04T00:30:00+08:00"))
	if err := runner.Run(context.Background(), drifter.ProviderID, connector.Credentials{}); err != nil {
		t.Fatalf("bootstrap sync: %v", err)
	}
	processDrifterSyncImports(t, service, enqueuer)
	assertDrifterSyncRun(t, db, rawfiles.IngestionModeFullSnapshot, "2026-11-11T10:00:00Z")
	assertDrifterSyncProgress(t, db, "completed", 100)
	if since := feed.requestedSince(); len(since) != 0 {
		t.Fatalf("bootstrap requested changes since %v, want a full snapshot only", since)
	}

	// 2. Steady state: the run asks for changes since the stored watermark and
	//    stores only the changed book.
	feed.serveChanges(drifterSyncPage("2026-11-11T15:00:00Z", "reading", 55, "2026-11-11T14:00:00Z", ""))
	if err := runner.Run(context.Background(), drifter.ProviderID, connector.Credentials{}); err != nil {
		t.Fatalf("incremental sync: %v", err)
	}
	processDrifterSyncImports(t, service, enqueuer)
	if since := feed.requestedSince(); len(since) != 1 || since[0] != "2026-11-11T10:00:00Z" {
		t.Fatalf("incremental requested changes since %v, want the bootstrap watermark", since)
	}
	assertDrifterSyncRun(t, db, rawfiles.IngestionModeIncremental, "2026-11-11T15:00:00Z")
	assertDrifterSyncProgress(t, db, "in_progress", 55)

	// 3. Expired horizon: the changes feed answers 410 Gone, so the same run
	//    falls back to a full snapshot, finishes completed, and stores the new
	//    watermark instead of failing the sync.
	feed.expireHorizon(drifterSyncPage("2026-11-12T09:00:00Z", "reading", 60, "2026-11-12T08:00:00Z", ""))
	if err := runner.Run(context.Background(), drifter.ProviderID, connector.Credentials{}); err != nil {
		t.Fatalf("fallback sync: %v", err)
	}
	processDrifterSyncImports(t, service, enqueuer)
	if since := feed.requestedSince(); len(since) != 2 || since[1] != "2026-11-11T15:00:00Z" {
		t.Fatalf("fallback requested changes since %v, want the incremental watermark", since)
	}
	assertDrifterSyncRun(t, db, rawfiles.IngestionModeFullSnapshot, "2026-11-12T09:00:00Z")
	assertDrifterSyncProgress(t, db, "in_progress", 60)
}

// TestIntegrationDrifterPeriodicSnapshotWithdrawsRemovedMapping proves what a
// changes feed cannot: a mapping Drifter deleted. A stale follower reads a full
// snapshot, and the snapshot that lists the book without a matched target
// removes the Hardcover ref the source asserted, while the book itself and its
// WeRead identity stay.
func TestIntegrationDrifterPeriodicSnapshotWithdrawsRemovedMapping(t *testing.T) {
	db := openImportsIntegrationDB(t)
	for _, statement := range []string{
		"delete from tb_source_receipts where source_kind = '" + coreimports.KindDrifter + "'",
		"delete from tb_source_instances where provider = '" + coreimports.KindDrifter + "'",
		"delete from tb_media_sync_state where connector_id = '" + drifter.ProviderID + "'",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("reset drifter sync state (%s): %v", statement, err)
		}
	}
	feed := &drifterSyncFeed{}
	server := httptest.NewServer(feed)
	t.Cleanup(server.Close)
	providers, err := provider.NewRegistry(drifter.NewAdapter())
	if err != nil {
		t.Fatalf("build provider registry: %v", err)
	}
	enqueuer := &drifterIntegrationEnqueuer{db: db}
	service := NewServiceWithRegistry(db, slog.New(slog.NewTextHandler(io.Discard, nil)), DefaultParserVersion, enqueuer, nil, providers)
	store, err := rawfiles.NewService(db, t.TempDir())
	if err != nil {
		t.Fatalf("create raw file service: %v", err)
	}
	connectors, err := connectorregistry.New(drifter.NewConnector(server.URL, "sync-token"))
	if err != nil {
		t.Fatalf("build connector registry: %v", err)
	}
	runner := NewSyncRunner(db, connectors, store, service)
	t.Cleanup(func() { cleanupDrifterSyncRows(t, db, enqueuer) })
	run := func() {
		t.Helper()
		if err := runner.Run(context.Background(), drifter.ProviderID, connector.Credentials{}); err != nil {
			t.Fatalf("sync run: %v", err)
		}
		processDrifterSyncImports(t, service, enqueuer)
	}

	mapped := drifterSyncPage("2026-11-01T10:00:00Z", "completed", 100, "2024-03-04T00:20:00+08:00", "2024-03-04T00:30:00+08:00")
	feed.serveSnapshot(mapped)
	run()
	if !drifterHardcoverRefExists(t, db) {
		t.Fatal("bootstrap snapshot did not record the matched Hardcover edition")
	}

	// The mapping is deleted in Drifter. The changes feed never lists the book,
	// so a recent follower keeps the stale ref.
	unmapped := strings.Replace(mapped, `"matched_target":{"target_provider":"hardcover","work_id":9001,"edition_id":424242}`, `"matched_target":null`, 1)
	feed.serveSnapshot(unmapped)
	feed.serveChanges(`{"version":"1.0","source_identity":"weread:23849102","watermark":"2026-11-03T10:00:00Z","has_more":false,"books":[]}`)
	ageFullSnapshots(t, db, drifter.ProviderID, 2*24*time.Hour)
	run()
	if since := feed.requestedSince(); len(since) != 1 {
		t.Fatalf("2 days after the snapshot requested changes %d times, want 1", len(since))
	}
	if !drifterHardcoverRefExists(t, db) {
		t.Fatal("a changes run withdrew the Hardcover ref; only a full snapshot may")
	}

	// A week later the follower reads the snapshot instead of changes.
	ageFullSnapshots(t, db, drifter.ProviderID, 6*24*time.Hour)
	run()
	if since := feed.requestedSince(); len(since) != 1 {
		t.Fatalf("8 days after the snapshot requested changes %d times in total, want still 1", len(since))
	}
	assertDrifterSyncRun(t, db, rawfiles.IngestionModeFullSnapshot, "2026-11-01T10:00:00Z")
	if drifterHardcoverRefExists(t, db) {
		t.Fatal("the full snapshot without a matched target left the Hardcover ref behind")
	}
	var itemRef models.MediaExternalRef
	if err := db.Where("provider = ? and external_id = ?", "weread", drifterIntegrationBookID).First(&itemRef).Error; err != nil {
		t.Fatalf("the book lost its WeRead identity: %v", err)
	}
}

func drifterHardcoverRefExists(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	var count int64
	if err := db.Raw(`select count(*) from tb_media_external_refs h
join tb_media_external_refs w on w.scope_type = h.scope_type and w.scope_id = h.scope_id
where h.provider = 'hardcover' and h.external_id = '424242' and w.provider = 'weread' and w.external_id = ?`, drifterIntegrationBookID).Scan(&count).Error; err != nil {
		t.Fatalf("count hardcover refs: %v", err)
	}
	return count > 0
}

// drifterSyncFeed is a stand-in drifter-server. It serves the snapshot and
// changes endpoints from programmable bodies and records the watermark each
// changes request asked for, so a test can prove what the runner resumed from.
type drifterSyncFeed struct {
	mu       sync.Mutex
	snapshot string
	changes  string
	gone     bool
	since    []string
}

func (f *drifterSyncFeed) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case drifter.SnapshotPath:
		_, _ = w.Write([]byte(f.snapshot))
	case drifter.ChangesPath:
		f.since = append(f.since, r.URL.Query().Get("since"))
		if f.gone {
			w.WriteHeader(http.StatusGone)
			_, _ = w.Write([]byte(`{"error":"horizon_exceeded","message":"watermark is older than 14 days; full snapshot required","snapshot_required":true}`))
			return
		}
		_, _ = w.Write([]byte(f.changes))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *drifterSyncFeed) serveSnapshot(body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapshot, f.gone = body, false
}

func (f *drifterSyncFeed) serveChanges(body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.changes, f.gone = body, false
}

// expireHorizon keeps serving the snapshot but refuses every changes request.
func (f *drifterSyncFeed) expireHorizon(body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapshot, f.gone = body, true
}

func (f *drifterSyncFeed) requestedSince() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.since...)
}

// drifterSyncPage renders one page of the reading feed for the fixture book.
// finishedAt is omitted for a changes page, which carries only what moved.
func drifterSyncPage(watermark, status string, progress int, lastReadAt, finishedAt string) string {
	finished := ""
	if finishedAt != "" {
		finished = `,"finished_at":"` + finishedAt + `"`
	}
	return `{"version":"1.0","source_identity":"weread:23849102","watermark":"` + watermark + `","has_more":false,"books":[{
  "source_provider":"weread","source_book_id":"` + drifterIntegrationBookID + `",
  "title":"克拉拉与太阳","author":"石黑一雄","translator":"赵松","publisher":"上海译文出版社",
  "isbn":"9787532786331","status":"` + status + `","progress_percent":` + strconv.Itoa(progress) + `,
  "last_read_at":"` + lastReadAt + `"` + finished + `,
  "is_custom":false,"matched_target":{"target_provider":"hardcover","work_id":9001,"edition_id":424242}
}]}`
}

func processDrifterSyncImports(t *testing.T, service *Service, enqueuer *drifterIntegrationEnqueuer) {
	t.Helper()
	for _, queued := range enqueuer.drain() {
		if queued.importJobID == uuid.Nil {
			t.Fatalf("enqueued parse job %s carries no import job id", queued.queueJob.ID)
		}
		ctx := jobs.WithClaim(context.Background(), jobs.ClaimFor(queued.queueJob, drifterIntegrationWorker))
		if err := service.ProcessContext(ctx, queued.importJobID); err != nil {
			t.Fatalf("process drifter import %s: %v", queued.importJobID, err)
		}
	}
}

// assertDrifterSyncRun checks the newest drifter run finished completed with a
// receipt proving the expected ingestion mode and watermark, and that its
// coverage assertion says the same thing.
func assertDrifterSyncRun(t *testing.T, db *gorm.DB, ingestionMode, watermark string) {
	t.Helper()
	var run models.MediaSyncRun
	if err := db.Where("connector_id = ?", drifter.ProviderID).Order("started_at desc").First(&run).Error; err != nil {
		t.Fatalf("load latest drifter sync run: %v", err)
	}
	if run.Status != syncRunStatusCompleted || run.FinishedAt == nil {
		t.Fatalf("sync run status = %q (finished %v), want completed", run.Status, run.FinishedAt)
	}
	var receipts []struct {
		IngestionMode string
		OrderingBasis string
	}
	if err := db.Raw(`select distinct r.ingestion_mode, r.ordering_basis
from tb_source_receipts r
join tb_import_jobs j on j.raw_file_id = r.raw_file_id
where j.sync_run_id = ?`, run.ID).Scan(&receipts).Error; err != nil {
		t.Fatalf("load run receipts: %v", err)
	}
	if len(receipts) != 1 || receipts[0].IngestionMode != ingestionMode || receipts[0].OrderingBasis != watermark {
		t.Fatalf("run receipts = %#v, want one %s receipt with watermark %s", receipts, ingestionMode, watermark)
	}
	var covered int64
	if err := db.Raw(`select count(*) from tb_source_coverage_assertions a
join tb_source_receipts r on r.id = a.source_receipt_id
join tb_import_jobs j on j.raw_file_id = r.raw_file_id
where j.sync_run_id = ? and a.category = 'media_list' and a.completeness = 'covered' and a.ingestion_mode = ?`, run.ID, ingestionMode).Scan(&covered).Error; err != nil || covered != 1 {
		t.Fatalf("covered media_list assertions = %d (err %v), want 1 with mode %s", covered, err, ingestionMode)
	}
}

func assertDrifterSyncProgress(t *testing.T, db *gorm.DB, status string, percent float64) {
	t.Helper()
	var itemRef models.MediaExternalRef
	if err := db.Where("provider = ? and external_id = ?", "weread", drifterIntegrationBookID).First(&itemRef).Error; err != nil {
		t.Fatalf("load weread external ref: %v", err)
	}
	var progress models.MediaProgress
	if err := db.First(&progress, "media_item_id = ?", itemRef.ScopeID).Error; err != nil {
		t.Fatalf("load media progress: %v", err)
	}
	if progress.Status != status {
		t.Fatalf("progress status = %q, want %q", progress.Status, status)
	}
	if progress.ProgressPercent == nil || *progress.ProgressPercent != percent {
		t.Fatalf("progress percent = %v, want %v", progress.ProgressPercent, percent)
	}
}

// cleanupDrifterSyncRows removes everything the state-machine test created:
// media rows for the fixture book, then its evidence, imports, queue jobs, raw
// files, source instances and the sync bookkeeping rows themselves.
func cleanupDrifterSyncRows(t *testing.T, db *gorm.DB, enqueuer *drifterIntegrationEnqueuer) {
	t.Helper()
	connectorID := drifter.ProviderID
	var rawFileIDs []uuid.UUID
	if err := db.Raw(`select distinct raw_file_id from tb_import_jobs where sync_run_id in (select id from tb_media_sync_runs where connector_id = ?)`, connectorID).Scan(&rawFileIDs).Error; err != nil {
		t.Errorf("collect sync raw files: %v", err)
	}
	var itemRef models.MediaExternalRef
	itemID := uuid.Nil
	var workID *uuid.UUID
	if err := db.Where("provider = ? and external_id = ?", "weread", drifterIntegrationBookID).First(&itemRef).Error; err == nil {
		itemID = itemRef.ScopeID
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
		{"delete from tb_source_coverage_assertions where source_receipt_id in (select id from tb_source_receipts where source_kind = ?)", []any{coreimports.KindDrifter}},
		{"delete from tb_source_receipts where source_kind = ?", []any{coreimports.KindDrifter}},
		{"delete from tb_import_snapshots where import_job_id in (select id from tb_import_jobs where sync_run_id in (select id from tb_media_sync_runs where connector_id = ?))", []any{connectorID}},
		{"delete from tb_import_jobs where sync_run_id in (select id from tb_media_sync_runs where connector_id = ?)", []any{connectorID}},
		{"delete from tb_source_instances where provider = ?", []any{coreimports.KindDrifter}},
		{"delete from tb_media_sync_state where connector_id = ?", []any{connectorID}},
		{"delete from tb_media_sync_runs where connector_id = ?", []any{connectorID}},
		{"delete from tb_media_creators c where not exists (select 1 from tb_media_creator_roles r where r.creator_id = c.id)", nil},
		{"delete from tb_media_lists l where l.name = 'weread library' and not exists (select 1 from tb_media_list_items li where li.list_id = l.id)", nil},
	}
	if workID != nil {
		statements = append(statements,
			cleanupStatement{"delete from tb_media_external_refs where scope_type = 'work' and scope_id = ?", []any{*workID}},
			cleanupStatement{"delete from tb_media_works where id = ?", []any{*workID}},
		)
	}
	for _, queued := range enqueuer.enqueued {
		statements = append(statements, cleanupStatement{"delete from tb_jobs where id = ?", []any{queued.queueJob.ID}})
	}
	for _, id := range rawFileIDs {
		statements = append(statements, cleanupStatement{"delete from tb_raw_files where id = ?", []any{id}})
	}
	for _, statement := range statements {
		if err := db.Exec(statement.query, statement.args...).Error; err != nil {
			t.Errorf("cleanup %q: %v", statement.query, err)
		}
	}
}

// TestIntegrationDrifterCrossEditionWorkLinking verifies the Phase 3a work-linking
// seam: when an English edition already exists linked to Hardcover work 9002,
// importing the Chinese WeRead edition with matched_target.work_id=9002 attaches
// it to the existing canonical tb_media_works row rather than creating a duplicate.
// Both edition items remain distinct, their languages and publishers are preserved,
// and an edition relation edge links them in tb_media_relations.
func TestIntegrationDrifterCrossEditionWorkLinking(t *testing.T) {
	db := openImportsIntegrationDB(t)
	now := time.Now().UTC()
	const crossEditionBookID = "integration-cross-edition-klara"

	// Clean up any stale artifacts
	_ = db.Exec("delete from tb_media_relations where provider = 'weread'").Error
	_ = db.Exec("delete from tb_media_external_refs where provider = 'hardcover_work' and external_id = '9002'").Error

	// 1. Seed an English edition linked to hardcover_work/9002
	workID := uuid.New()
	if err := db.Create(&models.MediaWork{
		ID: workID, WorkKind: "work", PrimaryTitle: "Klara and the Sun",
		OriginalTitle: "Klara and the Sun", OriginalLanguage: "en",
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create canonical work: %v", err)
	}
	defer func() {
		_ = db.Exec("delete from tb_media_relations where provider = 'weread'").Error
		_ = db.Exec("delete from tb_media_items where work_id = ?", workID).Error
		_ = db.Exec("delete from tb_media_external_refs where scope_type = 'work' and scope_id = ?", workID).Error
		_ = db.Exec("delete from tb_media_works where id = ?", workID).Error
	}()

	workRefID := uuid.New()
	if err := db.Create(&models.MediaExternalRef{
		ID: workRefID, ScopeType: "work", ScopeID: workID,
		Provider: "hardcover_work", ExternalID: "9002", MatchedBy: "provider_id",
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create work external ref: %v", err)
	}

	englishItemID := uuid.New()
	if err := db.Create(&models.MediaItem{
		ID: englishItemID, WorkID: &workID, MediaType: "book", ItemRole: "book",
		Title: "Klara and the Sun", OriginalTitle: "Klara and the Sun", Language: "en",
		Publisher: "Faber & Faber", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create english edition item: %v", err)
	}
	defer func() {
		_ = db.Exec("delete from tb_media_external_refs where scope_type = 'item' and scope_id = ?", englishItemID).Error
		_ = db.Exec("delete from tb_media_items where id = ?", englishItemID).Error
	}()

	englishItemRefID := uuid.New()
	if err := db.Create(&models.MediaExternalRef{
		ID: englishItemRefID, ScopeType: "item", ScopeID: englishItemID,
		Provider: "hardcover", ExternalID: "424243", MatchedBy: "provider_id",
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create english item external ref: %v", err)
	}

	// 2. Now run WeRead import with matched_target.work_id = 9002
	crossEditionSnapshot := strings.ReplaceAll(
		strings.ReplaceAll(drifterIntegrationSnapshot, drifterIntegrationBookID, crossEditionBookID),
		`"work_id": 9001`, `"work_id": 9002`,
	)
	body := []byte(crossEditionSnapshot)
	rawID := uuid.New()
	hash := sha256.Sum256(body)
	storagePath := filepath.Join(t.TempDir(), "drifter_reading_cross_edition.json")
	if err := os.WriteFile(storagePath, body, 0o600); err != nil {
		t.Fatalf("write drifter snapshot: %v", err)
	}
	shaStr := hex.EncodeToString(hash[:])
	_ = db.Exec("delete from tb_import_snapshots where import_job_id in (select id from tb_import_jobs where raw_file_id in (select id from tb_raw_files where sha256 = ?))", shaStr)
	_ = db.Exec("delete from tb_import_jobs where raw_file_id in (select id from tb_raw_files where sha256 = ?)", shaStr)
	_ = db.Exec("delete from tb_raw_files where sha256 = ?", shaStr)
	if err := db.Create(&models.RawFile{
		ID: rawID, SHA256: shaStr, OriginalFilename: "drifter_reading_cross_edition.json",
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
		t.Fatalf("create import job: %v", err)
	}
	defer func() {
		_ = db.Exec("delete from tb_import_snapshots where import_job_id = ?", importJob.ID).Error
		_ = db.Exec("delete from tb_import_jobs where id = ?", importJob.ID).Error
		_ = db.Exec("delete from tb_jobs where id = ?", enqueuer.job.ID).Error
		_ = db.Exec("delete from tb_raw_files where id = ?", rawID).Error
	}()

	ctx := jobs.WithClaim(context.Background(), jobs.Claim{JobID: enqueuer.job.ID, WorkerID: "drifter-integration", Attempt: 1})
	if err := service.ProcessContext(ctx, importJob.ID); err != nil {
		t.Fatalf("process drifter import: %v", err)
	}

	// 3. Assertions
	var chineseRef models.MediaExternalRef
	if err := db.Where("provider = ? and external_id = ?", "weread", crossEditionBookID).First(&chineseRef).Error; err != nil {
		t.Fatalf("find chinese edition ref: %v", err)
	}
	chineseItemID := chineseRef.ScopeID
	defer func() {
		_ = db.Exec("delete from tb_media_creator_roles where scope_type = 'item' and scope_id = ?", chineseItemID).Error
		_ = db.Exec("delete from tb_media_consumption_events where media_item_id = ?", chineseItemID).Error
		_ = db.Exec("delete from tb_media_progress where media_item_id = ?", chineseItemID).Error
		_ = db.Exec("delete from tb_media_titles where scope_type = 'item' and scope_id = ?", chineseItemID).Error
		_ = db.Exec("delete from tb_media_external_refs where scope_type = 'item' and scope_id = ?", chineseItemID).Error
		_ = db.Exec("delete from tb_media_items where id = ?", chineseItemID).Error
	}()

	if chineseItemID == englishItemID {
		t.Fatalf("chinese edition collapsed into english item %v (items must remain separate editions)", englishItemID)
	}

	var chineseItem models.MediaItem
	if err := db.First(&chineseItem, "id = ?", chineseItemID).Error; err != nil {
		t.Fatalf("find chinese item: %v", err)
	}
	if chineseItem.WorkID == nil || *chineseItem.WorkID != workID {
		t.Fatalf("chinese item work_id = %v, want shared work_id %v", chineseItem.WorkID, workID)
	}
	if chineseItem.Title != "克拉拉与太阳" || chineseItem.Language != "zh" {
		t.Fatalf("chinese item title/language = %q/%q, want 克拉拉与太阳/zh", chineseItem.Title, chineseItem.Language)
	}

	// Check edition relation edge in tb_media_relations
	var rel models.MediaRelation
	if err := db.Where("from_id = ? and to_id = ? and relation_type = 'edition'", chineseItemID, englishItemID).First(&rel).Error; err != nil {
		t.Fatalf("edition relation edge between chinese and english items not found: %v", err)
	}
}
