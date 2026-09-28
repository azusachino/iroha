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
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/intakecredential"
)

const healthIntakeMaxBytes int64 = 10 << 20

type healthIntakeResponse struct {
	RawFileID string `json:"raw_file_id"`
	ImportID  string `json:"import_id"`
	Status    string `json:"status"`
}

// HealthIntakeVerifier checks the dedicated HAE intake credential.
type HealthIntakeVerifier interface {
	Verify(ctx context.Context, token string) error
}

func (s *Server) handleHealthIntake(w http.ResponseWriter, r *http.Request) {
	err := intakecredential.ErrNotProvisioned
	if verifier := s.deps.HealthIntakeCredentials; verifier != nil {
		err = verifier.Verify(r.Context(), bearerToken(r.Header.Get("Authorization")))
	}
	switch {
	case err == nil:
	case errors.Is(err, intakecredential.ErrNotProvisioned):
		writeContractError(w, http.StatusServiceUnavailable, "intake_not_provisioned", "health intake credential is not provisioned")
		return
	case errors.Is(err, intakecredential.ErrInvalid):
		w.Header().Set("WWW-Authenticate", `Bearer realm="iroha-health-intake"`)
		writeContractError(w, http.StatusUnauthorized, "unauthorized", "valid bearer credentials are required")
		return
	default:
		s.deps.Logger.Error("verify health intake credential", "error", err)
		writeContractError(w, http.StatusInternalServerError, "intake_failed", "failed to verify health intake credential")
		return
	}

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

	sourceKind := coreimports.KindHealthAutoExport
	filename := "health-auto-export.json"
	instanceKey := haeMeta.SourceInstanceKey
	if deviceHeader := r.Header.Get("X-Device-Id"); deviceHeader != "" {
		instanceKey = deviceHeader
	}
	capturedAt := haeMeta.CapturedAt

	rawFile, err := s.deps.RawFileService.StoreSnapshot(r.Context(), connector.Snapshot{
		ContentType:       "application/json",
		Body:              body,
		SourceKind:        sourceKind,
		Filename:          filename,
		SourceInstanceKey: instanceKey,
		IngestionMode:     rawfiles.IngestionModeBoundedReplacement,
		ObservedAt:        capturedAt,
	})
	if err != nil {
		s.deps.Logger.Error("store health intake snapshot", "error", err)
		writeContractError(w, http.StatusInternalServerError, "intake_failed", "failed to store health intake")
		return
	}

	job, err := s.deps.ImportService.Create(imports.CreateInput{
		RawFileID:  ids.Encode(ids.RawFilePrefix, rawFile.ID),
		ParserKind: sourceKind,
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
