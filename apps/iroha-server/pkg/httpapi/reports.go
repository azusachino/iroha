package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/reports"
)

func (s *Server) handleMonthlyReport(w http.ResponseWriter, r *http.Request) {
	month, timezone, ok := s.reportMonthScope(w, r, "month")
	if !ok {
		return
	}
	report, err := reports.GenerateMonthly(month, timezone, s.reportServices(), s.clockNow().UTC())
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

func (s *Server) handleMonthlyReportSeries(w http.ResponseWriter, r *http.Request) {
	endMonth, timezone, ok := s.reportMonthScope(w, r, "end")
	if !ok {
		return
	}
	months := reports.DefaultSeriesMonths
	if value := r.URL.Query().Get("months"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, reports.ErrInvalidSeriesMonths.Error())
			return
		}
		months = parsed
	}
	series, err := reports.GenerateMonthlySeries(endMonth, timezone, months, s.reportServices(), s.clockNow().UTC())
	if err != nil {
		if errors.Is(err, reports.ErrInvalidMonth) || errors.Is(err, reports.ErrInvalidTimezone) || errors.Is(err, reports.ErrInvalidSeriesMonths) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.deps.Logger.Error("generate monthly report series", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate monthly report series")
		return
	}
	writeJSON(w, http.StatusOK, series)
}

func (s *Server) reportMonthScope(w http.ResponseWriter, r *http.Request, legacy string) (string, string, bool) {
	scope, active, err := s.resolveReadScope(r)
	if err != nil {
		writeReadScopeError(w, err)
		return "", "", false
	}
	if active {
		if scope.Kind != ScopeMonth {
			writeError(w, http.StatusBadRequest, "invalid report month")
			return "", "", false
		}
		return scope.Calendar.From.Format(calendarMonthLayout), scope.Timezone, true
	}
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = s.deps.Config.Server.Timezone
	}
	return r.URL.Query().Get(legacy), timezone, true
}

func (s *Server) reportServices() reports.Services {
	return reports.Services{
		Activities: s.deps.ActivityService,
		Sleep:      s.deps.SleepService,
		Daily:      s.deps.DailyService,
		Media:      s.deps.MediaService,
		Expenses:   s.deps.ExpenseService,
	}
}
