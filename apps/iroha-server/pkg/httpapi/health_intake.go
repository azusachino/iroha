package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-providers/parsers"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/intakecredential"
)

const healthIntakeMaxBytes int64 = 10 << 20

type healthIntakeResponse struct {
	RawFileID string `json:"raw_file_id"`
	ImportID  string `json:"import_id"`
	Status    string `json:"status"`
}

// HealthIntakeVerifier resolves a presented HAE intake token to its credential.
type HealthIntakeVerifier interface {
	Verify(ctx context.Context, token string) (models.IntakeCredential, error)
}

type intakeCredentialKey struct{}

// requireIntakeCredential authenticates the request with an HAE intake
// credential before the handler reads the body. It fails closed: no verifier
// or no active credential yields 503, a missing or wrong token 401.
func (s *Server) requireIntakeCredential(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var credential models.IntakeCredential
		err := intakecredential.ErrNotProvisioned
		if verifier := s.deps.HealthIntakeCredentials; verifier != nil {
			credential, err = verifier.Verify(r.Context(), bearerToken(r.Header.Get("Authorization")))
		}
		switch {
		case err == nil:
			s.deps.Logger.Info("intake credential accepted", "credential_id", ids.Encode(ids.IntakeCredentialPrefix, credential.ID), "credential_name", credential.Name)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), intakeCredentialKey{}, credential)))
		case errors.Is(err, intakecredential.ErrNotProvisioned):
			writeContractError(w, http.StatusServiceUnavailable, "intake_not_provisioned", "health intake credential is not provisioned")
		case errors.Is(err, intakecredential.ErrInvalid):
			s.deps.Logger.Warn("intake credential rejected", "remote_addr", r.RemoteAddr)
			w.Header().Set("WWW-Authenticate", `Bearer realm="iroha-health-intake"`)
			writeContractError(w, http.StatusUnauthorized, "unauthorized", "valid bearer credentials are required")
		default:
			s.deps.Logger.Error("verify health intake credential", "error", err)
			writeContractError(w, http.StatusInternalServerError, "intake_failed", "failed to verify health intake credential")
		}
	})
}

func (s *Server) handleHealthIntake(w http.ResponseWriter, r *http.Request) {
	credential, _ := r.Context().Value(intakeCredentialKey{}).(models.IntakeCredential)

	r.Body = http.MaxBytesReader(w, r.Body, healthIntakeMaxBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_body", "health intake body is too large or unreadable")
		return
	}

	haeMeta, err := parsers.ValidateHealthAutoExport(body)
	if err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_health_payload", "invalid health payload format")
		return
	}

	rawFile, err := s.deps.RawFileService.StoreSnapshot(r.Context(), connector.Snapshot{
		ContentType:       "application/json",
		Body:              body,
		SourceKind:        coreimports.KindHealthAutoExport,
		Filename:          "health-auto-export.json",
		SourceInstanceKey: parsers.HealthAutoExportInstance(credential.Name),
		IngestionMode:     rawfiles.IngestionModeBoundedReplacement,
		ObservedAt:        haeMeta.CapturedAt,
	})
	if err != nil {
		s.deps.Logger.Error("store health intake snapshot", "error", err)
		writeContractError(w, http.StatusInternalServerError, "intake_failed", "failed to store health intake")
		return
	}

	job, err := s.deps.ImportService.Create(imports.CreateInput{
		RawFileID:  ids.Encode(ids.RawFilePrefix, rawFile.ID),
		ParserKind: coreimports.KindHealthAutoExport,
	})
	if err != nil {
		s.deps.Logger.Error("create health intake import", "error", err)
		writeContractError(w, http.StatusInternalServerError, "intake_failed", "failed to queue health intake")
		return
	}

	writeJSON(w, http.StatusAccepted, healthIntakeResponse{
		RawFileID: ids.Encode(ids.RawFilePrefix, rawFile.ID),
		ImportID:  ids.Encode(ids.ImportPrefix, job.ID),
		Status:    job.Status,
	})
}

// bearerToken extracts the token from an Authorization header, or "" when the
// header is not a single-token Bearer credential.
func bearerToken(header string) string {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.Contains(token, " ") {
		return ""
	}
	return token
}
