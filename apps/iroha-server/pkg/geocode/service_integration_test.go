//go:build integration

package geocode

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
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

type recordingCacheStore struct {
	invalidated []string
}

func (s *recordingCacheStore) Get(_ context.Context, _, _ string) ([]byte, bool, error) {
	return nil, false, nil
}

func (s *recordingCacheStore) Set(_ context.Context, _, _ string, _ []byte, _ time.Duration) error {
	return nil
}

func (s *recordingCacheStore) InvalidateNamespace(_ context.Context, namespace string) error {
	s.invalidated = append(s.invalidated, namespace)
	return nil
}

func (s *recordingCacheStore) Close() error { return nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
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

func TestRefreshInvalidatesGeocodeDependentCaches(t *testing.T) {
	db := openGeocodeIntegrationDB(t)
	latitude, longitude := 12.345, 67.89
	key := CoordinateKey(latitude, longitude)
	t.Cleanup(func() { _ = db.Exec("delete from tb_geocode_cache where coordinate_key = ?", key).Error })

	store := &recordingCacheStore{}
	service := NewService(db, nil, cache.NewWithStore(store))
	service.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"address":{"city":"Tokyo"}}`)),
			Request:    request,
		}, nil
	})}

	if err := service.Refresh(context.Background(), RefreshPayload{CoordinateKey: key, Latitude: latitude, Longitude: longitude}); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	want := []string{cache.NamespaceActivities, cache.NamespacePublicRoutes}
	if len(store.invalidated) != len(want) {
		t.Fatalf("invalidated namespaces = %v, want %v", store.invalidated, want)
	}
	for index := range want {
		if store.invalidated[index] != want[index] {
			t.Fatalf("invalidated namespaces = %v, want %v", store.invalidated, want)
		}
	}
}

func openGeocodeIntegrationDB(t *testing.T) *gorm.DB {
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
		t.Fatalf("get integration db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
