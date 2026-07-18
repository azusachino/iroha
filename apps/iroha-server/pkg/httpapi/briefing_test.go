package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/briefing"
)

type briefingTestContributor struct{}

func (briefingTestContributor) Key() string    { return "daily" }
func (briefingTestContributor) Schema() string { return "daily.day.v1" }
func (briefingTestContributor) Contribute(context.Context, briefing.Day) (briefing.Section, error) {
	return briefing.Section{Data: map[string]int{"steps": 3}}, nil
}

func TestHandleBriefingReturnsVersionedSections(t *testing.T) {
	registry, err := briefing.NewRegistry(briefingTestContributor{})
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	server := NewServer(Dependencies{
		Config:           config.Config{Auth: config.AuthConfig{LocalNoAuth: true}},
		BriefingRegistry: registry,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/briefing?date=2026-07-14", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q", got)
	}
	want := `{"date":"2026-07-14","previous_date":"2026-07-13","next_date":"2026-07-15","sections":[{"key":"daily","schema":"daily.day.v1","state":"ready","data":{"steps":3}}]}` + "\n"
	if recorder.Body.String() != want {
		t.Fatalf("body = %s, want %s", recorder.Body.String(), want)
	}
}

func TestHandleBriefingRejectsInvalidDate(t *testing.T) {
	server := NewServer(Dependencies{Config: config.Config{Auth: config.AuthConfig{LocalNoAuth: true}}})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/briefing?date=bad", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}
