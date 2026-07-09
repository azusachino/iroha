package jobs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/internal/models"
)

func TestMarshalPayloadDefaultsNilToObject(t *testing.T) {
	payload, err := marshalPayload(nil)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if string(payload) != "{}" {
		t.Fatalf("payload = %s, want {}", payload)
	}
}

func TestMarshalPayloadRejectsInvalidRawJSON(t *testing.T) {
	_, err := marshalPayload(json.RawMessage(`{broken`))
	if err == nil {
		t.Fatalf("expected invalid raw JSON error")
	}
}

func TestNextScheduleRunInterval(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	next, enabled, err := nextScheduleRun(models.JobSchedule{
		Enabled:      true,
		ScheduleKind: ScheduleKindInterval,
		ScheduleExpr: "6h",
	}, now)
	if err != nil {
		t.Fatalf("next schedule run: %v", err)
	}
	if !enabled {
		t.Fatalf("enabled = false, want true")
	}
	if next == nil || !next.Equal(now.Add(6*time.Hour)) {
		t.Fatalf("next = %v, want %v", next, now.Add(6*time.Hour))
	}
}

func TestNextScheduleRunManualDisablesAfterRun(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	next, enabled, err := nextScheduleRun(models.JobSchedule{
		Enabled:      true,
		ScheduleKind: ScheduleKindManual,
		ScheduleExpr: "once",
	}, now)
	if err != nil {
		t.Fatalf("next schedule run: %v", err)
	}
	if enabled {
		t.Fatalf("enabled = true, want false")
	}
	if next != nil {
		t.Fatalf("next = %v, want nil", next)
	}
}
