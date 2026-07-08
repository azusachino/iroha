package activities

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidCursor is returned when an opaque cursor cannot be decoded.
var ErrInvalidCursor = errors.New("invalid cursor")

// EncodeCursor renders a Cursor as an opaque base64url token. The payload is
// "<started_at RFC3339Nano>|<uuid>"; clients must treat it as opaque.
func EncodeCursor(c Cursor) string {
	raw := c.StartedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a token produced by EncodeCursor.
func DecodeCursor(value string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	ts, idStr, ok := strings.Cut(string(decoded), "|")
	if !ok {
		return Cursor{}, ErrInvalidCursor
	}
	startedAt, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{StartedAt: startedAt, ID: id}, nil
}
