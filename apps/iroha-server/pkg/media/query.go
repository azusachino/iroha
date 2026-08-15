package media

import (
	"time"

	"github.com/google/uuid"
)

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
			progress.started_on_value,
			progress.started_on_precision,
			progress.completed_on_value,
			progress.completed_on_precision,
			coalesce(progress.last_update_at, item.updated_at) AS last_update_at,
			rating.rating,
			rating.rating_scale,
			coalesce(progress.hidden_from_continue, false) AS hidden_from_continue,
			(SELECT t.title FROM tb_media_titles t
			 WHERE t.scope_type = 'item' AND t.scope_id = item.id AND t.title_kind = 'original'
			 ORDER BY t.is_primary DESC, t.created_at ASC LIMIT 1) AS native_title`).
		Joins("LEFT JOIN tb_media_progress AS progress ON progress.media_item_id = item.id").
		Joins(`LEFT JOIN LATERAL (
			SELECT state.rating, state.rating_scale, state.observed_at AS rated_at
			FROM tb_media_state_history AS state
			WHERE state.media_item_id = item.id AND state.rating IS NOT NULL
			UNION ALL
			SELECT event.rating, event.rating_scale, event.event_at AS rated_at
			FROM tb_media_consumption_events AS event
			WHERE event.media_item_id = item.id AND event.rating IS NOT NULL
			ORDER BY rated_at DESC
			LIMIT 1
		) AS rating ON true`)
	query = query.Joins(`LEFT JOIN LATERAL (
		SELECT max(completed_at) AS completed_at
		FROM (
			SELECT completed_on_value::timestamptz AS completed_at
			FROM tb_media_progress
			WHERE media_item_id = item.id AND completed_on_precision = 'day'
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
				SELECT completed_on_value::timestamptz AS completed_at FROM tb_media_progress
				WHERE media_item_id = item.id AND completed_on_precision = 'day'
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

func (s *Service) Get(id uuid.UUID) (Detail, bool, error) {
	var row struct {
		ID                   uuid.UUID  `gorm:"column:id"`
		Title                string     `gorm:"column:title"`
		MediaType            string     `gorm:"column:media_type"`
		ItemRole             string     `gorm:"column:item_role"`
		CoverImageURL        string     `gorm:"column:cover_image_url"`
		Status               *string    `gorm:"column:status"`
		Position             *float64   `gorm:"column:position"`
		Total                *float64   `gorm:"column:total"`
		ProgressPercent      *float64   `gorm:"column:progress_percent"`
		StartedOnValue       *time.Time `gorm:"column:started_on_value"`
		StartedOnPrecision   string     `gorm:"column:started_on_precision"`
		CompletedOnValue     *time.Time `gorm:"column:completed_on_value"`
		CompletedOnPrecision string     `gorm:"column:completed_on_precision"`
		LastUpdateAt         time.Time  `gorm:"column:last_update_at"`
		Rating               *float64   `gorm:"column:rating"`
		RatingScale          *float64   `gorm:"column:rating_scale"`
		HiddenFromContinue   bool       `gorm:"column:hidden_from_continue"`
		NativeTitle          *string    `gorm:"column:native_title"`
		EpisodeCount         *int       `gorm:"column:episode_count"`
		ChapterCount         *int       `gorm:"column:chapter_count"`
		WorkID               uuid.UUID  `gorm:"column:work_id"`
		WorkKind             string     `gorm:"column:work_kind"`
		PrimaryTitle         string     `gorm:"column:primary_title"`
		OriginalTitle        string     `gorm:"column:original_title"`
		OriginalLanguage     string     `gorm:"column:original_language"`
		FirstReleaseDate     *time.Time `gorm:"column:first_release_date"`
		Description          string     `gorm:"column:description"`
	}
	result := s.db.Table("tb_media_items AS item").
		Select(`item.id, item.title, item.media_type, item.item_role, item.cover_image_url,
			item.episode_count, item.chapter_count,
			progress.status, progress.position, progress.total, progress.progress_percent,
			progress.started_on_value, progress.started_on_precision,
			progress.completed_on_value, progress.completed_on_precision,
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
			SELECT state.rating, state.rating_scale, state.observed_at AS rated_at
			FROM tb_media_state_history AS state
			WHERE state.media_item_id = item.id AND state.rating IS NOT NULL
			UNION ALL
			SELECT event.rating, event.rating_scale, event.event_at AS rated_at
			FROM tb_media_consumption_events AS event
			WHERE event.media_item_id = item.id AND event.rating IS NOT NULL
			ORDER BY rated_at DESC
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
			StartedOnValue: row.StartedOnValue, StartedOnPrecision: row.StartedOnPrecision,
			CompletedOnValue: row.CompletedOnValue, CompletedOnPrecision: row.CompletedOnPrecision,
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
	if err := s.db.Table("tb_media_consumption_events").Where("media_item_id = ?", id).Order("event_at DESC, created_at DESC, id DESC").Scan(&detail.Events).Error; err != nil {
		return Detail{}, false, err
	}
	changes, err := s.Changes(ChangeListFilters{MediaItemID: &id, Limit: 50})
	if err != nil {
		return Detail{}, false, err
	}
	detail.Updates = changes.Items
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
