package imports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func dailySummarySourceKey(summary observations.DailySummary) string {
	return summary.Day.Format("2006-01-02")
}

func dailySummaryContentHash(summary observations.DailySummary) string {
	content := fmt.Sprintf(
		"%s|%.17g|%.17g|%.17g|%.17g|%.17g|%.17g|%s",
		dailySummarySourceKey(summary),
		summary.MoveKcal,
		summary.MoveGoalKcal,
		summary.ExerciseMin,
		summary.ExerciseGoalMin,
		summary.StandHours,
		summary.StandGoalHours,
		summary.Source,
	)
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func dailyMetricSourceKey(metric observations.DailyMetric) string {
	return strings.Join([]string{metric.Day.Format("2006-01-02"), metric.Metric}, "|")
}

func dailyMetricContentHash(metric observations.DailyMetric) string {
	content := fmt.Sprintf("%s|%.17g|%s|%s", dailyMetricSourceKey(metric), metric.Value, metric.Unit, metric.Source)
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func (s *Service) persistDailySummary(tx *gorm.DB, rawFile models.RawFile, summary observations.DailySummary, snapshotID uuid.UUID) error {
	sourceKey := dailySummarySourceKey(summary)
	if sourceKey == "" {
		return fmt.Errorf("parsed daily summary missing source key")
	}
	contentHash := dailySummaryContentHash(summary)

	var existing models.AppleSourceItem
	res := tx.Limit(1).Find(&existing, "source_key = ?", sourceKey)
	if res.Error != nil {
		return res.Error
	}
	found := res.RowsAffected > 0
	if found && existing.ItemType != appleSourceItemTypeDailySummary {
		return fmt.Errorf("source key %q already belongs to item type %q", sourceKey, existing.ItemType)
	}
	if found && existing.ContentHash == contentHash {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            time.Now().UTC(),
		}).Error
	}

	summaryID := uuid.Nil
	if found && existing.DailySummaryID != nil {
		summaryID = *existing.DailySummaryID
	}
	if summaryID == uuid.Nil {
		var err error
		summaryID, err = ids.New()
		if err != nil {
			return err
		}
	}
	if err := upsertDailySummary(tx, rawFile, summaryID, summary); err != nil {
		return err
	}
	if err := s.persistDailySummaryObservation(tx, rawFile, summary, summaryID, snapshotID); err != nil {
		return err
	}
	now := time.Now().UTC()
	if found {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"content_hash":          contentHash,
			"daily_summary_id":      summaryID,
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error
	}
	itemID, err := ids.New()
	if err != nil {
		return err
	}
	return tx.Create(&models.AppleSourceItem{
		ID:                 itemID,
		SourceKey:          sourceKey,
		ItemType:           appleSourceItemTypeDailySummary,
		ContentHash:        contentHash,
		DailySummaryID:     &summaryID,
		LastSeenSnapshotID: &snapshotID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error
}

func upsertDailySummary(tx *gorm.DB, rawFile models.RawFile, summaryID uuid.UUID, parsed observations.DailySummary) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"day":               parsed.Day,
		"move_kcal":         parsed.MoveKcal,
		"move_goal_kcal":    parsed.MoveGoalKcal,
		"exercise_min":      parsed.ExerciseMin,
		"exercise_goal_min": parsed.ExerciseGoalMin,
		"stand_hours":       parsed.StandHours,
		"stand_goal_hours":  parsed.StandGoalHours,
		"source":            parsed.Source,
		"updated_at":        now,
	}
	var existing models.DailySummary
	if err := tx.First(&existing, "id = ?", summaryID).Error; err == nil {
		return tx.Model(&models.DailySummary{}).Where("id = ?", summaryID).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Create(&models.DailySummary{
		ID:              summaryID,
		Day:             parsed.Day,
		MoveKcal:        parsed.MoveKcal,
		MoveGoalKcal:    parsed.MoveGoalKcal,
		ExerciseMin:     parsed.ExerciseMin,
		ExerciseGoalMin: parsed.ExerciseGoalMin,
		StandHours:      parsed.StandHours,
		StandGoalHours:  parsed.StandGoalHours,
		Source:          parsed.Source,
		FirstRawFileID:  rawFile.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error
}

func (s *Service) persistDailySummaryObservation(tx *gorm.DB, rawFile models.RawFile, summary observations.DailySummary, summaryID, snapshotID uuid.UUID) error {
	observationID, err := upsertSourceObservation(tx, rawFile, "daily_summary", "apple_health", dailySummarySourceKey(summary), dailySummaryContentHash(summary), snapshotID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.DailySummaryObservation{ID: observationID, DailySummaryID: summaryID, Day: summary.Day, MoveKcal: summary.MoveKcal, MoveGoalKcal: summary.MoveGoalKcal, ExerciseMin: summary.ExerciseMin, ExerciseGoalMin: summary.ExerciseGoalMin, StandHours: summary.StandHours, StandGoalHours: summary.StandGoalHours, Source: summary.Source, MatchStatus: "canonical", CreatedAt: now, UpdatedAt: now}
	var existing models.DailySummaryObservation
	if err := tx.First(&existing, "id = ?", observationID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&models.DailySummaryObservation{}).Where("id = ?", observationID).Updates(map[string]any{
		"daily_summary_id": summaryID, "day": summary.Day, "move_kcal": summary.MoveKcal, "move_goal_kcal": summary.MoveGoalKcal,
		"exercise_min": summary.ExerciseMin, "exercise_goal_min": summary.ExerciseGoalMin, "stand_hours": summary.StandHours,
		"stand_goal_hours": summary.StandGoalHours, "source": summary.Source, "updated_at": now,
	}).Error; err != nil {
		return err
	}
	return tx.Model(&models.DailySummary{}).Where("id = ?", summaryID).Update("selected_observation_id", observationID).Error
}

func (s *Service) persistDailyMetric(tx *gorm.DB, rawFile models.RawFile, metric observations.DailyMetric, snapshotID uuid.UUID) error {
	sourceKey := dailyMetricSourceKey(metric)
	if sourceKey == "" {
		return fmt.Errorf("parsed daily metric missing source key")
	}
	contentHash := dailyMetricContentHash(metric)

	var existing models.AppleSourceItem
	res := tx.Limit(1).Find(&existing, "source_key = ?", sourceKey)
	if res.Error != nil {
		return res.Error
	}
	found := res.RowsAffected > 0
	if found && existing.ItemType != appleSourceItemTypeDailyMetric {
		return fmt.Errorf("source key %q already belongs to item type %q", sourceKey, existing.ItemType)
	}
	if found && existing.ContentHash == contentHash {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            time.Now().UTC(),
		}).Error
	}

	metricID := uuid.Nil
	if found && existing.DailyMetricID != nil {
		metricID = *existing.DailyMetricID
	}
	if metricID == uuid.Nil {
		var err error
		metricID, err = ids.New()
		if err != nil {
			return err
		}
	}
	if err := upsertDailyMetric(tx, rawFile, metricID, metric); err != nil {
		return err
	}
	if err := s.persistDailyMetricObservation(tx, rawFile, metric, metricID, snapshotID); err != nil {
		return err
	}
	now := time.Now().UTC()
	if found {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"content_hash":          contentHash,
			"daily_metric_id":       metricID,
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error
	}
	itemID, err := ids.New()
	if err != nil {
		return err
	}
	return tx.Create(&models.AppleSourceItem{
		ID:                 itemID,
		SourceKey:          sourceKey,
		ItemType:           appleSourceItemTypeDailyMetric,
		ContentHash:        contentHash,
		DailyMetricID:      &metricID,
		LastSeenSnapshotID: &snapshotID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error
}

func upsertDailyMetric(tx *gorm.DB, rawFile models.RawFile, metricID uuid.UUID, parsed observations.DailyMetric) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"day":        parsed.Day,
		"metric":     parsed.Metric,
		"value":      parsed.Value,
		"unit":       parsed.Unit,
		"source":     parsed.Source,
		"updated_at": now,
	}
	var existing models.DailyMetric
	if err := tx.First(&existing, "id = ?", metricID).Error; err == nil {
		return tx.Model(&models.DailyMetric{}).Where("id = ?", metricID).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Create(&models.DailyMetric{
		ID:             metricID,
		Day:            parsed.Day,
		Metric:         parsed.Metric,
		Value:          parsed.Value,
		Unit:           parsed.Unit,
		Source:         parsed.Source,
		FirstRawFileID: rawFile.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error
}

func (s *Service) persistDailyMetricObservation(tx *gorm.DB, rawFile models.RawFile, metric observations.DailyMetric, metricID, snapshotID uuid.UUID) error {
	observationID, err := upsertSourceObservation(tx, rawFile, "daily_metric", "apple_health", dailyMetricSourceKey(metric), dailyMetricContentHash(metric), snapshotID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.DailyMetricObservation{ID: observationID, DailyMetricID: metricID, Day: metric.Day, Metric: metric.Metric, Value: metric.Value, Unit: metric.Unit, Source: metric.Source, Reducer: "source_priority", MatchStatus: "canonical", CreatedAt: now, UpdatedAt: now}
	var existing models.DailyMetricObservation
	if err := tx.First(&existing, "id = ?", observationID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&models.DailyMetricObservation{}).Where("id = ?", observationID).Updates(map[string]any{
		"daily_metric_id": metricID, "day": metric.Day, "metric": metric.Metric, "value": metric.Value,
		"unit": metric.Unit, "source": metric.Source, "updated_at": now,
	}).Error; err != nil {
		return err
	}
	return tx.Model(&models.DailyMetric{}).Where("id = ?", metricID).Update("selected_observation_id", observationID).Error
}
