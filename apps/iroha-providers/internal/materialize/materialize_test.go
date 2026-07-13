package materialize

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestContextReaderStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (contextReader{ctx: ctx, reader: strings.NewReader("data")}).Read(make([]byte, 4))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Read() error = %v", err)
	}
}
