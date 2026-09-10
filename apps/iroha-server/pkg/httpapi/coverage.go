package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/coverage"
	"github.com/google/uuid"
)

func (s *Server) handleCoverage(w http.ResponseWriter, r *http.Request) {
	if s.deps.CoverageService == nil {
		writeError(w, http.StatusServiceUnavailable, "coverage service unavailable")
		return
	}
	sourceInstanceID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("source_instance_id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid source_instance_id")
		return
	}
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if category == "" {
		writeError(w, http.StatusBadRequest, "category is required")
		return
	}
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = s.deps.Config.Server.Timezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid timezone")
		return
	}
	from, err := parseCoverageTime(r.URL.Query().Get("from"), location)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from")
		return
	}
	to, err := parseCoverageTime(r.URL.Query().Get("to"), location)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to")
		return
	}
	result, err := s.deps.CoverageService.Query(r.Context(), coverage.QueryFilters{
		SourceInstanceID: sourceInstanceID, Category: category, From: from, To: to,
	})
	if err != nil {
		if errors.Is(err, coverage.ErrInvalidQuery) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.deps.Logger.Error("query source coverage", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to query source coverage")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseCoverageTime(value string, location *time.Location) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("coverage time is required")
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
