package metricseries

import (
	"context"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
)

type DailyServiceSource struct {
	Service *daily.Service
}

func (s DailyServiceSource) MetricValues(ctx context.Context, metric string, from, to time.Time) ([]DailyMetricValue, error) {
	values, err := s.Service.MetricValues(ctx, metric, from, to)
	if err != nil {
		return nil, err
	}
	result := make([]DailyMetricValue, len(values))
	for index, value := range values {
		result[index] = DailyMetricValue{Day: value.Day, Value: value.Value, Unit: value.Unit, Source: value.Source}
	}
	return result, nil
}
