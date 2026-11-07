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

func (s *Service) persistSleepSession(tx *gorm.DB, rawFile models.RawFile, session observations.Sleep, snapshotID uuid.UUID, reprocess bool) error {
	sourceKey := sleepSessionSourceKey(session)
	if sourceKey == "" {
		return fmt.Errorf("parsed sleep session missing source key")
	}
	sourceInstanceID, _, err := ensureRawFileReceiptContext(tx, rawFile)
	if err != nil {
		return err
	}
	sessionID := uuid.Nil
	var sourceObservation models.SourceObservation
	result := tx.Where("source_instance_id = ? and source_kind = ? and source_key = ?", sourceInstanceID, "sleep", sourceKey).First(&sourceObservation)
	if result.Error == nil {
		var existing models.SleepObservation
		if err := tx.First(&existing, "id = ?", sourceObservation.ID).Error; err != nil {
			return err
		}
		sessionID = existing.SleepSessionID
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	if sessionID == uuid.Nil {
		// Look for an existing sleep session with matching started_at and ended_at
		// (e.g. across Health Auto Export and native Apple Health export).
		var matchingSession models.SleepSession
		matchRes := tx.Where("started_at = ? and ended_at = ?", session.StartedAt, session.EndedAt).First(&matchingSession)
		if matchRes.Error == nil {
			sessionID = matchingSession.ID
		} else if !errors.Is(matchRes.Error, gorm.ErrRecordNotFound) {
			return matchRes.Error
		}
	}
	if sessionID == uuid.Nil {
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
	if err := s.persistSleepObservation(tx, rawFile, session, sessionID, snapshotID, reprocess); err != nil {
		return err
	}
	return nil
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

// Keep each source's timeline independently so selecting a different source
// cannot destroy the evidence needed to restore or compare its segments.
func replaceSleepObservationSegments(tx *gorm.DB, observationID uuid.UUID, segments []observations.SleepSegment) error {
	if err := tx.Exec("delete from tb_sleep_observation_segments where sleep_observation_id = ?", observationID).Error; err != nil {
		return err
	}
	if len(segments) == 0 {
		return nil
	}
	type sourceSegment struct {
		ID                 uuid.UUID
		SleepObservationID uuid.UUID
		Stage              string
		StartedAt          time.Time
		EndedAt            time.Time
		Seq                int
	}
	rows := make([]sourceSegment, 0, len(segments))
	for seq, segment := range segments {
		id, err := ids.New()
		if err != nil {
			return err
		}
		rows = append(rows, sourceSegment{ID: id, SleepObservationID: observationID, Stage: segment.Stage, StartedAt: segment.StartedAt, EndedAt: segment.EndedAt, Seq: seq})
	}
	return tx.Table("tb_sleep_observation_segments").CreateInBatches(rows, sleepSegmentInsertBatchSize).Error
}

func (s *Service) persistSleepObservation(tx *gorm.DB, rawFile models.RawFile, session observations.Sleep, sessionID, snapshotID uuid.UUID, reprocess bool) error {
	contentHash := sleepSessionContentHash(session)
	observationID, err := upsertSourceObservation(tx, rawFile, "sleep", "apple_health", sleepSessionSourceKey(session), contentHash, snapshotID)
	if err != nil {
		return err
	}
	// Multi-source sleep observations attach to the canonical session.
	// Prefer native Apple Health export over Health Auto Export when both are present.
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
	if err := replaceSleepObservationSegments(tx, observationID, session.Segments); err != nil {
		return err
	}

	baseImporter := NewBaseDomainImporter("tb_sleep_sessions")
	incomingKind := ParseSourceKind(rawFile.SourceKind)
	shouldSelect, err := baseImporter.ShouldSelectIncoming(tx, sessionID, incomingKind)
	if err != nil {
		return err
	}

	if err := tx.Exec(`insert into tb_sleep_session_observations (sleep_session_id, sleep_observation_id, is_preferred)
values (?, ?, ?) on conflict (sleep_session_id, sleep_observation_id) do update set is_preferred = excluded.is_preferred`, sessionID, observationID, shouldSelect).Error; err != nil {
		return err
	}
	if shouldSelect {
		selected, err := baseImporter.SelectObservation(tx, sessionID, observationID, reprocess)
		if err != nil {
			return err
		}
		if !selected {
			return restoreSelectedSleep(tx, sessionID)
		}
	} else {
		return restoreSelectedSleep(tx, sessionID)
	}
	return nil
}
