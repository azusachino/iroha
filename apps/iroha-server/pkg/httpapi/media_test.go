package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/imagefetch"
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

func TestMediaCoverURL(t *testing.T) {
	id := uuid.MustParse("018cc251-7b2e-7d52-9b0d-6bd6f2c9c9e4")
	encoded := ids.Encode(ids.MediaPrefix, id)

	if got := mediaCoverURL(id, ""); got != "" {
		t.Fatalf("mediaCoverURL with empty rawURL = %q, want empty", got)
	}

	want := "/api/v1/media/" + encoded + "/cover"
	if got := mediaCoverURL(id, "https://cdn.weread.qq.com/cover/123.jpg"); got != want {
		t.Fatalf("mediaCoverURL = %q, want %q", got, want)
	}
}

func newCoverTestServer(t *testing.T, opts ...imagefetch.Option) *Server {
	t.Helper()
	return &Server{
		deps:         Dependencies{Logger: slog.New(slog.DiscardHandler)},
		coverFetcher: imagefetch.New(t.TempDir(), opts...),
	}
}

func serveCoverTest(s *Server, coverURL string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	s.serveCover(rec, httptest.NewRequest("GET", "/api/v1/media/x/cover", nil), coverURL)
	return rec
}

func allowLoopback(netip.AddrPort) bool { return true }

var coverPNG = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)

func TestServeCoverRefusesPrivateAddress(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(coverPNG) }))
	defer upstream.Close()
	rec := serveCoverTest(newCoverTestServer(t), upstream.URL) // production policy
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}

func TestServeCoverRefusesRedirectToPrivateAddress(t *testing.T) {
	var hits atomic.Int32
	private := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(coverPNG)
	}))
	defer private.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, private.URL, http.StatusFound)
	}))
	defer redirector.Close()
	onlyRedirector := func(ap netip.AddrPort) bool { return ap.String() == strings.TrimPrefix(redirector.URL, "http://") }
	rec := serveCoverTest(newCoverTestServer(t, imagefetch.WithDialPolicy(onlyRedirector)), redirector.URL)
	if rec.Code != http.StatusBadGateway || hits.Load() != 0 {
		t.Fatalf("status = %d, hits = %d", rec.Code, hits.Load())
	}
}

func TestServeCoverRefusesHTML(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("<html><script>alert(1)</script></html>"))
	}))
	defer upstream.Close()
	rec := serveCoverTest(newCoverTestServer(t, imagefetch.WithDialPolicy(allowLoopback)), upstream.URL)
	if rec.Code != http.StatusBadGateway || strings.Contains(rec.Body.String(), "script") {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestServeCoverRefusesOversizedBodyAndDoesNotCache(t *testing.T) {
	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(coverPNG)
		chunk := make([]byte, 1<<20)
		for range imagefetch.MaxBytes>>20 + 1 {
			_, _ = w.Write(chunk)
		}
	}))
	defer upstream.Close()
	s := newCoverTestServer(t, imagefetch.WithDialPolicy(allowLoopback))
	for range 2 {
		if rec := serveCoverTest(s, upstream.URL); rec.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, want 502", rec.Code)
		}
	}
	if hits.Load() != 2 {
		t.Fatalf("hits = %d, want 2: an oversized body must not be cached", hits.Load())
	}
}

func TestServeCoverSetsSafetyHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(coverPNG)
	}))
	defer upstream.Close()
	rec := serveCoverTest(newCoverTestServer(t, imagefetch.WithDialPolicy(allowLoopback)), upstream.URL)
	h := rec.Header()
	if rec.Code != 200 || h.Get("Content-Type") != "image/png" ||
		h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Content-Security-Policy") != "sandbox" {
		t.Fatalf("status = %d, headers = %v", rec.Code, h)
	}
}

func TestServeCoverFetchesAgainWhenURLChanges(t *testing.T) {
	var hits atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(coverPNG)
	}))
	defer upstream.Close()
	s := newCoverTestServer(t, imagefetch.WithDialPolicy(allowLoopback))
	first := serveCoverTest(s, upstream.URL+"/a")
	serveCoverTest(s, upstream.URL+"/a")
	second := serveCoverTest(s, upstream.URL+"/b")
	if hits.Load() != 2 {
		t.Fatalf("hits = %d, want 2", hits.Load())
	}
	if first.Header().Get("ETag") == second.Header().Get("ETag") {
		t.Fatal("ETag must change with the cover URL")
	}
}
