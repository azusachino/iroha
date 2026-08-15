//go:build integration

package expenses

import (
	"testing"
	"time"
)

func TestServiceBoundsCapsMaxAtNowAndIgnoresFutureRowsForMax(t *testing.T) {
	db := openIntegrationDB(t)
	clearExpenses(t, db)
	t.Cleanup(func() { clearExpenses(t, db) })
	svc := NewService(db)

	now := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)

	if minDate, maxDate, ok, err := svc.Bounds(now, "UTC"); err != nil {
		t.Fatalf("bounds on empty table: %v", err)
	} else if ok || minDate != "" || maxDate != "" {
		t.Fatalf("bounds on empty table = (%q, %q, %v), want (\"\", \"\", false)", minDate, maxDate, ok)
	}

	if _, err := svc.Create(testCreateInput("bounds-past", time.Date(2019, time.March, 2, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("create past expense: %v", err)
	}
	if _, err := svc.Create(testCreateInput("bounds-future", time.Date(2099, time.April, 1, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("create future expense: %v", err)
	}

	minDate, maxDate, ok, err := svc.Bounds(now, "UTC")
	if err != nil {
		t.Fatalf("bounds: %v", err)
	}
	if !ok {
		t.Fatal("bounds ok = false, want true")
	}
	if minDate != "2019-03-02" {
		t.Fatalf("min = %q, want 2019-03-02", minDate)
	}
	if maxDate != "2026-08-15" {
		t.Fatalf("max = %q, want 2026-08-15 (capped at now, not the 2099 row)", maxDate)
	}
}

func TestServiceBoundsReportsNoDataWhenOnlyFutureRowsExist(t *testing.T) {
	db := openIntegrationDB(t)
	clearExpenses(t, db)
	t.Cleanup(func() { clearExpenses(t, db) })
	svc := NewService(db)

	if _, err := svc.Create(testCreateInput("bounds-only-future", time.Date(2099, time.April, 1, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("create future expense: %v", err)
	}

	now := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)
	minDate, maxDate, ok, err := svc.Bounds(now, "UTC")
	if err != nil {
		t.Fatalf("bounds: %v", err)
	}
	if ok || minDate != "" || maxDate != "" {
		t.Fatalf("bounds with only future rows = (%q, %q, %v), want (\"\", \"\", false) -- a future-only table must never widen the navigable range", minDate, maxDate, ok)
	}
}
