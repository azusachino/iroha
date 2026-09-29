package imports

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// coverageCompletenessUnknown is what a bounded source asserts when it
	// cannot prove it read the whole window -- a locked phone or an
	// ambiguous Health permission returns data without proving completeness.
	// The intake parser accepts it, so the pipeline must too.
	coverageCompletenessUnknown      = "unknown"
	coverageCompletenessPartial      = "partial"
	coverageCompletenessCovered      = "covered"
	coverageCompletenessCoveredEmpty = "covered_empty"
)

func (s *Service) persistCoverageTx(tx *gorm.DB, rawFile models.RawFile, assertions []provider.CoverageAssertion, snapshot models.ImportSnapshot) error {
	for _, assertion := range assertions {
		if err := insertCoverageAssertion(tx, rawFile, assertion, &snapshot.ID); err != nil {
			return err
		}
	}
	return nil
}

// insertCoverageAssertion stores one assertion, evidenced by the raw file's
// receipt and, when the caller has one, its import snapshot.
func insertCoverageAssertion(tx *gorm.DB, rawFile models.RawFile, assertion provider.CoverageAssertion, snapshotID *uuid.UUID) error {
	sourceInstanceID, sourceReceiptID, err := ensureRawFileReceiptContext(tx, rawFile)
	if err != nil {
		return err
	}
	if len(assertion.ScopeJSON) == 0 {
		assertion.ScopeJSON = json.RawMessage(`{}`)
	}
	if err := validateCoverageAssertion(assertion); err != nil {
		return err
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.SourceCoverageAssertion{
		ID:               id,
		SourceInstanceID: sourceInstanceID,
		SourceReceiptID:  &sourceReceiptID,
		ImportSnapshotID: snapshotID,
		Category:         assertion.Category,
		ScopeJSON:        assertion.ScopeJSON,
		IntervalStart:    assertion.From,
		IntervalEnd:      assertion.To,
		Timezone:         assertion.Timezone,
		IngestionMode:    assertion.IngestionMode,
		Completeness:     assertion.Completeness,
		RecordedAt:       now,
		CreatedAt:        now,
	}
	return tx.Create(&row).Error
}

func validateCoverageAssertion(assertion provider.CoverageAssertion) error {
	if strings.TrimSpace(assertion.Category) == "" || assertion.From.IsZero() || !assertion.From.Before(assertion.To) || strings.TrimSpace(assertion.Timezone) == "" {
		return errors.New("invalid coverage assertion")
	}
	if _, err := time.LoadLocation(assertion.Timezone); err != nil {
		return fmt.Errorf("invalid coverage timezone: %w", err)
	}
	if assertion.IngestionMode != "bounded_replacement" && assertion.IngestionMode != "full_snapshot" && assertion.IngestionMode != "incremental" {
		return fmt.Errorf("invalid coverage ingestion mode %q", assertion.IngestionMode)
	}
	switch assertion.Completeness {
	case coverageCompletenessUnknown, coverageCompletenessPartial, coverageCompletenessCovered, coverageCompletenessCoveredEmpty:
	default:
		return fmt.Errorf("invalid coverage completeness %q", assertion.Completeness)
	}
	scope := assertion.ScopeJSON
	if len(scope) == 0 {
		scope = json.RawMessage(`{}`)
	}
	var object map[string]any
	if err := json.Unmarshal(scope, &object); err != nil || object == nil {
		return errors.New("coverage scope must be an object")
	}
	return nil
}

// recordSyncCoverage records what one connector run proved it read. A run is
// one assertion, not one per page: a single page proves nothing about
// completeness. Current-state lists are full snapshots over the run; the
// activity feed adds to what is stored, so it is incremental over its window.
func (s *SyncRunner) recordSyncCoverage(rawFile models.RawFile, sourceKind string, from, to time.Time, completeness string) error {
	category, mode := "media_list", "full_snapshot"
	if sourceKind == coreimports.KindAniListActivity {
		category, mode = "media_activity", "incremental"
	}
	if !to.After(from) {
		to = from.Add(time.Second)
	}
	timezone := s.imports.timezone
	if timezone == "" {
		timezone = "UTC"
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return insertCoverageAssertion(tx, rawFile, provider.CoverageAssertion{
			Category: category, From: from, To: to, Timezone: timezone,
			IngestionMode: mode, Completeness: completeness,
		}, nil)
	})
}
