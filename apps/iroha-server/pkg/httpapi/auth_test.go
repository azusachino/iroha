package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testJWTSecret   = "test-jwt-secret"
	testJWTIssuer   = "iroha-test"
	testJWTAudience = "iroha-api-test"
)

func TestRequireJWT(t *testing.T) {
	server := &Server{deps: Dependencies{
		Config: config.Config{Auth: config.AuthConfig{
			JWTSecret:   testJWTSecret,
			JWTIssuer:   testJWTIssuer,
			JWTAudience: testJWTAudience,
		}},
	}}

	tests := []struct {
		name        string
		localNoAuth bool
		token       string
		scope       string
		wantStatus  int
	}{
		{name: "local no auth bypass", localNoAuth: true, wantStatus: http.StatusOK},
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "valid read token", token: testToken(t, "client", "iroha:read", time.Hour), wantStatus: http.StatusOK},
		{name: "write implies read", token: testToken(t, "uploader", "iroha:write", time.Hour), wantStatus: http.StatusOK},
		{name: "insufficient scope", token: testToken(t, "reader", "iroha:read", time.Hour), scope: "iroha:write", wantStatus: http.StatusForbidden},
		{name: "expired token", token: testToken(t, "client", "iroha:read", -time.Hour), wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/activities", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			server.deps.Config.Auth.LocalNoAuth = tt.localNoAuth
			server.requireJWT(defaultScope(tt.scope))(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if nextCalled != (tt.wantStatus == http.StatusOK) {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantStatus == http.StatusOK)
			}
		})
	}
}

func TestJWTRejectsWrongIssuerAndAudience(t *testing.T) {
	server := &Server{deps: Dependencies{Config: config.Config{Auth: config.AuthConfig{
		JWTSecret: testJWTSecret, JWTIssuer: testJWTIssuer, JWTAudience: testJWTAudience,
	}}}}
	for name, claims := range map[string]jwtClaims{
		"wrong issuer": {Scope: "iroha:read", RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "other", Subject: "client", Audience: []string{testJWTAudience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}},
		"wrong audience": {Scope: "iroha:read", RegisteredClaims: jwt.RegisteredClaims{
			Issuer: testJWTIssuer, Subject: "client", Audience: []string{"other"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}},
	} {
		t.Run(name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/activities", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			server.requireJWT("iroha:read")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestRateLimitResponse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := limitByIP(1)(next)

	first := httptest.NewRequest(http.MethodGet, "/public/v1/summary", nil)
	first.RemoteAddr = "192.0.2.10:1234"
	handler.ServeHTTP(httptest.NewRecorder(), first)

	second := httptest.NewRequest(http.MethodGet, "/public/v1/summary", nil)
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

func TestBearerToken(t *testing.T) {
	for _, tt := range []struct {
		header string
		want   string
		ok     bool
	}{
		{header: "Bearer abc", want: "abc", ok: true},
		{header: "bEaReR abc", want: "abc", ok: true},
		{header: "Bearer   abc  ", want: "abc", ok: true},
		{header: "Basic abc", ok: false},
		{header: "Bearer ", ok: false},
	} {
		token, ok := bearerToken(tt.header)
		if token != tt.want || ok != tt.ok {
			t.Errorf("bearerToken(%q) = (%q, %v), want (%q, %v)", tt.header, token, ok, tt.want, tt.ok)
		}
	}
}

func testToken(t *testing.T, subject, scope string, lifetime time.Duration) string {
	t.Helper()
	claims := jwtClaims{Scope: scope, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: testJWTIssuer, Subject: subject, Audience: []string{testJWTAudience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(lifetime)), IssuedAt: jwt.NewNumericDate(time.Now()),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func defaultScope(scope string) string {
	if scope == "" {
		return "iroha:read"
	}
	return scope
}
