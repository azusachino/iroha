package media

import (
	"errors"
	"sort"
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
	"manga_book": {"manga", "one_shot", "light_novel", "book"},
	"game":       {"game"},
}

func IsFamily(value string) bool {
	_, ok := familyMediaTypes[value]
	return ok
}

type Item struct {
	ID                 uuid.UUID
	Title              string
	MediaType          string
	ItemRole           string
	CoverImageURL      string
	Status             *string
	Position           *float64
	Total              *float64
	Unit               *string
	ProgressPercent    *float64
	LastUpdateAt       time.Time
	Rating             *float64
	RatingScale        *float64
	HiddenFromContinue bool
	NativeTitle        *string
	EpisodeCount       *int
	ChapterCount       *int
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
	ItemCount         int     `json:"item_count"`
	CompletedCount    int     `json:"completed_count"`
	ThisYearCompleted int     `json:"this_year_completed"`
	AverageRating     float64 `json:"average_rating"`
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
	ID              uuid.UUID  `gorm:"column:id"`
	EventType       string     `gorm:"column:event_type"`
	EventAt         *time.Time `gorm:"column:event_at"`
	Unit            string     `gorm:"column:unit"`
	Position        *float64   `gorm:"column:position"`
	Total           *float64   `gorm:"column:total"`
	ProgressPercent *float64   `gorm:"column:progress_percent"`
	Rating          *float64   `gorm:"column:rating"`
	RatingScale     *float64   `gorm:"column:rating_scale"`
	Note            string     `gorm:"column:note"`
}

type ProgressDetail struct {
	Status          string     `gorm:"column:status"`
	Unit            string     `gorm:"column:unit"`
	Position        *float64   `gorm:"column:position"`
	Total           *float64   `gorm:"column:total"`
	ProgressPercent *float64   `gorm:"column:progress_percent"`
	StartedAt       *time.Time `gorm:"column:started_at"`
	LastUpdateAt    *time.Time `gorm:"column:last_update_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at"`
	PlayCount       int        `gorm:"column:play_count"`
}

