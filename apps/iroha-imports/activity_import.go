package imports

import (
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

func (s *Service) persistActivitiesTx(tx *gorm.DB, rawFile models.RawFile, parsed []observations.Activity, parsedSleep []observations.Sleep, parsedDailySummaries []observations.DailySummary, parsedDailyMetrics []observations.DailyMetric, snapshot models.ImportSnapshot, reprocess bool) error {
	if reprocess {
		if err := purgeDerivedForRawFile(tx, rawFile.ID); err != nil {
			return err
		}
	}

	if err := tx.Create(&snapshot).Error; err != nil {
		return err
	}

	for _, activity := range parsed {
		if activity.ExternalID == "" {
			return fmt.Errorf("parsed activity missing external id")
		}

		activityID, err := s.upsertActivity(tx, rawFile, activity)
		if err != nil {
			return err
		}
		if err := replaceRoutePoints(tx, activityID, activity.RoutePoints); err != nil {
			return err
		}
		if err := replaceLaps(tx, activityID, activity.Laps); err != nil {
			return err
		}
		if err := replaceSamplings(tx, activityID, activity.Samplings); err != nil {
			return err
		}
		if err := s.persistActivityObservation(tx, rawFile, activity, activityID, snapshot.ID); err != nil {
			return err
		}
	}
	for _, session := range parsedSleep {
		if err := s.persistSleepSession(tx, rawFile, session, snapshot.ID); err != nil {
			return err
		}
	}
	for _, summary := range parsedDailySummaries {
		if err := s.persistDailySummary(tx, rawFile, summary, snapshot.ID); err != nil {
			return err
		}
	}
	for _, metric := range parsedDailyMetrics {
		if err := s.persistDailyMetric(tx, rawFile, metric, snapshot.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) persistActivityObservation(tx *gorm.DB, rawFile models.RawFile, activity observations.Activity, activityID, snapshotID uuid.UUID) error {
	observationID, err := upsertSourceObservation(tx, rawFile, "activity", activity.Provider, activity.ExternalID, activity.ContentHash, snapshotID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.ActivityObservation{
		ID:               observationID,
		ActivityID:       activityID,
		SourceActivityID: activity.SourceActivityID,
		SportType:        activity.SportType,
		Title:            activity.Title,
		StartedAt:        activity.StartedAt,
		EndedAt:          activity.EndedAt,
		DistanceM:        activity.DistanceM,
		DurationS:        activity.DurationS,
		AvgHR:            activity.AvgHR,
		MaxHR:            activity.MaxHR,
		AvgPaceSPerKM:    activity.AvgPaceSPerKM,
		CaloriesKcal:     activity.CaloriesKcal,
		MatchStatus:      "canonical",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	var existing models.ActivityObservation
	if err := tx.First(&existing, "id = ?", observationID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&models.ActivityObservation{}).Where("id = ?", observationID).Updates(map[string]any{
		"activity_id": activityID, "source_activity_id": activity.SourceActivityID, "sport_type": activity.SportType,
		"title": activity.Title, "started_at": activity.StartedAt, "ended_at": activity.EndedAt,
		"distance_m": activity.DistanceM, "duration_s": activity.DurationS, "avg_hr": activity.AvgHR,
		"max_hr": activity.MaxHR, "avg_pace_s_per_km": activity.AvgPaceSPerKM, "calories_kcal": activity.CaloriesKcal,
		"updated_at": now,
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Activity{}).Where("id = ?", activityID).Update("selected_observation_id", observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`delete from tb_activity_observation_route_points where activity_observation_id = ?`, observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`insert into tb_activity_observation_route_points (activity_observation_id, seq, ts, lat, lon, elevation_m, distance_m, speed_mps, heart_rate, geom)
select ?, seq, ts, lat, lon, elevation_m, distance_m, speed_mps, heart_rate, geom from tb_activity_route_points where activity_id = ?`, observationID, activityID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`delete from tb_activity_observation_samplings where activity_observation_id = ?`, observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`insert into tb_activity_observation_samplings (id, activity_observation_id, sampling_type, ts, value, unit)
select id, ?, sampling_type, ts, value, unit from tb_activity_samplings where activity_id = ?`, observationID, activityID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`delete from tb_activity_observation_laps where activity_observation_id = ?`, observationID).Error; err != nil {
		return err
	}
	return tx.Exec(`insert into tb_activity_observation_laps (id, activity_observation_id, lap_no, start_ts, end_ts, distance_m, duration_s, avg_hr, avg_pace_s_per_km, calories_kcal)
select id, ?, lap_no, start_ts, end_ts, distance_m, duration_s, avg_hr, avg_pace_s_per_km, calories_kcal from tb_activity_laps where activity_id = ?`, observationID, activityID).Error
}

func upsertSourceObservation(tx *gorm.DB, rawFile models.RawFile, sourceKind, provider, sourceKey, contentHash string, snapshotID uuid.UUID) (uuid.UUID, error) {
	sourceInstanceID, sourceReceiptID, err := ensureRawFileReceiptContext(tx, rawFile)
	if err != nil {
		return uuid.Nil, err
	}
	var existing models.SourceObservation
	err = tx.Where("source_instance_id = ? and source_kind = ? and source_key = ?", sourceInstanceID, sourceKind, sourceKey).First(&existing).Error
	now := time.Now().UTC()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, idErr := ids.New()
		if idErr != nil {
			return uuid.Nil, idErr
		}
		first := snapshotID
		row := models.SourceObservation{ID: id, SourceInstanceID: sourceInstanceID, Provider: provider, SourceKind: sourceKind, SourceKey: sourceKey, ContentHash: contentHash, RawFileID: rawFile.ID, FirstSeenSnapshotID: &first, LastSeenSnapshotID: &snapshotID, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&row).Error; err != nil {
			return uuid.Nil, err
		}
		return id, recordObservationReceipt(tx, sourceInstanceID, id, sourceReceiptID, snapshotID, contentHash, now)
	}
	if err != nil {
		return uuid.Nil, err
	}
	if err := tx.Model(&models.SourceObservation{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"content_hash": contentHash, "raw_file_id": rawFile.ID, "last_seen_snapshot_id": snapshotID, "updated_at": now,
	}).Error; err != nil {
		return uuid.Nil, err
	}
	return existing.ID, recordObservationReceipt(tx, sourceInstanceID, existing.ID, sourceReceiptID, snapshotID, contentHash, now)
}

func ensureRawFileReceiptContext(tx *gorm.DB, rawFile models.RawFile) (uuid.UUID, uuid.UUID, error) {
	var receipt models.SourceReceipt
	result := tx.Where("raw_file_id = ?", rawFile.ID).Order("received_at desc, id desc").First(&receipt)
	if result.Error == nil {
		return receipt.SourceInstanceID, receipt.ID, nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return uuid.Nil, uuid.Nil, result.Error
	}
	return uuid.Nil, uuid.Nil, fmt.Errorf("raw file %s has no source receipt", rawFile.ID)
}

func recordObservationReceipt(tx *gorm.DB, sourceInstanceID, observationID, receiptID, snapshotID uuid.UUID, contentHash string, now time.Time) error {
	return tx.Exec(`insert into tb_source_observation_receipts (source_instance_id, source_observation_id, source_receipt_id, import_snapshot_id, content_hash, created_at)
values (?, ?, ?, ?, ?, ?)
on conflict (source_receipt_id, source_observation_id, import_snapshot_id)
do update set content_hash = excluded.content_hash`, sourceInstanceID, observationID, receiptID, snapshotID, contentHash, now).Error
}

func (s *Service) upsertActivity(tx *gorm.DB, rawFile models.RawFile, parsed observations.Activity) (uuid.UUID, error) {
	var externalRef models.ExternalRef
	res := tx.Limit(1).Find(&externalRef, "provider = ? and external_id = ?", parsed.Provider, parsed.ExternalID)
	if res.Error != nil {
		return uuid.Nil, res.Error
	}
	found := res.RowsAffected > 0

	now := time.Now().UTC()
	if found {
		updates := map[string]any{
			"sport_type":         parsed.SportType,
			"title":              parsed.Title,
			"started_at":         parsed.StartedAt,
			"ended_at":           parsed.EndedAt,
			"distance_m":         parsed.DistanceM,
			"duration_s":         parsed.DurationS,
			"avg_hr":             parsed.AvgHR,
			"max_hr":             parsed.MaxHR,
			"avg_pace_s_per_km":  parsed.AvgPaceSPerKM,
			"source_kind":        parsed.SourceKind,
			"source_activity_id": parsed.SourceActivityID,
			"updated_at":         now,
		}
		return externalRef.ActivityID, tx.Model(&models.Activity{}).Where("id = ?", externalRef.ActivityID).Updates(updates).Error
	}

	activityID, err := ids.New()
	if err != nil {
		return uuid.Nil, err
	}
	refID, err := ids.New()
	if err != nil {
		return uuid.Nil, err
	}

	activity := models.Activity{
		ID:               activityID,
		SportType:        parsed.SportType,
		Title:            parsed.Title,
		StartedAt:        parsed.StartedAt,
		EndedAt:          parsed.EndedAt,
		DistanceM:        parsed.DistanceM,
		DurationS:        parsed.DurationS,
		AvgHR:            parsed.AvgHR,
		MaxHR:            parsed.MaxHR,
		AvgPaceSPerKM:    parsed.AvgPaceSPerKM,
		CaloriesKcal:     parsed.CaloriesKcal,
		SourceKind:       parsed.SourceKind,
		SourceActivityID: parsed.SourceActivityID,
		FirstRawFileID:   rawFile.ID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := tx.Create(&activity).Error; err != nil {
		return uuid.Nil, err
	}

	externalRef = models.ExternalRef{
		ID:         refID,
		ActivityID: activityID,
		Provider:   parsed.Provider,
		ExternalID: parsed.ExternalID,
		RawFileID:  rawFile.ID,
		CreatedAt:  now,
	}
	if err := tx.Create(&externalRef).Error; err != nil {
		return uuid.Nil, err
	}

	return activityID, nil
}

// replaceLaps deletes any existing laps for the activity and inserts the
// newly parsed ones. Mirrors replaceRoutePoints, but laps are few per
// workout so a simple create-slice loop (rather than a hand-built batched
// multi-row INSERT) is fine here.
func replaceLaps(tx *gorm.DB, activityID uuid.UUID, laps []observations.Lap) error {
	if err := tx.Delete(&models.ActivityLap{}, "activity_id = ?", activityID).Error; err != nil {
		return err
	}
	if len(laps) == 0 {
		return nil
	}

	rows := make([]models.ActivityLap, 0, len(laps))
	for _, lap := range laps {
		id, err := ids.New()
		if err != nil {
			return err
		}
		rows = append(rows, models.ActivityLap{
			ID:            id,
			ActivityID:    activityID,
			LapNo:         lap.LapNo,
			StartTs:       lap.StartTs,
			EndTs:         lap.EndTs,
			DistanceM:     nil,
			DurationS:     lap.DurationS,
			AvgHR:         nil,
			AvgPaceSPerKM: nil,
			CaloriesKcal:  lap.CaloriesKcal,
		})
	}
	return tx.Create(&rows).Error
}

// samplingInsertBatchSize caps how many samplings are inserted per batch via
// GORM's CreateInBatches. Unlike route points, samplings have no PostGIS
// geom column to hand-build SQL for, so CreateInBatches is clean here.
const samplingInsertBatchSize = 1000

// replaceSamplings deletes any existing samplings for the activity and
// bulk-inserts the newly parsed ones.
func replaceSamplings(tx *gorm.DB, activityID uuid.UUID, samplings []observations.Sampling) error {
	if err := tx.Delete(&models.ActivitySampling{}, "activity_id = ?", activityID).Error; err != nil {
		return err
	}
	if len(samplings) == 0 {
		return nil
	}

	rows := make([]models.ActivitySampling, 0, len(samplings))
	for _, sampling := range samplings {
		id, err := ids.New()
		if err != nil {
			return err
		}
		rows = append(rows, models.ActivitySampling{
			ID:           id,
			ActivityID:   activityID,
			SamplingType: sampling.SamplingType,
			Ts:           sampling.Ts,
			Value:        sampling.Value,
			Unit:         sampling.Unit,
		})
	}
	return tx.CreateInBatches(rows, samplingInsertBatchSize).Error
}

// routePointInsertBatchSize caps how many route points are inserted per
// multi-row INSERT statement. Each row binds 8 params, so 1000 rows/batch
// binds 8000 params - well under PostgreSQL's 65535 bind-parameter limit.
const routePointInsertBatchSize = 1000

func replaceRoutePoints(tx *gorm.DB, activityID uuid.UUID, points []observations.RoutePoint) error {
	if err := tx.Delete(&models.ActivityRoutePoint{}, "activity_id = ?", activityID).Error; err != nil {
		return err
	}
	for start := 0; start < len(points); start += routePointInsertBatchSize {
		end := start + routePointInsertBatchSize
		if end > len(points) {
			end = len(points)
		}
		sql, args := buildRoutePointsInsertSQL(activityID, points[start:end], start)
		if err := tx.Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

// buildRoutePointsInsertSQL builds a single multi-row INSERT statement (and
// its flat arg list) for one batch of route points. startSeq is the
// absolute index of points[0] within the full slice passed to
// replaceRoutePoints, so seq numbering stays correct across chunks.
func buildRoutePointsInsertSQL(activityID uuid.UUID, points []observations.RoutePoint, startSeq int) (string, []any) {
	var sb strings.Builder
	sb.WriteString(`insert into tb_activity_route_points
		  (activity_id, seq, ts, lat, lon, elevation_m, geom)
		  values `)

	args := make([]any, 0, len(points)*8)
	for i, point := range points {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?, ?, ?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)")
		args = append(
			args,
			activityID,
			startSeq+i,
			point.Ts,
			point.Lat,
			point.Lon,
			point.ElevationM,
			point.Lon,
			point.Lat,
		)
	}
	return sb.String(), args
}
