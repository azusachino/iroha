package activities

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
)

func encode(payload string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func TestCursorRoundTrip(t *testing.T) {
	want := Cursor{
		StartedAt: time.Date(2024, 6, 1, 12, 30, 45, 123456789, time.UTC),
		ID:        uuid.MustParse("018f7c2a-1b2c-7d3e-8f4a-5b6c7d8e9f00"),
	}

	got, err := DecodeCursor(EncodeCursor(want))
	if err != nil {
		t.Fatalf("DecodeCursor: %v", err)
	}
	if !got.StartedAt.Equal(want.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", got.StartedAt, want.StartedAt)
	}
	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}
}

func TestDecodeCursorInvalid(t *testing.T) {
	cases := map[string]string{
		"not base64":    "!!!not-base64!!!",
		"missing sep":   encode("2024-06-01T00:00:00Z"),
		"bad timestamp": encode("nope|018f7c2a-1b2c-7d3e-8f4a-5b6c7d8e9f00"),
		"bad uuid":      encode("2024-06-01T00:00:00Z|not-a-uuid"),
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCursor(token); err == nil {
				t.Fatalf("expected error for %q", token)
			}
		})
	}
}
