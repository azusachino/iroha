package media

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultPageLimit = 50

type Service struct {
	db *gorm.DB
}

type ListFilters struct {
	Status    string
	MediaType string
	Limit     int
	Cursor    *Cursor
}

type Item struct {
	ID              uuid.UUID
	Title           string
	MediaType       string
	ItemRole        string
	CoverImageURL   string
	Status          *string
	Position        *float64
	Total           *float64
	ProgressPercent *float64
	LastUpdateAt    time.Time
	Rating          *float64
	RatingScale     *float64
}

type Page struct {
	Items      []Item
	NextCursor *Cursor
	HasMore    bool
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

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
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
			progress.total,
			progress.progress_percent,
			coalesce(progress.last_update_at, item.updated_at) AS last_update_at,
			rating.rating,
			rating.rating_scale`).
		Joins("LEFT JOIN tb_media_progress AS progress ON progress.media_item_id = item.id").
		Joins(`LEFT JOIN LATERAL (
			SELECT event.rating, event.rating_scale
			FROM tb_media_consumption_events AS event
			WHERE event.media_item_id = item.id AND event.rating IS NOT NULL
			ORDER BY event.event_at DESC NULLS LAST, event.created_at DESC, event.id DESC
			LIMIT 1
		) AS rating ON true`)

	if filters.Status != "" {
		query = query.Where("progress.status = ?", filters.Status)
	}
	if filters.MediaType != "" {
		query = query.Where("item.media_type = ?", filters.MediaType)
	}
	if filters.Cursor != nil {
		query = query.Where("(coalesce(progress.last_update_at, item.updated_at), item.id) < (?, ?)", filters.Cursor.LastUpdateAt, filters.Cursor.ID)
	}

	var rows []Item
	if err := query.Order("last_update_at DESC, item.id DESC").Limit(limit + 1).Scan(&rows).Error; err != nil {
		return Page{}, err
	}

	page := Page{Items: rows}
	if len(rows) > limit {
		last := rows[limit-1]
		page.Items = rows[:limit]
		page.HasMore = true
		page.NextCursor = &Cursor{LastUpdateAt: last.LastUpdateAt, ID: last.ID}
	}
	return page, nil
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
			SELECT coalesce(work.work_kind, item.media_type) AS type,
				count(*)::int AS count,
				avg(latest_ratings.normalized_rating) AS average_rating,
				count(latest_ratings.normalized_rating)::int AS rating_count,
				count(completions.media_item_id)::int AS completed_count
			FROM tb_media_items AS item
			LEFT JOIN tb_media_works AS work ON work.id = item.work_id
			LEFT JOIN latest_ratings ON latest_ratings.media_item_id = item.id
			LEFT JOIN completions ON completions.media_item_id = item.id
			GROUP BY type
		)
		SELECT type, count, average_rating, completed_count,
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
		ID               uuid.UUID  `gorm:"column:id"`
		Title            string     `gorm:"column:title"`
		MediaType        string     `gorm:"column:media_type"`
		ItemRole         string     `gorm:"column:item_role"`
		CoverImageURL    string     `gorm:"column:cover_image_url"`
		Status           *string    `gorm:"column:status"`
		Position         *float64   `gorm:"column:position"`
		Total            *float64   `gorm:"column:total"`
		ProgressPercent  *float64   `gorm:"column:progress_percent"`
		LastUpdateAt     time.Time  `gorm:"column:last_update_at"`
		Rating           *float64   `gorm:"column:rating"`
		RatingScale      *float64   `gorm:"column:rating_scale"`
		WorkID           uuid.UUID  `gorm:"column:work_id"`
		WorkKind         string     `gorm:"column:work_kind"`
		PrimaryTitle     string     `gorm:"column:primary_title"`
		OriginalTitle    string     `gorm:"column:original_title"`
		OriginalLanguage string     `gorm:"column:original_language"`
		FirstReleaseDate *time.Time `gorm:"column:first_release_date"`
		Description      string     `gorm:"column:description"`
	}
	result := s.db.Table("tb_media_items AS item").
		Select(`item.id, item.title, item.media_type, item.item_role, item.cover_image_url,
			progress.status, progress.position, progress.total, progress.progress_percent,
			coalesce(progress.last_update_at, item.updated_at) AS last_update_at,
			rating.rating, rating.rating_scale,
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
			Rating: row.Rating, RatingScale: row.RatingScale,
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
	if err := s.db.Table("tb_media_consumption_events").Where("media_item_id = ?", id).Order("event_at DESC NULLS LAST, created_at DESC, id DESC").Scan(&detail.Events).Error; err != nil {
		return Detail{}, false, err
	}
	return detail, true, nil
}
