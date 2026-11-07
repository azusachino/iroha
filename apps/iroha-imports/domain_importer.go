package imports

import (
	"context"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ObservationPersister defines the contract for persisting domain observations
// and selecting canonical winners according to the source priority ladder.
type ObservationPersister interface {
	Persist(ctx context.Context, tx *gorm.DB, rawFile models.RawFile, snapshotID uuid.UUID, reprocess bool) error
}

// BaseDomainImporter provides common observation persistence and priority-based
// selection logic for domain entities (activities, sleep sessions, daily metrics).
type BaseDomainImporter struct {
	tableName string
}

func NewBaseDomainImporter(tableName string) BaseDomainImporter {
	return BaseDomainImporter{tableName: tableName}
}

// ShouldSelectIncoming evaluates the source priority ladder between the incoming raw file
// and the currently selected observation for an entity.
func (b *BaseDomainImporter) ShouldSelectIncoming(tx *gorm.DB, entityID uuid.UUID, incomingKind SourceKind) (bool, error) {
	var current struct {
		SelectedObservationID *uuid.UUID `gorm:"column:selected_observation_id"`
	}
	if err := tx.Table(b.tableName).Select("selected_observation_id").Where("id = ?", entityID).Take(&current).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, err
	}
	if current.SelectedObservationID == nil {
		return true, nil
	}

	// Lookup existing source observation's source kind
	var existingRaw struct {
		SourceKind string `gorm:"column:source_kind"`
	}
	err := tx.Table("tb_source_observations as so").
		Select("rf.source_kind").
		Joins("join tb_raw_files as rf on rf.id = so.raw_file_id").
		Where("so.id = ?", *current.SelectedObservationID).
		Take(&existingRaw).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, err
	}

	existingKind := ParseSourceKind(existingRaw.SourceKind)
	return SourceWins(incomingKind, existingKind), nil
}

// SelectObservation updates the selected_observation_id pointer if allowed.
func (b *BaseDomainImporter) SelectObservation(tx *gorm.DB, entityID, observationID uuid.UUID, reprocess bool) (bool, error) {
	return selectImportedObservation(tx, b.tableName, entityID, observationID, reprocess)
}
