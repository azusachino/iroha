package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
)

func TestClientIPTrustsForwardedHeaderOnlyFromTrustedProxy(t *testing.T) {
	s := &Server{}
	s.trustedProxies, _ = parseTrustedProxies([]string{"10.42.0.0/16"})
	for name, tc := range map[string]struct {
		peer, header, want string
	}{
		"trusted proxy":         {"10.42.0.7:5000", "203.0.113.9", "203.0.113.9"},
		"untrusted peer":        {"198.51.100.4:5000", "203.0.113.9", "198.51.100.4"},
		"trusted, no header":    {"10.42.0.7:5000", "", "10.42.0.7"},
		"trusted, junk header":  {"10.42.0.7:5000", "not-an-ip", "10.42.0.7"},
		"trusted, list spoofed": {"10.42.0.7:5000", "1.1.1.1, 2.2.2.2", "10.42.0.7"},
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = tc.peer
		if tc.header != "" {
			r.Header.Set(forwardedClientHeader, tc.header)
		}
		if got := s.clientIP(r); got != tc.want {
			t.Errorf("%s: clientIP = %q, want %q", name, got, tc.want)
		}
	}
}

func TestLoginIsRateLimitedPerClient(t *testing.T) {
	handler := NewServer(Dependencies{Auth: tokenAuth{setUp: true}})
	body := `{"username":"owner","password":"wrong"}`
	var last int
	for i := 0; i <= authRateLimitPerMin; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.RemoteAddr = "198.51.100.4:5000"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		last = rec.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("attempt %d status = %d, want 429", authRateLimitPerMin+1, last)
	}
}

func TestIntakeIsRateLimitedPerForwardedClient(t *testing.T) {
	handler := NewServer(Dependencies{
		Config:                  config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo", TrustedProxies: []string{"10.42.0.0/16"}}},
		HealthIntakeCredentials: fakeIntakeVerifier{token: "secret"},
	})
	post := func(client string) int {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/intake/health", strings.NewReader("{}"))
		r.RemoteAddr = "10.42.0.7:5000"
		r.Header.Set(forwardedClientHeader, client)
		r.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Code
	}
	for i := 0; i < intakeRateLimitPerMin; i++ {
		post("203.0.113.9")
	}
	if code := post("203.0.113.9"); code != http.StatusTooManyRequests {
		t.Fatalf("flooding client status = %d, want 429", code)
	}
	if code := post("198.51.100.20"); code != http.StatusUnauthorized {
		t.Fatalf("other client status = %d, want 401 (own bucket)", code)
	}
}

func TestIntakeQuotaResetsDaily(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	q := newIntakeQuota(func() time.Time { return now })
	if !q.take("c", intakeBytesPerCredentialPerDay) {
		t.Fatal("first take within quota refused")
	}
	if q.take("c", 1) {
		t.Fatal("take over quota accepted")
	}
	if !q.take("other", 1) {
		t.Fatal("quota leaked across credentials")
	}
	now = now.Add(24 * time.Hour)
	if !q.take("c", 1) {
		t.Fatal("quota did not reset on a new day")
	}
}
