package httpapi

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/google/uuid"
)

func TestParseDailyFiltersRejectsInvalidQuery(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/daily?from=bad", nil)
	recorder := httptest.NewRecorder()

	if _, ok := parseDailyFilters(recorder, request); ok {
		t.Fatal("parseDailyFilters accepted an invalid date")
	}
	if recorder.Code != 400 {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestToDailyResponseEncodesIDsAndMetrics(t *testing.T) {
	rawFileID := uuid.New()
	summaryID := uuid.New()
	steps := 1234.0
	row := daily.Row{
		DailySummary: models.DailySummary{
			ID:             summaryID,
			Day:            time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			MoveKcal:       600,
			FirstRawFileID: rawFileID,
		},
		Steps: &steps,
	}

	got := toDailyResponse(row)
	if got.ID != ids.Encode(ids.DailySummaryPrefix, summaryID) {
		t.Errorf("id = %q, want encoded summary id", got.ID)
	}
	if got.FirstRawFileID != ids.Encode(ids.RawFilePrefix, rawFileID) {
		t.Errorf("first_raw_file_id = %q, want encoded raw id", got.FirstRawFileID)
	}
	if got.Steps == nil || *got.Steps != steps {
		t.Errorf("steps = %v, want %v", got.Steps, steps)
	}
}
