package imports

import (
	"testing"
	"time"

	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

func TestValidateCoverageAssertionCompleteness(t *testing.T) {
	from := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	assertion := func(completeness string) provider.CoverageAssertion {
		return provider.CoverageAssertion{
			Category:      "activities",
			From:          from,
			To:            from.AddDate(0, 0, 7),
			Timezone:      "Asia/Tokyo",
			IngestionMode: "bounded_replacement",
			Completeness:  completeness,
		}
	}

	cases := []struct {
		name         string
		completeness string
		wantErr      bool
	}{
		// Every value parsers.ParseAppleHealthShortcut accepts must survive
		// the pipeline: the intake endpoint returns 202 before the import job
		// runs, so a narrower set here means accepting evidence that can only
		// fail later. "unknown" is the locked-phone/ambiguous-permission case.
		{"unknown round-trips from a bounded source", coverageCompletenessUnknown, false},
		{"partial round-trips", coverageCompletenessPartial, false},
		{"covered round-trips", coverageCompletenessCovered, false},
		{"covered_empty round-trips", coverageCompletenessCoveredEmpty, false},
		{"an unsupported value is still rejected", "mostly", true},
		{"an empty value is still rejected", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCoverageAssertion(assertion(tc.completeness))
			if (err != nil) != tc.wantErr {
				t.Errorf("validateCoverageAssertion(%q) error = %v, wantErr %v", tc.completeness, err, tc.wantErr)
			}
		})
	}
}
