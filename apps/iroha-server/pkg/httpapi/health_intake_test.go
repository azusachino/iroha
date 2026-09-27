package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
)

func TestValidBearerToken(t *testing.T) {
	for name, header := range map[string]string{
		"exact":       "Bearer secret",
		"scheme-case": "bearer secret",
	} {
		t.Run(name, func(t *testing.T) {
			if !validBearerToken(header, "secret") {
				t.Fatal("valid bearer token rejected")
			}
		})
	}
	for _, header := range []string{"", "secret", "Bearer", "Bearer wrong", "Bearer secret extra"} {
		if validBearerToken(header, "secret") {
			t.Fatalf("invalid bearer token accepted: %q", header)
		}
	}
}

func TestHealthIntakeEnabledWithoutTokenOnTailnet(t *testing.T) {
	server := NewServer(Dependencies{Config: config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}}})
	recorder := httptest.NewRecorder()
	// An invalid body should reach body processing (yielding 400) rather than 503
	request := httptest.NewRequest(http.MethodPost, "/api/v1/intake/health", strings.NewReader(`not-json`))
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 bad request", recorder.Code)
	}
}

func TestHealthIntakeRejectsInvalidCredentialsBeforeBodyProcessing(t *testing.T) {
	server := NewServer(Dependencies{Config: config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo", HealthIntakeToken: "secret"}}})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/intake/health", strings.NewReader("not-json"))
	request.Header.Set("Authorization", "Bearer wrong")
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}
