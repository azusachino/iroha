package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metricseries"
)

type fakeMetricDailySource struct {
	values []metricseries.DailyMetricValue
}

func (f fakeMetricDailySource) MetricValues(context.Context, string, time.Time, time.Time) ([]metricseries.DailyMetricValue, error) {
	return f.values, nil
}

func TestHandleListMetricsReturnsCatalog(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewServer(Dependencies{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Schema  string               `json:"schema"`
		Metrics []metrics.Definition `json:"metrics"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Schema != "metric-catalog.v1" {
		t.Fatalf("schema = %q, want metric-catalog.v1", response.Schema)
	}
	if len(response.Metrics) != 16 {
		t.Fatalf("metric count = %d, want 16", len(response.Metrics))
	}
}

func TestHandleGetMetricReturnsDefinition(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewServer(Dependencies{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/metrics/expenses.amount_minor", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Schema string             `json:"schema"`
		Metric metrics.Definition `json:"metric"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Schema != "metric-catalog.v1" || response.Metric.ID != "expenses.amount_minor" {
		t.Fatalf("response = %+v", response)
	}
}

func TestHandleGetMetricReturnsNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewServer(Dependencies{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/metrics/missing.metric", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestHandleMetricSeriesReturnsServerAggregatedSeries(t *testing.T) {
	registry, err := metrics.DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	seriesService := metricseries.NewService(registry, fakeMetricDailySource{values: []metricseries.DailyMetricValue{
		{Day: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC), Value: 1000, Source: "watch"},
	}}, nil, nil, nil, nil)
	recorder := httptest.NewRecorder()
	NewServer(Dependencies{MetricSeriesService: seriesService}).ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/metrics/health.steps/series?from=2026-01-01&to=2026-03-01&grain=month&timezone=UTC",
		nil,
	))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"schema":"metric-series.v1"`) || !strings.Contains(recorder.Body.String(), `"value":1000`) {
		t.Fatalf("series response = %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"value":null`) {
		t.Fatalf("series response does not preserve missing period: %s", recorder.Body.String())
	}
}

func TestHandleMetricSeriesRejectsInvalidTimezone(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewServer(Dependencies{MetricSeriesService: metricseries.NewService(nil, nil, nil, nil, nil, nil)}).ServeHTTP(recorder, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/metrics/health.steps/series?from=2026-01-01&to=2026-02-01&grain=month&timezone=Not%2FATimezone",
		nil,
	))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
