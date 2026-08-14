package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePageLimit(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  int
		valid bool
	}{
		{name: "omitted", want: 50, valid: true},
		{name: "minimum", query: "?limit=1", want: 1, valid: true},
		{name: "maximum", query: "?limit=100", want: 100, valid: true},
		{name: "non numeric", query: "?limit=bad"},
		{name: "zero", query: "?limit=0"},
		{name: "negative", query: "?limit=-1"},
		{name: "too large", query: "?limit=101"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			got, ok := parsePageLimit(recorder, httptest.NewRequest("GET", "/api/v1/activities"+test.query, nil))
			if ok != test.valid || (test.valid && got != test.want) {
				t.Fatalf("parsePageLimit() = (%d, %t), want (%d, %t)", got, ok, test.want, test.valid)
			}
			if !test.valid && recorder.Code != 400 {
				t.Fatalf("status = %d, want 400", recorder.Code)
			}
		})
	}
}

func TestListFilterParsersRejectInvalidPageLimits(t *testing.T) {
	parsers := []struct {
		name  string
		parse func(*httptest.ResponseRecorder, *http.Request) bool
	}{
		{name: "activities", parse: func(w *httptest.ResponseRecorder, r *http.Request) bool {
			_, ok := parseActivityFilters(w, r)
			return ok
		}},
		{name: "daily", parse: func(w *httptest.ResponseRecorder, r *http.Request) bool { _, ok := parseDailyFilters(w, r); return ok }},
		{name: "sleep", parse: func(w *httptest.ResponseRecorder, r *http.Request) bool { _, ok := parseSleepFilters(w, r); return ok }},
		{name: "media", parse: func(w *httptest.ResponseRecorder, r *http.Request) bool { _, ok := parseMediaFilters(w, r); return ok }},
		{name: "media events", parse: func(w *httptest.ResponseRecorder, r *http.Request) bool {
			_, ok := parseMediaEventFilters(w, r)
			return ok
		}},
	}

	for _, parser := range parsers {
		t.Run(parser.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			if parser.parse(recorder, httptest.NewRequest("GET", "/api/v1/example?limit=0", nil)) {
				t.Fatal("parser accepted zero limit")
			}
			if recorder.Code != 400 {
				t.Fatalf("status = %d, want 400", recorder.Code)
			}
		})
	}
}
