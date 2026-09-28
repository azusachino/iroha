package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/intakecredential"
)

type fakeIntakeVerifier struct {
	token string
	err   error
}

func (f fakeIntakeVerifier) Verify(_ context.Context, token string) error {
	if f.err != nil {
		return f.err
	}
	if token != f.token {
		return intakecredential.ErrInvalid
	}
	return nil
}

func TestBearerToken(t *testing.T) {
	for header, want := range map[string]string{
		"Bearer secret":       "secret",
		"bearer secret":       "secret",
		"":                    "",
		"secret":              "",
		"Bearer":              "",
		"Basic secret":        "",
		"Bearer secret extra": "",
	} {
		if got := bearerToken(header); got != want {
			t.Errorf("bearerToken(%q) = %q, want %q", header, got, want)
		}
	}
}

func postHealthIntake(t *testing.T, verifier HealthIntakeVerifier, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	server := NewServer(Dependencies{
		Config:                  config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}},
		HealthIntakeCredentials: verifier,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/intake/health", strings.NewReader("not-json"))
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	server.ServeHTTP(recorder, request)
	return recorder
}

func TestHealthIntakeFailsClosedWithoutVerifier(t *testing.T) {
	if code := postHealthIntake(t, nil, "Bearer anything").Code; code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", code)
	}
}

func TestHealthIntakeFailsClosedWhenNotProvisioned(t *testing.T) {
	verifier := fakeIntakeVerifier{err: intakecredential.ErrNotProvisioned}
	if code := postHealthIntake(t, verifier, "Bearer anything").Code; code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", code)
	}
}

func TestHealthIntakeRejectsMissingOrWrongCredentialBeforeBodyProcessing(t *testing.T) {
	verifier := fakeIntakeVerifier{token: "secret"}
	for _, authorization := range []string{"", "Bearer wrong", "secret"} {
		recorder := postHealthIntake(t, verifier, authorization)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("authorization %q: status = %d, want 401", authorization, recorder.Code)
		}
		if recorder.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("authorization %q: missing WWW-Authenticate", authorization)
		}
	}
}

func TestHealthIntakeAcceptsValidCredential(t *testing.T) {
	// A valid credential reaches body validation, so an invalid body yields 400.
	if code := postHealthIntake(t, fakeIntakeVerifier{token: "secret"}, "Bearer secret").Code; code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestHealthIntakeVerifierFailureIsServerError(t *testing.T) {
	server := NewServer(Dependencies{
		Config:                  config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}},
		HealthIntakeCredentials: fakeIntakeVerifier{err: errors.New("db down")},
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/intake/health", strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer secret")
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
