package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ceremonyCookieName binds a WebAuthn ceremony to the browser that began it.
const ceremonyCookieName = "iroha_webauthn"

// PasskeyManager is the WebAuthn surface (ADR-0008 item 6).
type PasskeyManager interface {
	PasskeysEnabled() bool
	Reauthenticate(ctx context.Context, who auth.Principal, password string) error
	ListPasskeys(ctx context.Context, who auth.Principal) ([]auth.PasskeyInfo, error)
	BeginPasskeyRegistration(ctx context.Context, who auth.Principal) (any, string, error)
	FinishPasskeyRegistration(ctx context.Context, who auth.Principal, ceremonyID, name string, r *http.Request) (auth.PasskeyInfo, error)
	RenamePasskey(ctx context.Context, who auth.Principal, id uuid.UUID, name string) error
	DeletePasskey(ctx context.Context, who auth.Principal, id uuid.UUID) error
	BeginPasskeyLogin() (any, string, error)
	FinishPasskeyLogin(ctx context.Context, ceremonyID string, r *http.Request) (string, auth.Principal, error)
}

func (s *Server) passkeysReady(w http.ResponseWriter) bool {
	if s.deps.Passkeys == nil || !s.deps.Passkeys.PasskeysEnabled() {
		writeContractError(w, http.StatusServiceUnavailable, "passkeys_disabled", "passkeys are not configured")
		return false
	}
	return true
}

func (s *Server) writePasskeyError(w http.ResponseWriter, err error, action string) {
	switch {
	case errors.Is(err, auth.ErrReauthRequired):
		writeContractError(w, http.StatusForbidden, "reauth_required", "confirm your password first")
	case errors.Is(err, auth.ErrCeremony):
		writeContractError(w, http.StatusBadRequest, "ceremony_expired", "the passkey request expired; try again")
	case errors.Is(err, auth.ErrInvalidInput):
		writeContractError(w, http.StatusBadRequest, "invalid_passkey", err.Error())
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeContractError(w, http.StatusUnauthorized, "invalid_credentials", "the passkey was not accepted")
	case errors.Is(err, auth.ErrPasskeyNotFound):
		writeContractError(w, http.StatusNotFound, "not_found", "passkey not found")
	case errors.Is(err, auth.ErrPasskeysDisabled):
		writeContractError(w, http.StatusServiceUnavailable, "passkeys_disabled", "passkeys are not configured")
	default:
		s.deps.Logger.Error(action, "error", err)
		writeContractError(w, http.StatusInternalServerError, "auth_failed", "passkey request failed")
	}
}

func principalFrom(r *http.Request) auth.Principal {
	who, _ := r.Context().Value(principalKey{}).(auth.Principal)
	return who
}

func setCeremonyCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name: ceremonyCookieName, Value: id, Path: "/api/v1",
		MaxAge: int((5 * time.Minute).Seconds()), HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode,
	})
}

func takeCeremonyCookie(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(ceremonyCookieName)
	http.SetCookie(w, &http.Cookie{Name: ceremonyCookieName, Path: "/api/v1", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Server) handleReauth(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysReady(w) {
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, authBodyMaxBytes)).Decode(&body); err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if err := s.deps.Passkeys.Reauthenticate(r.Context(), principalFrom(r), body.Password); err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeContractError(w, http.StatusUnauthorized, "invalid_credentials", "wrong password")
			return
		}
		s.writePasskeyError(w, err, "reauthenticate")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListPasskeys(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysReady(w) {
		return
	}
	items, err := s.deps.Passkeys.ListPasskeys(r.Context(), principalFrom(r))
	if err != nil {
		s.writePasskeyError(w, err, "list passkeys")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleBeginPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysReady(w) {
		return
	}
	options, id, err := s.deps.Passkeys.BeginPasskeyRegistration(r.Context(), principalFrom(r))
	if err != nil {
		s.writePasskeyError(w, err, "begin passkey registration")
		return
	}
	setCeremonyCookie(w, id)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, options)
}

func (s *Server) handleFinishPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysReady(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	info, err := s.deps.Passkeys.FinishPasskeyRegistration(r.Context(), principalFrom(r), takeCeremonyCookie(w, r), r.URL.Query().Get("name"), r)
	if err != nil {
		s.writePasskeyError(w, err, "finish passkey registration")
		return
	}
	s.deps.Logger.Info("passkey added", "passkey_id", info.ID)
	writeJSON(w, http.StatusCreated, info)
}

func (s *Server) handleRenamePasskey(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysReady(w) {
		return
	}
	id, err := ids.Decode(ids.PasskeyPrefix, chi.URLParam(r, "passkeyId"))
	if err != nil {
		writeContractError(w, http.StatusNotFound, "not_found", "passkey not found")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, authBodyMaxBytes)).Decode(&body); err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	if err := s.deps.Passkeys.RenamePasskey(r.Context(), principalFrom(r), id, body.Name); err != nil {
		s.writePasskeyError(w, err, "rename passkey")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeletePasskey(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysReady(w) {
		return
	}
	id, err := ids.Decode(ids.PasskeyPrefix, chi.URLParam(r, "passkeyId"))
	if err != nil {
		writeContractError(w, http.StatusNotFound, "not_found", "passkey not found")
		return
	}
	if err := s.deps.Passkeys.DeletePasskey(r.Context(), principalFrom(r), id); err != nil {
		s.writePasskeyError(w, err, "delete passkey")
		return
	}
	s.deps.Logger.Info("passkey removed", "passkey_id", chi.URLParam(r, "passkeyId"))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBeginPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysReady(w) {
		return
	}
	options, id, err := s.deps.Passkeys.BeginPasskeyLogin()
	if err != nil {
		s.writePasskeyError(w, err, "begin passkey login")
		return
	}
	setCeremonyCookie(w, id)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, options)
}

func (s *Server) handleFinishPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysReady(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	token, who, err := s.deps.Passkeys.FinishPasskeyLogin(r.Context(), takeCeremonyCookie(w, r), r)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			s.deps.Logger.Warn("passkey login rejected", "client_ip", s.clientIP(r))
		}
		s.writePasskeyError(w, err, "finish passkey login")
		return
	}
	s.deps.Logger.Info("owner session created", "session_id", who.SessionID, "method", "passkey")
	http.SetCookie(w, sessionCookie(token, time.Now().Add(auth.SessionTTL)))
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, s.signedInResponse(who))
}
