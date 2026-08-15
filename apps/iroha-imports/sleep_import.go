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

func sleepSessionSourceKey(session observations.Sleep) string {
	return strings.Join([]string{
		session.Source,
		session.WakeDate.Format("2006-01-02"),
		session.StartedAt.Format(time.RFC3339Nano),
		session.EndedAt.Format(time.RFC3339Nano),
	}, "|")
}

func sleepSessionContentHash(session observations.Sleep) string {
	var content strings.Builder
	for _, segment := range session.Segments {
		fmt.Fprintf(
			&content, "%s|%s|%s|%s\n",
			segment.Stage,
			segment.StartedAt.Format(time.RFC3339Nano),
			segment.EndedAt.Format(time.RFC3339Nano),
			segment.Source,
		)
	}
	sum := sha256.Sum256([]byte(content.String()))
	return hex.EncodeToString(sum[:])
}

func (s *Service) persistSleepSession(tx *gorm.DB, rawFile models.RawFile, session observations.Sleep, snapshotID uuid.UUID) error {
	sourceKey := sleepSessionSourceKey(session)
	if sourceKey == "" {
		return fmt.Errorf("parsed sleep session missing source key")
	}
	contentHash := sleepSessionContentHash(session)

	var existing models.AppleSourceItem
	res := tx.Limit(1).Find(&existing, "source_key = ?", sourceKey)
	if res.Error != nil {
		return res.Error
	}
	found := res.RowsAffected > 0
	if found && existing.ItemType != appleSourceItemTypeSleepSession {
		return fmt.Errorf("source key %q already belongs to item type %q", sourceKey, existing.ItemType)
	}

	var existingHash *string
	if found {
		existingHash = &existing.ContentHash
	}
	now := time.Now().UTC()
	switch decideSourceItem(existingHash, contentHash) {
	case sourceItemUnchanged:
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error
	}

	sessionID := uuid.Nil
	if found && existing.SleepSessionID != nil {
		sessionID = *existing.SleepSessionID
	}
	if sessionID == uuid.Nil {
		var err error
		sessionID, err = ids.New()
		if err != nil {
			return err
		}
	}
	if err := upsertSleepSession(tx, rawFile, sessionID, session); err != nil {
		return err
	}
	if err := replaceSleepSegments(tx, sessionID, session.Segments); err != nil {
		return err
	}
	if err := s.persistSleepObservation(tx, rawFile, session, sessionID, snapshotID); err != nil {
		return err
	}

	if found {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"content_hash":          contentHash,
			"sleep_session_id":      sessionID,
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error
	}
	itemID, err := ids.New()
	if err != nil {
		return err
	}
	item := models.AppleSourceItem{
		ID:                 itemID,
		SourceKey:          sourceKey,
		ItemType:           appleSourceItemTypeSleepSession,
		ContentHash:        contentHash,
		SleepSessionID:     &sessionID,
		LastSeenSnapshotID: &snapshotID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return tx.Create(&item).Error
}

func upsertSleepSession(tx *gorm.DB, rawFile models.RawFile, sessionID uuid.UUID, parsed observations.Sleep) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"wake_date":     parsed.WakeDate,
		"started_at":    parsed.StartedAt,
		"ended_at":      parsed.EndedAt,
		"time_in_bed_s": parsed.TimeInBedS,
		"asleep_s":      parsed.AsleepS,
		"efficiency":    parsed.Efficiency,
		"is_main_sleep": parsed.IsMainSleep,
		"core_s":        parsed.CoreS,
		"deep_s":        parsed.DeepS,
		"rem_s":         parsed.RemS,
		"awake_s":       parsed.AwakeS,
		"unspecified_s": parsed.UnspecifiedS,
		"source":        parsed.Source,
		"updated_at":    now,
	}
	var existing models.SleepSession
	if err := tx.First(&existing, "id = ?", sessionID).Error; err == nil {
		return tx.Model(&models.SleepSession{}).Where("id = ?", sessionID).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return tx.Create(&models.SleepSession{
		ID:             sessionID,
		WakeDate:       parsed.WakeDate,
		StartedAt:      parsed.StartedAt,
		EndedAt:        parsed.EndedAt,
		TimeInBedS:     parsed.TimeInBedS,
		AsleepS:        parsed.AsleepS,
		Efficiency:     parsed.Efficiency,
		IsMainSleep:    parsed.IsMainSleep,
		CoreS:          parsed.CoreS,
		DeepS:          parsed.DeepS,
		RemS:           parsed.RemS,
		AwakeS:         parsed.AwakeS,
		UnspecifiedS:   parsed.UnspecifiedS,
		Source:         parsed.Source,
		FirstRawFileID: rawFile.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error
}

const sleepSegmentInsertBatchSize = 1000

func replaceSleepSegments(tx *gorm.DB, sessionID uuid.UUID, segments []observations.SleepSegment) error {
	if err := tx.Delete(&models.SleepSegment{}, "session_id = ?", sessionID).Error; err != nil {
		return err
	}
	if len(segments) == 0 {
		return nil
	}
	rows := make([]models.SleepSegment, 0, len(segments))
	for seq, segment := range segments {
		id, err := ids.New()
		if err != nil {
			return err
		}
		rows = append(rows, models.SleepSegment{
			ID:        id,
			SessionID: sessionID,
			Stage:     segment.Stage,
			StartedAt: segment.StartedAt,
			EndedAt:   segment.EndedAt,
			Seq:       seq,
		})
	}
	return tx.CreateInBatches(rows, sleepSegmentInsertBatchSize).Error
}

func (s *Service) persistSleepObservation(tx *gorm.DB, rawFile models.RawFile, session observations.Sleep, sessionID, snapshotID uuid.UUID) error {
	contentHash := sleepSessionContentHash(session)
	observationID, err := upsertSourceObservation(tx, rawFile, "sleep", "apple_health", sleepSessionSourceKey(session), contentHash, snapshotID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.SleepObservation{ID: observationID, SleepSessionID: sessionID, WakeDate: session.WakeDate, StartedAt: session.StartedAt, EndedAt: session.EndedAt, TimeInBedS: session.TimeInBedS, AsleepS: session.AsleepS, Efficiency: session.Efficiency, IsMainSleep: session.IsMainSleep, CoreS: session.CoreS, DeepS: session.DeepS, RemS: session.RemS, AwakeS: session.AwakeS, UnspecifiedS: session.UnspecifiedS, Source: session.Source, MatchStatus: "canonical", CreatedAt: now, UpdatedAt: now}
	var existing models.SleepObservation
	if err := tx.First(&existing, "id = ?", observationID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&models.SleepObservation{}).Where("id = ?", observationID).Updates(map[string]any{
		"sleep_session_id": sessionID, "wake_date": session.WakeDate, "started_at": session.StartedAt, "ended_at": session.EndedAt,
		"time_in_bed_s": session.TimeInBedS, "asleep_s": session.AsleepS, "efficiency": session.Efficiency, "is_main_sleep": session.IsMainSleep,
		"core_s": session.CoreS, "deep_s": session.DeepS, "rem_s": session.RemS, "awake_s": session.AwakeS, "unspecified_s": session.UnspecifiedS,
		"source": session.Source, "updated_at": now,
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.SleepSession{}).Where("id = ?", sessionID).Update("selected_observation_id", observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`insert into tb_sleep_session_observations (sleep_session_id, sleep_observation_id, is_preferred)
values (?, ?, true) on conflict (sleep_session_id, sleep_observation_id) do update set is_preferred = excluded.is_preferred`, sessionID, observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`delete from tb_sleep_observation_segments where sleep_observation_id = ?`, observationID).Error; err != nil {
		return err
	}
	return tx.Exec(`insert into tb_sleep_observation_segments (id, sleep_observation_id, stage, started_at, ended_at, seq)
select id, ?, stage, started_at, ended_at, seq from tb_sleep_segments where session_id = ?`, observationID, sessionID).Error
}
