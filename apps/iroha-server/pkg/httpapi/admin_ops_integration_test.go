//go:build integration

package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
)

func adminStatus(t *testing.T, handler http.Handler, method, path, body string, want int) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(csrfHeaderName, testCSRFToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("%s %s = %d, want %d: %s", method, path, rec.Code, want, rec.Body.String())
	}
}

func TestIntegrationAdminJobControlsAndSystem(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	db.Exec("delete from tb_job_schedules where kind = 'test_admin_schedule'")
	t.Cleanup(func() { db.Exec("delete from tb_job_schedules where kind = 'test_admin_schedule'") })

	jobsService := jobs.NewService(db, slog.Default(), nil)
	server := NewServer(Dependencies{Auth: allowAllAuth{}, DB: db, JobsService: jobsService, ParserVersion: "test-parser"})

	job, err := jobsService.Enqueue(jobs.EnqueueInput{Kind: "test_admin_job", Payload: map[string]string{"a": "b"}})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	id := ids.Encode(ids.JobPrefix, job.ID)

	adminStatus(t, server, http.MethodPost, "/api/v1/jobs/"+id+"/retry", "", http.StatusConflict)
	adminStatus(t, server, http.MethodPost, "/api/v1/jobs/"+id+"/cancel", "", http.StatusNoContent)
	adminStatus(t, server, http.MethodPost, "/api/v1/jobs/"+id+"/cancel", "", http.StatusConflict)
	var retried string
	requestJSON(t, server, http.MethodPost, "/api/v1/jobs/"+id+"/retry", "", http.StatusCreated, func(body map[string]any) {
		retried = stringValue(t, body, "id")
		if body["status"] != "queued" || body["kind"] != "test_admin_job" {
			t.Fatalf("retried job = %v", body)
		}
	})
	if retried == id {
		t.Fatal("retry reused the original job instead of enqueuing a new one")
	}
	adminStatus(t, server, http.MethodPost, "/api/v1/jobs/job_00000000-0000-7000-8000-000000000000/cancel", "", http.StatusNotFound)

	if _, err := jobsService.CreateSchedule(jobs.ScheduleInput{Kind: "test_admin_schedule", ScheduleKind: jobs.ScheduleKindInterval, ScheduleExpr: "24h", Enabled: true}); err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	adminStatus(t, server, http.MethodPatch, "/api/v1/admin/schedules/test_admin_schedule", "", http.StatusBadRequest)
	adminStatus(t, server, http.MethodPatch, "/api/v1/admin/schedules/test_admin_schedule", `{"enabled":false}`, http.StatusNoContent)
	adminStatus(t, server, http.MethodPost, "/api/v1/admin/schedules/test_admin_schedule/run", "", http.StatusAccepted)
	adminStatus(t, server, http.MethodPost, "/api/v1/admin/schedules/no_such_kind/run", "", http.StatusNotFound)
	var enabled bool
	db.Raw("select enabled from tb_job_schedules where kind = 'test_admin_schedule'").Scan(&enabled)
	if enabled {
		t.Fatal("schedule still enabled after PATCH enabled=false")
	}

	requestJSON(t, server, http.MethodGet, "/api/v1/admin/system", "", http.StatusOK, func(body map[string]any) {
		if body["parser_version"] != "test-parser" {
			t.Fatalf("parser_version = %v", body["parser_version"])
		}
		if v, _ := body["migration_version"].(float64); v < 24 {
			t.Fatalf("migration_version = %v, want >= 24", body["migration_version"])
		}
		if v, _ := body["database_bytes"].(float64); v <= 0 {
			t.Fatalf("database_bytes = %v", body["database_bytes"])
		}
	})
}
