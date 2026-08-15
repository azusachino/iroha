package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
)

var readScopeNow = time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)

func TestResolveReadScopeScalarForms(t *testing.T) {
	tests := []struct {
		name string
		date string
		kind ScopeKind
		from string
		to   string
	}{
		{"year", "2026", ScopeYear, "2026-01-01T00:00:00+09:00", "2027-01-01T00:00:00+09:00"},
		{"month", "2026-08", ScopeMonth, "2026-08-01T00:00:00+09:00", "2026-09-01T00:00:00+09:00"},
		{"day", "2026-08-13", ScopeDay, "2026-08-13T00:00:00+09:00", "2026-08-14T00:00:00+09:00"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scope, err := ResolveReadScope(ReadScopeInput{Date: test.date}, readScopeNow)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if scope.Kind != test.kind || scope.Timezone != DefaultReadScopeTimezone {
				t.Fatalf("scope = %+v", scope)
			}
			if scope.Instant.From.Format(time.RFC3339) != test.from || scope.Instant.ToExclusive.Format(time.RFC3339) != test.to {
				t.Fatalf("instant = %s .. %s", scope.Instant.From.Format(time.RFC3339), scope.Instant.ToExclusive.Format(time.RFC3339))
			}
		})
	}
}

func TestResolveReadScopeRangeUsesHalfOpenBoundsAndTimezone(t *testing.T) {
	scope, err := ResolveReadScope(ReadScopeInput{
		From:     "2026-08-01",
		To:       "2026-09-01",
		Timezone: "America/New_York",
	}, readScopeNow)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if scope.Kind != ScopeRange || scope.Timezone != "America/New_York" {
		t.Fatalf("scope = %+v", scope)
	}
	if got := scope.Calendar.From.Format(dateLayout); got != "2026-08-01" {
		t.Errorf("calendar from = %s", got)
	}
	if got := scope.Calendar.ToExclusive.Format(dateLayout); got != "2026-09-01" {
		t.Errorf("calendar to = %s", got)
	}
	if got := scope.Instant.From.Format(time.RFC3339); got != "2026-08-01T00:00:00-04:00" {
		t.Errorf("instant from = %s", got)
	}
	if got := scope.Instant.ToExclusive.Format(time.RFC3339); got != "2026-09-01T00:00:00-04:00" {
		t.Errorf("instant to = %s", got)
	}
}

func TestResolveReadScopeLifetimeIsExplicit(t *testing.T) {
	scope, err := ResolveReadScope(ReadScopeInput{Lifetime: true}, readScopeNow)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if scope.Kind != ScopeLifetime || !scope.Instant.From.IsZero() || !scope.Calendar.From.IsZero() {
		t.Fatalf("scope = %+v", scope)
	}
}

func TestResolveReadScopeRejectsInvalidAndConflictingInputs(t *testing.T) {
	tests := []ReadScopeInput{
		{Date: "2026-2"},
		{Date: "2026-02-30"},
		{Date: "2026-08", From: "2026-08-01", To: "2026-09-01"},
		{From: "2026-08-01"},
		{From: "2026-08-02", To: "2026-08-01"},
		{Lifetime: true, Date: "2026"},
		{Timezone: "Not/A/Timezone", Date: "2026"},
	}
	for _, input := range tests {
		if _, err := ResolveReadScope(input, readScopeNow); err == nil {
			t.Errorf("input %+v succeeded", input)
		}
	}
}

func TestResolveReadScopeRejectsFutureAndExcessiveRanges(t *testing.T) {
	for _, input := range []ReadScopeInput{
		{Date: "2026-08-16"},
		{Date: "2027"},
		{From: "2026-08-16", To: "2026-08-17"},
		{From: "2010-01-01", To: "2026-08-16"},
	} {
		_, err := ResolveReadScope(input, readScopeNow)
		if err == nil {
			t.Errorf("input %+v succeeded", input)
		}
	}
	if _, err := ResolveReadScope(ReadScopeInput{Date: "2026-08-16"}, time.Time{}); !errors.Is(err, ErrMissingScopeNow) {
		t.Fatalf("zero now error = %v, want ErrMissingScopeNow", err)
	}
}

