package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/reports"
)

func (s *Server) handleMonthlyReport(w http.ResponseWriter, r *http.Request) {
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = s.deps.Config.Server.Timezone
	}
	report, err := reports.GenerateMonthly(r.URL.Query().Get("month"), timezone, reports.Services{
		Activities: s.deps.ActivityService, Sleep: s.deps.SleepService, Daily: s.deps.DailyService,
		Media: s.deps.MediaService, Expenses: s.deps.ExpenseService,
	}, time.Now().UTC())
	if err != nil {
		if errors.Is(err, reports.ErrInvalidMonth) || errors.Is(err, reports.ErrInvalidTimezone) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.deps.Logger.Error("generate monthly report", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate monthly report")
		return
	}
	writeJSON(w, http.StatusOK, report)
}
