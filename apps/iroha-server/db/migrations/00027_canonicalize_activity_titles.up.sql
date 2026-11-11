-- Canonicalize historical activity titles and record where each title came from.
--
-- ADR-0012 makes the activity taxonomy (apps/iroha-core/i18n) the owner of the
-- canonical title and sport slug for system labels, and keeps user-authored
-- titles verbatim. Historical rows predate that taxonomy: a workout recorded
-- while iOS was Japanese kept its localized label ("ラン", "屋外ラン"), an old
-- native export record kept the stripped enum body ("Running",
-- "TraditionalStrengthTraining"), and Chinese labels left sport_type empty.
-- This migration heals those rows in place.
--
-- Safety rules for every rewrite:
--   * exact equality on the full title - never LIKE/substring matching - so a
--     user title such as '朝のウォーキング' or 'Sub-40 10k Tempo' cannot match;
--   * scoped to source_kind in ('health_auto_export', 'apple_health_export'),
--     so another provider's titles are never touched;
--   * the before/after values are recorded in tb_activity_title_remediation in
--     the same statement that performs the update;
--   * rows whose title and sport already agree with the taxonomy are skipped
--     entirely (no audit row, no updated_at bump).
--
-- Raw evidence is not rewritten: the observation row keeps its canonical
-- projection, and replaying a retained raw file through the current parser
-- writes the same title, sport and origin.

alter table tb_activity_observations
  add column if not exists title_origin text not null default 'unknown'
  constraint tb_activity_observations_title_origin_check check (title_origin in ('system', 'custom', 'unknown'));

alter table tb_activities
  add column if not exists title_origin text not null default 'unknown'
  constraint tb_activities_title_origin_check check (title_origin in ('system', 'custom', 'unknown'));

create table if not exists tb_activity_title_remediation (
  id bigserial primary key,
  target_table text not null,
  row_id uuid not null,
  old_title text not null,
  new_title text not null,
  old_sport_type text not null,
  new_sport_type text not null,
  migrated_at timestamptz not null default now()
);

-- Snapshot of the taxonomy synonyms as they exist at this migration. The rows
-- are the exact strings providers emit: Health Auto Export's localized names and
-- the native export's enum bodies (Apple strips the HKWorkoutActivityType
-- prefix before storing a title). A later taxonomy addition belongs in the
-- importer, not in this applied migration.
create temporary table activity_title_map (
  old_title text primary key,
  new_title text not null,
  new_sport_type text not null
);

