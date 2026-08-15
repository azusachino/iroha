package media

import (
	"errors"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateEvent(input CreateEventInput) (Event, error) {
	if input.MediaItemID == uuid.Nil {
		return Event{}, ErrMediaItemNotFound
	}
	if input.EventAt.IsZero() {
		return Event{}, ErrEventAtRequired
	}
	if !validEventType(input.EventType) {
		return Event{}, ErrInvalidEventType
	}
	if input.SourceKind == "" {
		input.SourceKind = "manual"
	}
	if input.SourceEventID == "" {
		return Event{}, ErrSourceEventIDRequired
	}
	input.EventAt = input.EventAt.UTC()
	var created Event
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var item struct{ ID uuid.UUID }
		if err := tx.Table("tb_media_items").Select("id").Where("id = ?", input.MediaItemID).First(&item).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMediaItemNotFound
		} else if err != nil {
			return err
		}
		var existing models.MediaConsumptionEvent
		result := tx.Where("source_kind = ? and source_event_id = ? and event_type = ?", input.SourceKind, input.SourceEventID, input.EventType).First(&existing)
		if result.Error == nil {
			if existing.MediaItemID != input.MediaItemID || !existing.EventAt.Equal(input.EventAt) || existing.Unit != input.Unit ||
				!floatPtrEqual(existing.Position, input.Position) || !floatPtrEqual(existing.Total, input.Total) ||
				!floatPtrEqual(existing.ProgressPercent, input.ProgressPercent) || !floatPtrEqual(existing.Rating, input.Rating) ||
				!floatPtrEqual(existing.RatingScale, input.RatingScale) || existing.Note != input.Note {
				return ErrEventConflict
			}
			created = Event{ID: existing.ID, MediaItemID: existing.MediaItemID, EventType: existing.EventType, OccurredAt: existing.EventAt, Unit: existing.Unit, Position: existing.Position, Total: existing.Total, ProgressPercent: existing.ProgressPercent, Rating: existing.Rating, RatingScale: existing.RatingScale}
			return nil
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		id, err := ids.New()
		if err != nil {
			return err
		}
		row := models.MediaConsumptionEvent{ID: id, MediaItemID: input.MediaItemID, EventType: input.EventType, EventAt: input.EventAt, SourceKind: input.SourceKind, SourceEventID: input.SourceEventID, Unit: input.Unit, Position: input.Position, Total: input.Total, ProgressPercent: input.ProgressPercent, Rating: input.Rating, RatingScale: input.RatingScale, Note: input.Note, CreatedAt: time.Now().UTC()}
		result = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err := tx.Where("source_kind = ? and source_event_id = ? and event_type = ?", input.SourceKind, input.SourceEventID, input.EventType).First(&existing).Error; err != nil {
				return err
			}
			if existing.MediaItemID != input.MediaItemID || !existing.EventAt.Equal(input.EventAt) || existing.Unit != input.Unit ||
				!floatPtrEqual(existing.Position, input.Position) || !floatPtrEqual(existing.Total, input.Total) ||
				!floatPtrEqual(existing.ProgressPercent, input.ProgressPercent) || !floatPtrEqual(existing.Rating, input.Rating) ||
				!floatPtrEqual(existing.RatingScale, input.RatingScale) || existing.Note != input.Note {
				return ErrEventConflict
			}
			created = Event{ID: existing.ID, MediaItemID: existing.MediaItemID, EventType: existing.EventType, OccurredAt: existing.EventAt, Unit: existing.Unit, Position: existing.Position, Total: existing.Total, ProgressPercent: existing.ProgressPercent, Rating: existing.Rating, RatingScale: existing.RatingScale}
			return nil
		}
		created = Event{ID: id, MediaItemID: row.MediaItemID, EventType: row.EventType, OccurredAt: row.EventAt, Unit: row.Unit, Position: row.Position, Total: row.Total, ProgressPercent: row.ProgressPercent, Rating: row.Rating, RatingScale: row.RatingScale}
		return nil
	})
	return created, err
}

func validEventType(value string) bool {
	return observations.ValidMediaEventType(value)
}
