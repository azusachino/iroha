package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/intakecredential"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// IntakeCredentialAdmin manages HAE intake credentials from the admin page.
type IntakeCredentialAdmin interface {
	Issue(ctx context.Context, name string) (models.IntakeCredential, string, error)
	List(ctx context.Context) ([]models.IntakeCredential, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type intakeCredentialResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

type issuedIntakeCredentialResponse struct {
	Credential intakeCredentialResponse `json:"credential"`
	// Token is shown once; it is never stored or returned again.
	Token string `json:"token"`
}

func (s *Server) handleListIntakeCredentials(w http.ResponseWriter, r *http.Request) {
	if !s.intakeAdminConfigured(w) {
		return
	}
	rows, err := s.deps.IntakeCredentialAdmin.List(r.Context())
	if err != nil {
		s.deps.Logger.Error("list intake credentials", "error", err)
		writeContractError(w, http.StatusInternalServerError, "intake_credentials_failed", "failed to list intake credentials")
		return
	}
	items := make([]intakeCredentialResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, toIntakeCredentialResponse(row))
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleIssueIntakeCredential(w http.ResponseWriter, r *http.Request) {
	if !s.intakeAdminConfigured(w) {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, authBodyMaxBytes)).Decode(&body); err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	row, token, err := s.deps.IntakeCredentialAdmin.Issue(r.Context(), body.Name)
	if err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_credential_name", "name must be lowercase letters, digits, and dashes")
		return
	}
	s.deps.Logger.Info("intake credential issued", "credential_id", ids.Encode(ids.IntakeCredentialPrefix, row.ID), "credential_name", row.Name)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, issuedIntakeCredentialResponse{Credential: toIntakeCredentialResponse(row), Token: token})
}

func (s *Server) handleRevokeIntakeCredential(w http.ResponseWriter, r *http.Request) {
	if !s.intakeAdminConfigured(w) {
		return
	}
	id, err := ids.Decode(ids.IntakeCredentialPrefix, chi.URLParam(r, "credentialId"))
	if err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_id", "invalid credential id")
		return
	}
	switch err := s.deps.IntakeCredentialAdmin.Revoke(r.Context(), id); {
	case err == nil:
		s.deps.Logger.Info("intake credential revoked", "credential_id", chi.URLParam(r, "credentialId"))
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, intakecredential.ErrNotFound):
		writeContractError(w, http.StatusNotFound, "not_found", "no active credential with that id")
	default:
		s.deps.Logger.Error("revoke intake credential", "error", err)
		writeContractError(w, http.StatusInternalServerError, "intake_credentials_failed", "failed to revoke intake credential")
	}
}

func (s *Server) intakeAdminConfigured(w http.ResponseWriter) bool {
	if s.deps.IntakeCredentialAdmin == nil {
		writeContractError(w, http.StatusServiceUnavailable, "intake_admin_not_configured", "intake credential management is not configured")
		return false
	}
	return true
}

func toIntakeCredentialResponse(row models.IntakeCredential) intakeCredentialResponse {
	return intakeCredentialResponse{
		ID:         ids.Encode(ids.IntakeCredentialPrefix, row.ID),
		Name:       row.Name,
		CreatedAt:  row.CreatedAt,
		LastUsedAt: row.LastUsedAt,
		RevokedAt:  row.RevokedAt,
	}
}
