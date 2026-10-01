package dbconnect

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestSlogGormLogger_ParamsFilter_SuppressesBoundParams(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)

	gormLog := &slogGormLogger{logger: logger}

	// Verify that slogGormLogger satisfies gorm.ParamsFilter
	var _ gorm.ParamsFilter = gormLog

	rawSQL := "SELECT * FROM tb_users WHERE token = $1 AND email = $2"
	filteredSQL, filteredParams := gormLog.ParamsFilter(context.Background(), rawSQL, "super-secret-token", "user@example.com")

	if filteredSQL != rawSQL {
		t.Fatalf("expected sql %q, got %q", rawSQL, filteredSQL)
	}
	if len(filteredParams) != 0 {
		t.Fatalf("expected filteredParams to be empty/nil, got %+v", filteredParams)
	}

	// Verify Trace logging
	gormLog.Trace(context.Background(), time.Now().Add(-300*time.Millisecond), func() (string, int64) {
		return filteredSQL, 1
	}, nil)

	logOutput := buf.String()
	if bytes.Contains(buf.Bytes(), []byte("super-secret-token")) {
		t.Fatalf("sensitive token leaked in log: %s", logOutput)
	}
}
