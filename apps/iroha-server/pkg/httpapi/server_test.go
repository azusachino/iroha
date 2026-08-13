package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestHealthzIsProcessLiveness(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewServer(Dependencies{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestReadyzRequiresAndChecksDatabase(t *testing.T) {
	withoutCheck := httptest.NewRecorder()
	NewServer(Dependencies{}).ServeHTTP(withoutCheck, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if withoutCheck.Code != http.StatusServiceUnavailable {
		t.Fatalf("without check status = %d, want %d", withoutCheck.Code, http.StatusServiceUnavailable)
	}

	called := false
	hasDeadline := false
	server := NewServer(Dependencies{ReadyCheck: func(ctx context.Context) error {
		called = true
		_, hasDeadline = ctx.Deadline()
		return nil
	}})
	ready := httptest.NewRecorder()
	server.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusOK || !called || !hasDeadline {
		t.Fatalf("ready response = %d, called = %t, deadline = %t", ready.Code, called, hasDeadline)
	}
	var body map[string]string
	if err := json.NewDecoder(ready.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ready" {
		t.Fatalf("ready body = %v", body)
	}
}

func TestReadyzReportsDatabaseFailure(t *testing.T) {
	server := NewServer(Dependencies{ReadyCheck: func(context.Context) error {
		return errors.New("database unavailable")
	}})
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestPrivateCORSAllowsMutationPreflight(t *testing.T) {
	handler := corsMiddleware([]string{"https://app.example"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/api/v1/imports", nil)
			req.Header.Set("Origin", "https://app.example")
			req.Header.Set("Access-Control-Request-Method", method)
			req.Header.Set("Access-Control-Request-Headers", "Content-Type")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
				t.Fatalf("allow origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
			}
			if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), method) {
				t.Fatalf("allow methods = %q", rec.Header().Get("Access-Control-Allow-Methods"))
			}
			if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Content-Type") {
				t.Fatalf("allow headers = %q", rec.Header().Get("Access-Control-Allow-Headers"))
			}
		})
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

func TestRateLimitResponse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := limitByIP(1)(next)

	first := httptest.NewRequest(http.MethodGet, "/api/v1/example", nil)
	first.RemoteAddr = "192.0.2.10:1234"
	handler.ServeHTTP(httptest.NewRecorder(), first)

	second := httptest.NewRequest(http.MethodGet, "/api/v1/example", nil)
	second.RemoteAddr = "192.0.2.10:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, second)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After header is missing")
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
}

func TestRequestIDResponseHeaderAndErrorBody(t *testing.T) {
	handler := middleware.RequestID(requestIDResponseHeader(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeContractError(w, http.StatusBadRequest, "bad_request", "invalid request")
	})))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/activities", nil))

	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("X-Request-ID header is missing")
	}
	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.RequestID != requestID {
		t.Fatalf("body request_id = %q, want %q", body.RequestID, requestID)
	}
}
