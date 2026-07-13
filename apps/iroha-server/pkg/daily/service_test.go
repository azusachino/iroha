package daily

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	want := Cursor{
		Day: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		ID:  uuid.MustParse("018cc251-7b2e-7d52-9b0d-6bd6f2c9c9e4"),
	}

	got, err := DecodeCursor(EncodeCursor(want))
	if err != nil {
		t.Fatalf("DecodeCursor returned error: %v", err)
	}
	if !got.Day.Equal(want.Day) || got.ID != want.ID {
		t.Fatalf("cursor = %+v, want %+v", got, want)
	}
}

func TestDecodeCursorRejectsMalformedValue(t *testing.T) {
	for _, value := range []string{"", "not-base64", "MjAyNC0wMS0wMg==", "MjAyNC0wMS0wMnxibnVsbA"} {
		if _, err := DecodeCursor(value); err == nil {
			t.Errorf("DecodeCursor(%q) returned nil error", value)
		}
	}
}
