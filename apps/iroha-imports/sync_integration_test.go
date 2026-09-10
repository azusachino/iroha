//go:build integration

package imports

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	connectorregistry "github.com/azusachino/iroha/apps/iroha-providers/connectors"
	providerregistry "github.com/azusachino/iroha/apps/iroha-providers/registry"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestIntegrationSyncRunLinksImportsAndRejectsOverlap(t *testing.T) {
	db := openImportsIntegrationDB(t)
	connectorRegistry, err := connectorregistry.New(syncTestConnector{})
	if err != nil {
		t.Fatalf("build connector registry: %v", err)
	}
	providers, err := providerregistry.New()
	if err != nil {
		t.Fatalf("build provider registry: %v", err)
	}
	enqueuer := syncTestEnqueuer{db: db}
	importService := NewServiceWithRegistry(db, nil, DefaultParserVersion, enqueuer, nil, providers)
	store := &syncTestSnapshotStore{db: db, dir: t.TempDir()}
	runner := NewSyncRunner(db, connectorRegistry, store, importService)

	if err := runner.Run(context.Background(), "sync-test", connector.Credentials{}); err != nil {
		t.Fatalf("run sync: %v", err)
	}
	var run models.MediaSyncRun
	if err := db.Where("connector_id = ?", "sync-test").First(&run).Error; err != nil {
		t.Fatalf("load sync run: %v", err)
	}
	if run.Status != syncRunStatusCompleted || run.FinishedAt == nil {
		t.Fatalf("sync run = %#v, want completed", run)
	}
	var child models.ImportJob
	if err := db.Where("sync_run_id = ?", run.ID).First(&child).Error; err != nil {
		t.Fatalf("load child import: %v", err)
	}
	if child.ParserKind != "bangumi" {
		t.Fatalf("child parser kind = %q, want bangumi", child.ParserKind)
	}
	var state models.MediaSyncState
	if err := db.Where("connector_id = ?", "sync-test").First(&state).Error; err != nil {
		t.Fatalf("load sync state: %v", err)
	}
	if state.Status != mediaSyncStatusCompleted || state.LastFetchedAt == nil {
		t.Fatalf("sync state = %#v, want completed with fetched timestamp", state)
	}

	active, err := runner.beginSyncRun("sync-test")
	if err != nil {
		t.Fatalf("begin active overlap run: %v", err)
	}
	t.Cleanup(func() { _ = runner.failSyncRun(active.ID, errors.New("test cleanup")) })
	if err := runner.Run(context.Background(), "sync-test", connector.Credentials{}); !errors.Is(err, ErrSyncAlreadyRunning) {
		t.Fatalf("overlapping sync error = %v, want ErrSyncAlreadyRunning", err)
	}
}

type syncTestConnector struct{}

func (syncTestConnector) Descriptor() connector.Descriptor {
	return connector.Descriptor{ID: "sync-test", DisplayName: "Sync test", SourceKind: "bangumi"}
}

func (syncTestConnector) Fetch(context.Context, connector.Credentials, *connector.Cursor) (connector.Snapshot, *connector.Cursor, error) {
	return connector.Snapshot{ContentType: "application/json", Body: []byte(`{"data":[]}`), SourceKind: "bangumi", Filename: "sync-test.json", ObservedAt: time.Now().UTC()}, nil, nil
}

type syncTestSnapshotStore struct {
	db  *gorm.DB
	dir string
}

func (s *syncTestSnapshotStore) StoreSnapshot(_ context.Context, snapshot connector.Snapshot) (models.RawFile, error) {
	id, err := ids.New()
	if err != nil {
		return models.RawFile{}, err
	}
	path := filepath.Join(s.dir, id.String()+".json")
	if err := os.WriteFile(path, snapshot.Body, 0o600); err != nil {
		return models.RawFile{}, err
	}
	rawFile := models.RawFile{ID: id, SHA256: id.String(), OriginalFilename: snapshot.Filename, ContentType: snapshot.ContentType, SizeBytes: int64(len(snapshot.Body)), StoragePath: path, SourceKind: snapshot.SourceKind, UploadedVia: "connector", CreatedAt: time.Now().UTC()}
	return rawFile, s.db.Create(&rawFile).Error
}

type syncTestEnqueuer struct {
	db *gorm.DB
}

func (e syncTestEnqueuer) EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return models.Job{}, err
	}
	job := models.Job{ID: uuid.New(), Kind: kind, Status: jobs.StatusQueued, PayloadJSON: encoded, MaxAttempts: jobs.DefaultMaxAttempts, RunAfter: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if tx == nil {
		tx = e.db
	}
	return job, tx.Create(&job).Error
}