type Detail struct {
	Item      Item
	Work      WorkDetail
	Progress  *ProgressDetail
	Creators  []CreatorDetail
	Relations []RelationDetail
	Events    []EventDetail
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

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) PeriodReport(filters PeriodFilters) (PeriodReport, error) {
	if !filters.From.Before(filters.To) {
		return PeriodReport{}, errors.New("period from must be before to")
	}

	type aggregateRow struct {
		Kind          string   `gorm:"column:kind"`
		EventCount    int      `gorm:"column:event_count"`
		RatedCount    int      `gorm:"column:rated_count"`
		AverageRating *float64 `gorm:"column:average_rating"`
	}
	var aggregateRows []aggregateRow
	if err := s.db.Raw(`
		SELECT item.media_type AS kind,
			count(*)::int AS event_count,
			count(*) FILTER (WHERE event.rating IS NOT NULL AND event.rating_scale IS NOT NULL AND event.rating_scale <> 0)::int AS rated_count,
			avg(least(greatest(event.rating / nullif(event.rating_scale, 0) * 10, 0), 10))
				FILTER (WHERE event.rating IS NOT NULL AND event.rating_scale IS NOT NULL AND event.rating_scale <> 0) AS average_rating
		FROM tb_media_consumption_events AS event
		JOIN tb_media_items AS item ON item.id = event.media_item_id
		WHERE event.event_at IS NOT NULL
			AND event.event_at >= ?
			AND event.event_at < ?
			AND event.event_type <> 'list_state'
		GROUP BY item.media_type
		ORDER BY item.media_type`, filters.From, filters.To).Scan(&aggregateRows).Error; err != nil {
		return PeriodReport{}, err
	}

	type completionRow struct {
		ID          uuid.UUID `gorm:"column:id"`
		ItemID      uuid.UUID `gorm:"column:item_id"`
		Title       string    `gorm:"column:title"`
		MediaType   string    `gorm:"column:media_type"`
		CompletedAt time.Time `gorm:"column:completed_at"`
	}
	var completionRows []completionRow
	if err := s.db.Raw(`
		SELECT item.id, item.id AS item_id, item.title, item.media_type, progress.finished_at AS completed_at
		FROM tb_media_progress AS progress
		JOIN tb_media_items AS item ON item.id = progress.media_item_id
		WHERE progress.finished_at IS NOT NULL
			AND progress.finished_at >= ?
			AND progress.finished_at < ?
		UNION ALL
		SELECT event.id, event.media_item_id AS item_id, item.title, item.media_type, event.event_at AS completed_at
		FROM tb_media_consumption_events AS event
		JOIN tb_media_items AS item ON item.id = event.media_item_id
		WHERE event.event_type IN ('finished', 'completed')
			AND event.event_at IS NOT NULL
			AND event.event_at >= ?
			AND event.event_at < ?
		ORDER BY completed_at ASC, item_id ASC, id ASC`, filters.From, filters.To, filters.From, filters.To).Scan(&completionRows).Error; err != nil {
		return PeriodReport{}, err
	}

	completedByItem := make(map[uuid.UUID]PeriodCompletedItem, len(completionRows))
	for _, row := range completionRows {
		candidate := PeriodCompletedItem{
			ID: row.ItemID, Title: row.Title, MediaType: row.MediaType, CompletedAt: row.CompletedAt,
		}
		current, ok := completedByItem[row.ItemID]
		if !ok || candidate.CompletedAt.After(current.CompletedAt) {
			completedByItem[row.ItemID] = candidate
		}
	}
	completedItems := make([]PeriodCompletedItem, 0, len(completedByItem))
	for _, item := range completedByItem {
		completedItems = append(completedItems, item)
	}
	sort.Slice(completedItems, func(i, j int) bool {
		if completedItems[i].CompletedAt.Equal(completedItems[j].CompletedAt) {
			return completedItems[i].ID.String() < completedItems[j].ID.String()
		}
		return completedItems[i].CompletedAt.Before(completedItems[j].CompletedAt)
	})

	byKind := make(map[string]PeriodKindTotal, len(aggregateRows))
	for _, row := range aggregateRows {
		byKind[row.Kind] = PeriodKindTotal{Kind: row.Kind, EventCount: row.EventCount, CompletedCount: 0}
	}
	for _, item := range completedItems {
		kind := byKind[item.MediaType]
		kind.Kind = item.MediaType
		kind.CompletedCount++
		byKind[item.MediaType] = kind
	}
	kinds := make([]PeriodKindTotal, 0, len(byKind))
	for _, kind := range byKind {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i].Kind < kinds[j].Kind })

	var ratedCount int
	eventCount := 0
	var averageRating *float64
	var weightedRating float64
	for _, row := range aggregateRows {
		eventCount += row.EventCount
		ratedCount += row.RatedCount
		if row.AverageRating != nil {
			weightedRating += *row.AverageRating * float64(row.RatedCount)
		}
	}
	if ratedCount > 0 {
		value := weightedRating / float64(ratedCount)
		averageRating = &value
	}

	return PeriodReport{
		EventCount: eventCount, CompletedCount: len(completedItems), RatedCount: ratedCount,
		AverageRating: averageRating, ByKind: kinds, CompletedItems: completedItems,
	}, nil
}

func (s *Service) List(filters ListFilters) (Page, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = defaultPageLimit
	}

	query := s.db.Table("tb_media_items AS item").
		Select(`
			item.id,
			item.title,
			item.media_type,
			item.item_role,
			item.cover_image_url,
			progress.status,
			progress.position,
			coalesce(progress.total, case
				when progress.unit = 'episodes' then item.episode_count
				when progress.unit = 'chapters' then item.chapter_count
			end) AS total,
			progress.unit,
			progress.progress_percent,
			coalesce(progress.last_update_at, item.updated_at) AS last_update_at,
			rating.rating,
			rating.rating_scale,
			coalesce(progress.hidden_from_continue, false) AS hidden_from_continue,
			(SELECT t.title FROM tb_media_titles t
			 WHERE t.scope_type = 'item' AND t.scope_id = item.id AND t.title_kind = 'original'
			 ORDER BY t.is_primary DESC, t.created_at ASC LIMIT 1) AS native_title`).
		Joins("LEFT JOIN tb_media_progress AS progress ON progress.media_item_id = item.id").
		Joins(`LEFT JOIN LATERAL (
			SELECT event.rating, event.rating_scale
			FROM tb_media_consumption_events AS event
			WHERE event.media_item_id = item.id AND event.rating IS NOT NULL
			ORDER BY event.event_at DESC NULLS LAST, event.created_at DESC, event.id DESC
			LIMIT 1
		) AS rating ON true`)
	query = query.Joins(`LEFT JOIN LATERAL (
		SELECT max(completed_at) AS completed_at
		FROM (
			SELECT finished_at AS completed_at
			FROM tb_media_progress
			WHERE media_item_id = item.id AND finished_at IS NOT NULL
			UNION ALL
			SELECT event_at AS completed_at
			FROM tb_media_consumption_events
			WHERE media_item_id = item.id
				AND event_type IN ('finished', 'completed')
				AND event_at IS NOT NULL
		) AS completion_dates
	) AS completion ON true`)

	if filters.Status != "" {
		query = query.Where("progress.status = ?", filters.Status)
	}
	if filters.MediaType != "" {
		query = query.Where("item.media_type = ?", filters.MediaType)
	}
	if types, ok := familyMediaTypes[filters.Family]; ok {
		query = query.Where("item.media_type IN ?", types)
	}
	if filters.CompletedYear != nil {
		query = query.Where("extract(year from completion.completed_at) = ?", *filters.CompletedYear)
	}
	if filters.Cursor != nil {
		query = query.Where("(coalesce(progress.last_update_at, item.updated_at), item.id) < (?, ?)", filters.Cursor.LastUpdateAt, filters.Cursor.ID)
	}

	var rows []Item
	if err := query.Order("last_update_at DESC, item.id DESC").Limit(limit + 1).Scan(&rows).Error; err != nil {
		return Page{}, err
	}

	statusCounts, activeCount, err := s.Counts(filters)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: rows, StatusCounts: statusCounts, ActiveCount: activeCount}
	if len(rows) > limit {
		last := rows[limit-1]
		page.Items = rows[:limit]
		page.HasMore = true
		page.NextCursor = &Cursor{LastUpdateAt: last.LastUpdateAt, ID: last.ID}
	}
	return page, nil
}

