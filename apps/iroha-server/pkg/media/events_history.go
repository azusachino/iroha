package media

import (
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
)

func (s *Service) CompletedMetricValues(filters PeriodFilters) ([]MetricValue, error) {
	if !filters.From.Before(filters.To) {
		return nil, errors.New("period from must be before to")
	}
	type completionRow struct {
		ItemID      uuid.UUID `gorm:"column:item_id"`
		MediaKind   string    `gorm:"column:media_kind"`
		CompletedAt time.Time `gorm:"column:completed_at"`
		Source      string    `gorm:"column:source"`
	}
	var rows []completionRow
	if err := s.db.Raw(`
		SELECT item.id AS item_id, item.media_type AS media_kind, progress.completed_on_value::timestamptz AS completed_at, progress.source_kind AS source
		FROM tb_media_progress AS progress
		JOIN tb_media_items AS item ON item.id = progress.media_item_id
		WHERE progress.completed_on_precision = 'day' AND progress.completed_on_value >= CAST(? AS date) AND progress.completed_on_value < CAST(? AS date)
		UNION ALL
		SELECT event.media_item_id AS item_id, item.media_type AS media_kind, event.event_at AS completed_at, event.source_kind AS source
		FROM tb_media_consumption_events AS event
		JOIN tb_media_items AS item ON item.id = event.media_item_id
		WHERE event.event_type IN ('finished', 'completed') AND event.event_at IS NOT NULL AND event.event_at >= ? AND event.event_at < ?
		ORDER BY completed_at ASC, item_id ASC`, filters.From, filters.To, filters.From, filters.To).Scan(&rows).Error; err != nil {
		return nil, err
	}
	latest := make(map[uuid.UUID]completionRow, len(rows))
	for _, row := range rows {
		current, ok := latest[row.ItemID]
		if !ok || row.CompletedAt.After(current.CompletedAt) {
			latest[row.ItemID] = row
		}
	}
	values := make([]MetricValue, 0, len(latest))
	for _, row := range latest {
		values = append(values, MetricValue{CompletedAt: row.CompletedAt, MediaKind: row.MediaKind, Source: row.Source})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].CompletedAt.Equal(values[j].CompletedAt) {
			return values[i].MediaKind < values[j].MediaKind
		}
		return values[i].CompletedAt.Before(values[j].CompletedAt)
	})
	return values, nil
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
		SELECT item.id, item.id AS item_id, item.title, item.media_type, progress.completed_on_value::timestamptz AS completed_at
		FROM tb_media_progress AS progress
		JOIN tb_media_items AS item ON item.id = progress.media_item_id
		WHERE progress.completed_on_precision = 'day'
			AND progress.completed_on_value >= CAST(? AS date)
			AND progress.completed_on_value < CAST(? AS date)
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
			event.event_type, event.event_at AS occurred_at,
			event.unit, event.position, event.total, event.progress_percent,
			event.rating, event.rating_scale`).
		Joins("JOIN tb_media_items AS item ON item.id = event.media_item_id")
	// The schema makes event_at non-null and rejects list_state, so the exact
	// event contract is structural rather than a best-effort filter.
	if filters.From != nil {
		query = query.Where("event.event_at >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("event.event_at < ?", *filters.To)
	}
	if filters.Cursor != nil {
		query = query.Where("(event.event_at, event.id) < (?, ?)", filters.Cursor.LastUpdateAt, filters.Cursor.ID)
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

func (s *Service) Changes(filters ChangeListFilters) (ChangePage, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = defaultPageLimit
	}
	query := s.db.Table("tb_media_state_history AS state").
		Select(`state.id, state.media_item_id, item.title, item.cover_image_url,
			(SELECT t.title FROM tb_media_titles t
			 WHERE t.scope_type = 'item' AND t.scope_id = item.id AND t.title_kind = 'original'
			 ORDER BY t.is_primary DESC, t.created_at ASC LIMIT 1) AS native_title,
			state.source_kind, state.change_kind, state.time_basis, state.observed_at,
			state.effective_at, state.effective_on_value, state.effective_on_precision,
			state.provider_recorded_at, state.status, state.unit, state.position, state.total,
			state.progress_percent, state.rating, state.rating_scale, state.note, state.repeat_count`).
		Joins("JOIN tb_media_items AS item ON item.id = state.media_item_id")
	if filters.MediaItemID != nil {
		query = query.Where("state.media_item_id = ?", *filters.MediaItemID)
		query = query.Where("(state.change_kind = 'provider_activity' OR state.time_basis IN ('source_date', 'source_fuzzy_date'))")
	}
	if filters.From != nil {
		query = query.Where("state.observed_at >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("state.observed_at < ?", *filters.To)
	}
	if filters.Cursor != nil {
		query = query.Where("(state.observed_at, state.id) < (?, ?)", filters.Cursor.LastUpdateAt, filters.Cursor.ID)
	}
	var rows []Change
	orderBy := "state.observed_at DESC, state.id DESC"
	if filters.MediaItemID != nil {
		orderBy = "coalesce(state.effective_at, state.effective_on_value::timestamptz, state.observed_at) DESC, state.id DESC"
	}
	if err := query.Order(orderBy).Limit(limit + 1).Scan(&rows).Error; err != nil {
		return ChangePage{}, err
	}
	page := ChangePage{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = &Cursor{LastUpdateAt: last.ObservedAt, ID: last.ID}
	}
	return page, nil
}

// DatedChanges returns provider-state changes that have a source-proven date
// inside the requested calendar window. It deliberately does not use
// observed_at: a connector may observe a snapshot today that describes a
// completion on an earlier source date.
func (s *Service) DatedChanges(from, to time.Time, limit int) (ChangePage, error) {
	if !from.Before(to) {
		return ChangePage{}, errors.New("dated change from must be before to")
	}
	if limit <= 0 || limit > 100 {
		limit = defaultPageLimit
	}
	fromDate := from.Format("2006-01-02")
	toDate := to.Format("2006-01-02")
	query := s.db.Table("tb_media_state_history AS state").
		Select(`state.id, state.media_item_id, item.title, item.cover_image_url,
			(SELECT t.title FROM tb_media_titles t
			 WHERE t.scope_type = 'item' AND t.scope_id = item.id AND t.title_kind = 'original'
			 ORDER BY t.is_primary DESC, t.created_at ASC LIMIT 1) AS native_title,
			state.source_kind, state.change_kind, state.time_basis, state.observed_at,
			state.effective_at, state.effective_on_value, state.effective_on_precision,
			state.provider_recorded_at, state.status, state.unit, state.position, state.total,
			state.progress_percent, state.rating, state.rating_scale, state.note, state.repeat_count`).
		Joins("JOIN tb_media_items AS item ON item.id = state.media_item_id").
		Where(`(
			(state.time_basis = 'source_date'
				AND state.effective_on_precision = 'day'
				AND state.effective_on_value >= CAST(? AS date)
				AND state.effective_on_value < CAST(? AS date))
			OR (state.time_basis = 'provider_activity'
				AND state.effective_at >= ?
				AND state.effective_at < ?)
		)`, fromDate, toDate, from, to)
	var rows []Change
	if err := query.Order("coalesce(state.effective_at, state.effective_on_value::timestamptz) DESC, state.id DESC").Limit(limit + 1).Scan(&rows).Error; err != nil {
		return ChangePage{}, err
	}
	page := ChangePage{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = &Cursor{LastUpdateAt: last.ObservedAt, ID: last.ID}
	}
	return page, nil
}

func floatPtrEqual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
