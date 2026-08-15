package media

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultPageLimit = 50

type Service struct {
	db *gorm.DB
}

type ListFilters struct {
	Status        string
	MediaType     string
	Family        string
	CompletedYear *int
	Limit         int
	Cursor        *Cursor
}

// familyMediaTypes maps a coarse family filter to the granular media_type
// values the sync stores, so the UI can offer an anime/manga-books/games filter.
var familyMediaTypes = map[string][]string{
	"anime":      {"anime_season", "movie", "ona", "ova", "special"},
	"manga_book": {"manga", "one_shot", "light_novel"},
	"book":       {"book"},
	"game":       {"game"},
}

func IsFamily(value string) bool {
	_, ok := familyMediaTypes[value]
	return ok
}

type Item struct {
	ID                   uuid.UUID
	Title                string
	MediaType            string
	ItemRole             string
	CoverImageURL        string
	Status               *string
	Position             *float64
	Total                *float64
	Unit                 *string
	ProgressPercent      *float64
	LastUpdateAt         time.Time
	Rating               *float64
	RatingScale          *float64
	HiddenFromContinue   bool
	NativeTitle          *string
	EpisodeCount         *int
	ChapterCount         *int
	StartedOnValue       *time.Time
	StartedOnPrecision   string
	CompletedOnValue     *time.Time
	CompletedOnPrecision string
}

type Page struct {
	Items        []Item
	NextCursor   *Cursor
	HasMore      bool
	StatusCounts map[string]int
	ActiveCount  int
}

type CompletionBucket struct {
	Year  int `gorm:"column:year" json:"year"`
	Count int `gorm:"column:count" json:"count"`
}

type ScoreBucket struct {
	Score float64 `gorm:"column:score" json:"score"`
	Count int     `gorm:"column:count" json:"count"`
}

type TypeBucket struct {
	Type  string `gorm:"column:type" json:"type"`
	Count int    `gorm:"column:count" json:"count"`
}

type Totals struct {
	ItemCount             int     `json:"item_count"`
	CompletedCount        int     `json:"completed_count"`
	CurrentCompletedCount int     `json:"current_completed_count"`
	ThisYearCompleted     int     `json:"this_year_completed"`
	AverageRating         float64 `json:"average_rating"`
}

type Aggregates struct {
	Totals            Totals             `json:"totals"`
	CompletionsByYear []CompletionBucket `json:"completions_by_year"`
	ScoreDistribution []ScoreBucket      `json:"score_distribution"`
	TypeSplit         []TypeBucket       `json:"type_split"`
}

type PeriodFilters struct {
	From time.Time
	To   time.Time
}

type PeriodKindTotal struct {
	Kind           string
	EventCount     int
	CompletedCount int
}

type PeriodCompletedItem struct {
	ID          uuid.UUID
	Title       string
	MediaType   string
	CompletedAt time.Time
}

type PeriodReport struct {
	EventCount     int
	CompletedCount int
	RatedCount     int
	AverageRating  *float64
	ByKind         []PeriodKindTotal
	CompletedItems []PeriodCompletedItem
}

type MetricValue struct {
	CompletedAt time.Time
	MediaKind   string
	Source      string
}

type WorkDetail struct {
	ID               uuid.UUID  `gorm:"column:id"`
	WorkKind         string     `gorm:"column:work_kind"`
	PrimaryTitle     string     `gorm:"column:primary_title"`
	OriginalTitle    string     `gorm:"column:original_title"`
	OriginalLanguage string     `gorm:"column:original_language"`
	FirstReleaseDate *time.Time `gorm:"column:first_release_date"`
	Description      string     `gorm:"column:description"`
}

type CreatorDetail struct {
	ID   uuid.UUID `gorm:"column:id"`
	Name string    `gorm:"column:name"`
	Role string    `gorm:"column:role"`
}

type RelationDetail struct {
	ID            uuid.UUID `gorm:"column:id"`
	RelationType  string    `gorm:"column:relation_type"`
	Direction     string    `gorm:"column:direction"`
	RelatedItemID uuid.UUID `gorm:"column:related_item_id"`
	RelatedTitle  string    `gorm:"column:related_title"`
	RelatedType   string    `gorm:"column:related_type"`
	CoverImageURL string    `gorm:"column:cover_image_url"`
}

