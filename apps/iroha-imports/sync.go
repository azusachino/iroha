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
	cursor, err := decodeCursor(state.CursorJSON)
	if err != nil {
		return s.failSyncState(state, err)
	}
	if err := s.updateSyncState(state, mediaSyncStatusRunning, nil, cursor, false); err != nil {
		return err
	}

	for {
		snapshot, nextCursor, fetchErr := item.Fetch(ctx, credentials, cursor)
		if fetchErr != nil {
			return s.failSyncState(state, fetchErr)
		}
		if snapshot.SourceKind == "" {
			snapshot.SourceKind = item.Descriptor().SourceKind
		}
		if snapshot.SourceKind != item.Descriptor().SourceKind {
			return s.failSyncState(state, fmt.Errorf("connector %q returned source kind %q, want %q", connectorID, snapshot.SourceKind, item.Descriptor().SourceKind))
		}
		if snapshot.Filename == "" {
			snapshot.Filename = connectorID + ".json"
		}
		rawFile, err := s.snapshots.StoreSnapshot(ctx, snapshot)
		if err != nil {
			return s.failSyncState(state, err)
		}
		if _, err := s.imports.Create(CreateInput{
			RawFileID:  ids.Encode(ids.RawFilePrefix, rawFile.ID),
			ParserKind: snapshot.SourceKind,
		}); err != nil {
			return s.failSyncState(state, err)
		}

		if err := s.updateSyncState(state, mediaSyncStatusRunning, nil, nextCursor, true); err != nil {
			return err
		}
		if nextCursor == nil {
			return s.updateSyncState(state, mediaSyncStatusCompleted, nil, nil, true)
		}
		cursor = nextCursor
	}
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

func (s *SyncRunner) failSyncState(state models.MediaSyncState, cause error) error {
	if err := s.updateSyncState(state, mediaSyncStatusFailed, cause, nil, false); err != nil {
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
