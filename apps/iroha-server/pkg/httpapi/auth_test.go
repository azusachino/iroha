package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/auth"
	"github.com/google/uuid"
)

// tokenAuth accepts only the session token "good" and the password "right".
type tokenAuth struct{ setUp bool }

func (a tokenAuth) SetupRequired(context.Context) (bool, error) { return !a.setUp, nil }
func (a tokenAuth) Setup(_ context.Context, username, _ string) (string, auth.Principal, error) {
	if a.setUp {
		return "", auth.Principal{}, auth.ErrAlreadySetUp
	}
	return "good", auth.Principal{Username: username, CSRFToken: "csrf"}, nil
}

func (tokenAuth) Login(_ context.Context, username, password string) (string, auth.Principal, error) {
	if password != "right" {
		return "", auth.Principal{}, auth.ErrInvalidCredentials
	}
	return "good", auth.Principal{Username: username, CSRFToken: "csrf"}, nil
}

func (tokenAuth) Authenticate(_ context.Context, token string) (auth.Principal, error) {
	if token != "good" {
		return auth.Principal{}, auth.ErrUnauthenticated
	}
	return auth.Principal{Username: "owner", CSRFToken: "csrf"}, nil
}
func (tokenAuth) Logout(context.Context, string) error { return nil }
func (tokenAuth) SetDisplayName(_ context.Context, _ uuid.UUID, name string) (string, error) {
	return name, nil
}

func serve(t *testing.T, deps Dependencies, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	NewServer(deps).ServeHTTP(recorder, request)
	return recorder
}

func TestPrivateRoutesRequireSession(t *testing.T) {
	deps := Dependencies{Auth: tokenAuth{setUp: true}}
	for _, path := range []string{"/api/v1/metrics", "/api/v1/expenses", "/api/v1/admin/intake-credentials"} {
		if code := serve(t, deps, http.MethodGet, path, "", nil).Code; code != http.StatusUnauthorized {
			t.Errorf("GET %s without session = %d, want 401", path, code)
		}
		bad := map[string]string{"Cookie": sessionCookieName + "=forged"}
		if code := serve(t, deps, http.MethodGet, path, "", bad).Code; code != http.StatusUnauthorized {
			t.Errorf("GET %s with forged session = %d, want 401", path, code)
		}
	}
	good := map[string]string{"Cookie": sessionCookieName + "=good"}
	if code := serve(t, deps, http.MethodGet, "/api/v1/metrics", "", good).Code; code != http.StatusOK {
		t.Fatalf("GET with session = %d, want 200", code)
	}
}

func TestPrivateRoutesFailClosedWithoutAuthenticator(t *testing.T) {
	if code := serve(t, Dependencies{}, http.MethodGet, "/api/v1/metrics", "", nil).Code; code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", code)
	}
}

func TestStateChangingRequestsRequireCSRF(t *testing.T) {
	deps := Dependencies{Auth: tokenAuth{setUp: true}}
	cookie := sessionCookieName + "=good"
	for name, headers := range map[string]map[string]string{
		"missing": {"Cookie": cookie},
		"wrong":   {"Cookie": cookie, csrfHeaderName: "nope"},
	} {
		if code := serve(t, deps, http.MethodPost, "/api/v1/auth/logout", "", headers).Code; code != http.StatusForbidden {
			t.Errorf("%s CSRF = %d, want 403", name, code)
		}
	}
	recorder := serve(t, deps, http.MethodPost, "/api/v1/auth/logout", "", map[string]string{"Cookie": cookie, csrfHeaderName: "csrf"})
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("logout with CSRF = %d, want 204", recorder.Code)
	}
	if set := recorder.Header().Get("Set-Cookie"); !strings.Contains(set, sessionCookieName+"=;") {
		t.Fatalf("logout Set-Cookie = %q, want cleared cookie", set)
	}
}

func TestLoginSetsHardenedCookie(t *testing.T) {
	deps := Dependencies{Auth: tokenAuth{setUp: true}}
	json := map[string]string{"Content-Type": "application/json"}
	if code := serve(t, deps, http.MethodPost, "/api/v1/auth/login", `{"username":"owner","password":"wrong"}`, json).Code; code != http.StatusUnauthorized {
		t.Fatalf("wrong password = %d, want 401", code)
	}
	form := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	if code := serve(t, deps, http.MethodPost, "/api/v1/auth/login", "username=owner&password=right", form).Code; code != http.StatusUnsupportedMediaType {
		t.Fatalf("form login = %d, want 415", code)
	}
	recorder := serve(t, deps, http.MethodPost, "/api/v1/auth/login", `{"username":"owner","password":"right"}`, json)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login = %d, want 200", recorder.Code)
	}
	set := recorder.Header().Get("Set-Cookie")
	for _, want := range []string{sessionCookieName + "=good", "HttpOnly", "Secure", "SameSite=Lax", "Path=/"} {
		if !strings.Contains(set, want) {
			t.Errorf("Set-Cookie %q missing %q", set, want)
		}
	}
	if !strings.Contains(recorder.Body.String(), `"csrf_token":"csrf"`) {
		t.Errorf("login body %s lacks csrf token", recorder.Body.String())
	}
}

func TestSetupOnlyWhileNoOwnerExists(t *testing.T) {
	json := map[string]string{"Content-Type": "application/json"}
	body := `{"username":"owner","password":"long enough password"}`
	if code := serve(t, Dependencies{Auth: tokenAuth{setUp: true}}, http.MethodPost, "/api/v1/auth/setup", body, json).Code; code != http.StatusConflict {
		t.Fatalf("setup after owner exists = %d, want 409", code)
	}
	if code := serve(t, Dependencies{Auth: tokenAuth{}}, http.MethodPost, "/api/v1/auth/setup", body, json).Code; code != http.StatusOK {
		t.Fatalf("first setup = %d, want 200", code)
	}
}

func TestAuthSessionReportsStateWithoutCaching(t *testing.T) {
	recorder := serve(t, Dependencies{Auth: tokenAuth{}}, http.MethodGet, "/api/v1/auth/session", "", nil)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"setup_required":true`) {
		t.Fatalf("session state = %d %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", recorder.Header().Get("Cache-Control"))
	}
}
