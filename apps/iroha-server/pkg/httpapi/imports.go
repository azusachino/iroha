package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/go-chi/chi/v5"
)

type createImportJobRequest struct {
	RawFileID  string `json:"raw_file_id"`
	ParserKind string `json:"parser_kind"`
}

type importJobResponse struct {
	ID            string     `json:"id"`
	RawFileID     string     `json:"raw_file_id"`
	Status        string     `json:"status"`
	ParserKind    string     `json:"parser_kind"`
	ParserVersion string     `json:"parser_version"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (s *Server) handleCreateImportJob(w http.ResponseWriter, r *http.Request) {
	var request createImportJobRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if request.RawFileID == "" {
		writeError(w, http.StatusBadRequest, "raw_file_id is required")
		return
	}
	if request.ParserKind != coreimports.KindAppleHealthExport &&
		request.ParserKind != coreimports.KindGPX &&
		request.ParserKind != coreimports.KindAniList &&
		request.ParserKind != coreimports.KindBangumi {
		writeError(w, http.StatusBadRequest, "unsupported parser_kind")
		return
	}

	job, err := s.deps.ImportService.Create(imports.CreateInput{
		RawFileID:  request.RawFileID,
		ParserKind: request.ParserKind,
	})
	if err != nil {
		s.deps.Logger.Error("create import job", "error", err)
		writeError(w, http.StatusBadRequest, "failed to create import job")
		return
	}

	writeJSON(w, http.StatusAccepted, toImportJobResponse(job))
}

func (s *Server) handleListImportJobs(w http.ResponseWriter, _ *http.Request) {
	jobs, err := s.deps.ImportService.List(50)
	if err != nil {
		s.deps.Logger.Error("list import jobs", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list import jobs")
		return
	}

	response := make([]importJobResponse, 0, len(jobs))
	for _, job := range jobs {
		response = append(response, toImportJobResponse(job))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetImportJob(w http.ResponseWriter, r *http.Request) {
	job, found, err := s.deps.ImportService.Get(chi.URLParam(r, "importId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid import id")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "import not found")
		return
	}
	writeJSON(w, http.StatusOK, toImportJobResponse(job))
}

func toImportJobResponse(job models.ImportJob) importJobResponse {
	return importJobResponse{
		ID:            ids.Encode(ids.ImportPrefix, job.ID),
		RawFileID:     ids.Encode(ids.RawFilePrefix, job.RawFileID),
		Status:        job.Status,
		ParserKind:    job.ParserKind,
		ParserVersion: job.ParserVersion,
		ErrorMessage:  job.ErrorMessage,
		StartedAt:     job.StartedAt,
		FinishedAt:    job.FinishedAt,
		CreatedAt:     job.CreatedAt,
	}
}
