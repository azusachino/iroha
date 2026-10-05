//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
)

func publicGet(t *testing.T, server http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil)) // no session cookie
	return rec
}

func TestIntegrationPublicProjectionIsAnonymousCachedAndFresh(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	server := newIntegrationServer(t, db)

	empty := publicGet(t, server, "/public/v1/activities")
	if empty.Code != http.StatusOK || empty.Body.String() != "[]\n" && empty.Body.String() != "[]" {
		t.Fatalf("empty activities = %d %q", empty.Code, empty.Body.String())
	}
	if got := empty.Header().Get("Cache-Control"); got != "public, max-age=86400" {
		t.Fatalf("Cache-Control = %q", got)
	}

	emptyTag := empty.Header().Get("ETag")
	if emptyTag == "" {
		t.Fatal("empty projection needs a validator")
	}
	conditional := httptest.NewRequest(http.MethodGet, "/public/v1/activities", nil)
	conditional.Header.Set("If-None-Match", emptyTag)
	matched := httptest.NewRecorder()
	server.ServeHTTP(matched, conditional)
	if matched.Code != http.StatusNotModified || matched.Body.Len() != 0 || matched.Header().Get("ETag") != emptyTag {
		t.Fatalf("matching empty projection: %d %v", matched.Code, matched.Header())
	}

	// Importing an activity bumps the activity revision, so the 24-hour
	// snapshot is rebuilt immediately rather than served stale.
	rawID := uploadRawFile(t, server, "run.gpx", "gpx", "cli", validGPX())
	var importID string
	requestJSON(t, server, http.MethodPost, "/api/v1/imports", `{"raw_file_id":"`+rawID+`","parser_kind":"gpx"}`, http.StatusAccepted, func(body map[string]any) {
		importID = stringValue(t, body, "id")
	})
	waitForImportStatus(t, server, importID, imports.StatusCompleted)

	var list []map[string]any
	rec := publicGet(t, server, "/public/v1/activities")
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("activities after import = %d %s (%v)", rec.Code, rec.Body.String(), err)
	}
	id, _ := list[0]["id"].(string)
	changed := httptest.NewRecorder()
	server.ServeHTTP(changed, conditional)
	if changed.Code != http.StatusOK || changed.Header().Get("ETag") == emptyTag || !bytes.Equal(changed.Body.Bytes(), rec.Body.Bytes()) {
		t.Fatalf("revision rebuild reused old validator: %d %v", changed.Code, changed.Header())
	}
	current := httptest.NewRequest(http.MethodGet, "/public/v1/activities", nil)
	current.Header.Set("If-None-Match", changed.Header().Get("ETag"))
	matched = httptest.NewRecorder()
	server.ServeHTTP(matched, current)
	if matched.Code != http.StatusNotModified || matched.Body.Len() != 0 {
		t.Fatalf("new revision match: %d %s", matched.Code, matched.Body.String())
	}

	for _, path := range []string{"/public/v1/summary", "/public/v1/routes", "/public/v1/meta", "/public/v1/activities/" + id} {
		if rec := publicGet(t, server, path); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d %s", path, rec.Code, rec.Body.String())
		}
	}
	for _, path := range []string{"/public/v1/activities/act_00000000-0000-7000-8000-000000000000", "/public/v1/activities/raw_nope"} {
		if rec := publicGet(t, server, path); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
	}
	post := httptest.NewRecorder()
	server.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/public/v1/activities", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST public = %d, want 405", post.Code)
	}
	// The private API stays closed to the same anonymous caller.
	if rec := publicGet(t, server, "/api/v1/activities"); rec.Code == http.StatusOK && rec.Header().Get("Cache-Control") == "public, max-age=86400" {
		t.Fatal("private API served as public")
	}
}
