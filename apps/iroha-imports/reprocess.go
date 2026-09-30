package imports

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// selectImportedObservation establishes the default source selection for a
// canonical object. A parser replay may refresh its own observation, but it
// must not silently replace a selection that was made from another source.
func selectImportedObservation(tx *gorm.DB, table string, objectID, observationID uuid.UUID, reprocess bool) (bool, error) {
	if reprocess {
		var current struct {
			SelectedObservationID *uuid.UUID `gorm:"column:selected_observation_id"`
		}
		if err := tx.Table(table).Select("selected_observation_id").Where("id = ?", objectID).Take(&current).Error; err != nil {
			return false, err
		}
		if current.SelectedObservationID != nil && *current.SelectedObservationID != observationID {
			return false, nil
		}
	}
	return true, tx.Table(table).Where("id = ?", objectID).Update("selected_observation_id", observationID).Error
}

func restoreSelectedActivity(tx *gorm.DB, activityID uuid.UUID) error {
	return tx.Exec(`update tb_activities a
set sport_type = o.sport_type, title = o.title, started_at = o.started_at,
    ended_at = o.ended_at, distance_m = o.distance_m, duration_s = o.duration_s,
    avg_hr = o.avg_hr, max_hr = o.max_hr, avg_pace_s_per_km = o.avg_pace_s_per_km,
    calories_kcal = o.calories_kcal, source_activity_id = o.source_activity_id,
    updated_at = now()
from tb_activity_observations o
where a.id = ? and o.id = a.selected_observation_id`, activityID).Error
}

func restoreSelectedSleep(tx *gorm.DB, sessionID uuid.UUID) error {
	return tx.Exec(`update tb_sleep_sessions s
set wake_date = o.wake_date, started_at = o.started_at, ended_at = o.ended_at,
    time_in_bed_s = o.time_in_bed_s, asleep_s = o.asleep_s, efficiency = o.efficiency,
    is_main_sleep = o.is_main_sleep, core_s = o.core_s, deep_s = o.deep_s,
    rem_s = o.rem_s, awake_s = o.awake_s, unspecified_s = o.unspecified_s,
    source = o.source, updated_at = now()
from tb_sleep_observations o
where s.id = ? and o.id = s.selected_observation_id`, sessionID).Error
}

// restoreSelectedDailySummary keeps the day's goals when the selected
// observation has none, as upsertDailySummary does (see hasDailySummaryGoals).
func restoreSelectedDailySummary(tx *gorm.DB, summaryID uuid.UUID) error {
	return tx.Exec(`update tb_daily_summaries d
set day = o.day, move_kcal = o.move_kcal, exercise_min = o.exercise_min,
    stand_hours = o.stand_hours, source = o.source, updated_at = now(),
    move_goal_kcal = case when g.has_goals then o.move_goal_kcal else d.move_goal_kcal end,
    exercise_goal_min = case when g.has_goals then o.exercise_goal_min else d.exercise_goal_min end,
    stand_goal_hours = case when g.has_goals then o.stand_goal_hours else d.stand_goal_hours end
from tb_daily_summary_observations o
cross join lateral (select o.move_goal_kcal > 0 or o.exercise_goal_min > 0 or o.stand_goal_hours > 0 as has_goals) g
where d.id = ? and o.id = d.selected_observation_id`, summaryID).Error
}

func restoreSelectedDailyMetric(tx *gorm.DB, metricID uuid.UUID) error {
	return tx.Exec(`update tb_daily_metrics d
set day = o.day, metric = o.metric, value = o.value, unit = o.unit,
    source = o.source, updated_at = now()
from tb_daily_metric_observations o
where d.id = ? and o.id = d.selected_observation_id`, metricID).Error
}
