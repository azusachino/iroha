package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/internal/config"
)

func TestRequireUploadAuth(t *testing.T) {
	const token = "s3cret-import-token"

	tests := []struct {
		name        string
		localNoAuth bool
		importToken string
		authHeader  string
		setHeader   bool
		wantStatus  int
		wantNext    bool
	}{
		{
			name:        "local no auth passes through without header",
			localNoAuth: true,
			importToken: "",
			wantStatus:  http.StatusOK,
			wantNext:    true,
		},
		{
			name:        "local no auth ignores any header",
			localNoAuth: true,
			importToken: token,
			authHeader:  "Bearer wrong",
			setHeader:   true,
			wantStatus:  http.StatusOK,
			wantNext:    true,
		},
		{
			name:        "correct bearer token passes",
			localNoAuth: false,
			importToken: token,
			authHeader:  "Bearer " + token,
			setHeader:   true,
			wantStatus:  http.StatusOK,
			wantNext:    true,
		},
		{
			name:        "case insensitive bearer scheme passes",
			localNoAuth: false,
			importToken: token,
			authHeader:  "bearer " + token,
			setHeader:   true,
			wantStatus:  http.StatusOK,
			wantNext:    true,
		},
		{
			name:        "missing header is unauthorized",
			localNoAuth: false,
			importToken: token,
			setHeader:   false,
			wantStatus:  http.StatusUnauthorized,
			wantNext:    false,
		},
		{
			name:        "malformed header without bearer prefix is unauthorized",
			localNoAuth: false,
			importToken: token,
			authHeader:  token,
			setHeader:   true,
			wantStatus:  http.StatusUnauthorized,
			wantNext:    false,
		},
		{
			name:        "bearer with empty token is unauthorized",
			localNoAuth: false,
			importToken: token,
			authHeader:  "Bearer   ",
			setHeader:   true,
			wantStatus:  http.StatusUnauthorized,
			wantNext:    false,
		},
		{
			name:        "wrong token is unauthorized",
			localNoAuth: false,
			importToken: token,
			authHeader:  "Bearer nope",
			setHeader:   true,
			wantStatus:  http.StatusUnauthorized,
			wantNext:    false,
		},
		{
			name:        "empty configured token fails closed even with bearer",
			localNoAuth: false,
			importToken: "",
			authHeader:  "Bearer anything",
			setHeader:   true,
			wantStatus:  http.StatusUnauthorized,
			wantNext:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{deps: Dependencies{
				Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
				Config: config.Config{
					Auth: config.AuthConfig{
						LocalNoAuth: tt.localNoAuth,
						ImportToken: tt.importToken,
					},
				},
			}}

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/raw-files", nil)
			if tt.setHeader {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			s.requireUploadAuth(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNext {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantNext)
			}
		})
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{name: "valid bearer", header: "Bearer abc", wantToken: "abc", wantOK: true},
		{name: "case insensitive scheme", header: "bEaReR abc", wantToken: "abc", wantOK: true},
		{name: "trims surrounding space", header: "Bearer   abc  ", wantToken: "abc", wantOK: true},
		{name: "empty header", header: "", wantOK: false},
		{name: "no bearer prefix", header: "Basic abc", wantOK: false},
		{name: "bearer without token", header: "Bearer ", wantOK: false},
		{name: "shorter than prefix", header: "Bear", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := bearerToken(tt.header)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if token != tt.wantToken {
				t.Errorf("token = %q, want %q", token, tt.wantToken)
			}
		})
	}
}
