package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

// TestActiveRouteInventory keeps the active OpenAPI surface aligned with the
// routes registered by the server. Deferred roadmap resources must not enter
// the active contract until their handlers exist.
func TestActiveRouteInventory(t *testing.T) {
	server := &Server{mux: chi.NewRouter()}
	server.routes()

	got := activeRoutes(t, server.mux)

	want := []string{
		"GET /api/v1/activities",
		"GET /api/v1/activities/routes",
		"GET /api/v1/activities/summary",
		"GET /api/v1/activities/{activityId}",
		"GET /api/v1/activities/{activityId}/laps",
		"GET /api/v1/activities/{activityId}/route",
		"GET /api/v1/activities/{activityId}/samplings",
		"GET /api/v1/briefing",
		"GET /api/v1/daily",
		"GET /api/v1/daily/aggregates",
		"GET /api/v1/expenses",
		"GET /api/v1/expenses/{expenseId}",
		"GET /api/v1/imports",
		"GET /api/v1/imports/{importId}",
		"GET /api/v1/jobs",
		"GET /api/v1/jobs/{jobId}",
		"GET /api/v1/media",
		"GET /api/v1/media/aggregates",
		"GET /api/v1/media/events",
		"GET /api/v1/media/resolution-tasks",
		"GET /api/v1/media/{mediaId}",
		"GET /api/v1/raw-files",
		"GET /api/v1/raw-files/{rawFileId}",
		"GET /api/v1/metrics",
		"GET /api/v1/metrics/{metricId}",
		"GET /api/v1/reports/monthly",
		"GET /api/v1/sleep",
		"GET /api/v1/sleep/aggregates",
		"GET /api/v1/sleep/{sleepId}",
		"GET /api/v1/sleep/{sleepId}/segments",
		"GET /api/v1/tasks",
		"GET /healthz",
		"PATCH /api/v1/media/resolution-tasks/{taskId}",
		"PATCH /api/v1/tasks/{taskId}",
		"POST /api/v1/actions/{action}",
		"POST /api/v1/expenses",
		"POST /api/v1/imports",
		"POST /api/v1/media/sync/{connectorId}",
		"POST /api/v1/raw-files",
		"POST /api/v1/tasks",
		"PUT /api/v1/expenses/{expenseId}",
		"DELETE /api/v1/expenses/{expenseId}",
	}
	sort.Strings(want)

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("active route inventory differs\nwant:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
	}

	document := loadOpenAPI(t)
	openAPIRoutes := openAPIRouteInventory(t, document)
	if strings.Join(got, "\n") != strings.Join(openAPIRoutes, "\n") {
		t.Fatalf("OpenAPI route inventory differs from Chi\nwant:\n%s\ngot:\n%s", strings.Join(got, "\n"), strings.Join(openAPIRoutes, "\n"))
	}
	assertOpenAPIContract(t, document)
}

type openAPIDocument struct {
	OpenAPI    string                          `yaml:"openapi"`
	Paths      map[string]map[string]yaml.Node `yaml:"paths"`
	Components openAPIComponents               `yaml:"components"`
}

type openAPIComponents struct {
	Schemas map[string]yaml.Node `yaml:"schemas"`
}

func activeRoutes(t *testing.T, router chi.Router) []string {
	t.Helper()
	routes := make([]string, 0)
	if err := chi.Walk(router, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes = append(routes, method+" "+normalizeRoute(route))
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	sort.Strings(routes)
	return routes
}

func loadOpenAPI(t *testing.T) openAPIDocument {
	t.Helper()
	data, err := os.ReadFile("../../../../docs/contracts/openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI: %v", err)
	}
	var document openAPIDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	return document
}

func openAPIRouteInventory(t *testing.T, document openAPIDocument) []string {
	t.Helper()
	routes := make([]string, 0)
	for path, operations := range document.Paths {
		for method := range operations {
			switch method {
			case "get", "post", "patch", "put", "delete":
				routes = append(routes, strings.ToUpper(method)+" "+normalizeRoute(path))
			case "parameters":
			default:
				t.Fatalf("OpenAPI path %q has unsupported key %q", path, method)
			}
		}
	}
	sort.Strings(routes)
	return routes
}

