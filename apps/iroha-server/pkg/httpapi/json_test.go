package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONBodyRejectsUnknownAndTrailingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"raw_file_id":"raw_1","extra":true}`},
		{name: "trailing value", body: `{"raw_file_id":"raw_1"}{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/imports", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			var destination createImportJobRequest
			if err := decodeJSONBody(rec, req, &destination); err == nil {
				t.Fatal("decodeJSONBody succeeded, want error")
			}
		})
	}
}

func TestDecodeJSONBodyRejectsOversizedBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/imports", strings.NewReader(`{"raw_file_id":"`+strings.Repeat("x", maxJSONBodyBytes)+`"}`))
	rec := httptest.NewRecorder()
	var destination createImportJobRequest
	if err := decodeJSONBody(rec, req, &destination); err == nil {
		t.Fatal("decodeJSONBody succeeded, want oversized body error")
	}
}
