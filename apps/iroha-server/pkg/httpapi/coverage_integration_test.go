//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/coverage"
	"github.com/google/uuid"
)

func TestIntegrationSourceCoveragePreservesGapsAndStates(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })

	now := time.Date(2099, time.January, 5, 12, 0, 0, 0, time.UTC)
	sourceID := uuid.New()
	rawID := uuid.New()
	receiptID := uuid.New()
	if err := db.Create(&models.SourceInstance{ID: sourceID, Provider: "coverage-test", InstanceKey: sourceID.String(), CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create source instance: %v", err)
	}
	if err := db.Create(&models.RawFile{ID: rawID, SHA256: "coverage-" + rawID.String(), OriginalFilename: "coverage.json", StoragePath: "/tmp/coverage.json", SourceKind: "coverage-test", UploadedVia: "test", CreatedAt: now}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	if err := db.Create(&models.SourceReceipt{ID: receiptID, SourceInstanceID: sourceID, RawFileID: rawID, SourceKind: "coverage-test", IngestionMode: "bounded_replacement", ScopeJSON: json.RawMessage(`{"account":"main"}`), ReceivedAt: now, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create source receipt: %v", err)
	}

	service := coverage.NewService(db)
	start := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	assertCoverage := func(from, to time.Time, completeness string) {
		t.Helper()
		if _, err := service.Assert(nil, coverage.AssertInput{
			SourceInstanceID: sourceID, SourceReceiptID: &receiptID, Category: "health.daily",
			From: from, To: to, Timezone: "UTC", IngestionMode: "bounded_replacement",
			Completeness: completeness, RecordedAt: from.Add(time.Hour),
		}); err != nil {
			t.Fatalf("assert coverage %s-%s: %v", from, to, err)
		}
	}
	assertCoverage(start, start.Add(2*day), coverage.CompletenessCovered)
	assertCoverage(start.Add(2*day), start.Add(3*day), coverage.CompletenessCoveredEmpty)

	result, err := service.Query(t.Context(), coverage.QueryFilters{SourceInstanceID: sourceID, Category: "health.daily", From: start, To: start.Add(4 * day)})
	if err != nil {
		t.Fatalf("query coverage: %v", err)
	}
	if result.State != coverage.CompletenessPartial || len(result.Covered) != 1 || len(result.Gaps) != 1 {
		t.Fatalf("coverage result = %#v, want covered interval plus one gap", result)
	}
	if !result.Covered[0].From.Equal(start) || !result.Covered[0].To.Equal(start.Add(3*day)) || !result.Gaps[0].From.Equal(start.Add(3*day)) {
		t.Fatalf("coverage intervals = %#v/%#v", result.Covered, result.Gaps)
	}

	server := newIntegrationServer(t, db)
	requestJSON(t, server, http.MethodGet, "/api/v1/coverage?source_instance_id="+sourceID.String()+"&category=health.daily&from=2099-01-01&to=2099-01-04&timezone=UTC", "", http.StatusOK, func(body map[string]any) {
		if body["state"] != coverage.CompletenessCovered {
			t.Fatalf("API coverage state = %#v, want covered", body["state"])
		}
	})
}
