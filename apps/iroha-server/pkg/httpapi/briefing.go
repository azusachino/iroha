package httpapi

import (
	"net/http"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/briefing"
)

func (s *Server) handleBriefing(w http.ResponseWriter, r *http.Request) {
	scope, active, err := s.resolveReadScope(r)
	if err != nil {
		writeReadScopeError(w, err)
		return
	}
	date := r.URL.Query().Get("date")
	timezone := s.deps.Config.Server.Timezone
	if active {
		if scope.Kind != ScopeDay {
			writeError(w, http.StatusBadRequest, "invalid date")
			return
		}
		date = scope.Calendar.From.Format(calendarDateLayout)
		timezone = scope.Timezone
	}
	day, err := briefing.ParseDayInLocation(date, timezone)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date")
		return
	}
	writeJSON(w, http.StatusOK, s.deps.BriefingRegistry.Build(r.Context(), day))
}
