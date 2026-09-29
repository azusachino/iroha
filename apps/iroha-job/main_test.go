package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestRawFilePurgeHandlerDisabledWhenRetentionZero(t *testing.T) {
	// A nil service would panic if the handler tried to purge.
	handler := rawFilePurgeHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, 0)
	if err := handler(context.Background(), struct{}{}); err != nil {
		t.Fatalf("disabled purge returned %v", err)
	}
}