func assertOpenAPIContract(t *testing.T, document openAPIDocument) {
	t.Helper()
	if document.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI version = %q, want 3.1.0", document.OpenAPI)
	}
	for path, operations := range document.Paths {
		for method, operation := range operations {
			if method == "parameters" {
				continue
			}
			if mappingValue(&operation, "responses") == nil {
				t.Fatalf("OpenAPI operation %s %s has no responses", strings.ToUpper(method), path)
			}
		}
	}
	for name, schema := range document.Components.Schemas {
		if yamlContainsKey(&schema, "nullable") {
			t.Fatalf("schema %q uses OpenAPI 3.0 nullable; use a 3.1 type union", name)
		}
	}
	for _, name := range []string{"Activity", "Sleep", "Daily", "Media", "Error"} {
		schema, ok := document.Components.Schemas[name]
		if !ok {
			t.Fatalf("missing schema %q", name)
		}
		if schemaValue(&schema, "additionalProperties") == "true" {
			t.Fatalf("schema %q is not an explicit wire contract", name)
		}
	}
}

func TestOpenAPIExamples(t *testing.T) {
	document := loadOpenAPI(t)
	routes := make(map[string]struct{})
	server := &Server{mux: chi.NewRouter()}
	server.routes()
	for _, route := range activeRoutes(t, server.mux) {
		routes[route] = struct{}{}
	}

	manifestData, err := os.ReadFile("../../../../docs/contracts/examples/manifest.json")
	if err != nil {
		t.Fatalf("read example manifest: %v", err)
	}
	var manifest struct {
		Examples []struct {
			File   string `json:"file"`
			Method string `json:"method"`
			Path   string `json:"path"`
			Schema string `json:"schema"`
		} `json:"examples"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse example manifest: %v", err)
	}
	if len(manifest.Examples) == 0 {
		t.Fatal("example manifest is empty")
	}
	for _, example := range manifest.Examples {
		t.Run(example.File, func(t *testing.T) {
			if _, ok := routes[example.Method+" "+example.Path]; !ok {
				t.Fatalf("example targets inactive route %s %s", example.Method, example.Path)
			}
			if _, ok := document.Components.Schemas[example.Schema]; !ok {
				t.Fatalf("example references missing schema %q", example.Schema)
			}
			data, err := os.ReadFile(filepath.Join("../../../../docs/contracts/examples", example.File))
			if err != nil {
				t.Fatalf("read example: %v", err)
			}
			var value map[string]any
			if err := json.Unmarshal(data, &value); err != nil {
				t.Fatalf("parse example JSON: %v", err)
			}
			for _, field := range exampleRequiredFields(example.Schema) {
				if _, ok := value[field]; !ok {
					t.Fatalf("example missing required field %q", field)
				}
			}
		})
	}
}

func exampleRequiredFields(schema string) []string {
	switch schema {
	case "ActivitySummary":
		return []string{"totals", "by_year", "by_month", "by_sport"}
	case "DailyPage", "ActivityPage":
		return []string{"items", "next_cursor", "has_more"}
	case "MediaPage":
		return []string{"items", "next_cursor", "has_more", "status_counts", "active_count"}
	case "SleepAggregateResponse":
		return []string{"granularity", "buckets"}
	case "Error":
		return []string{"code", "message", "request_id"}
	default:
		return nil
	}
}

func yamlContainsKey(node *yaml.Node, key string) bool {
	if node == nil {
		return false
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				return true
			}
			if yamlContainsKey(node.Content[i+1], key) {
				return true
			}
		}
	}
	for _, child := range node.Content {
		if yamlContainsKey(child, key) {
			return true
		}
	}
	return false
}

func schemaValue(node *yaml.Node, key string) string {
	value := mappingValue(node, key)
	if value == nil {
		return ""
	}
	return value.Value
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func normalizeRoute(route string) string {
	if route != "/" {
		route = strings.TrimSuffix(route, "/")
	}
	return route
}
