package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RawFile struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	SHA256           string    `gorm:"column:sha256"`
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
	StoragePath      string
	SourceKind       string
	UploadedVia      string
	ObservedAt       *time.Time
	CreatedAt        time.Time
}

func (RawFile) TableName() string {
	return "tb_raw_files"
}

type ImportJob struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	RawFileID     uuid.UUID `gorm:"type:uuid"`
	Status        string
	ParserKind    string
	ParserVersion string
	ErrorMessage  *string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
}

func (ImportJob) TableName() string {
	return "tb_import_jobs"
}

type Activity struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	SportType             string
	Title                 string
	StartedAt             time.Time
	EndedAt               *time.Time
	Timezone              string
	DistanceM             *float64
	DurationS             *int
	MovingTimeS           *int
	ElevationGainM        *float64
	AvgHR                 *int
	MaxHR                 *int
	AvgPaceSPerKM         *float64
	CaloriesKcal          *float64
	SourceKind            string
	SourceActivityID      string
	FirstRawFileID        uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SelectedObservationID *uuid.UUID `gorm:"type:uuid"`
}

func (Activity) TableName() string {
	return "tb_activities"
}

type SourceObservation struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey"`
	Provider            string
	SourceKind          string
	SourceKey           string
	ContentHash         string
	RawFileID           uuid.UUID  `gorm:"type:uuid"`
	FirstSeenSnapshotID *uuid.UUID `gorm:"type:uuid"`
	LastSeenSnapshotID  *uuid.UUID `gorm:"type:uuid"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (SourceObservation) TableName() string { return "tb_source_observations" }

type ActivityObservation struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActivityID       uuid.UUID `gorm:"type:uuid"`
	SourceActivityID string
	SportType        string
	Title            string
	StartedAt        time.Time
	EndedAt          *time.Time
	DistanceM        *float64
	DurationS        *int
	MovingTimeS      *int
	ElevationGainM   *float64
	AvgHR            *int
	MaxHR            *int
	AvgPaceSPerKM    *float64
	CaloriesKcal     *float64
	MatchStatus      string
	MatchConfidence  *float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (ActivityObservation) TableName() string { return "tb_activity_observations" }

type ExternalRef struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActivityID uuid.UUID `gorm:"type:uuid"`
	Provider   string
	ExternalID string
	RawFileID  uuid.UUID `gorm:"type:uuid"`
	CreatedAt  time.Time
}

func (ExternalRef) TableName() string {
	return "tb_external_refs"
}

type ActivityRoutePoint struct {
	ActivityID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Seq        int       `gorm:"primaryKey"`
	Ts         *time.Time
	Lat        float64
	Lon        float64
	ElevationM *float64
	DistanceM  *float64
	SpeedMPS   *float64
	HeartRate  *int
}

func (ActivityRoutePoint) TableName() string {
	return "tb_activity_route_points"
}

type ActivitySampling struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActivityID   uuid.UUID `gorm:"type:uuid"`
	SamplingType string
	Ts           time.Time
	Value        float64
	Unit         string
}

func (ActivitySampling) TableName() string {
	return "tb_activity_samplings"
}

type ActivityLap struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActivityID    uuid.UUID `gorm:"type:uuid"`
	LapNo         int
	StartTs       *time.Time
	EndTs         *time.Time
	DistanceM     *float64
	DurationS     *int
	AvgHR         *int
	AvgPaceSPerKM *float64
	CaloriesKcal  *float64
}

func (ActivityLap) TableName() string {
	return "tb_activity_laps"
}

type SleepSession struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	WakeDate              time.Time `gorm:"type:date"`
	StartedAt             time.Time
	EndedAt               time.Time
	TimeInBedS            int
	AsleepS               int
	Efficiency            float64
	IsMainSleep           bool
	CoreS                 int
	DeepS                 int
	RemS                  int
	AwakeS                int
	UnspecifiedS          int
	Source                string
	FirstRawFileID        uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SelectedObservationID *uuid.UUID `gorm:"type:uuid"`
}

func (SleepSession) TableName() string {
	return "tb_sleep_sessions"
}

type SleepObservation struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	SleepSessionID  uuid.UUID `gorm:"type:uuid"`
	WakeDate        time.Time `gorm:"type:date"`
	StartedAt       time.Time
	EndedAt         time.Time
	TimeInBedS      int
	AsleepS         int
	Efficiency      float64
	IsMainSleep     bool
	CoreS           int
	DeepS           int
	RemS            int
	AwakeS          int
	UnspecifiedS    int
	Source          string
	MatchStatus     string
	MatchConfidence *float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (SleepObservation) TableName() string { return "tb_sleep_observations" }

type SleepSegment struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	SessionID uuid.UUID `gorm:"type:uuid"`
	Stage     string
	StartedAt time.Time
	EndedAt   time.Time
	Seq       int
}

func (SleepSegment) TableName() string {
	return "tb_sleep_segments"
}

type DailySummary struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Day                   time.Time `gorm:"type:date"`
	MoveKcal              float64
	MoveGoalKcal          float64
	ExerciseMin           float64
	ExerciseGoalMin       float64
	StandHours            float64
	StandGoalHours        float64
	Source                string
	FirstRawFileID        uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SelectedObservationID *uuid.UUID `gorm:"type:uuid"`
}

func (DailySummary) TableName() string {
	return "tb_daily_summaries"
}

type DailySummaryObservation struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	DailySummaryID  uuid.UUID `gorm:"type:uuid"`
	Day             time.Time `gorm:"type:date"`
	MoveKcal        float64
	MoveGoalKcal    float64
	ExerciseMin     float64
	ExerciseGoalMin float64
	StandHours      float64
	StandGoalHours  float64
	Source          string
	MatchStatus     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (DailySummaryObservation) TableName() string { return "tb_daily_summary_observations" }

type DailyMetric struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Day                   time.Time `gorm:"type:date"`
	Metric                string
	Value                 float64
	Unit                  string
	Source                string
	FirstRawFileID        uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SelectedObservationID *uuid.UUID `gorm:"type:uuid"`
}

func (DailyMetric) TableName() string {
	return "tb_daily_metrics"
}

type DailyMetricObservation struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	DailyMetricID uuid.UUID `gorm:"type:uuid"`
	Day           time.Time `gorm:"type:date"`
	Metric        string
	Value         float64
	Unit          string
	Source        string
	Reducer       string
	MatchStatus   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (DailyMetricObservation) TableName() string { return "tb_daily_metric_observations" }

type ImportSnapshot struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ImportJobID   uuid.UUID `gorm:"type:uuid"`
	RawFileID     uuid.UUID `gorm:"type:uuid"`
	SHA256        string    `gorm:"column:sha256"`
	ParserVersion string
	TakenAt       *time.Time
	CreatedAt     time.Time
}

func (ImportSnapshot) TableName() string {
	return "tb_import_snapshots"
}

type AppleSourceItem struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	SourceKey          string
	ItemType           string
	ContentHash        string
	ActivityID         *uuid.UUID `gorm:"type:uuid"`
	SleepSessionID     *uuid.UUID `gorm:"type:uuid"`
	DailySummaryID     *uuid.UUID `gorm:"type:uuid"`
	DailyMetricID      *uuid.UUID `gorm:"type:uuid"`
	LastSeenSnapshotID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (AppleSourceItem) TableName() string {
	return "tb_apple_source_items"
}

type MediaWork struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	WorkKind         string
	PrimaryTitle     string
	OriginalTitle    string
	OriginalLanguage string
	FirstReleaseDate *time.Time `gorm:"type:date"`
	Description      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (MediaWork) TableName() string { return "tb_media_works" }

type MediaItem struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkID               *uuid.UUID `gorm:"type:uuid"`
	ParentItemID         *uuid.UUID `gorm:"type:uuid"`
	MediaType            string
	ItemRole             string
	Title                string
	SortTitle            string
	OriginalTitle        string
	Description          string
	ReleaseDate          *time.Time `gorm:"type:date"`
	ReleaseDatePrecision string
	SeasonNumber         *int
	EpisodeNumber        *int
	ChapterNumber        *float64
	VolumeNumber         *float64
	DurationSeconds      *int
	PageCount            *int
	EpisodeCount         *int
	ChapterCount         *int
	Language             string
	Country              string
	CoverImageURL        string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (MediaItem) TableName() string { return "tb_media_items" }

type MediaTitle struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	ScopeType  string
	ScopeID    uuid.UUID `gorm:"type:uuid"`
	Title      string
	Language   string
	Script     string
	Region     string
	TitleKind  string
	Provider   string
	IsPrimary  bool
	Confidence *float64
	CreatedAt  time.Time
}

func (MediaTitle) TableName() string { return "tb_media_titles" }

type MediaRelation struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	FromType     string
	FromID       uuid.UUID `gorm:"type:uuid"`
	ToType       string
	ToID         uuid.UUID `gorm:"type:uuid"`
	RelationType string
	Provider     string
	Confidence   *float64
	CreatedAt    time.Time
}

func (MediaRelation) TableName() string { return "tb_media_relations" }

type MediaExternalRef struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	ScopeType   string
	ScopeID     uuid.UUID `gorm:"type:uuid"`
	Provider    string
	ExternalID  string
	ExternalURL string
	Confidence  *float64
	MatchedBy   string
	CreatedAt   time.Time
}

func (MediaExternalRef) TableName() string { return "tb_media_external_refs" }

type MediaCreator struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name         string
	SortName     string
	OriginalName string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (MediaCreator) TableName() string { return "tb_media_creators" }

type MediaCreatorRole struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatorID uuid.UUID `gorm:"type:uuid"`
	ScopeType string
	ScopeID   uuid.UUID `gorm:"type:uuid"`
	Role      string
	Provider  string
	CreatedAt time.Time
}

func (MediaCreatorRole) TableName() string { return "tb_media_creator_roles" }

type MediaConsumptionEvent struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	MediaItemID     uuid.UUID `gorm:"type:uuid"`
	EventType       string
	EventAt         time.Time
	SourceKind      string
	SourceEventID   string
	Unit            string
	Position        *float64
	Total           *float64
	ProgressPercent *float64
	Rating          *float64
	RatingScale     *float64
	Note            string
	RawFileID       *uuid.UUID `gorm:"type:uuid"`
	CreatedAt       time.Time
}

func (MediaConsumptionEvent) TableName() string { return "tb_media_consumption_events" }

type MediaProgress struct {
	MediaItemID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Status               string
	Unit                 string
	Position             *float64
	Total                *float64
	ProgressPercent      *float64
	StartedOnValue       *time.Time `gorm:"column:started_on_value;type:date"`
	StartedOnPrecision   string
	LastUpdateAt         *time.Time
	CompletedOnValue     *time.Time `gorm:"column:completed_on_value;type:date"`
	CompletedOnPrecision string
	PlayCount            int
	HiddenFromContinue   bool
	SourceKind           string
	UpdatedAt            time.Time
}

func (MediaProgress) TableName() string { return "tb_media_progress" }

type MediaStateHistory struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey"`
	MediaItemID          uuid.UUID `gorm:"type:uuid"`
	SourceKind           string
	SourceEventID        string
	ObservedAt           time.Time
	EffectiveAt          *time.Time
	TimeBasis            string
	ChangeKind           string
	StateFingerprint     string
	Status               string
	Unit                 string
	Position             *float64
	Total                *float64
	ProgressPercent      *float64
	Rating               *float64
	RatingScale          *float64
	Note                 string
	RepeatCount          int
	StartedOnValue       *time.Time `gorm:"type:date"`
	StartedOnPrecision   string
	CompletedOnValue     *time.Time `gorm:"type:date"`
	CompletedOnPrecision string
	EffectiveOnValue     *time.Time `gorm:"type:date"`
	EffectiveOnPrecision string
	ProviderRecordedAt   *time.Time
	RawFileID            *uuid.UUID `gorm:"type:uuid"`
	CreatedAt            time.Time
}

