package httpapi

import (
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/mediaresolution"
)

func TestHandleListMediaResolutionTasks_NoServiceIsUnavailable(t *testing.T) {
	server := &Server{deps: Dependencies{Logger: slog.Default()}}
	request := httptest.NewRequest("GET", "/api/v1/media/resolution-tasks", nil)
	recorder := httptest.NewRecorder()
	server.handleListMediaResolutionTasks(recorder, request)
	if recorder.Code != 503 {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}

func TestHandleListMediaResolutionTasks_RejectsUnknownStatus(t *testing.T) {
	// Status validation must happen before any DB call, so a service backed
	// by a nil *gorm.DB is enough to exercise the 400 path without a real
	// database.
	server := &Server{deps: Dependencies{Logger: slog.Default(), MediaResolutionService: mediaresolution.NewService(nil)}}
	request := httptest.NewRequest("GET", "/api/v1/media/resolution-tasks?status=bogus", nil)
	recorder := httptest.NewRecorder()
	server.handleListMediaResolutionTasks(recorder, request)
	if recorder.Code != 400 {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestHandleUpdateMediaResolutionTask_NoServiceIsUnavailable(t *testing.T) {
	server := &Server{deps: Dependencies{Logger: slog.Default()}}
	request := httptest.NewRequest("PATCH", "/api/v1/media/resolution-tasks/medres_bogus", nil)
	recorder := httptest.NewRecorder()
	server.handleUpdateMediaResolutionTask(recorder, request)
	if recorder.Code != 503 {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}
