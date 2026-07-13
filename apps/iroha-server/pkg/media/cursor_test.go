package media

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	want := Cursor{
		LastUpdateAt: time.Date(2026, 7, 13, 12, 34, 56, 123456789, time.FixedZone("JST", 9*60*60)),
		ID:           uuid.MustParse("019810d6-7f8f-7b2f-9a4f-8b9c7e1d2f30"),
	}

	got, err := DecodeCursor(EncodeCursor(want))
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !got.LastUpdateAt.Equal(want.LastUpdateAt) || got.ID != want.ID {
		t.Fatalf("cursor = %+v, want %+v", got, want)
	}
}

func TestDecodeCursorRejectsMalformedValue(t *testing.T) {
	if _, err := DecodeCursor("not-a-cursor"); err != ErrInvalidCursor {
		t.Fatalf("error = %v, want %v", err, ErrInvalidCursor)
	}
}
