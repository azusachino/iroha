package httpapi

import (
	"net/http"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
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
