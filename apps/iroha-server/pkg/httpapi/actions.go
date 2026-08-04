package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	switch chi.URLParam(r, "action") {
	case "media-sync-anilist":
		s.enqueueMediaSync(w, "anilist")
	case "media-sync-bangumi":
		s.enqueueMediaSync(w, "bangumi")
	default:
		writeError(w, http.StatusBadRequest, "unsupported action")
	}
}
