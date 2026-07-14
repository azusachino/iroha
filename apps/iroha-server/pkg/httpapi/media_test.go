package httpapi

import (
	"net/http/httptest"
	"testing"
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