// Counts returns exact status totals for the current family/completion-year
// facet. Status itself is intentionally ignored so the UI can keep category
// counts visible while one status is selected. ActiveCount excludes paused
// items hidden from the continue shelf.
func (s *Service) Counts(filters ListFilters) (map[string]int, int, error) {
	query := s.db.Table("tb_media_items AS item").
		Select("case when progress.status = 'in_progress' and coalesce(progress.hidden_from_continue, false) then 'paused' else coalesce(progress.status, 'unknown') end AS status, count(*)::int AS count, count(*) filter (where progress.status = 'in_progress' and coalesce(progress.hidden_from_continue, false) = false)::int AS active_count").
		Joins("LEFT JOIN tb_media_progress AS progress ON progress.media_item_id = item.id").
		Joins(`LEFT JOIN LATERAL (
			SELECT max(completed_at) AS completed_at
			FROM (
				SELECT finished_at AS completed_at FROM tb_media_progress
				WHERE media_item_id = item.id AND finished_at IS NOT NULL
				UNION ALL
				SELECT event_at AS completed_at FROM tb_media_consumption_events
				WHERE media_item_id = item.id
					AND event_type IN ('finished', 'completed')
					AND event_at IS NOT NULL
			) AS completion_dates
		) AS completion ON true`)
	if types, ok := familyMediaTypes[filters.Family]; ok {
		query = query.Where("item.media_type IN ?", types)
	}
	if filters.CompletedYear != nil {
		query = query.Where("extract(year from completion.completed_at) = ?", *filters.CompletedYear)
	}
	type statusCountRow struct {
		Status      string `gorm:"column:status"`
		Count       int    `gorm:"column:count"`
		ActiveCount int    `gorm:"column:active_count"`
	}
	var rows []statusCountRow
	if err := query.Group("progress.status, progress.hidden_from_continue").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	counts := make(map[string]int, len(rows))
	activeCount := 0
	for _, row := range rows {
		counts[row.Status] = row.Count
		activeCount += row.ActiveCount
	}
	return counts, activeCount, nil
}

