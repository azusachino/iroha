//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

func TestIntegrationConnectionsFreshnessPolicy(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	now := time.Date(2099, time.December, 31, 12, 0, 0, 0, time.UTC)
	name := "freshness-" + uuid.NewString()
	credential := models.IntakeCredential{ID: uuid.New(), Name: name, TokenSHA256: uuid.NewString(), CreatedAt: now}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	source := models.SourceInstance{ID: uuid.New(), Provider: "health_auto_export", InstanceKey: "iphone-hae:" + name, CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{DB: db}, now: func() time.Time { return now }}
	response, err := server.connection(context.Background(), source)
	if err != nil || response.Freshness != "overdue" || response.ExpectedIntervalS == nil || *response.ExpectedIntervalS != 86400 || response.Collection != "unknown" {
		t.Fatalf("never-delivered source: %+v, %v", response, err)
	}
	raw := models.RawFile{ID: uuid.New(), SHA256: uuid.NewString(), OriginalFilename: "synthetic", StoragePath: "synthetic", SourceKind: "health_auto_export", UploadedVia: "test", CreatedAt: now}
	if err := db.Create(&raw).Error; err != nil {
		t.Fatal(err)
	}
	receipt := models.SourceReceipt{ID: uuid.New(), SourceInstanceID: source.ID, RawFileID: raw.ID, SourceKind: "health_auto_export", IngestionMode: "incremental", ScopeJSON: json.RawMessage(`{}`), ReceivedAt: now.Add(-25 * time.Hour), CreatedAt: now}
	if err := db.Create(&receipt).Error; err != nil {
		t.Fatal(err)
	}
	response, err = server.connection(context.Background(), source)
	if err != nil || response.Freshness != "overdue" || !response.NextExpectedAt.Equal(now.Add(-time.Hour)) {
		t.Fatalf("late delivery: %+v, %v", response, err)
	}
	if err := db.Model(&receipt).Update("received_at", now).Error; err != nil {
		t.Fatal(err)
	}
	response, err = server.connection(context.Background(), source)
	if err != nil || response.Freshness != "within_cadence" || response.Collection != "unknown" {
		t.Fatalf("new delivery is not coverage: %+v, %v", response, err)
	}
	if err := db.Model(&credential).Update("revoked_at", now).Error; err != nil {
		t.Fatal(err)
	}
	response, err = server.connection(context.Background(), source)
	if err != nil || response.Freshness != "not_scheduled" || response.NextExpectedAt != nil {
		t.Fatalf("revoked source: %+v, %v", response, err)
	}
	source.Provider = "anilist"
	schedule := models.JobSchedule{ID: uuid.New(), Kind: jobs.KindMediaSyncAniList, Enabled: true, ScheduleKind: jobs.ScheduleKindInterval, ScheduleExpr: "48h", PayloadJSON: json.RawMessage(`{}`), CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&schedule).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Delete(&schedule) })
	response, err = server.connection(context.Background(), source)
	if err != nil || response.Freshness != "within_cadence" || *response.ExpectedIntervalS != int64((48*time.Hour)/time.Second) {
		t.Fatalf("connector interval: %+v, %v", response, err)
	}
	if err := db.Model(&schedule).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	response, err = server.connection(context.Background(), source)
	if err != nil || response.Freshness != "not_scheduled" {
		t.Fatalf("disabled schedule: %+v, %v", response, err)
	}
	if err := db.Model(&schedule).Updates(map[string]any{"enabled": true, "schedule_expr": "invalid"}).Error; err != nil {
		t.Fatal(err)
	}
	response, err = server.connection(context.Background(), source)
	if err != nil || response.Freshness != "unknown" {
		t.Fatalf("unknown cadence: %+v, %v", response, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := server.connection(ctx, source); err == nil {
		t.Fatal("connection queries ignored cancellation")
	}
}

func TestIntegrationConnectionsExposeEvidenceAndExecutableActions(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })

	now := time.Date(2099, time.January, 5, 12, 0, 0, 0, time.UTC)
	sourceID := uuid.New()
	rawID := uuid.New()
	receiptID := uuid.New()
	importID := uuid.New()
	if err := db.Create(&models.SourceInstance{ID: sourceID, Provider: "gpx", InstanceKey: "watch-1", DisplayName: "Watch 1", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create source instance: %v", err)
	}
	if err := db.Create(&models.RawFile{ID: rawID, SHA256: "connections-" + rawID.String(), OriginalFilename: "retry.gpx", StoragePath: "/tmp/retry.gpx", SourceKind: "gpx", UploadedVia: "test", CreatedAt: now}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	if err := db.Create(&models.SourceReceipt{ID: receiptID, SourceInstanceID: sourceID, RawFileID: rawID, SourceKind: "gpx", IngestionMode: "incremental", ScopeJSON: json.RawMessage(`{"device":"watch-1"}`), ReceivedAt: now, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create source receipt: %v", err)
	}
	if err := db.Create(&models.SourceCoverageAssertion{ID: uuid.New(), SourceInstanceID: sourceID, SourceReceiptID: &receiptID, Category: "activity", ScopeJSON: json.RawMessage(`{"device":"watch-1"}`), IntervalStart: now.Add(-24 * time.Hour), IntervalEnd: now, Timezone: "UTC", IngestionMode: "incremental", Completeness: "partial", RecordedAt: now}).Error; err != nil {
		t.Fatalf("create coverage assertion: %v", err)
	}
	errMessage := "parse failed"
	if err := db.Create(&models.ImportJob{ID: importID, RawFileID: rawID, Status: imports.StatusFailed, ParserKind: "gpx", ErrorMessage: &errMessage, CreatedAt: now, FinishedAt: ptrTime(now.Add(time.Minute))}).Error; err != nil {
		t.Fatalf("create import job: %v", err)
	}

	server := newIntegrationServer(t, db)
	requestJSON(t, server, http.MethodGet, "/api/v1/connections", "", http.StatusOK, func(body map[string]any) {
		connections := body["connections"].([]any)
		if len(connections) != 1 {
			t.Fatalf("connections = %#v, want one", connections)
		}
		connection := connections[0].(map[string]any)
		if connection["id"] != ids.Encode(ids.SourceInstancePrefix, sourceID) || connection["provider"] != "gpx" || connection["collection"] != "partial" || connection["operation"] != "failed" {
			t.Fatalf("connection summary = %#v", connection)
		}
		if connection["last_receipt"].(map[string]any)["id"] != ids.Encode(ids.ReceiptPrefix, receiptID) {
			t.Fatalf("last receipt = %#v", connection["last_receipt"])
		}
		if connection["last_import"].(map[string]any)["status"] != imports.StatusFailed {
			t.Fatalf("last import = %#v", connection["last_import"])
		}
		actions := connection["next_actions"].([]any)
		if len(actions) != 1 || actions[0].(map[string]any)["kind"] != "retry_import" {
			t.Fatalf("next actions = %#v", actions)
		}
	})
}
