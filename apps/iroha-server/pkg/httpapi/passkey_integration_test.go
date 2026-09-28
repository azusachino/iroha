//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/auth"
	"github.com/descope/virtualwebauthn"
)

// passkeyClient drives the owner API over HTTPS with a cookie jar, the way a
// browser would, so Secure cookies and CSRF behave as in production.
type passkeyClient struct {
	t    *testing.T
	base string
	http *http.Client
	csrf string
}

func (c *passkeyClient) do(method, path string, body []byte, want int) []byte {
	c.t.Helper()
	req, _ := http.NewRequest(method, c.base+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c.csrf != "" && method != http.MethodGet {
		req.Header.Set(csrfHeaderName, c.csrf)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		c.t.Fatalf("%s %s = %d, want %d: %s", method, path, res.StatusCode, want, out)
	}
	return out
}

// publicKey extracts the `publicKey` member the browser would pass to the
// authenticator.
func publicKey(t *testing.T, raw []byte) string {
	t.Helper()
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapper); err != nil || wrapper["publicKey"] == nil {
		t.Fatalf("options lack publicKey: %s", raw)
	}
	return string(wrapper["publicKey"])
}

func TestIntegrationPasskeyLifecycle(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	db.Exec("delete from tb_users")
	t.Cleanup(func() { db.Exec("delete from tb_users") })

	svc := auth.NewService(db)
	if err := svc.ConfigurePasskeys("localhost", []string{"https://localhost"}); err != nil {
		t.Fatalf("configure passkeys: %v", err)
	}
	server := httptest.NewTLSServer(NewServer(Dependencies{Auth: svc, Passkeys: svc, Logger: slog.Default()}))
	t.Cleanup(server.Close)
	newClient := func() *passkeyClient {
		// server.Client() is shared; copy it so each "browser" has its own jar.
		jar, _ := cookiejar.New(nil)
		h := *server.Client()
		h.Jar = jar
		return &passkeyClient{t: t, base: server.URL, http: &h}
	}
	rp := virtualwebauthn.RelyingParty{ID: "localhost", Name: "iroha", Origin: "https://localhost"}

	owner := newClient()
	var session authSessionResponse
	json.Unmarshal(owner.do(http.MethodPost, "/api/v1/auth/setup", []byte(`{"username":"owner","password":"correct horse battery"}`), http.StatusOK), &session)
	owner.csrf = session.CSRFToken
	if !session.PasskeysEnabled {
		t.Fatal("setup response does not report passkeys enabled")
	}
	json.Unmarshal(owner.do(http.MethodGet, "/api/v1/auth/session", nil, http.StatusOK), &session)
	if !session.PasskeysEnabled {
		t.Fatal("session does not report passkeys enabled")
	}

	// Adding a passkey needs a fresh password confirmation.
	owner.do(http.MethodPost, "/api/v1/account/passkeys/register/begin", nil, http.StatusForbidden)
	owner.do(http.MethodPost, "/api/v1/account/reauth", []byte(`{"password":"wrong password here"}`), http.StatusUnauthorized)
	owner.do(http.MethodPost, "/api/v1/account/reauth", []byte(`{"password":"correct horse battery"}`), http.StatusNoContent)

	authenticator := virtualwebauthn.NewAuthenticator()
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	creation := owner.do(http.MethodPost, "/api/v1/account/passkeys/register/begin", nil, http.StatusOK)
	attOptions, err := virtualwebauthn.ParseAttestationOptions(publicKey(t, creation))
	if err != nil {
		t.Fatalf("parse creation options: %v", err)
	}
	attestation := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOptions)
	var added auth.PasskeyInfo
	json.Unmarshal(owner.do(http.MethodPost, "/api/v1/account/passkeys/register/finish?name="+url.QueryEscape("Haru's iPhone"), []byte(attestation), http.StatusCreated), &added)
	if added.Name != "Haru's iPhone" || !strings.HasPrefix(added.ID, "pk_") {
		t.Fatalf("added passkey = %+v", added)
	}
	authenticator.AddCredential(credential)
	// ParseAttestationOptions has already decoded the user handle.
	authenticator.Options.UserHandle = []byte(attOptions.UserID)

	// The same ceremony cannot be finished twice.
	owner.do(http.MethodPost, "/api/v1/account/passkeys/register/finish?name=again", []byte(attestation), http.StatusBadRequest)

	owner.do(http.MethodPatch, "/api/v1/account/passkeys/"+added.ID, []byte(`{"name":"iPhone"}`), http.StatusNoContent)
	if list := string(owner.do(http.MethodGet, "/api/v1/account/passkeys", nil, http.StatusOK)); !strings.Contains(list, `"name":"iPhone"`) {
		t.Fatalf("list = %s", list)
	}

	// A fresh browser signs in with the passkey alone, no username.
	visitor := newClient()
	visitor.do(http.MethodGet, "/api/v1/metrics", nil, http.StatusUnauthorized)
	assertionJSON := visitor.do(http.MethodPost, "/api/v1/auth/passkey/begin", nil, http.StatusOK)
	asOptions, err := virtualwebauthn.ParseAssertionOptions(publicKey(t, assertionJSON))
	if err != nil {
		t.Fatalf("parse request options: %v", err)
	}
	assertion := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *asOptions)
	var loggedIn authSessionResponse
	json.Unmarshal(visitor.do(http.MethodPost, "/api/v1/auth/passkey/finish", []byte(assertion), http.StatusOK), &loggedIn)
	if !loggedIn.Authenticated || loggedIn.Username != "owner" || !loggedIn.PasskeysEnabled {
		t.Fatalf("passkey login = %+v", loggedIn)
	}
	visitor.csrf = loggedIn.CSRFToken
	visitor.do(http.MethodGet, "/api/v1/metrics", nil, http.StatusOK)

	// Replaying the same assertion without a ceremony is refused.
	newClient().do(http.MethodPost, "/api/v1/auth/passkey/finish", []byte(assertion), http.StatusBadRequest)

	// Removing needs a fresh confirmation too; after removal the passkey no
	// longer signs in.
	visitor.do(http.MethodDelete, "/api/v1/account/passkeys/"+added.ID, nil, http.StatusForbidden)
	owner.do(http.MethodDelete, "/api/v1/account/passkeys/"+added.ID, nil, http.StatusNoContent)
	late := newClient()
	lateOptions, _ := virtualwebauthn.ParseAssertionOptions(publicKey(t, late.do(http.MethodPost, "/api/v1/auth/passkey/begin", nil, http.StatusOK)))
	late.do(http.MethodPost, "/api/v1/auth/passkey/finish", []byte(virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *lateOptions)), http.StatusUnauthorized)
}
