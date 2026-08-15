package imports

import (
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// purgeDerivedForRawFile deletes everything derived from a raw file so it
// can be re-persisted fresh (dispositionReprocess), instead of appending
// alongside stale rows. The delete order matters:
//
//  1. tb_apple_source_items first. These carry the content_hash used by
//     persistAppleWorkout's change-detection. If they survived a purge, the
//     freshly re-parsed workouts would look "unchanged" against the
//     just-deleted activities and persistAppleWorkout would only bump
//     last_seen_snapshot_id instead of re-creating the activity -
//     resurrecting nothing and silently losing data. They're matched two
//     ways: by activity_id (workouts produced from this raw file) and by
//     last_seen_snapshot_id (source items last touched by a snapshot of
//     this raw file, covering unchanged workouts that were never
//     re-persisted but still got their last_seen bumped).
//  2. tb_import_snapshots for this raw file, now safe to remove since no
//     source item or source observation still references them.
//  3. tb_activities with first_raw_file_id = this raw file. ON DELETE
//     CASCADE removes tb_external_refs, tb_activity_route_points,
//     tb_activity_samplings, and tb_activity_laps for those activities.
//     This step must come after (1) since tb_apple_source_items.activity_id
//     is ON DELETE SET NULL, not CASCADE - deleting activities first would
//     just null out the source items rather than removing them, defeating
//     step (1)'s purpose.
func purgeDerivedForRawFile(tx *gorm.DB, rawFileID uuid.UUID) error {
	if err := tx.Where("raw_file_id = ?", rawFileID).Delete(&models.MediaConsumptionEvent{}).Error; err != nil {
		return err
	}
	if err := tx.Where("raw_file_id = ?", rawFileID).Delete(&models.MediaStateHistory{}).Error; err != nil {
		return err
	}

	if err := tx.Exec(`
		delete from tb_apple_source_items
		where activity_id in (select id from tb_activities where first_raw_file_id = ?)
		   or sleep_session_id in (select id from tb_sleep_sessions where first_raw_file_id = ?)
		   or daily_summary_id in (select id from tb_daily_summaries where first_raw_file_id = ?)
		   or daily_metric_id in (select id from tb_daily_metrics where first_raw_file_id = ?)
		   or last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?)
	`, rawFileID, rawFileID, rawFileID, rawFileID, rawFileID).Error; err != nil {
		return err
	}
	for _, statement := range []string{
		`update tb_activities set selected_observation_id = null where selected_observation_id in (select ao.id from tb_activity_observations ao join tb_source_observations so on so.id = ao.id where so.raw_file_id = ? or so.first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?) or so.last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?))`,
		`update tb_sleep_sessions set selected_observation_id = null where selected_observation_id in (select so.id from tb_sleep_observations so join tb_source_observations src on src.id = so.id where src.raw_file_id = ? or src.first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?) or src.last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?))`,
		`update tb_daily_summaries set selected_observation_id = null where selected_observation_id in (select so.id from tb_daily_summary_observations so join tb_source_observations src on src.id = so.id where src.raw_file_id = ? or src.first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?) or src.last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?))`,
		`update tb_daily_metrics set selected_observation_id = null where selected_observation_id in (select so.id from tb_daily_metric_observations so join tb_source_observations src on src.id = so.id where src.raw_file_id = ? or src.first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?) or src.last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?))`,
	} {
		if err := tx.Exec(statement, rawFileID, rawFileID, rawFileID).Error; err != nil {
			return err
		}
	}
	if err := tx.Exec(`
		delete from tb_source_observations
		where raw_file_id = ?
		   or first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?)
		   or last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?)
	`, rawFileID, rawFileID, rawFileID).Error; err != nil {
		return err
	}

	if err := tx.Where("raw_file_id = ?", rawFileID).Delete(&models.ImportSnapshot{}).Error; err != nil {
		return err
	}

	if err := tx.Where("first_raw_file_id = ?", rawFileID).Delete(&models.Activity{}).Error; err != nil {
		return err
	}

	if err := tx.Where("first_raw_file_id = ?", rawFileID).Delete(&models.SleepSession{}).Error; err != nil {
		return err
	}

	if err := tx.Where("first_raw_file_id = ?", rawFileID).Delete(&models.DailyMetric{}).Error; err != nil {
		return err
	}

	if err := tx.Where("first_raw_file_id = ?", rawFileID).Delete(&models.DailySummary{}).Error; err != nil {
		return err
	}

	return nil
}
