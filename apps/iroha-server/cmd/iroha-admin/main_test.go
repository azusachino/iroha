package main

import (
	"errors"
	"io"
	"testing"
)

func TestInvalidExportsFailBeforeOpeningDatabase(t *testing.T) {
	for _, args := range [][]string{
		{"export"},
		{"export", "json", "activities"},
		{"export", "ndjson", "tb_users"},
		{"export", "gpx", "invalid"},
		{"export", "ndjson", "activities", "extra"},
	} {
		if err := run(args, io.Discard); !errors.Is(err, errUsage) {
			t.Fatalf("%v: %v, want usage error", args, err)
		}
	}
}
