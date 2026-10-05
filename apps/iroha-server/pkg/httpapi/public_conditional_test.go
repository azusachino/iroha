package httpapi

import (
	"bytes"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/publicexport"
)

func TestPublicRouterConditionalMethodsAndRateLimit(t *testing.T) {
	now := time.Now()
	encoded, err := encodePublicSnapshot(publicexport.Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(Dependencies{Now: func() time.Time { return now }}).(*Server)
	s.publicCache = &publicSnapshotCache{encoded: encoded, builtAt: now, revision: "map[]"}
	for name, headers := range map[string]http.Header{
		"strong":   {"If-Match": {encoded["meta"].etag}},
		"wildcard": {"If-Match": {"*"}},
		"repeated": {"If-Match": {"\"other\"", encoded["meta"].etag}},
	} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/public/v1/meta", nil)
			r.Header = headers
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), encoded["meta"].body) {
				t.Fatalf("matching If-Match: %d %s", w.Code, w.Body.String())
			}
		})
	}
	for _, method := range []string{http.MethodHead, http.MethodPost} {
		r := httptest.NewRequest(method, "/public/v1/meta", nil)
		r.Header.Set("If-None-Match", "*")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != http.StatusMethodNotAllowed || w.Header().Get("ETag") != "" {
			t.Fatalf("method %s: %d %v", method, w.Code, w.Header())
		}
	}
	s = NewServer(Dependencies{Now: func() time.Time { return now }}).(*Server)
	s.publicCache = &publicSnapshotCache{encoded: encoded, builtAt: now, revision: "map[]"}
	for index := 0; index <= publicRateLimitPerMin; index++ {
		r := httptest.NewRequest(http.MethodGet, "/public/v1/meta", nil)
		r.Header.Set("Origin", "https://fixture.example.invalid")
		r.Header.Set("If-None-Match", "*")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if index < publicRateLimitPerMin {
			if w.Code != http.StatusNotModified || w.Body.Len() != 0 || w.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Fatalf("router304 metadata: %d %v", w.Code, w.Header())
			}
		} else if w.Code != http.StatusTooManyRequests || w.Header().Get("ETag") != "" || w.Header().Get("Cache-Control") == publicCacheControl {
			t.Fatalf("rate-limit precedes validator: %d %v", w.Code, w.Header())
		}
	}
}

func TestPublicBuildErrorPrecedesValidators(t *testing.T) {
	// DryRun deliberately fails the real summary scan without connecting to a database.
	db, err := gorm.Open(postgres.Open("host=fixture.invalid dbname=synthetic"), &gorm.Config{DisableAutomaticPing: true, DryRun: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(Dependencies{ActivityService: activities.NewService(db)}).(*Server)
	for _, header := range []string{"If-Match", "If-None-Match"} {
		r := httptest.NewRequest(http.MethodGet, "/public/v1/meta", nil)
		r.Header.Set(header, "*")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "public_unavailable") || w.Header().Get("ETag") != "" || !s.publicCache.builtAt.IsZero() {
			t.Fatalf("build error masked by %s: %d %s", header, w.Code, w.Body.String())
		}
	}
}

func TestPublicSnapshotValidator(t *testing.T) {
	now := time.Now()
	encoded, err := encodePublicSnapshot(publicexport.Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{now: func() time.Time { return now }, publicCache: &publicSnapshotCache{encoded: encoded, builtAt: now, revision: "map[]"}}
	handler := s.servePublic("meta")
	first := httptest.NewRecorder()
	handler(first, httptest.NewRequest(http.MethodGet, "/public/v1/meta", nil))
	etag := first.Header().Get("ETag")
	if first.Code != http.StatusOK || !strings.HasPrefix(etag, "\"") || len(etag) != 66 {
		t.Fatalf("want 200 and strong sha256 validator; got %d %v", first.Code, first.Header())
	}
	if !bytes.Equal(first.Body.Bytes(), encoded["meta"].body) {
		t.Fatal("cached representation bytes changed")
	}
	if !bytes.HasSuffix(first.Body.Bytes(), []byte("\n")) {
		t.Fatal("encoder newline lost")
	}
	for name, headers := range map[string]http.Header{
		"exact":               {"If-None-Match": {etag}},
		"weak":                {"If-None-Match": {"W/" + etag}},
		"list":                {"If-None-Match": {"\"other\", W/" + etag}},
		"wildcard":            {"If-None-Match": {"*"}},
		"repeated":            {"If-None-Match": {"\"other\"", etag}},
		"empty list elements": {"If-None-Match": {", , " + etag + ", "}},
		"etag before date":    {"If-None-Match": {etag}, "If-Modified-Since": {time.Now().Add(time.Hour).Format(http.TimeFormat)}},
	} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/public/v1/meta", nil)
			r.Header = headers
			w := httptest.NewRecorder()
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			handler(w, r)
			if w.Code != http.StatusNotModified || w.Body.Len() != 0 {
				t.Fatalf("want bodyless304; got %d %q", w.Code, w.Body.String())
			}
			if w.Header().Get("ETag") != etag || w.Header().Get("Cache-Control") != publicCacheControl || w.Header().Get("Vary") != "Origin" || w.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Fatal("304 metadata lost")
			}
		})
	}
	for _, value := range []string{"\"stale\"", "unquoted", "W/", "\"unterminated"} {
		r := httptest.NewRequest(http.MethodGet, "/public/v1/meta", nil)
		r.Header.Set("If-None-Match", value)
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), first.Body.Bytes()) {
			t.Fatalf("invalid/stale validator %q changed response", value)
		}
	}
	r := httptest.NewRequest(http.MethodGet, "/public/v1/meta", nil)
	r.Header.Set("Range", "bytes=0-1")
	r.Header.Set("If-Range", etag)
	w := httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusOK || w.Header().Get("Accept-Ranges") != "" || !bytes.Equal(w.Body.Bytes(), first.Body.Bytes()) || r.Header.Get("Range") == "" {
		t.Fatal("range semantics or caller request changed")
	}
	r.Header.Set("If-Match", "\"stale\"")
	r.Header.Set("If-None-Match", etag)
	w = httptest.NewRecorder()
	handler(w, r)
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("If-Match must precede INM: %d", w.Code)
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("ETag") != "" || w.Header().Get("Content-Type") != "" || w.Body.Len() != 0 {
		t.Fatal("412 must not be cached as a public representation")
	}
	reused, err := s.publicSnapshot(r)
	if err != nil || &reused["meta"].body[0] != &encoded["meta"].body[0] {
		t.Fatal("cache hit re-encoded representation")
	}
}

func TestPublicRepresentationIdentity(t *testing.T) {
	snapshot := publicexport.Snapshot{}
	first, err := encodePublicSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Meta.GeneratedAt = time.Now().UTC()
	second, err := encodePublicSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if first["meta"].etag == second["meta"].etag {
		t.Fatal("TTL metadata change reused validator")
	}
	if first["summary"].etag != second["summary"].etag {
		t.Fatal("unchanged summary bytes changed validator")
	}
	snapshot.Summary.Totals.DistanceM = math.NaN()
	if encoded, err := encodePublicSnapshot(snapshot); err == nil || encoded != nil {
		t.Fatal("failed encoding published partial snapshot")
	}
}
