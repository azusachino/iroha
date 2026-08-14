//go:build integration

package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresStoreRoundTripAndNamespaceInvalidation(t *testing.T) {
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

	namespace := "cache_test_" + uuid.NewString()
	t.Cleanup(func() {
		_ = db.Exec("delete from tb_cache_entries where namespace = ?", namespace).Error
		_ = db.Exec("delete from tb_cache_namespaces where namespace = ?", namespace).Error
	})

	client := NewWithStore(NewPostgresStore(db))
	Set(context.Background(), client, namespace, "key", time.Minute, map[string]string{"value": "one"})
	value, ok := Get[map[string]string](context.Background(), client, namespace, "key")
	if !ok || value["value"] != "one" {
		t.Fatalf("got %#v/%v, want cached value", value, ok)
	}
	_, generation, found := GetWithGeneration[map[string]string](context.Background(), client, namespace, "key")
	if !found || generation < 1 {
		t.Fatalf("generation lookup = found %v, generation %d; want cached value and positive generation", found, generation)
	}

	if err := client.InvalidateNamespace(context.Background(), namespace); err != nil {
		t.Fatalf("invalidate namespace: %v", err)
	}
	if _, ok := Get[map[string]string](context.Background(), client, namespace, "key"); ok {
		t.Fatal("invalidated entry was still visible")
	}
	if stored := SetAtGeneration(context.Background(), client, namespace, "key", generation, time.Minute, map[string]string{"value": "stale"}); stored {
		t.Fatal("stale generation write was stored")
	}
	_, currentGeneration, found := GetWithGeneration[map[string]string](context.Background(), client, namespace, "key")
	if found || currentGeneration != generation+1 {
		t.Fatalf("post-invalidation lookup = found %v, generation %d; want false, %d", found, currentGeneration, generation+1)
	}
}
