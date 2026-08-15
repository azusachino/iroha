package httpapi

import (
	"net/http"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/briefing"
)

func (s *Server) handleBriefing(w http.ResponseWriter, r *http.Request) {
	timezone := s.deps.Config.Server.Timezone
	if timezone == "" {
		timezone = "Asia/Tokyo"
	}
	day, err := briefing.ParseDayInLocation(r.URL.Query().Get("date"), timezone)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date")
		return
	}
	writeJSON(w, http.StatusOK, s.deps.BriefingRegistry.Build(r.Context(), day))
}
