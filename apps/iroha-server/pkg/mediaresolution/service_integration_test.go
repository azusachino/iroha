//go:build integration

package mediaresolution

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func seedResolutionTask(t *testing.T, db *gorm.DB, status string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	task := models.MediaResolutionTask{
		ID: id, TaskType: "dedupe_candidate", Status: status,
		CandidatesJSON: json.RawMessage(`{"candidates":["` + uuid.NewString() + `"]}`),
		ResolutionJSON: json.RawMessage(`{}`),
		CreatedAt:      time.Now().UTC(),
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("seed resolution task: %v", err)
	}
	t.Cleanup(func() {
		db.Delete(&models.MediaResolutionTask{}, "id = ?", id)
	})
	return id
}

func TestService_ListDefaultsToOpen(t *testing.T) {
	db := openIntegrationDB(t)
	openID := seedResolutionTask(t, db, StatusOpen)
	seedResolutionTask(t, db, StatusResolved)

	svc := NewService(db)
	rows, err := svc.List(ListFilters{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, row := range rows {
		if row.Status != StatusOpen {
			t.Fatalf("list without a status filter returned a %s row", row.Status)
		}
	}
	found := false
	for _, row := range rows {
		if row.ID == openID {
			found = true
		}
	}
	if !found {
		t.Fatal("seeded open task missing from default list")
	}
}

func TestService_ResolveClosesOpenTaskOnce(t *testing.T) {
	db := openIntegrationDB(t)
	id := seedResolutionTask(t, db, StatusOpen)
	svc := NewService(db)

	resolution := json.RawMessage(`{"decision":"duplicate","canonical_item_id":"abc"}`)
	task, err := svc.Resolve(id, StatusResolved, resolution)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if task.Status != StatusResolved || task.ResolvedAt == nil {
		t.Fatalf("resolved task = %+v, want status=resolved with resolved_at set", task)
	}
	// jsonb round-trips through Postgres reformatted (added spacing), so
	// compare decoded values rather than raw bytes.
	var got, want map[string]any
	if err := json.Unmarshal(task.ResolutionJSON, &got); err != nil {
		t.Fatalf("unmarshal stored resolution_json: %v", err)
	}
	if err := json.Unmarshal(resolution, &want); err != nil {
		t.Fatalf("unmarshal expected resolution: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolution_json = %v, want %v", got, want)
	}

	// Resolving an already-closed task must not silently succeed again.
	if _, err := svc.Resolve(id, StatusDismissed, nil); err != ErrNotFound {
		t.Fatalf("re-resolving a closed task: err = %v, want ErrNotFound", err)
	}
}
