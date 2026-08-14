package metricseries

import (
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
)

type MediaServiceSource struct {
	Service *media.Service
}

func (s MediaServiceSource) MediaValues(from, to time.Time) ([]MediaMetricValue, error) {
	rows, err := s.Service.CompletedMetricValues(media.PeriodFilters{From: from, To: to})
	if err != nil {
		return nil, err
	}
	values := make([]MediaMetricValue, len(rows))
	for index, row := range rows {
		values[index] = MediaMetricValue{CompletedAt: row.CompletedAt, MediaKind: row.MediaKind, Source: row.Source}
	}
	return values, nil
}
