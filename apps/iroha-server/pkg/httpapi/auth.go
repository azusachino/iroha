package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/golang-jwt/jwt/v5"
)

type authContextKey struct{}

type jwtClaims struct {
	Scope string `json:"scope"`
	jwt.RegisteredClaims
}

// requireJWT protects the private API in authenticated deployments. Local
// no-auth mode remains an explicit trusted-development bypass.
func (s *Server) requireJWT(requiredScope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.deps.Config.Auth.LocalNoAuth {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := parseJWT(r, s.deps.Config.Auth)
			if err != nil {
				writeContractError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing bearer token")
				return
			}
			if !claims.hasScope(requiredScope) {
				writeContractError(w, http.StatusForbidden, "forbidden", "insufficient scope")
				return
			}

			ctx := context.WithValue(r.Context(), authContextKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseJWT(r *http.Request, auth config.AuthConfig) (jwtClaims, error) {
	tokenValue, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok || auth.JWTSecret == "" {
		return jwtClaims{}, jwt.ErrTokenMalformed
	}

	claims := jwtClaims{}
	_, err := jwt.ParseWithClaims(tokenValue, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(auth.JWTSecret), nil
	}, jwt.WithIssuer(auth.JWTIssuer), jwt.WithAudience(auth.JWTAudience), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return jwtClaims{}, err
	}
	return claims, nil
}

func (claims jwtClaims) hasScope(required string) bool {
	for _, scope := range strings.Fields(claims.Scope) {
		if scope == required || (required == "iroha:read" && scope == "iroha:write") {
			return true
		}
	}
	return false
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

func authSubject(r *http.Request) string {
	claims, ok := r.Context().Value(authContextKey{}).(jwtClaims)
	if !ok || claims.Subject == "" {
		return ""
	}
	return claims.Subject
}
