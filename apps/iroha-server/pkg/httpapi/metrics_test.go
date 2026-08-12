package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
)

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