func (s *Service) Aggregates(now time.Time) (Aggregates, error) {
	completionSQL := `
		WITH completion_dates AS (
			SELECT media_item_id, finished_at AS completed_at
			FROM tb_media_progress
			WHERE finished_at IS NOT NULL
			UNION ALL
			SELECT media_item_id, event_at AS completed_at
			FROM tb_media_consumption_events
			WHERE event_type IN ('finished', 'completed') AND event_at IS NOT NULL
		), completions AS (
			SELECT media_item_id, max(completed_at) AS completed_at
			FROM completion_dates
			GROUP BY media_item_id
		)
		SELECT extract(year FROM completed_at)::int AS year, count(*)::int AS count
		FROM completions
		GROUP BY year
		ORDER BY year`
	var completionRows []CompletionBucket
	if err := s.db.Raw(completionSQL).Scan(&completionRows).Error; err != nil {
		return Aggregates{}, err
	}

	ratingSQL := `
		WITH latest_ratings AS (
			SELECT DISTINCT ON (media_item_id)
				media_item_id,
				rating / nullif(rating_scale, 0) * 10 AS normalized_rating
			FROM tb_media_consumption_events
			WHERE rating IS NOT NULL AND rating_scale IS NOT NULL
			ORDER BY media_item_id, event_at DESC NULLS LAST, created_at DESC, id DESC
		)
		SELECT round(least(greatest(normalized_rating, 0), 10)) AS score, count(*)::int AS count
		FROM latest_ratings
		GROUP BY score
		ORDER BY score`
	var scoreRows []ScoreBucket
	if err := s.db.Raw(ratingSQL).Scan(&scoreRows).Error; err != nil {
		return Aggregates{}, err
	}

	typeSQL := `
		WITH latest_ratings AS (
			SELECT DISTINCT ON (media_item_id)
				media_item_id,
				rating / nullif(rating_scale, 0) * 10 AS normalized_rating
			FROM tb_media_consumption_events
			WHERE rating IS NOT NULL AND rating_scale IS NOT NULL
			ORDER BY media_item_id, event_at DESC NULLS LAST, created_at DESC, id DESC
		), completion_dates AS (
			SELECT media_item_id, finished_at AS completed_at
			FROM tb_media_progress
			WHERE finished_at IS NOT NULL
			UNION ALL
			SELECT media_item_id, event_at AS completed_at
			FROM tb_media_consumption_events
			WHERE event_type IN ('finished', 'completed') AND event_at IS NOT NULL
		), completions AS (
			SELECT media_item_id, max(completed_at) AS completed_at
			FROM completion_dates
			GROUP BY media_item_id
		), grouped AS (
			-- Kind lives on the item (media_type: anime_season, manga, movie…);
			-- the parent work_kind is a generic 'media', so group by media_type.
			SELECT item.media_type AS type,
				count(*)::int AS count,
				avg(latest_ratings.normalized_rating) AS average_rating,
				count(latest_ratings.normalized_rating)::int AS rating_count,
				count(completions.media_item_id)::int AS completed_count
			FROM tb_media_items AS item
			LEFT JOIN latest_ratings ON latest_ratings.media_item_id = item.id
			LEFT JOIN completions ON completions.media_item_id = item.id
			GROUP BY type
		)
		SELECT type, count, average_rating, rating_count, completed_count,
			sum(count) OVER ()::int AS item_count
		FROM grouped
		ORDER BY type`
	var typeRows []struct {
		Type           string  `gorm:"column:type"`
		Count          int     `gorm:"column:count"`
		AverageRating  float64 `gorm:"column:average_rating"`
		RatingCount    int     `gorm:"column:rating_count"`
		CompletedCount int     `gorm:"column:completed_count"`
		ItemCount      int     `gorm:"column:item_count"`
	}
	if err := s.db.Raw(typeSQL).Scan(&typeRows).Error; err != nil {
		return Aggregates{}, err
	}

	result := Aggregates{
		CompletionsByYear: completionRows,
		ScoreDistribution: scoreRows,
		TypeSplit:         make([]TypeBucket, 0, len(typeRows)),
	}
	for _, row := range typeRows {
		result.TypeSplit = append(result.TypeSplit, TypeBucket{Type: row.Type, Count: row.Count})
		result.Totals.ItemCount += row.Count
		result.Totals.CompletedCount += row.CompletedCount
		result.Totals.AverageRating += row.AverageRating * float64(row.RatingCount)
	}
	var ratedCount int
	for _, row := range typeRows {
		ratedCount += row.RatingCount
	}
	if ratedCount > 0 {
		result.Totals.AverageRating /= float64(ratedCount)
	}
	for _, row := range completionRows {
		if row.Year == now.UTC().Year() {
			result.Totals.ThisYearCompleted = row.Count
			break
		}
	}
	return result, nil
}

