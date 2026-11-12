package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	connectorregistry "github.com/azusachino/iroha/apps/iroha-providers/connectors"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"gorm.io/gorm"
)

const (
	mediaSyncStatusRunning   = "running"
	mediaSyncStatusCompleted = "completed"
	mediaSyncStatusFailed    = "failed"
)

type SnapshotStore interface {
	StoreSnapshot(context.Context, connector.Snapshot) (models.RawFile, error)
}

type SyncRunner struct {
	db         *gorm.DB
	connectors *connectorregistry.Registry
	snapshots  SnapshotStore
	imports    *Service
}

func NewSyncRunner(db *gorm.DB, connectors *connectorregistry.Registry, snapshots SnapshotStore, imports *Service) *SyncRunner {
	return &SyncRunner{db: db, connectors: connectors, snapshots: snapshots, imports: imports}
}

// syncRun is one media sync run's shared bookkeeping: the run row, the
// connector's resumable state, the pages stored so far, and the phase the run
// is currently in.
type syncRun struct {
	runner      *SyncRunner
	item        connector.Connector
	run         models.MediaSyncRun
	state       models.MediaSyncState
	credentials connector.Credentials
	// cursor is where a resumed full snapshot would restart. It stays nil for
	// an incremental run: a changes feed resumes from its watermark instead.
	cursor      *connector.Cursor
	phase       syncPhase
	lastRawFile *models.RawFile
}

// syncPhase is what the current attempt is doing, in the terms coverage is
// reported in: how the pages were fetched and which window they claim.
type syncPhase struct {
	ingestionMode string
	windowStart   time.Time
	runStart      time.Time
}

// Run executes one connector sync. A connector that can serve changes follows
// the follower state machine; every other connector reads its whole current
// state, as before.
func (s *SyncRunner) Run(ctx context.Context, connectorID string, credentials connector.Credentials) error {
	if connectorID == "" {
		return errors.New("connector id is required")
	}
	item, ok := s.connectors.Get(connectorID)
	if !ok {
		return fmt.Errorf("connector %q is not registered", connectorID)
	}
	state, err := s.syncState(connectorID)
	if err != nil {
		return err
	}
	run, err := s.beginSyncRun(connectorID)
	if err != nil {
		return err
	}
	cursor, err := decodeCursor(state.CursorJSON)
	if err != nil {
		_ = s.failSyncRun(run.ID, err)
		return s.failSyncState(state, err, nil)
	}
	if err := s.updateSyncState(state, mediaSyncStatusRunning, nil, cursor, false); err != nil {
		_ = s.failSyncRun(run.ID, err)
		return err
	}
	job := &syncRun{runner: s, item: item, run: run, state: state, credentials: credentials}
	return s.runFollower(ctx, job, cursor)
}

// runFollower executes the follower state machine. A connector that can serve
// changes resumes from the watermark its last completed run proved; when the
// gateway can no longer serve that watermark the same run falls back to a full
// snapshot, so the horizon signal is a state transition and never a failure.
//
// The sync-state cursor means "a full snapshot stopped mid-pagination", while a
// receipt's ordering basis means "the incremental watermark". Keeping them
// separate is what makes a resumed snapshot and a resumed changes feed
// distinguishable.
func (s *SyncRunner) runFollower(ctx context.Context, job *syncRun, cursor *connector.Cursor) error {
	runStart := time.Now().UTC()
	windowStart := runStart
	if windowed, ok := job.item.(connector.CoverageWindowProvider); ok {
		windowStart = windowed.CoverageWindowStart(cursor, runStart)
	}
	changes, servesChanges := job.item.(connector.ChangeFetcher)
	if servesChanges && cursor == nil {
		since, resumable, err := s.resumeWatermark(job.item.Descriptor().ID)
		if err != nil {
			return job.fail(err)
		}
		if resumable {
			err := s.runChanges(ctx, job, changes, since, runStart)
			if err == nil {
				return nil
			}
			if !errors.Is(err, connector.ErrSnapshotRequired) {
				return err
			}
			s.logNote("change watermark horizon exceeded, falling back to full snapshot",
				"connector_id", job.item.Descriptor().ID,
				"watermark", since.Format(time.RFC3339))
			// The full snapshot starts at its first page, never mid-feed.
			cursor = nil
			windowStart = runStart
		}
	}
	return s.runSnapshot(ctx, job, cursor, windowStart, runStart)
}

