package httpapi

import (
	"crypto/subtle"
	"io"
	"net/http"
	"strings"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-providers/parsers"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
)

const healthIntakeMaxBytes int64 = 2 << 20

type healthIntakeResponse struct {
	RawFileID string `json:"raw_file_id"`
	ImportID  string `json:"import_id"`
	Status    string `json:"status"`
}

func (s *Server) handleHealthIntake(w http.ResponseWriter, r *http.Request) {
	token := s.deps.Config.Server.HealthIntakeToken
	if token == "" {
		writeContractError(w, http.StatusServiceUnavailable, "health_intake_disabled", "health intake is not configured")
		return
	}
	if !validBearerToken(r.Header.Get("Authorization"), token) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="iroha-health-intake"`)
		writeContractError(w, http.StatusUnauthorized, "unauthorized", "valid bearer credentials are required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, healthIntakeMaxBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_body", "health intake body is too large or unreadable")
		return
	}
	metadata, err := parsers.ValidateAppleHealthShortcut(body)
	if err != nil {
		writeContractError(w, http.StatusBadRequest, "invalid_health_payload", "invalid Apple Health Shortcut payload")
		return
	}

	rawFile, err := s.deps.RawFileService.StoreSnapshot(r.Context(), connector.Snapshot{
		ContentType:       "application/json",
		Body:              body,
		SourceKind:        coreimports.KindAppleHealthShortcut,
		Filename:          "apple-health-shortcut.json",
		SourceInstanceKey: metadata.SourceInstanceKey,
		IngestionMode:     rawfiles.IngestionModeBoundedReplacement,
		ObservedAt:        metadata.CapturedAt,
	})
	if err != nil {
		s.deps.Logger.Error("store health intake snapshot", "error", err)
		writeContractError(w, http.StatusInternalServerError, "intake_failed", "failed to store health intake")
		return
	}

	job, err := s.deps.ImportService.Create(imports.CreateInput{
		RawFileID:  ids.Encode(ids.RawFilePrefix, rawFile.ID),
		ParserKind: coreimports.KindAppleHealthShortcut,
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

func validBearerToken(header, expected string) bool {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}