func (s *Service) Get(id uuid.UUID) (Detail, bool, error) {
	var row struct {
		ID                 uuid.UUID  `gorm:"column:id"`
		Title              string     `gorm:"column:title"`
		MediaType          string     `gorm:"column:media_type"`
		ItemRole           string     `gorm:"column:item_role"`
		CoverImageURL      string     `gorm:"column:cover_image_url"`
		Status             *string    `gorm:"column:status"`
		Position           *float64   `gorm:"column:position"`
		Total              *float64   `gorm:"column:total"`
		ProgressPercent    *float64   `gorm:"column:progress_percent"`
		LastUpdateAt       time.Time  `gorm:"column:last_update_at"`
		Rating             *float64   `gorm:"column:rating"`
		RatingScale        *float64   `gorm:"column:rating_scale"`
		HiddenFromContinue bool       `gorm:"column:hidden_from_continue"`
		NativeTitle        *string    `gorm:"column:native_title"`
		EpisodeCount       *int       `gorm:"column:episode_count"`
		ChapterCount       *int       `gorm:"column:chapter_count"`
		WorkID             uuid.UUID  `gorm:"column:work_id"`
		WorkKind           string     `gorm:"column:work_kind"`
		PrimaryTitle       string     `gorm:"column:primary_title"`
		OriginalTitle      string     `gorm:"column:original_title"`
		OriginalLanguage   string     `gorm:"column:original_language"`
		FirstReleaseDate   *time.Time `gorm:"column:first_release_date"`
		Description        string     `gorm:"column:description"`
	}
	result := s.db.Table("tb_media_items AS item").
		Select(`item.id, item.title, item.media_type, item.item_role, item.cover_image_url,
			item.episode_count, item.chapter_count,
			progress.status, progress.position, progress.total, progress.progress_percent,
			coalesce(progress.last_update_at, item.updated_at) AS last_update_at,
			rating.rating, rating.rating_scale,
			coalesce(progress.hidden_from_continue, false) AS hidden_from_continue,
			(SELECT t.title FROM tb_media_titles t
			 WHERE t.scope_type = 'item' AND t.scope_id = item.id AND t.title_kind = 'original'
			 ORDER BY t.is_primary DESC, t.created_at ASC LIMIT 1) AS native_title,
			work.id AS work_id, work.work_kind, work.primary_title, work.original_title,
			work.original_language, work.first_release_date, work.description`).
		Joins("LEFT JOIN tb_media_progress AS progress ON progress.media_item_id = item.id").
		Joins("LEFT JOIN tb_media_works AS work ON work.id = item.work_id").
		Joins(`LEFT JOIN LATERAL (
			SELECT event.rating, event.rating_scale
			FROM tb_media_consumption_events AS event
			WHERE event.media_item_id = item.id AND event.rating IS NOT NULL
			ORDER BY event.event_at DESC NULLS LAST, event.created_at DESC, event.id DESC
			LIMIT 1
		) AS rating ON true`).
		Where("item.id = ?", id).
		Scan(&row)
	if result.Error != nil {
		return Detail{}, false, result.Error
	}
	if row.ID == uuid.Nil {
		return Detail{}, false, nil
	}

	detail := Detail{
		Item: Item{
			ID: row.ID, Title: row.Title, MediaType: row.MediaType, ItemRole: row.ItemRole,
			CoverImageURL: row.CoverImageURL, Status: row.Status, Position: row.Position,
			Total: row.Total, ProgressPercent: row.ProgressPercent, LastUpdateAt: row.LastUpdateAt,
			Rating: row.Rating, RatingScale: row.RatingScale, HiddenFromContinue: row.HiddenFromContinue,
			NativeTitle: row.NativeTitle, EpisodeCount: row.EpisodeCount, ChapterCount: row.ChapterCount,
		},
		Work: WorkDetail{
			ID: row.WorkID, WorkKind: row.WorkKind, PrimaryTitle: row.PrimaryTitle,
			OriginalTitle: row.OriginalTitle, OriginalLanguage: row.OriginalLanguage,
			FirstReleaseDate: row.FirstReleaseDate, Description: row.Description,
		},
	}

	var progress ProgressDetail
	if err := s.db.Table("tb_media_progress").Where("media_item_id = ?", id).Scan(&progress).Error; err != nil {
		return Detail{}, false, err
	}
	if progress.Status != "" {
		detail.Progress = &progress
	}

	if err := s.db.Raw(`
		SELECT creator.id, creator.name, role.role
		FROM tb_media_creator_roles AS role
		JOIN tb_media_creators AS creator ON creator.id = role.creator_id
		WHERE role.scope_type = 'item' AND role.scope_id = ?
		ORDER BY role.role, creator.sort_name, creator.name`, id).Scan(&detail.Creators).Error; err != nil {
		return Detail{}, false, err
	}
	if err := s.db.Raw(`
		SELECT relation.id, relation.relation_type,
			CASE WHEN relation.from_id = ? THEN 'outgoing' ELSE 'incoming' END AS direction,
			related.id AS related_item_id, related.title AS related_title,
			related.media_type AS related_type, related.cover_image_url
		FROM tb_media_relations AS relation
		JOIN tb_media_items AS related
			ON related.id = CASE WHEN relation.from_id = ? THEN relation.to_id ELSE relation.from_id END
		WHERE relation.from_type = 'item' AND relation.to_type = 'item'
			AND (relation.from_id = ? OR relation.to_id = ?)
		ORDER BY relation.relation_type, related.sort_title, related.title`, id, id, id, id).Scan(&detail.Relations).Error; err != nil {
		return Detail{}, false, err
	}
	detail.Relations = dedupeRelations(detail.Relations)
	if err := s.db.Table("tb_media_consumption_events").Where("media_item_id = ?", id).Order("event_at DESC NULLS LAST, created_at DESC, id DESC").Scan(&detail.Events).Error; err != nil {
		return Detail{}, false, err
	}
	return detail, true, nil
}