// resumeWatermark returns the ordering fact the last completed run proved. The
// resume point belongs to a run, not to a page: pages a failed run stored are
// kept as evidence but cannot move it. A source with no completed run, or one
// whose ordering basis is not a readable timestamp, has nothing to resume from
// and must read a full snapshot.
func (s *SyncRunner) resumeWatermark(connectorID string) (time.Time, bool, error) {
	var orderingBasis string
	result := s.db.Raw(`select r.ordering_basis
from tb_media_sync_runs m
join tb_import_jobs j on j.sync_run_id = m.id
join tb_source_receipts r on r.raw_file_id = j.raw_file_id
where m.connector_id = ? and m.status = ? and r.ordering_basis <> ''
order by m.finished_at desc, r.received_at desc
limit 1`, connectorID, syncRunStatusCompleted).Scan(&orderingBasis)
	if result.Error != nil {
		return time.Time{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return time.Time{}, false, nil
	}
	since, err := time.Parse(time.RFC3339, orderingBasis)
	if err != nil {
		s.logNote("unreadable source watermark; falling back to full snapshot",
			"connector_id", connectorID, "ordering_basis", orderingBasis)
		return time.Time{}, false, nil
	}
	return since.UTC(), true, nil
}

// runSnapshot reads the connector's whole current state, page by page. It is
// both the bootstrap for a change-feed connector and the fallback when its
// watermark is out of horizon.
func (s *SyncRunner) runSnapshot(ctx context.Context, job *syncRun, cursor *connector.Cursor, windowStart, runStart time.Time) error {
	job.cursor = cursor
	job.phase = syncPhase{ingestionMode: rawfiles.IngestionModeFullSnapshot, windowStart: windowStart, runStart: runStart}
	for {
		snapshot, nextCursor, err := job.item.Fetch(ctx, job.credentials, cursor)
		if err != nil {
			return job.fail(err)
		}
		if err := job.storePage(ctx, snapshot); err != nil {
			return job.fail(err)
		}
		if err := s.updateSyncState(job.state, mediaSyncStatusRunning, nil, nextCursor, true); err != nil {
			return job.fail(err)
		}
		if nextCursor == nil {
			return job.complete(nil)
		}
		cursor = nextCursor
		job.cursor = nextCursor
	}
}

// runChanges fetches the incremental feed from since until it is exhausted.
// Each page is stored as its own evidence and queued for parsing, so a changed
// book is updated without re-reading the whole shelf. The pages' watermark
// becomes the next run's resume point only once this run completes; a run that
// fails part-way leaves the resume point where it was.
//
// It returns connector.ErrSnapshotRequired unwrapped when the feed refuses the
// watermark, so the caller can fall back inside the same run.
func (s *SyncRunner) runChanges(ctx context.Context, job *syncRun, changes connector.ChangeFetcher, since, runStart time.Time) error {
	job.phase = syncPhase{ingestionMode: rawfiles.IngestionModeIncremental, windowStart: since, runStart: runStart}
	var cursor *connector.Cursor
	for {
		page, err := changes.FetchChanges(ctx, job.credentials, since, cursor)
		if err != nil {
			if errors.Is(err, connector.ErrSnapshotRequired) {
				return err
			}
			return job.fail(err)
		}
		// The watermark a changes page proves is the receipt's ordering basis,
		// whatever the connector named it on the page.
		if page.Watermark != "" {
			page.Snapshot.OrderingBasis = page.Watermark
		}
		if err := job.storePage(ctx, page.Snapshot); err != nil {
			return job.fail(err)
		}
		if err := s.updateSyncState(job.state, mediaSyncStatusRunning, nil, nil, true); err != nil {
			return job.fail(err)
		}
		if page.Cursor == nil {
			return job.complete(nil)
		}
		cursor = page.Cursor
	}
}

// storePage persists one fetched page as raw evidence and queues its import.
// The connector's own ingestion mode and ordering basis travel with the page
// into its receipt; a connector that states neither gets the run's mode.
func (s *syncRun) storePage(ctx context.Context, snapshot connector.Snapshot) error {
	descriptor := s.item.Descriptor()
	if snapshot.SourceKind == "" {
		snapshot.SourceKind = descriptor.SourceKind
	}
	if snapshot.SourceKind != descriptor.SourceKind {
		return fmt.Errorf("connector %q returned source kind %q, want %q", descriptor.ID, snapshot.SourceKind, descriptor.SourceKind)
	}
	if snapshot.Filename == "" {
		snapshot.Filename = descriptor.ID + ".json"
	}
	if snapshot.IngestionMode == "" {
		snapshot.IngestionMode = s.phase.ingestionMode
	}
	rawFile, err := s.runner.snapshots.StoreSnapshot(ctx, snapshot)
	if err != nil {
		return err
	}
	s.lastRawFile = &rawFile
	if _, err := s.runner.imports.Create(CreateInput{
		RawFileID:  ids.Encode(ids.RawFilePrefix, rawFile.ID),
		ParserKind: snapshot.SourceKind,
		SyncRunID:  &s.run.ID,
	}); err != nil {
		return err
	}
	return s.runner.touchSyncRun(s.run.ID)
}

// fail closes a run that stopped early. The pages it did store are recorded as
// a partial assertion so nothing later mistakes them for a covered window.
func (s *syncRun) fail(cause error) error {
	if s.lastRawFile != nil {
		_ = s.runner.recordSyncCoverage(*s.lastRawFile, s.item.Descriptor().SourceKind, s.phase.ingestionMode, s.phase.windowStart, s.phase.runStart, coverageCompletenessPartial)
	}
	if err := s.runner.failSyncRun(s.run.ID, cause); err != nil {
		return err
	}
	return s.runner.failSyncState(s.state, cause, s.cursor)
}

// complete closes a run that read its whole window, recording what it proved.
func (s *syncRun) complete(nextCursor *connector.Cursor) error {
	if resumable, ok := s.item.(connector.ResumeCursorProvider); ok {
		nextCursor = resumable.ResumeCursor()
	}
	if err := s.runner.updateSyncState(s.state, mediaSyncStatusCompleted, nil, nextCursor, true); err != nil {
		_ = s.runner.failSyncRun(s.run.ID, err)
		return err
	}
	if s.lastRawFile != nil {
		if err := s.runner.recordSyncCoverage(*s.lastRawFile, s.item.Descriptor().SourceKind, s.phase.ingestionMode, s.phase.windowStart, s.phase.runStart, coverageCompletenessCovered); err != nil {
			_ = s.runner.failSyncRun(s.run.ID, err)
			return err
		}
	}
	return s.runner.completeSyncRun(s.run.ID)
}

// logNote records a state-machine note when the runner has a service to log
// through.
func (s *SyncRunner) logNote(message string, args ...any) {
	if s.imports == nil {
		return
	}
	s.imports.logger.Info(message, args...)
}

func (s *SyncRunner) syncState(connectorID string) (models.MediaSyncState, error) {
	var state models.MediaSyncState
	result := s.db.Where("connector_id = ?", connectorID).First(&state)
	if result.Error == nil {
		return state, nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return models.MediaSyncState{}, result.Error
	}
	id, err := ids.New()
	if err != nil {
		return models.MediaSyncState{}, err
	}
	now := time.Now().UTC()
	state = models.MediaSyncState{ID: id, ConnectorID: connectorID, CursorJSON: json.RawMessage(`{}`), Status: mediaSyncStatusCompleted, CreatedAt: now, UpdatedAt: now}
	return state, s.db.Create(&state).Error
}

func (s *SyncRunner) updateSyncState(state models.MediaSyncState, status string, cause error, cursor *connector.Cursor, fetched bool) error {
	encoded, err := encodeCursor(cursor)
	if err != nil {
		return err
	}
	updates := map[string]any{"status": status, "cursor_json": encoded, "updated_at": time.Now().UTC()}
	if cause != nil {
		message := cause.Error()
		updates["last_error"] = &message
	} else {
		updates["last_error"] = nil
	}
	if fetched {
		now := time.Now().UTC()
		updates["last_fetched_at"] = &now
	}
	return s.db.Model(&models.MediaSyncState{}).Where("id = ?", state.ID).Updates(updates).Error
}

func (s *SyncRunner) failSyncState(state models.MediaSyncState, cause error, cursor *connector.Cursor) error {
	if err := s.updateSyncState(state, mediaSyncStatusFailed, cause, cursor, false); err != nil {
		return err
	}
	return cause
}

func encodeCursor(cursor *connector.Cursor) (json.RawMessage, error) {
	if cursor == nil {
		return json.RawMessage(`{}`), nil
	}
	encoded, err := json.Marshal(cursor)
	return encoded, err
}

func decodeCursor(raw json.RawMessage) (*connector.Cursor, error) {
	if len(raw) == 0 || string(raw) == "{}" {
		return nil, nil
	}
	var cursor connector.Cursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return nil, err
	}
	return &cursor, nil
}
