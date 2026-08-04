package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/go-chi/chi/v5"
)

type jobResponse struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"`
	Status       string     `json:"status"`
	Attempts     int        `json:"attempts"`
	MaxAttempts  int        `json:"max_attempts"`
	RunAfter     time.Time  `json:"run_after"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	if s.deps.JobsService == nil {
		writeError(w, http.StatusServiceUnavailable, "job service unavailable")
		return
	}
	limit := jobs.DefaultLimit
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsed
	}
	rows, err := s.deps.JobsService.List(jobs.ListFilters{
		Kind: r.URL.Query().Get("kind"), Status: r.URL.Query().Get("status"), Limit: limit,
	})
	if err != nil {
		s.deps.Logger.Error("list jobs", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}
	response := make([]jobResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, toJobResponse(row))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	if s.deps.JobsService == nil {
		writeError(w, http.StatusServiceUnavailable, "job service unavailable")
		return
	}
	id, err := ids.Decode(ids.JobPrefix, chi.URLParam(r, "jobId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	job, found, err := s.deps.JobsService.Get(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get job")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, toJobResponse(job))
}

func toJobResponse(job models.Job) jobResponse {
	return jobResponse{
		ID: ids.Encode(ids.JobPrefix, job.ID), Kind: job.Kind, Status: job.Status,
		Attempts: job.Attempts, MaxAttempts: job.MaxAttempts, RunAfter: job.RunAfter.UTC(),
		ErrorMessage: job.ErrorMessage, StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
		CreatedAt: job.CreatedAt.UTC(), UpdatedAt: job.UpdatedAt.UTC(),
	}
}
