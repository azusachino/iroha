package expenses

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Cursor is the keyset position for occurred_on DESC, id DESC ordering.
type Cursor struct {
	OccurredOn time.Time
	ID         uuid.UUID
}

// ErrInvalidCursor indicates that a cursor was not produced by EncodeCursor.
var ErrInvalidCursor = errors.New("invalid cursor")

// EncodeCursor renders a cursor as an opaque base64url token.
func EncodeCursor(cursor Cursor) string {
	raw := dateOnly(cursor.OccurredOn).Format("2006-01-02") + "|" + cursor.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a cursor produced by EncodeCursor.
func DecodeCursor(value string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	dateValue, idValue, ok := strings.Cut(string(decoded), "|")
	if !ok {
		return Cursor{}, ErrInvalidCursor
	}
	date, err := time.Parse("2006-01-02", dateValue)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	id, err := uuid.Parse(idValue)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{OccurredOn: date, ID: id}, nil
}