func (MediaStateHistory) TableName() string { return "tb_media_state_history" }

type MediaList struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name          string
	ListKind      string
	SourceKind    string
	ExternalRefID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (MediaList) TableName() string { return "tb_media_lists" }

type MediaListItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	ListID      uuid.UUID `gorm:"type:uuid"`
	MediaItemID uuid.UUID `gorm:"type:uuid"`
	Position    *float64
	CreatedAt   time.Time
}

func (MediaListItem) TableName() string { return "tb_media_list_items" }

type MediaResolutionTask struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	TaskType       string
	Status         string
	CandidatesJSON json.RawMessage `gorm:"column:candidates_json;type:jsonb"`
	ResolutionJSON json.RawMessage `gorm:"column:resolution_json;type:jsonb"`
	CreatedAt      time.Time
	ResolvedAt     *time.Time
}

func (MediaResolutionTask) TableName() string { return "tb_media_resolution_tasks" }

type MediaSyncState struct {
	ID            uuid.UUID       `gorm:"type:uuid;primaryKey"`
	ConnectorID   string          `gorm:"uniqueIndex"`
	CursorJSON    json.RawMessage `gorm:"column:cursor_json;type:jsonb"`
	Status        string
	LastError     *string
	LastFetchedAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (MediaSyncState) TableName() string { return "tb_media_sync_state" }

type IntakePayload struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	SourceKind    string
	SourceActor   string
	SourceEventID string
	ContentType   string
	SHA256        string `gorm:"column:sha256"`
	SizeBytes     int64
	StoragePath   string
	PayloadJSON   *json.RawMessage `gorm:"column:payload_json;type:jsonb"`
	ReceivedAt    time.Time
	ParsedAt      *time.Time
	CreatedAt     time.Time
}

