package reports

import "github.com/azusachino/iroha/apps/iroha-server/pkg/daily"

func DailyHealth(service *daily.Service, period Period) (*DailyHealthData, error) {
	result, err := service.PeriodReport(daily.PeriodFilters{From: period.FromDate, To: period.ToDateExclusive})
	if err != nil {
		return nil, err
	}
	if result.ObservedDays == 0 {
		return nil, nil
	}
	metrics := make([]MetricAverage, 0, len(result.MetricAverages))
	for _, metric := range result.MetricAverages {
		metrics = append(metrics, MetricAverage{Metric: metric.Metric, Value: metric.Value, Unit: metric.Unit, ObservedDays: metric.ObservedDays})
	}
	return &DailyHealthData{ObservedDays: result.ObservedDays, MetricAverages: metrics}, nil
}
