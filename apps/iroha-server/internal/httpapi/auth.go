package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// requireUploadAuth guards the write endpoints (raw-file and import creation).
//
// When Auth.LocalNoAuth is true the request passes through unchanged, matching
// the local-only development posture. Otherwise the request must carry a valid
// "Authorization: Bearer <import_token>" header. This lets an external upload
// client such as a personal Telegram bot push raw bytes without exposing the
// write surface to unauthenticated callers. Read endpoints are never guarded.
func (s *Server) requireUploadAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.deps.Config.Auth.LocalNoAuth {
			next.ServeHTTP(w, r)
			return
		}

		expected := s.deps.Config.Auth.ImportToken
		if expected == "" {
			s.deps.Logger.Error("upload auth enabled but import_token is empty")
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}
