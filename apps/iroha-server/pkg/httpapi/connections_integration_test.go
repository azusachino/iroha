//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

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
