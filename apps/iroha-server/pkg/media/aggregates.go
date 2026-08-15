package media

import (
	"fmt"
	"strings"
	"time"
)

func (s *Service) Aggregates(now time.Time) (Aggregates, error) {
	return s.AggregatesFiltered(now, ListFilters{})
}

func (s *Service) AggregatesFiltered(now time.Time, filters ListFilters) (Aggregates, error) {
	filterSQL, filterArgs := aggregateFilterSQL(filters)
	completionSQL := fmt.Sprintf(`
		WITH completion_dates AS (
			SELECT media_item_id, completed_on_value::timestamptz AS completed_at
			FROM tb_media_progress
			WHERE completed_on_precision = 'day'
			UNION ALL
			SELECT media_item_id, event_at AS completed_at
			FROM tb_media_consumption_events
			WHERE event_type IN ('finished', 'completed') AND event_at IS NOT NULL
		), completions AS (
			SELECT media_item_id, max(completed_at) AS completed_at
			FROM completion_dates
			GROUP BY media_item_id
		), filtered_items AS (
			SELECT item.id, item.media_type, progress.status, completions.completed_at
			FROM tb_media_items AS item
			LEFT JOIN tb_media_progress AS progress ON progress.media_item_id = item.id
			LEFT JOIN completions ON completions.media_item_id = item.id
			WHERE %s
		)
		SELECT extract(year FROM completed_at)::int AS year, count(*)::int AS count
		FROM filtered_items
		WHERE completed_at IS NOT NULL
		GROUP BY year
		ORDER BY year`, filterSQL)
	var completionRows []CompletionBucket
	if err := s.db.Raw(completionSQL, filterArgs...).Scan(&completionRows).Error; err != nil {
		return Aggregates{}, err
	}

	ratingSQL := fmt.Sprintf(`
		WITH latest_ratings AS (
			SELECT DISTINCT ON (media_item_id)
				media_item_id,
				rating / nullif(rating_scale, 0) * 10 AS normalized_rating
			FROM (
				SELECT media_item_id, rating, rating_scale, observed_at AS rated_at, created_at, id
				FROM tb_media_state_history
				WHERE rating IS NOT NULL AND rating_scale IS NOT NULL
				UNION ALL
				SELECT media_item_id, rating, rating_scale, event_at AS rated_at, created_at, id
				FROM tb_media_consumption_events
				WHERE rating IS NOT NULL AND rating_scale IS NOT NULL
			) AS ratings
			ORDER BY media_item_id, rated_at DESC, created_at DESC, id DESC
		), completion_dates AS (
			SELECT media_item_id, completed_on_value::timestamptz AS completed_at
			FROM tb_media_progress
			WHERE completed_on_precision = 'day'
			UNION ALL
			SELECT media_item_id, event_at AS completed_at
			FROM tb_media_consumption_events
			WHERE event_type IN ('finished', 'completed') AND event_at IS NOT NULL
		), completions AS (
			SELECT media_item_id, max(completed_at) AS completed_at
			FROM completion_dates
			GROUP BY media_item_id
		), filtered_items AS (
			SELECT item.id, item.media_type, progress.status, completions.completed_at
			FROM tb_media_items AS item
			LEFT JOIN tb_media_progress AS progress ON progress.media_item_id = item.id
			LEFT JOIN completions ON completions.media_item_id = item.id
			WHERE %s
		)
		SELECT round(least(greatest(normalized_rating, 0), 10)) AS score, count(*)::int AS count
		FROM latest_ratings
		JOIN filtered_items ON filtered_items.id = latest_ratings.media_item_id
		GROUP BY score
		ORDER BY score`, filterSQL)
	var scoreRows []ScoreBucket
	if err := s.db.Raw(ratingSQL, filterArgs...).Scan(&scoreRows).Error; err != nil {
		return Aggregates{}, err
	}

	typeSQL := fmt.Sprintf(`
		WITH latest_ratings AS (
			SELECT DISTINCT ON (media_item_id)
				media_item_id,
				rating / nullif(rating_scale, 0) * 10 AS normalized_rating
			FROM (
				SELECT media_item_id, rating, rating_scale, observed_at AS rated_at, created_at, id
				FROM tb_media_state_history
				WHERE rating IS NOT NULL AND rating_scale IS NOT NULL
				UNION ALL
				SELECT media_item_id, rating, rating_scale, event_at AS rated_at, created_at, id
				FROM tb_media_consumption_events
				WHERE rating IS NOT NULL AND rating_scale IS NOT NULL
			) AS ratings
			ORDER BY media_item_id, rated_at DESC, created_at DESC, id DESC
		), completion_dates AS (
			SELECT media_item_id, completed_on_value::timestamptz AS completed_at
			FROM tb_media_progress
			WHERE completed_on_precision = 'day'
			UNION ALL
			SELECT media_item_id, event_at AS completed_at
			FROM tb_media_consumption_events
			WHERE event_type IN ('finished', 'completed') AND event_at IS NOT NULL
		), completions AS (
			SELECT media_item_id, max(completed_at) AS completed_at
			FROM completion_dates
			GROUP BY media_item_id
		), filtered_items AS (
			SELECT item.id, item.media_type, progress.status, completions.completed_at
			FROM tb_media_items AS item
			LEFT JOIN tb_media_progress AS progress ON progress.media_item_id = item.id
			LEFT JOIN completions ON completions.media_item_id = item.id
			WHERE %s
		), grouped AS (
			-- Kind lives on the item (media_type: anime_season, manga, movie…);
			-- the parent work_kind is a generic 'media', so group by media_type.
			SELECT item.media_type AS type,
				count(*)::int AS count,
				avg(latest_ratings.normalized_rating) AS average_rating,
				count(latest_ratings.normalized_rating)::int AS rating_count,
				count(item.completed_at)::int AS completed_count,
				count(*) FILTER (WHERE item.status = 'completed')::int AS current_completed_count
			FROM filtered_items AS item
			LEFT JOIN latest_ratings ON latest_ratings.media_item_id = item.id
			GROUP BY type
		)
		SELECT type, count, average_rating, rating_count, completed_count,
			current_completed_count,
			sum(count) OVER ()::int AS item_count
		FROM grouped
		ORDER BY type`, filterSQL)
	var typeRows []struct {
		Type                  string  `gorm:"column:type"`
		Count                 int     `gorm:"column:count"`
		AverageRating         float64 `gorm:"column:average_rating"`
		RatingCount           int     `gorm:"column:rating_count"`
		CompletedCount        int     `gorm:"column:completed_count"`
		CurrentCompletedCount int     `gorm:"column:current_completed_count"`
		ItemCount             int     `gorm:"column:item_count"`
	}
	if err := s.db.Raw(typeSQL, filterArgs...).Scan(&typeRows).Error; err != nil {
		return Aggregates{}, err
	}

	result := Aggregates{
		CompletionsByYear: make([]CompletionBucket, 0, len(completionRows)),
		ScoreDistribution: make([]ScoreBucket, 0, len(scoreRows)),
		TypeSplit:         make([]TypeBucket, 0, len(typeRows)),
	}
	result.CompletionsByYear = append(result.CompletionsByYear, completionRows...)
	result.ScoreDistribution = append(result.ScoreDistribution, scoreRows...)
	for _, row := range typeRows {
		result.TypeSplit = append(result.TypeSplit, TypeBucket{Type: row.Type, Count: row.Count})
		result.Totals.ItemCount += row.Count
		result.Totals.CompletedCount += row.CompletedCount
		result.Totals.CurrentCompletedCount += row.CurrentCompletedCount
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

func aggregateFilterSQL(filters ListFilters) (string, []any) {
	clauses := []string{"TRUE"}
	args := make([]any, 0, 3)
	if filters.Status != "" {
		clauses = append(clauses, "progress.status = ?")
		args = append(args, filters.Status)
	}
	if filters.MediaType != "" {
		clauses = append(clauses, "item.media_type = ?")
		args = append(args, filters.MediaType)
	}
	if types, ok := familyMediaTypes[filters.Family]; ok {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(types)), ",")
		clauses = append(clauses, "item.media_type IN ("+placeholders+")")
		for _, mediaType := range types {
			args = append(args, mediaType)
		}
	}
	if filters.CompletedYear != nil {
		clauses = append(clauses, "extract(year from completions.completed_at) = ?")
		args = append(args, *filters.CompletedYear)
	}
	return strings.Join(clauses, " AND "), args
}