insert into activity_title_map (old_title, new_title, new_sport_type) values
    -- Running
    ('Outdoor Run', 'Outdoor Run', 'run'),
    ('Outdoor Running', 'Outdoor Run', 'run'),
    ('户外跑步', 'Outdoor Run', 'run'),
    ('户外跑', 'Outdoor Run', 'run'),
    ('路跑', 'Outdoor Run', 'run'),
    ('屋外ラン', 'Outdoor Run', 'run'),
    ('屋外 ラン', 'Outdoor Run', 'run'),
    ('屋外ランニング', 'Outdoor Run', 'run'),
    ('Indoor Run', 'Indoor Run', 'run'),
    ('Treadmill', 'Indoor Run', 'run'),
    ('Treadmill Run', 'Indoor Run', 'run'),
    ('室内跑步', 'Indoor Run', 'run'),
    ('跑步机', 'Indoor Run', 'run'),
    ('室内跑', 'Indoor Run', 'run'),
    ('屋内ラン', 'Indoor Run', 'run'),
    ('屋内 ラン', 'Indoor Run', 'run'),
    ('トレッドミル', 'Indoor Run', 'run'),
    ('室内ラン', 'Indoor Run', 'run'),
    ('Run', 'Run', 'run'),
    ('Running', 'Run', 'run'),
    ('HKWorkoutActivityTypeRunning', 'Run', 'run'),
    ('跑步', 'Run', 'run'),
    ('ラン', 'Run', 'run'),
    -- Walking
    ('Outdoor Walk', 'Outdoor Walk', 'walk'),
    ('Outdoor Walking', 'Outdoor Walk', 'walk'),
    ('户外步行', 'Outdoor Walk', 'walk'),
    ('户外散步', 'Outdoor Walk', 'walk'),
    ('健走', 'Outdoor Walk', 'walk'),
    ('屋外ウォーキング', 'Outdoor Walk', 'walk'),
    ('屋外 ウォーキング', 'Outdoor Walk', 'walk'),
    ('Indoor Walk', 'Indoor Walk', 'walk'),
    ('室内步行', 'Indoor Walk', 'walk'),
    ('室内散步', 'Indoor Walk', 'walk'),
    ('屋内ウォーキング', 'Indoor Walk', 'walk'),
    ('屋内 ウォーキング', 'Indoor Walk', 'walk'),
    ('Walk', 'Walk', 'walk'),
    ('Walking', 'Walk', 'walk'),
    ('HKWorkoutActivityTypeWalking', 'Walk', 'walk'),
    ('步行', 'Walk', 'walk'),
    ('散步', 'Walk', 'walk'),
    ('ウォーキング', 'Walk', 'walk'),
    ('徒歩', 'Walk', 'walk'),
    -- Cycling
    ('Outdoor Cycling', 'Outdoor Cycling', 'ride'),
    ('Cycling', 'Outdoor Cycling', 'ride'),
    ('Biking', 'Outdoor Cycling', 'ride'),
    ('HKWorkoutActivityTypeCycling', 'Outdoor Cycling', 'ride'),
    ('户外骑行', 'Outdoor Cycling', 'ride'),
    ('骑行', 'Outdoor Cycling', 'ride'),
    ('自行车', 'Outdoor Cycling', 'ride'),
    ('屋外サイクリング', 'Outdoor Cycling', 'ride'),
    ('屋外 サイクリング', 'Outdoor Cycling', 'ride'),
    ('サイクリング', 'Outdoor Cycling', 'ride'),
    ('自転車', 'Outdoor Cycling', 'ride'),
    ('Indoor Cycling', 'Indoor Cycling', 'ride'),
    ('Spinning', 'Indoor Cycling', 'ride'),
    ('室内骑行', 'Indoor Cycling', 'ride'),
    ('室内单车', 'Indoor Cycling', 'ride'),
    ('动感单车', 'Indoor Cycling', 'ride'),
    ('屋内サイクリング', 'Indoor Cycling', 'ride'),
    ('屋内 サイクリング', 'Indoor Cycling', 'ride'),
    ('スピンバイク', 'Indoor Cycling', 'ride'),
    ('Ride', 'Ride', 'ride'),
    -- Swimming
    ('Pool Swim', 'Pool Swim', 'swim'),
    ('Swimming', 'Pool Swim', 'swim'),
    ('Pool Swimming', 'Pool Swim', 'swim'),
    ('HKWorkoutActivityTypeSwimming', 'Pool Swim', 'swim'),
    ('泳池游泳', 'Pool Swim', 'swim'),
    ('游泳', 'Pool Swim', 'swim'),
    ('プールスイミング', 'Pool Swim', 'swim'),
    ('スイム', 'Pool Swim', 'swim'),
    ('水泳', 'Pool Swim', 'swim'),
    ('Open Water Swim', 'Open Water Swim', 'swim'),
    ('公开水域游泳', 'Open Water Swim', 'swim'),
    ('オープンウォータースイミング', 'Open Water Swim', 'swim'),
    -- Hiking
    ('Hiking', 'Hiking', 'hike'),
    ('Hike', 'Hiking', 'hike'),
    ('HKWorkoutActivityTypeHiking', 'Hiking', 'hike'),
    ('徒步', 'Hiking', 'hike'),
    ('远足', 'Hiking', 'hike'),
    ('爬山', 'Hiking', 'hike'),
    ('ハイキング', 'Hiking', 'hike'),
    ('登山', 'Hiking', 'hike'),
    -- Strength training
    ('Functional Strength Training', 'Functional Strength Training', 'traditional_strength_training'),
    ('Traditional Strength Training', 'Functional Strength Training', 'traditional_strength_training'),
    ('Strength Training', 'Functional Strength Training', 'traditional_strength_training'),
    ('HKWorkoutActivityTypeTraditionalStrengthTraining', 'Functional Strength Training', 'traditional_strength_training'),
    ('HKWorkoutActivityTypeFunctionalStrengthTraining', 'Functional Strength Training', 'traditional_strength_training'),
    ('TraditionalStrengthTraining', 'Functional Strength Training', 'traditional_strength_training'),
    ('FunctionalStrengthTraining', 'Functional Strength Training', 'traditional_strength_training'),
    ('功能性力量训练', 'Functional Strength Training', 'traditional_strength_training'),
    ('力量训练', 'Functional Strength Training', 'traditional_strength_training'),
    ('健身', 'Functional Strength Training', 'traditional_strength_training'),
    ('機能的筋力トレーニング', 'Functional Strength Training', 'traditional_strength_training'),
    ('筋トレ', 'Functional Strength Training', 'traditional_strength_training'),
    ('筋力トレーニング', 'Functional Strength Training', 'traditional_strength_training'),
    -- HIIT
    ('High Intensity Interval Training', 'High Intensity Interval Training', 'hiit'),
    ('HIIT', 'High Intensity Interval Training', 'hiit'),
    ('HKWorkoutActivityTypeHighIntensityIntervalTraining', 'High Intensity Interval Training', 'hiit'),
    ('HighIntensityIntervalTraining', 'High Intensity Interval Training', 'hiit'),
    ('高强度间歇训练', 'High Intensity Interval Training', 'hiit'),
    ('高強度インターバルトレーニング', 'High Intensity Interval Training', 'hiit'),
    -- Yoga
    ('Yoga', 'Yoga', 'yoga'),
    ('HKWorkoutActivityTypeYoga', 'Yoga', 'yoga'),
    ('瑜伽', 'Yoga', 'yoga'),
    ('ヨガ', 'Yoga', 'yoga'),
    -- Fitness gaming
    ('Fitness Gaming', 'Fitness Gaming', 'fitness_gaming'),
    ('FitnessGaming', 'Fitness Gaming', 'fitness_gaming'),
    ('HKWorkoutActivityTypeFitnessGaming', 'Fitness Gaming', 'fitness_gaming'),
    ('健身游戏', 'Fitness Gaming', 'fitness_gaming'),
    ('フィットネスゲーム', 'Fitness Gaming', 'fitness_gaming'),
    -- Generic workout
    ('Workout', 'Workout', 'other'),
    ('Other', 'Workout', 'other'),
    ('HKWorkoutActivityTypeOther', 'Workout', 'other');

