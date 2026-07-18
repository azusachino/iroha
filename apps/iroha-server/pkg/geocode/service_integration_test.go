//go:build integration

package geocode

import (
	"context"
	"os"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type recordingEnqueuer struct {
	calls int
}

func (e *recordingEnqueuer) EnqueueTx(_ *gorm.DB, _ string, _ any) (models.Job, error) {
	e.calls++
	return models.Job{}, nil
}

func TestEnqueueRefreshCoalescesCoordinate(t *testing.T) {
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
		t.Fatalf("get integration db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	key := CoordinateKey(35.681236, 139.767125)
	t.Cleanup(func() { _ = db.Exec("delete from tb_geocode_cache where coordinate_key = ?", key).Error })

	enqueuer := &recordingEnqueuer{}
	service := NewService(db, enqueuer, nil)
	if err := service.EnqueueRefresh(context.Background(), 35.681236, 139.767125); err != nil {
		t.Fatalf("enqueue first refresh: %v", err)
	}
	if err := service.EnqueueRefresh(context.Background(), 35.681236, 139.767125); err != nil {
		t.Fatalf("enqueue duplicate refresh: %v", err)
	}
	if enqueuer.calls != 1 {
		t.Fatalf("enqueuer calls = %d, want 1", enqueuer.calls)
	}
}
