package httpapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

// TestPublicActivityResponse_DropsPrivateFields is the sanitization-boundary
// guard: the public DTO must never serialize internal or source-linking
// fields, even when the source activity carries them.
func TestPublicActivityResponse_DropsPrivateFields(t *testing.T) {
	activity := models.Activity{
		ID:               uuid.New(),
		SportType:        "run",
		Title:            "Morning Run",
		StartedAt:        time.Now(),
		Timezone:         "UTC",
		SourceKind:       "apple_health",
		SourceActivityID: "workout-secret-123",
		FirstRawFileID:   uuid.New(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	raw, err := json.Marshal(toPublicActivityResponse(activity))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	forbidden := []string{
		"first_raw_file_id",
		"source_activity_id",
		"source_kind",
		"created_at",
		"updated_at",
	}
	for _, key := range forbidden {
		if _, present := decoded[key]; present {
			t.Errorf("public activity leaked private field %q: %s", key, raw)
		}
	}

	// Guard against accidental leakage by raw value too (e.g. a renamed field).
	if string(raw) == "" {
		t.Fatal("empty response")
	}
	for _, secret := range []string{"workout-secret-123", "apple_health"} {
		if containsValue(decoded, secret) {
			t.Errorf("public activity leaked secret value %q: %s", secret, raw)
		}
	}
}

func containsValue(m map[string]any, want string) bool {
	for _, v := range m {
		if s, ok := v.(string); ok && s == want {
			return true
		}
	}
	return false
}
