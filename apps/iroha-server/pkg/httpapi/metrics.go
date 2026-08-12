package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metricseries"
	"github.com/go-chi/chi/v5"
)

type metricCatalogResponse struct {
	Schema  string               `json:"schema"`
	Metrics []metrics.Definition `json:"metrics,omitempty"`
	Metric  *metrics.Definition  `json:"metric,omitempty"`
}

func (s *Server) handleListMetrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, metricCatalogResponse{
		Schema:  "metric-catalog.v1",
		Metrics: s.deps.MetricRegistry.List(),
	})
}

func (s *Server) handleGetMetric(w http.ResponseWriter, r *http.Request) {
	definition, err := s.deps.MetricRegistry.Get(chi.URLParam(r, "metricId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "metric not found")
		return
	}
	writeJSON(w, http.StatusOK, metricCatalogResponse{
		Schema: "metric-catalog.v1",
		Metric: &definition,
	})
}

func (s *Server) handleMetricSeries(w http.ResponseWriter, r *http.Request) {
	if s.deps.MetricSeriesService == nil {
		writeError(w, http.StatusServiceUnavailable, "metric series service unavailable")
		return
	}
	request, ok := parseMetricSeriesRequest(w, r)
	if !ok {
		return
	}
	series, err := s.deps.MetricSeriesService.Series(r.Context(), request)
	if err != nil {
		switch err {
		case metricseries.ErrMetricNotFound:
			writeError(w, http.StatusNotFound, "metric not found")
		case metricseries.ErrInvalidRequest:
			writeError(w, http.StatusBadRequest, "invalid metric series request")
		default:
			s.deps.Logger.Error("build metric series", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to build metric series")
		}
		return
	}
	writeJSON(w, http.StatusOK, series)
}

func parseMetricSeriesRequest(w http.ResponseWriter, r *http.Request) (metricseries.Request, bool) {
	query := r.URL.Query()
	from, err := time.Parse("2006-01-02", query.Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid metric series request")
		return metricseries.Request{}, false
	}
	to, err := time.Parse("2006-01-02", query.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid metric series request")
		return metricseries.Request{}, false
	}
	grain := query.Get("grain")
	if grain == "" {
		writeError(w, http.StatusBadRequest, "invalid metric series request")
		return metricseries.Request{}, false
	}
	timezone := query.Get("timezone")
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid metric series request")
		return metricseries.Request{}, false
	}
	dimensions := make(map[string][]string)
	for _, raw := range query["dimension"] {
		name, value, found := strings.Cut(raw, ":")
		if !found || name == "" || value == "" {
			writeError(w, http.StatusBadRequest, "invalid metric series request")
			return metricseries.Request{}, false
		}
		dimensions[name] = append(dimensions[name], value)
	}
	return metricseries.Request{
		MetricID:   chi.URLParam(r, "metricId"),
		From:       time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, location),
		To:         time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, location),
		Grain:      grain,
		Timezone:   location,
		Dimensions: dimensions,
	}, true
}
