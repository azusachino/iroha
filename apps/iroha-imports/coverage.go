package imports

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
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
	if len(assertions) == 0 {
		return nil
	}

	sourceInstanceID, sourceReceiptID, err := ensureRawFileReceiptContext(tx, rawFile)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, assertion := range assertions {
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
		row := models.SourceCoverageAssertion{
			ID:               id,
			SourceInstanceID: sourceInstanceID,
			SourceReceiptID:  &sourceReceiptID,
			ImportSnapshotID: &snapshot.ID,
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
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
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
