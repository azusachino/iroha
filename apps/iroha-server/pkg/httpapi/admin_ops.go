package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/go-chi/chi/v5"
)

type scheduleResponse struct {
	Kind         string     `json:"kind"`
	Enabled      bool       `json:"enabled"`
	ScheduleKind string     `json:"schedule_kind"`
	ScheduleExpr string     `json:"schedule_expr"`
	NextRunAt    *time.Time `json:"next_run_at"`
	LastRunAt    *time.Time `json:"last_run_at"`
}

type systemResponse struct {
	ParserVersion    string    `json:"parser_version"`
	MigrationVersion int64     `json:"migration_version"`
	DatabaseBytes    int64     `json:"database_bytes"`
	RawFileCount     int64     `json:"raw_file_count"`
	RawFileBytes     int64     `json:"raw_file_bytes"`
	CacheBackend     string    `json:"cache_backend"`
	Timezone         string    `json:"timezone"`
	PasskeysEnabled  bool      `json:"passkeys_enabled"`
	ServerTime       time.Time `json:"server_time"`
}

func (s *Server) jobsReady(w http.ResponseWriter) bool {
	if s.deps.JobsService == nil {
		writeContractError(w, http.StatusServiceUnavailable, "jobs_unavailable", "job service unavailable")
		return false
	}
	return true
}

func (s *Server) writeJobControlError(w http.ResponseWriter, err error, action string) {
	switch {
	case errors.Is(err, jobs.ErrJobNotFound):
		writeContractError(w, http.StatusNotFound, "not_found", "job or schedule not found")
	case errors.Is(err, jobs.ErrJobState):
		writeContractError(w, http.StatusConflict, "job_state", "the job cannot do that in its current state")
	default:
		s.deps.Logger.Error(action, "error", err)
		writeContractError(w, http.StatusInternalServerError, "jobs_failed", "job request failed")
	}
}

// handleRetryJob re-enqueues a failed or canceled job as a new job.
func (s *Server) handleRetryJob(w http.ResponseWriter, r *http.Request) {
	if !s.jobsReady(w) {
		return
	}
	id, err := ids.Decode(ids.JobPrefix, chi.URLParam(r, "jobId"))
	if err != nil {
		writeContractError(w, http.StatusNotFound, "not_found", "job not found")
		return
	}
	job, err := s.deps.JobsService.Retry(id)
	if err != nil {
		s.writeJobControlError(w, err, "retry job")
		return
	}
	s.deps.Logger.Info("job retried", "from", chi.URLParam(r, "jobId"), "job_id", ids.Encode(ids.JobPrefix, job.ID))
	writeJSON(w, http.StatusCreated, toJobResponse(job))
}

// handleCancelJob stops a queued job before it runs.
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	if !s.jobsReady(w) {
		return
	}
	id, err := ids.Decode(ids.JobPrefix, chi.URLParam(r, "jobId"))
	if err != nil {
		writeContractError(w, http.StatusNotFound, "not_found", "job not found")
		return
	}
	if err := s.deps.JobsService.Cancel(id); err != nil {
		s.writeJobControlError(w, err, "cancel job")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListSchedules(w http.ResponseWriter, _ *http.Request) {
	if !s.jobsReady(w) {
		return
	}
	rows, err := s.deps.JobsService.ListSchedules()
	if err != nil {
		s.writeJobControlError(w, err, "list schedules")
		return
	}
	items := make([]scheduleResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, scheduleResponse{
			Kind: row.Kind, Enabled: row.Enabled, ScheduleKind: row.ScheduleKind,
			ScheduleExpr: row.ScheduleExpr, NextRunAt: row.NextRunAt, LastRunAt: row.LastRunAt,
		})
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	if !s.jobsReady(w) {
		return
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, authBodyMaxBytes)).Decode(&body); err != nil || body.Enabled == nil {
		writeContractError(w, http.StatusBadRequest, "invalid_body", "expected {\"enabled\": true|false}")
		return
	}
	if err := s.deps.JobsService.SetScheduleEnabled(chi.URLParam(r, "kind"), *body.Enabled); err != nil {
		s.writeJobControlError(w, err, "update schedule")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRunSchedule(w http.ResponseWriter, r *http.Request) {
	if !s.jobsReady(w) {
		return
	}
	if err := s.deps.JobsService.RunScheduleNow(chi.URLParam(r, "kind")); err != nil {
		s.writeJobControlError(w, err, "run schedule")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// handleSystem reports deployment facts for the admin page.
func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	if s.deps.DB == nil {
		writeContractError(w, http.StatusServiceUnavailable, "system_unavailable", "database unavailable")
		return
	}
	db := s.deps.DB.WithContext(r.Context())
	out := systemResponse{
		ParserVersion:   s.deps.ParserVersion,
		CacheBackend:    s.deps.Config.Cache.Backend,
		Timezone:        s.deps.Config.Server.Timezone,
		PasskeysEnabled: s.passkeysEnabled(),
		ServerTime:      s.now().UTC(),
	}
	var raw struct {
		Count int64
		Bytes int64
	}
	queries := []error{
		db.Raw("select coalesce(max(version), 0) from _sqlx_migrations where success").Scan(&out.MigrationVersion).Error,
		db.Raw("select pg_database_size(current_database())").Scan(&out.DatabaseBytes).Error,
		db.Raw("select count(*) as count, coalesce(sum(size_bytes), 0) as bytes from tb_raw_files").Scan(&raw).Error,
	}
	if err := errors.Join(queries...); err != nil {
		s.deps.Logger.Error("read system state", "error", err)
		writeContractError(w, http.StatusInternalServerError, "system_unavailable", "failed to read system state")
		return
	}
	out.RawFileCount, out.RawFileBytes = raw.Count, raw.Bytes
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, out)
}