-- tb_activity_observations: the evidence row for one source's reading of the
-- workout. Activity observations are not keyed by the raw format they came from
-- (tb_source_observations.source_kind is 'activity'), so the source guard is the
-- raw file's source_kind. title_origin becomes 'system' because the row now
-- carries a taxonomy label rather than a provider-localized or user-authored one.
with targets as (
    select
        o.id,
        o.title as old_title,
        m.new_title,
        o.sport_type as old_sport_type,
        m.new_sport_type
    from tb_activity_observations o
    join tb_source_observations so on so.id = o.id
    join tb_raw_files r on r.id = so.raw_file_id
    join activity_title_map m on m.old_title = o.title
    where r.source_kind in ('health_auto_export', 'apple_health_export')
      and (o.title is distinct from m.new_title or o.sport_type is distinct from m.new_sport_type)
),
audited as (
    insert into tb_activity_title_remediation
        (target_table, row_id, old_title, new_title, old_sport_type, new_sport_type)
    select 'tb_activity_observations', id, old_title, new_title, old_sport_type, new_sport_type
    from targets
    returning row_id, new_title, new_sport_type
)
update tb_activity_observations o
set title = a.new_title,
    sport_type = a.new_sport_type,
    title_origin = 'system',
    updated_at = now()
from audited a
where o.id = a.row_id;

-- tb_activities: the canonical projection the API and UI read. title_origin is
-- copied alongside the title so a localized presentation can tell a taxonomy
-- label from a user-authored one without joining the observation.
with targets as (
    select
        c.id,
        c.title as old_title,
        m.new_title,
        c.sport_type as old_sport_type,
        m.new_sport_type
    from tb_activities c
    join activity_title_map m on m.old_title = c.title
    where c.source_kind in ('health_auto_export', 'apple_health_export')
      and (c.title is distinct from m.new_title or c.sport_type is distinct from m.new_sport_type)
),
audited as (
    insert into tb_activity_title_remediation
        (target_table, row_id, old_title, new_title, old_sport_type, new_sport_type)
    select 'tb_activities', id, old_title, new_title, old_sport_type, new_sport_type
    from targets
    returning row_id, new_title, new_sport_type
)
update tb_activities c
set title = a.new_title,
    sport_type = a.new_sport_type,
    title_origin = 'system',
    updated_at = now()
from audited a
where c.id = a.row_id;

drop table activity_title_map;
