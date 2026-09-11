package parsers

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseAppleHealthShortcutFixture(t *testing.T) {
	path := filepath.Join("testdata", "apple_health_shortcut.json")
	batch, err := ParseAppleHealthShortcut(path, "shortcut-hash")
	if err != nil {
		t.Fatalf("parse shortcut: %v", err)
	}
	if len(batch.Activities) != 1 || batch.Activities[0].SourceKind != KindAppleHealthShortcut {
		t.Fatalf("activities = %#v", batch.Activities)
	}
	if len(batch.Sleep) != 1 || len(batch.Sleep[0].Segments) != 1 {
		t.Fatalf("sleep = %#v", batch.Sleep)
	}
	if len(batch.Daily.Summaries) != 1 || len(batch.Daily.Metrics) != 1 {
		t.Fatalf("daily = %#v", batch.Daily)
	}
	if len(batch.Coverage) != 3 || batch.Coverage[0].IngestionMode != "bounded_replacement" {
		t.Fatalf("coverage = %#v", batch.Coverage)
	}
}

func TestValidateAppleHealthShortcutPreservesUnknownCoverage(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "apple_health_shortcut.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	body = bytes.Replace(body, []byte(`"completeness": "partial"`), []byte(`"completeness": "unknown"`), 1)
	if _, err := ValidateAppleHealthShortcut(body); err != nil {
		t.Fatalf("validate unknown coverage: %v", err)
	}
	path := filepath.Join(t.TempDir(), "unknown.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write unknown fixture: %v", err)
	}
	batch, err := ParseAppleHealthShortcut(path, "unknown-hash")
	if err != nil {
		t.Fatalf("parse unknown coverage: %v", err)
	}
	if len(batch.Coverage) != 3 || batch.Coverage[1].Completeness != "unknown" {
		t.Fatalf("coverage = %#v, want unknown coverage retained", batch.Coverage)
	}
}

func TestValidateAppleHealthShortcut(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "apple_health_shortcut.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	metadata, err := ValidateAppleHealthShortcut(body)
	if err != nil {
		t.Fatalf("validate shortcut: %v", err)
	}
	if metadata.SourceInstanceKey != "iphone-health:primary" || metadata.CapturedAt.IsZero() {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestValidateAppleHealthShortcutRejectsTrailingDataAndUnknownFields(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "apple_health_shortcut.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	for name, input := range map[string][]byte{
		"trailing": append(append([]byte{}, body...), []byte("{}")...),
		"unknown":  append([]byte(`{"unexpected":true,`), body[1:]...),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateAppleHealthShortcut(input); err == nil {
				t.Fatal("invalid shortcut was accepted")
			}
		})
	}
}
