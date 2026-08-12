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

type fakeExpenseSource struct {
	values []ExpenseMetricValue
}

type fakeSleepSource struct {
	values []SleepMetricValue
}

type fakeMediaSource struct {
	values []MediaMetricValue
}

func (f fakeMediaSource) MediaValues(time.Time, time.Time) ([]MediaMetricValue, error) {
	return f.values, nil
}

func (f fakeSleepSource) SleepValues(time.Time, time.Time) ([]SleepMetricValue, error) {
	return f.values, nil
}

func (f fakeExpenseSource) ExpenseValues(time.Time, time.Time) ([]ExpenseMetricValue, error) {
	return f.values, nil
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
	}}, nil, nil, nil, nil)
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
	}}, nil, nil, nil)
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

func TestExpenseSeriesRequiresCurrencyAndPreservesMinorUnits(t *testing.T) {
	registry, err := metrics.DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	location := time.UTC
	service := NewService(registry, nil, nil, fakeExpenseSource{values: []ExpenseMetricValue{
		{OccurredOn: time.Date(2026, time.January, 2, 0, 0, 0, 0, location), Currency: "JPY", Category: "food", AmountMinor: 800, Source: "local_agent"},
		{OccurredOn: time.Date(2026, time.January, 3, 0, 0, 0, 0, location), Currency: "JPY", Category: "transport", AmountMinor: 200, Source: "local_agent"},
	}}, nil, nil)
	series, err := service.Series(context.Background(), Request{
		MetricID: "expenses.amount_minor",
		From:     time.Date(2026, time.January, 1, 0, 0, 0, 0, location),
		To:       time.Date(2026, time.February, 1, 0, 0, 0, 0, location),
		Grain:    "month",
		Timezone: location,
		Dimensions: map[string][]string{
			"currency": {"JPY"},
		},
	})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	point := series.Series[0].Points[0]
	if !point.Minor || point.ValueMinor == nil || *point.ValueMinor != 1000 {
		t.Fatalf("minor-unit point = %+v", point)
	}
	if _, err := service.Series(context.Background(), Request{
		MetricID: "expenses.amount_minor", From: time.Date(2026, time.January, 1, 0, 0, 0, 0, location), To: time.Date(2026, time.February, 1, 0, 0, 0, 0, location), Grain: "month", Timezone: location,
	}); err != ErrInvalidRequest {
		t.Fatalf("missing currency error = %v, want invalid request", err)
	}
}

func TestSleepSeriesAveragesSelectedSleepKind(t *testing.T) {
	registry, err := metrics.DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	location := time.UTC
	service := NewService(registry, nil, nil, nil, fakeSleepSource{values: []SleepMetricValue{
		{WakeDate: time.Date(2026, time.January, 2, 0, 0, 0, 0, location), SleepKind: "main", AsleepS: 20000, Efficiency: 0.8, Source: "watch"},
		{WakeDate: time.Date(2026, time.January, 3, 0, 0, 0, 0, location), SleepKind: "main", AsleepS: 24000, Efficiency: 0.9, Source: "watch"},
	}}, nil)
	series, err := service.Series(context.Background(), Request{
		MetricID: "sleep.asleep_s",
		From:     time.Date(2026, time.January, 1, 0, 0, 0, 0, location),
		To:       time.Date(2026, time.February, 1, 0, 0, 0, 0, location),
		Grain:    "month", Timezone: location,
		Dimensions: map[string][]string{"sleep_kind": {"main"}},
	})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	point := series.Series[0].Points[0]
	if point.Value == nil || *point.Value != 22000 || point.ObservedDays != 2 {
		t.Fatalf("sleep point = %+v", point)
	}
}

func TestMediaSeriesCountsCompletedItemsByKind(t *testing.T) {
	registry, err := metrics.DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	location := time.UTC
	service := NewService(registry, nil, nil, nil, nil, fakeMediaSource{values: []MediaMetricValue{
		{CompletedAt: time.Date(2026, time.January, 2, 0, 0, 0, 0, location), MediaKind: "book", Source: "manual"},
		{CompletedAt: time.Date(2026, time.January, 3, 0, 0, 0, 0, location), MediaKind: "book", Source: "manual"},
	}})
	series, err := service.Series(context.Background(), Request{
		MetricID: "media.completed_count",
		From:     time.Date(2026, time.January, 1, 0, 0, 0, 0, location),
		To:       time.Date(2026, time.February, 1, 0, 0, 0, 0, location),
		Grain:    "month", Timezone: location,
		Dimensions: map[string][]string{"media_kind": {"book"}},
	})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	point := series.Series[0].Points[0]
	if point.Value == nil || *point.Value != 2 || point.ObservedDays != 2 {
		t.Fatalf("media point = %+v", point)
	}
}
