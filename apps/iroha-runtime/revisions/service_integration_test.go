//go:build integration

package revisions

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestIntegrationBumpIsTransactional(t *testing.T) {
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

	namespace := "revision_test_" + uuid.NewString()
	t.Cleanup(func() { _ = db.Exec("delete from tb_read_revisions where namespace = ?", namespace).Error })

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := Bump(tx, namespace, namespace); err != nil {
			return err
		}
		current, err := Read(tx, namespace)
		if err != nil {
			return err
		}
		if current[namespace] != 1 {
			t.Fatalf("in-transaction revision = %d, want 1", current[namespace])
		}
		return gorm.ErrInvalidData
	}); err == nil {
		t.Fatal("rollback transaction succeeded")
	}
	current, err := Read(db, namespace)
	if err != nil {
		t.Fatalf("read rolled-back revision: %v", err)
	}
	if current[namespace] != 0 {
		t.Fatalf("rolled-back revision = %d, want 0", current[namespace])
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return Bump(tx, namespace)
	}); err != nil {
		t.Fatalf("commit revision: %v", err)
	}
	current, err = Read(db, namespace)
	if err != nil {
		t.Fatalf("read committed revision: %v", err)
	}
	if current[namespace] != 1 {
		t.Fatalf("committed revision = %d, want 1", current[namespace])
	}
}
