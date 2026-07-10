package ids

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	RawFilePrefix  = "raw"
	ImportPrefix   = "imp"
	ActivityPrefix = "act"
)

func New() (uuid.UUID, error) {
	return uuid.NewV7()
}

func Encode(prefix string, id uuid.UUID) string {
	return prefix + "_" + id.String()
}

func Decode(prefix string, value string) (uuid.UUID, error) {
	expected := prefix + "_"
	if !strings.HasPrefix(value, expected) {
		return uuid.Nil, fmt.Errorf("expected %s-prefixed id", prefix)
	}

	id, err := uuid.Parse(strings.TrimPrefix(value, expected))
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}
