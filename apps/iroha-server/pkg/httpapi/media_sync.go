package httpapi

import (
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/go-chi/chi/v5"
)

type mediaSyncJobResponse struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) handleEnqueueMediaSync(w http.ResponseWriter, r *http.Request) {
	if s.deps.JobEnqueuer == nil {
		writeError(w, http.StatusServiceUnavailable, "job dispatcher unavailable")
		return
	}

	connectorID := chi.URLParam(r, "connectorId")
	kind, ok := mediaSyncJobKind(connectorID)
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported media connector")
		return
	}

	job, err := s.deps.JobEnqueuer.EnqueueTx(nil, kind, map[string]string{
		"connector_id": connectorID,
	})
	if err != nil {
		s.deps.Logger.Error("enqueue media sync", "connector_id", connectorID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to enqueue media sync")
		return
	}

	writeJSON(w, http.StatusAccepted, toMediaSyncJobResponse(job))
}

func mediaSyncJobKind(connectorID string) (string, bool) {
	switch connectorID {
	case "anilist":
		return jobs.KindMediaSyncAniList, true
	case "bangumi":
		return jobs.KindMediaSyncBangumi, true
	default:
		return "", false
	}
}

func toMediaSyncJobResponse(job models.Job) mediaSyncJobResponse {
	return mediaSyncJobResponse{
		ID:        ids.Encode(ids.JobPrefix, job.ID),
		Kind:      job.Kind,
		Status:    job.Status,
		CreatedAt: job.CreatedAt.UTC(),
	}
}