func TestResolveReadScopeAllowsCurrentEnvelopeAndDSTBoundary(t *testing.T) {
	scope, err := ResolveReadScope(ReadScopeInput{Date: "2026-08", Timezone: "America/New_York"}, readScopeNow)
	if err != nil {
		t.Fatalf("resolve current month: %v", err)
	}
	if got := scope.Instant.ToExclusive.Format(time.RFC3339); got != "2026-09-01T00:00:00-04:00" {
		t.Errorf("current month end = %s", got)
	}

	now := time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC)
	scope, err = ResolveReadScope(ReadScopeInput{Date: "2026-11", Timezone: "America/New_York"}, now)
	if err != nil {
		t.Fatalf("resolve DST month: %v", err)
	}
	if got := scope.Instant.ToExclusive.Format(time.RFC3339); got != "2026-12-01T00:00:00-05:00" {
		t.Errorf("DST month end = %s", got)
	}
}

func TestScopeFilterAdaptersKeepCalendarAndInstantImplementationsSeparate(t *testing.T) {
	server := &Server{
		deps: Dependencies{
			Config: config.Config{Server: config.ServerConfig{Timezone: "America/New_York"}},
		},
		now: func() time.Time { return readScopeNow },
	}

	dailyRecorder := httptest.NewRecorder()
	dailyFilters, ok := server.parseDailyFilters(
		dailyRecorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/daily?date=2026-08-01", nil),
	)
	if !ok || dailyFilters.From == nil || dailyFilters.To == nil {
		t.Fatalf("daily filters = %+v, ok=%t", dailyFilters, ok)
	}
	if got := dailyFilters.From.Format(dateLayout); got != "2026-08-01" {
		t.Fatalf("daily from = %s", got)
	}

	activityRecorder := httptest.NewRecorder()
	activityFilters, ok := server.parseActivityFilters(
		activityRecorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/activities?date=2026-08-01", nil),
	)
	if !ok || activityFilters.StartedFrom == nil || activityFilters.StartedTo == nil {
		t.Fatalf("activity filters = %+v, ok=%t", activityFilters, ok)
	}
	if got := activityFilters.StartedFrom.Format(time.RFC3339); got != "2026-08-01T00:00:00-04:00" {
		t.Fatalf("activity from = %s", got)
	}

	rangeRequest := func(path string) *http.Request {
		return httptest.NewRequest(http.MethodGet, path+"?from=2026-08-01&to=2026-08-03&timezone=America%2FNew_York", nil)
	}

	sleepRecorder := httptest.NewRecorder()
	sleepFilters, ok := server.parseSleepFilters(sleepRecorder, rangeRequest("/api/v1/sleep"))
	if !ok || sleepFilters.From == nil || sleepFilters.To == nil {
		t.Fatalf("sleep filters = %+v, ok=%t", sleepFilters, ok)
	}
	if got := sleepFilters.From.Format(dateLayout); got != "2026-08-01" {
		t.Fatalf("sleep from = %s", got)
	}
	if got := sleepFilters.To.Format(dateLayout); got != "2026-08-03" {
		t.Fatalf("sleep to = %s", got)
	}

	expenseRecorder := httptest.NewRecorder()
	expenseFilters, ok := server.parseExpenseFilters(expenseRecorder, rangeRequest("/api/v1/expenses"))
	if !ok || expenseFilters.From == nil || expenseFilters.To == nil {
		t.Fatalf("expense filters = %+v, ok=%t", expenseFilters, ok)
	}
	if got := expenseFilters.From.Format(dateLayout); got != "2026-08-01" {
		t.Fatalf("expense from = %s", got)
	}

	mediaRecorder := httptest.NewRecorder()
	mediaFilters, ok := server.parseMediaEventFilters(mediaRecorder, rangeRequest("/api/v1/media/events"))
	if !ok || mediaFilters.From == nil || mediaFilters.To == nil {
		t.Fatalf("media filters = %+v, ok=%t", mediaFilters, ok)
	}
	if got := mediaFilters.From.Format(time.RFC3339); got != "2026-08-01T00:00:00-04:00" {
		t.Fatalf("media from = %s", got)
	}
	if got := mediaFilters.To.Format(time.RFC3339); got != "2026-08-03T00:00:00-04:00" {
		t.Fatalf("media to = %s", got)
	}
}
