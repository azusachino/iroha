package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestPrivateCORSAllowsJWTPostPreflight(t *testing.T) {
	handler := corsMiddleware([]string{"https://app.example"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/imports", nil)
	req.Header.Set("Origin", "https://app.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("allow origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), http.MethodPost) {
		t.Fatalf("allow methods = %q", rec.Header().Get("Access-Control-Allow-Methods"))
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("allow headers = %q", rec.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestAccessLogIncludesRequestMetadata(t *testing.T) {
	var logs bytes.Buffer
	server := &Server{deps: Dependencies{Logger: slog.New(slog.NewJSONHandler(&logs, nil))}}
	handler := middleware.RequestID(server.accessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	for _, field := range []string{"request_id", "method", "route", "status", "duration_ms"} {
		if !strings.Contains(logs.String(), `"`+field+`"`) {
			t.Fatalf("access log missing %q: %s", field, logs.String())
		}
	}
}
