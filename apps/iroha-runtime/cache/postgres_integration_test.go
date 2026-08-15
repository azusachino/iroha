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

func TestPostgresStoreCleanupIsBoundedAndKeepsCurrentGeneration(t *testing.T) {
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

	namespace := "cache_cleanup_test_" + uuid.NewString()
	t.Cleanup(func() {
		_ = db.Exec("delete from tb_cache_entries where namespace = ?", namespace).Error
		_ = db.Exec("delete from tb_cache_namespaces where namespace = ?", namespace).Error
	})

	client := NewWithStore(NewPostgresStore(db))
	for _, key := range []string{"old-a", "old-b", "old-c"} {
		Set(context.Background(), client, namespace, key, time.Minute, key)
	}
	if err := client.InvalidateNamespace(context.Background(), namespace); err != nil {
		t.Fatalf("invalidate namespace: %v", err)
	}
	_, generation, found := GetWithGeneration[string](context.Background(), client, namespace, "fresh")
	if found || generation != 2 {
		t.Fatalf("fresh generation lookup = found %v, generation %d; want false, 2", found, generation)
	}
	if stored := SetAtGeneration(context.Background(), client, namespace, "fresh", generation, time.Minute, "fresh"); !stored {
		t.Fatal("current generation value was not stored")
	}

	for i := 0; i < 3; i++ {
		result, err := client.Cleanup(context.Background(), 1)
		if err != nil {
			t.Fatalf("cleanup batch %d: %v", i, err)
		}
		if result.DeletedEntries > 1 {
			t.Fatalf("cleanup batch %d deleted %d entries, want at most 1", i, result.DeletedEntries)
		}
	}
	if value, found := Get[string](context.Background(), client, namespace, "fresh"); !found || value != "fresh" {
		t.Fatalf("current generation value = %q/%v, want fresh/true", value, found)
	}
	var remaining int64
	if err := db.Table("tb_cache_entries").Where("namespace = ?", namespace).Count(&remaining).Error; err != nil {
		t.Fatalf("count remaining cache entries: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("remaining cache entries = %d, want 1 current entry", remaining)
	}
}