func (IntakePayload) TableName() string {
	return "tb_intake_payloads"
}

type Job struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Kind         string
	Status       string
	Priority     int
	PayloadJSON  json.RawMessage `gorm:"column:payload_json;type:jsonb"`
	Attempts     int
	MaxAttempts  int
	RunAfter     time.Time
	LockedBy     *string
	LockedAt     *time.Time
	ErrorMessage *string
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Job) TableName() string {
	return "tb_jobs"
}

type JobSchedule struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Kind         string
	Enabled      bool
	ScheduleKind string
	ScheduleExpr string
	PayloadJSON  json.RawMessage `gorm:"column:payload_json;type:jsonb"`
	NextRunAt    *time.Time
	LastRunAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (JobSchedule) TableName() string {
	return "tb_job_schedules"
}

type Task struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	Title       string
	Notes       string
	Status      string
	DueDate     *time.Time `gorm:"column:due_date;type:date"`
	Priority    int
	Source      string
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Task) TableName() string {
	return "tb_tasks"
}

type Expense struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	OccurredOn        time.Time `gorm:"type:date"`
	Currency          string
	AmountMinor       int64
	Category          string
	Merchant          string
	Note              string
	ItemsJSON         json.RawMessage `gorm:"column:items_json;type:jsonb"`
	SourceKind        string
	SourceRef         string
	CreateFingerprint string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

func (Expense) TableName() string {
	return "tb_expenses"
}