type EventDetail struct {
	ID              uuid.UUID `gorm:"column:id"`
	EventType       string    `gorm:"column:event_type"`
	EventAt         time.Time `gorm:"column:event_at"`
	Unit            string    `gorm:"column:unit"`
	Position        *float64  `gorm:"column:position"`
	Total           *float64  `gorm:"column:total"`
	ProgressPercent *float64  `gorm:"column:progress_percent"`
	Rating          *float64  `gorm:"column:rating"`
	RatingScale     *float64  `gorm:"column:rating_scale"`
	Note            string    `gorm:"column:note"`
}

type ProgressDetail struct {
	Status               string     `gorm:"column:status"`
	Unit                 string     `gorm:"column:unit"`
	Position             *float64   `gorm:"column:position"`
	Total                *float64   `gorm:"column:total"`
	ProgressPercent      *float64   `gorm:"column:progress_percent"`
	StartedOnValue       *time.Time `gorm:"column:started_on_value"`
	StartedOnPrecision   string     `gorm:"column:started_on_precision"`
	LastUpdateAt         *time.Time `gorm:"column:last_update_at"`
	CompletedOnValue     *time.Time `gorm:"column:completed_on_value"`
	CompletedOnPrecision string     `gorm:"column:completed_on_precision"`
	PlayCount            int        `gorm:"column:play_count"`
}

type Detail struct {
	Item      Item
	Work      WorkDetail
	Progress  *ProgressDetail
	Creators  []CreatorDetail
	Relations []RelationDetail
	Events    []EventDetail
	Updates   []Change
}

type Event struct {
	ID              uuid.UUID `gorm:"column:id"`
	MediaItemID     uuid.UUID `gorm:"column:media_item_id"`
	Title           string    `gorm:"column:title"`
	NativeTitle     *string   `gorm:"column:native_title"`
	CoverImageURL   string    `gorm:"column:cover_image_url"`
	EventType       string    `gorm:"column:event_type"`
	OccurredAt      time.Time `gorm:"column:occurred_at"`
	Unit            string    `gorm:"column:unit"`
	Position        *float64  `gorm:"column:position"`
	Total           *float64  `gorm:"column:total"`
	ProgressPercent *float64  `gorm:"column:progress_percent"`
	Rating          *float64  `gorm:"column:rating"`
	RatingScale     *float64  `gorm:"column:rating_scale"`
}

type EventListFilters struct {
	From   *time.Time
	To     *time.Time
	Limit  int
	Cursor *Cursor
}

type EventPage struct {
	Items      []Event
	NextCursor *Cursor
	HasMore    bool
}

type CreateEventInput struct {
	MediaItemID     uuid.UUID
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
}

var (
	ErrEventConflict         = errors.New("media event idempotency key conflicts with existing event")
	ErrEventAtRequired       = errors.New("event_at is required")
	ErrInvalidEventType      = errors.New("invalid event_type")
	ErrSourceEventIDRequired = errors.New("source_event_id is required")
	ErrMediaItemNotFound     = errors.New("media item not found")
)

type Change struct {
	ID                   uuid.UUID
	MediaItemID          uuid.UUID
	Title                string
	NativeTitle          *string
	CoverImageURL        string
	SourceKind           string
	ChangeKind           string
	TimeBasis            string
	ObservedAt           time.Time
	EffectiveAt          *time.Time
	EffectiveOnValue     *time.Time
	EffectiveOnPrecision string
	ProviderRecordedAt   *time.Time
	Status               string
	Unit                 string
	Position             *float64
	Total                *float64
	ProgressPercent      *float64
	Rating               *float64
	RatingScale          *float64
	Note                 string
	RepeatCount          int
}

type ChangeListFilters struct {
	From        *time.Time
	To          *time.Time
	Limit       int
	Cursor      *Cursor
	MediaItemID *uuid.UUID
}

type ChangePage struct {
	Items      []Change
	NextCursor *Cursor
	HasMore    bool
}
