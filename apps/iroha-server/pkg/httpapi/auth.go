package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/auth"
	"github.com/google/uuid"
)

const (
	sessionCookieName = "iroha_session"
	csrfHeaderName    = "X-CSRF-Token"
	authBodyMaxBytes  = 8 << 10
)

// Authenticator is the owner-login surface the HTTP layer needs (ADR-0008).
type Authenticator interface {
	SetupRequired(ctx context.Context) (bool, error)
	Setup(ctx context.Context, username, password string) (string, auth.Principal, error)
	Login(ctx context.Context, username, password string) (string, auth.Principal, error)
	Authenticate(ctx context.Context, token string) (auth.Principal, error)
	Logout(ctx context.Context, token string) error
	SetDisplayName(ctx context.Context, userID uuid.UUID, name string) (string, error)
}

type principalKey struct{}

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authSessionResponse struct {
	SetupRequired bool   `json:"setup_required"`
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
	DisplayName   string `json:"display_name,omitempty"`
	// PasskeysEnabled tells the sign-in screen whether to offer passkeys.
	PasskeysEnabled bool   `json:"passkeys_enabled"`
	CSRFToken       string `json:"csrf_token,omitempty"`
}

// requireSession admits only requests carrying a valid owner session cookie.
// It fails closed when no authenticator is configured. Network origin grants
// nothing: tailnet requests are checked the same way.
func (s *Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authConfigured(w) {
			return
		}
		who, err := s.deps.Auth.Authenticate(r.Context(), sessionToken(r))
		switch {
		case err == nil:
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, who)))
		case errors.Is(err, auth.ErrUnauthenticated):
			writeContractError(w, http.StatusUnauthorized, "unauthenticated", "login required")
		default:
			s.deps.Logger.Error("authenticate session", "error", err)
			writeContractError(w, http.StatusInternalServerError, "auth_failed", "failed to verify session")
		}
	})
}

// requireCSRF rejects state-changing requests whose X-CSRF-Token header does
// not match the session's token. It must run after requireSession.
func (s *Server) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		who, _ := r.Context().Value(principalKey{}).(auth.Principal)
		got := r.Header.Get(csrfHeaderName)
		if who.CSRFToken == "" || subtle.ConstantTimeCompare([]byte(got), []byte(who.CSRFToken)) != 1 {
			writeContractError(w, http.StatusForbidden, "csrf_failed", "missing or invalid CSRF token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleAuthSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !s.authConfigured(w) {
		return
	}
	required, err := s.deps.Auth.SetupRequired(r.Context())
	if err != nil {
		s.deps.Logger.Error("read setup state", "error", err)
		writeContractError(w, http.StatusInternalServerError, "auth_failed", "failed to read authentication state")
		return
	}
	response := authSessionResponse{SetupRequired: required}
	if who, err := s.deps.Auth.Authenticate(r.Context(), sessionToken(r)); err == nil {
		response = s.signedInResponse(who)
	}
	response.PasskeysEnabled = s.passkeysEnabled()
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	if s.authConfigured(w) {
		s.handleCredentials(w, r, s.deps.Auth.Setup)
	}
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if s.authConfigured(w) {
		s.handleCredentials(w, r, s.deps.Auth.Login)
	}
}

func (s *Server) authConfigured(w http.ResponseWriter) bool {
	if s.deps.Auth == nil {
		writeContractError(w, http.StatusServiceUnavailable, "auth_not_configured", "authentication is not configured")
		return false
	}
	return true
}

// handleCredentials serves setup and login. Requiring a JSON body means a
// cross-site HTML form cannot submit it without a CORS preflight.
func (s *Server) handleCredentials(w http.ResponseWriter, r *http.Request, action func(context.Context, string, string) (string, auth.Principal, error)) {
	w.Header().Set("Cache-Control", "no-store")
	if mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mediaType != "application/json" {
		writeContractError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "expected application/json")
		return
	}
	var body credentialsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, authBodyMaxBytes)).Decode(&body); err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	token, who, err := action(r.Context(), body.Username, body.Password)
	switch {
	case err == nil:
	case errors.Is(err, auth.ErrInvalidInput):
		writeContractError(w, http.StatusBadRequest, "invalid_credentials_input", err.Error())
		return
	case errors.Is(err, auth.ErrAlreadySetUp):
		writeContractError(w, http.StatusConflict, "already_set_up", "the owner account already exists")
		return
	case errors.Is(err, auth.ErrInvalidCredentials):
		s.deps.Logger.Warn("login rejected", "client_ip", s.clientIP(r))
		writeContractError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
		return
	default:
		s.deps.Logger.Error("authenticate owner", "error", err)
		writeContractError(w, http.StatusInternalServerError, "auth_failed", "authentication failed")
		return
	}
	s.deps.Logger.Info("owner session created", "session_id", who.SessionID)
	http.SetCookie(w, sessionCookie(token, time.Now().Add(auth.SessionTTL)))
	writeJSON(w, http.StatusOK, s.signedInResponse(who))
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.Auth.Logout(r.Context(), sessionToken(r)); err != nil {
		s.deps.Logger.Error("logout", "error", err)
		writeContractError(w, http.StatusInternalServerError, "auth_failed", "failed to log out")
		return
	}
	http.SetCookie(w, sessionCookie("", time.Unix(0, 0)))
	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateAccount changes the owner's profile (currently the display
// name; an empty value clears it).
func (s *Server) handleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	who, _ := r.Context().Value(principalKey{}).(auth.Principal)
	userID, err := ids.Decode(ids.UserPrefix, who.UserID)
	if err != nil {
		writeContractError(w, http.StatusUnauthorized, "unauthenticated", "login required")
		return
	}
	var body struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, authBodyMaxBytes)).Decode(&body); err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	name, err := s.deps.Auth.SetDisplayName(r.Context(), userID, body.DisplayName)
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		writeContractError(w, http.StatusBadRequest, "invalid_display_name", err.Error())
	case err != nil:
		s.deps.Logger.Error("update account", "error", err)
		writeContractError(w, http.StatusInternalServerError, "auth_failed", "failed to update account")
	default:
		writeJSON(w, http.StatusOK, map[string]string{"username": who.Username, "display_name": name})
	}
}

func (s *Server) passkeysEnabled() bool {
	return s.deps.Passkeys != nil && s.deps.Passkeys.PasskeysEnabled()
}

// signedInResponse is the session state returned after any sign-in.
func (s *Server) signedInResponse(who auth.Principal) authSessionResponse {
	return authSessionResponse{
		Authenticated:   true,
		Username:        who.Username,
		DisplayName:     who.DisplayName,
		PasskeysEnabled: s.passkeysEnabled(),
		CSRFToken:       who.CSRFToken,
	}
}

func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// sessionCookie is HttpOnly, Secure, and SameSite=Lax (ADR-0008). Browsers
// accept Secure cookies from http://localhost, so local development works.
func sessionCookie(value string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}
