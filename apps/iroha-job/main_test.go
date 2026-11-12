package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestRawFilePurgeHandlerDisabledWhenRetentionZero(t *testing.T) {
	// A nil service would panic if the handler tried to purge.
	handler := rawFilePurgeHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, 0)
	if err := handler(context.Background(), struct{}{}); err != nil {
		t.Fatalf("disabled purge returned %v", err)
	}
}

func TestMediaSyncEnabledWarnsOnlyForMisconfiguredConnector(t *testing.T) {
	cases := []struct {
		name       string
		interval   string
		credential bool
		wantEnable bool
		wantLevel  string
	}{
		{"interval set without credential warns", "6h", false, false, "WARN"},
		{"explicit off does not warn", "off", false, false, "INFO"},
		{"explicit off with credential does not warn", "disabled", true, false, "INFO"},
		{"nothing configured is info", "", false, false, "INFO"},
		{"credential present runs silently", "6h", true, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			got := mediaSyncEnabled(logger, "media.sync.drifter", "IROHA_DRIFTER_API_TOKEN", tc.interval, tc.credential)
			if got != tc.wantEnable {
				t.Fatalf("enabled = %v, want %v", got, tc.wantEnable)
			}
			out := buf.String()
			if tc.wantLevel == "" {
				if out != "" {
					t.Fatalf("unexpected log: %s", out)
				}
				return
			}
			if !strings.Contains(out, "level="+tc.wantLevel) {
				t.Fatalf("log %q lacks level %s", out, tc.wantLevel)
			}
			if tc.wantLevel == "WARN" && (!strings.Contains(out, "IROHA_DRIFTER_API_TOKEN") || !strings.Contains(out, "media.sync.drifter")) {
				t.Fatalf("warning must name kind and setting: %s", out)
			}
		})
	}
}
