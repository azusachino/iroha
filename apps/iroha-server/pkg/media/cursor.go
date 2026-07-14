package media

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("invalid cursor")

type Cursor struct {
	LastUpdateAt time.Time
	ID           uuid.UUID
}

func EncodeCursor(cursor Cursor) string {
	raw := cursor.LastUpdateAt.UTC().Format(time.RFC3339Nano) + "|" + cursor.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(value string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	dateValue, idValue, ok := strings.Cut(string(decoded), "|")
	if !ok {
		return Cursor{}, ErrInvalidCursor
	}
	lastUpdateAt, err := time.Parse(time.RFC3339Nano, dateValue)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	id, err := uuid.Parse(idValue)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{LastUpdateAt: lastUpdateAt, ID: id}, nil
}
