package httpapi

import (
	"net/http/httptest"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/google/uuid"
)

func TestParseMediaFiltersSupportsCompletedYear(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/media?family=anime&completed_year=2026", nil)
	filters, ok := parseMediaFilters(httptest.NewRecorder(), request)
	if !ok || filters.CompletedYear == nil || *filters.CompletedYear != 2026 {
		t.Fatalf("filters = %+v, ok = %t", filters, ok)
	}
}

func TestParseMediaFiltersRejectsUnknownFamilyAndYear(t *testing.T) {
	for _, query := range []string{"family=cartoon", "completed_year=20x6", "completed_year=0"} {
		request := httptest.NewRequest("GET", "/api/v1/media?"+query, nil)
		recorder := httptest.NewRecorder()
		if _, ok := parseMediaFilters(recorder, request); ok || recorder.Code != 400 {
			t.Fatalf("query %q: ok = %t, status = %d", query, ok, recorder.Code)
		}
	}
}

func TestNormalizedRating(t *testing.T) {
	rating, scale := 4.2, 5.0
	got := normalizedRating(&rating, &scale)
	if got == nil || *got != 8.4 {
		t.Fatalf("normalized rating = %v, want 8.4", got)
	}
}

func TestNormalizedRatingWithoutScale(t *testing.T) {
	rating := 8.0
	if got := normalizedRating(&rating, nil); got != nil {
		t.Fatalf("normalized rating = %v, want nil", got)
	}
}

func TestMediaEventIDUsesDedicatedPrefix(t *testing.T) {
	id := uuid.MustParse("018cc251-7b2e-7d52-9b0d-6bd6f2c9c9e4")
	if got, want := mediaEventID(id), ids.Encode(ids.MediaEventPrefix, id); got != want {
		t.Fatalf("media event id = %q, want %q", got, want)
	}
}

func TestMediaChangeIDUsesSeparatePrefix(t *testing.T) {
	id := uuid.MustParse("018cc251-7b2e-7d52-9b0d-6bd6f2c9c9e4")
	if got, want := ids.Encode(ids.MediaChangePrefix, id), "medchg_018cc251-7b2e-7d52-9b0d-6bd6f2c9c9e4"; got != want {
		t.Fatalf("media change id = %q, want %q", got, want)
	}
}
