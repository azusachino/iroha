package metricseries

import (
	"context"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
)

type fakeDailySource struct {
	values []DailyMetricValue
}

type fakeActivitySource struct {
	values []ActivityMetricValue
}

func (f fakeActivitySource) ActivityValues(time.Time, time.Time, string) ([]ActivityMetricValue, error) {
	return f.values, nil
}

func (f fakeDailySource) MetricValues(context.Context, string, time.Time, time.Time) ([]DailyMetricValue, error) {
	return f.values, nil
}

func TestSeriesRollsDailyStepsIntoCompleteMonthlyPoints(t *testing.T) {
	registry, err := metrics.DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	location := time.UTC
	service := NewService(registry, fakeDailySource{values: []DailyMetricValue{
		{Day: time.Date(2026, time.January, 2, 0, 0, 0, 0, location), Value: 1000, Unit: "count", Source: "watch"},
		{Day: time.Date(2026, time.January, 3, 0, 0, 0, 0, location), Value: 2000, Unit: "count", Source: "watch"},
	}})
	series, err := service.Series(context.Background(), Request{
		MetricID: "health.steps",
		From:     time.Date(2026, time.January, 1, 0, 0, 0, 0, location),
		To:       time.Date(2026, time.March, 1, 0, 0, 0, 0, location),
		Grain:    "month",
		Timezone: location,
	})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(series.Series) != 1 || len(series.Series[0].Points) != 2 {
		t.Fatalf("series = %+v", series)
	}
	points := series.Series[0].Points
	if points[0].Value == nil || *points[0].Value != 3000 || points[0].ObservedDays != 2 {
		t.Fatalf("January point = %+v", points[0])
	}
	if points[1].Value != nil || points[1].ObservedDays != 0 {
		t.Fatalf("February point = %+v, want null zero-coverage point", points[1])
	}
	if series.Series[0].Coverage.ExpectedPeriods != 2 || series.Series[0].Coverage.ObservedPeriods != 1 {
		t.Fatalf("coverage = %+v", series.Series[0].Coverage)
	}
}

func TestSeriesExpandsActivitySportDimensions(t *testing.T) {
	registry, err := metrics.DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	location := time.UTC
	distance := 5000.0
	service := NewService(registry, nil, fakeActivitySource{values: []ActivityMetricValue{
		{StartedAt: time.Date(2026, time.January, 5, 8, 0, 0, 0, location), Sport: "run", DistanceM: &distance, Source: "gpx"},
	}})
	series, err := service.Series(context.Background(), Request{
		MetricID: "movement.distance_m",
		From:     time.Date(2026, time.January, 1, 0, 0, 0, 0, location),
		To:       time.Date(2026, time.February, 1, 0, 0, 0, 0, location),
		Grain:    "month",
		Timezone: location,
		Dimensions: map[string][]string{
			"sport": {"run"},
		},
	})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(series.Series) != 1 || series.Series[0].Dimensions["sport"] != "run" {
		t.Fatalf("series dimensions = %+v", series.Series)
	}
	point := series.Series[0].Points[0]
	if point.Value == nil || *point.Value != distance || point.ObservedDays != 1 {
		t.Fatalf("distance point = %+v", point)
	}
}
