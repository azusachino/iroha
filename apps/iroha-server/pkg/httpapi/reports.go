package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/reports"
)

func (s *Server) handleMonthlyReport(w http.ResponseWriter, r *http.Request) {
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = s.deps.Config.Server.Timezone
	}
	report, err := reports.GenerateMonthly(r.URL.Query().Get("month"), timezone, s.reportServices(), time.Now().UTC())
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
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = s.deps.Config.Server.Timezone
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
	series, err := reports.GenerateMonthlySeries(r.URL.Query().Get("end"), timezone, months, s.reportServices(), time.Now().UTC())
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

func (s *Server) reportServices() reports.Services {
	return reports.Services{
		Activities: s.deps.ActivityService,
		Sleep:      s.deps.SleepService,
		Daily:      s.deps.DailyService,
		Media:      s.deps.MediaService,
		Expenses:   s.deps.ExpenseService,
	}
}
