package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mediaSyncEnqueuer struct {
	kind    string
	payload any
}

func (e *mediaSyncEnqueuer) EnqueueTx(_ *gorm.DB, kind string, payload any) (models.Job, error) {
	e.kind = kind
	e.payload = payload
	return models.Job{ID: uuid.MustParse("01900000-0000-7000-8000-000000000001"), Kind: kind, Status: jobs.StatusQueued, CreatedAt: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)}, nil
}

func TestHandleEnqueueMediaSync(t *testing.T) {
	enqueuer := &mediaSyncEnqueuer{}
	server := &Server{deps: Dependencies{JobEnqueuer: enqueuer}}
	router := chi.NewRouter()
	router.Post("/media/sync/{connectorId}", server.handleEnqueueMediaSync)

	req := httptest.NewRequest(http.MethodPost, "/media/sync/anilist", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if enqueuer.kind != jobs.KindMediaSyncAniList {
		t.Fatalf("kind = %q, want %q", enqueuer.kind, jobs.KindMediaSyncAniList)
	}
	if got := recorder.Body.String(); got != `{"id":"job_01900000-0000-7000-8000-000000000001","kind":"media_sync_anilist","status":"queued","created_at":"2026-07-20T00:00:00Z"}
` {
		t.Fatalf("body = %q", got)
	}
}

func TestHandleEnqueueMediaSyncRejectsUnsupportedConnector(t *testing.T) {
	server := &Server{deps: Dependencies{JobEnqueuer: &mediaSyncEnqueuer{}}}
	router := chi.NewRouter()
	router.Post("/media/sync/{connectorId}", server.handleEnqueueMediaSync)

	req := httptest.NewRequest(http.MethodPost, "/media/sync/unknown", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestMediaSyncJobKind(t *testing.T) {
	for _, test := range []struct {
		connector string
		want      string
	}{
		{connector: "anilist", want: jobs.KindMediaSyncAniList},
		{connector: "bangumi", want: jobs.KindMediaSyncBangumi},
	} {
		got, ok := mediaSyncJobKind(test.connector)
		if !ok || got != test.want {
			t.Errorf("mediaSyncJobKind(%q) = (%q, %t), want (%q, true)", test.connector, got, ok, test.want)
		}
	}
}