// inverseRelationType holds the pairs where a relation observed from the
// *other* item's own sync reads backwards from this item's point of view --
// AniList reports each side of a season split independently (season 1's own
// relations list says SEQUEL, season 2's own list says PREQUEL for the same
// pair), so the "incoming" row's type must be flipped to read correctly here.
// Types outside this map (ADAPTATION, ALTERNATIVE, SOURCE, ...) are reported
// identically from both sides in provider data, so they pass through as-is.
var inverseRelationType = map[string]string{
	"PREQUEL": "SEQUEL",
	"SEQUEL":  "PREQUEL",
	"PARENT":  "SIDE_STORY",
}

// dedupeRelations collapses the reciprocal edges that come from syncing both
// endpoints of a relation independently (see persistMediaRelations) into one
// entry per related item. The "outgoing" row -- written from this item's own
// sync -- is authoritative when present; an "incoming"-only row is flipped
// through inverseRelationType so PREQUEL/SEQUEL etc. still read correctly
// from this item's perspective.
func dedupeRelations(relations []RelationDetail) []RelationDetail {
	byRelated := make(map[uuid.UUID]RelationDetail, len(relations))
	order := make([]uuid.UUID, 0, len(relations))
	for _, rel := range relations {
		existing, seen := byRelated[rel.RelatedItemID]
		if !seen {
			order = append(order, rel.RelatedItemID)
			byRelated[rel.RelatedItemID] = rel
			continue
		}
		if existing.Direction == "outgoing" {
			continue
		}
		if rel.Direction == "outgoing" {
			byRelated[rel.RelatedItemID] = rel
		}
	}
	result := make([]RelationDetail, 0, len(order))
	for _, relatedID := range order {
		rel := byRelated[relatedID]
		if rel.Direction == "incoming" {
			if inverse, ok := inverseRelationType[rel.RelationType]; ok {
				rel.RelationType = inverse
			}
		}
		result = append(result, rel)
	}
	return result
}

func (s *Service) Events(filters EventListFilters) (EventPage, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = defaultPageLimit
	}
	query := s.db.Table("tb_media_consumption_events AS event").
		Select(`event.id, event.media_item_id, item.title, item.cover_image_url,
			(SELECT t.title FROM tb_media_titles t
			 WHERE t.scope_type = 'item' AND t.scope_id = item.id AND t.title_kind = 'original'
			 ORDER BY t.is_primary DESC, t.created_at ASC LIMIT 1) AS native_title,
			event.event_type, coalesce(event.event_at, event.created_at) AS occurred_at,
			event.unit, event.position, event.total, event.progress_percent,
			event.rating, event.rating_scale`).
		Joins("JOIN tb_media_items AS item ON item.id = event.media_item_id")
	if filters.From != nil {
		query = query.Where("coalesce(event.event_at, event.created_at) >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("coalesce(event.event_at, event.created_at) < ?", *filters.To)
	}
	if filters.Cursor != nil {
		query = query.Where("(coalesce(event.event_at, event.created_at), event.id) < (?, ?)", filters.Cursor.LastUpdateAt, filters.Cursor.ID)
	}
	var rows []Event
	if err := query.Order("occurred_at DESC, event.id DESC").Limit(limit + 1).Scan(&rows).Error; err != nil {
		return EventPage{}, err
	}
	page := EventPage{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = &Cursor{LastUpdateAt: last.OccurredAt, ID: last.ID}
	}
	return page, nil
}
