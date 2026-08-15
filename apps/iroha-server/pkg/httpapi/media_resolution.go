package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/mediaresolution"
	"github.com/go-chi/chi/v5"
)

type mediaResolutionTaskResponse struct {
	ID         string          `json:"id"`
	TaskType   string          `json:"task_type"`
	Status     string          `json:"status"`
	Candidates json.RawMessage `json:"candidates"`
	Resolution json.RawMessage `json:"resolution"`
	CreatedAt  time.Time       `json:"created_at"`
	ResolvedAt *time.Time      `json:"resolved_at,omitempty"`
}

type updateMediaResolutionTaskRequest struct {
	Status     string          `json:"status"`
	Resolution json.RawMessage `json:"resolution"`
}

func (s *Server) handleListMediaResolutionTasks(w http.ResponseWriter, r *http.Request) {
	if s.deps.MediaResolutionService == nil {
		writeError(w, http.StatusServiceUnavailable, "media resolution service unavailable")
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != mediaresolution.StatusOpen && status != mediaresolution.StatusResolved && status != mediaresolution.StatusDismissed {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	rows, err := s.deps.MediaResolutionService.List(mediaresolution.ListFilters{Status: status})
	if err != nil {
		s.deps.Logger.Error("list media resolution tasks", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list media resolution tasks")
		return
	}
	response := make([]mediaResolutionTaskResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, toMediaResolutionTaskResponse(row))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateMediaResolutionTask(w http.ResponseWriter, r *http.Request) {
	if s.deps.MediaResolutionService == nil {
		writeError(w, http.StatusServiceUnavailable, "media resolution service unavailable")
		return
	}
	id, err := ids.Decode(ids.MediaResolutionTaskPrefix, chi.URLParam(r, "taskId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid media resolution task id")
		return
	}
	var request updateMediaResolutionTaskRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	task, err := s.deps.MediaResolutionService.Resolve(id, request.Status, request.Resolution)
	if err != nil {
		if err == mediaresolution.ErrNotFound {
			writeError(w, http.StatusNotFound, "media resolution task not found or already closed")
			return
		}
		s.deps.Logger.Error("resolve media resolution task", "error", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.deps.Cache != nil {
		if err := s.deps.Cache.InvalidateChange(r.Context(), cache.ChangeMediaResolution); err != nil {
			s.deps.Logger.Error("invalidate caches after media resolution", "error", err)
		}
	}
	writeJSON(w, http.StatusOK, toMediaResolutionTaskResponse(task))
}

func toMediaResolutionTaskResponse(row models.MediaResolutionTask) mediaResolutionTaskResponse {
	return mediaResolutionTaskResponse{
		ID:         ids.Encode(ids.MediaResolutionTaskPrefix, row.ID),
		TaskType:   row.TaskType,
		Status:     row.Status,
		Candidates: row.CandidatesJSON,
		Resolution: row.ResolutionJSON,
		CreatedAt:  row.CreatedAt.UTC(),
		ResolvedAt: row.ResolvedAt,
	}
}
