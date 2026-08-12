package metricseries

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
)

var (
	ErrMetricNotFound = errors.New("metric not found")
	ErrInvalidRequest = errors.New("invalid metric series request")
)

type Request struct {
	MetricID string
	From     time.Time
	To       time.Time
	Grain    string
	Timezone *time.Location
}

type DailyMetricValue struct {
	Day    time.Time
	Value  float64
	Unit   string
	Source string
}

type DailyMetricSource interface {
	MetricValues(context.Context, string, time.Time, time.Time) ([]DailyMetricValue, error)
}

type Service struct {
	registry *metrics.Registry
	daily    DailyMetricSource
}

func NewService(registry *metrics.Registry, daily DailyMetricSource) *Service {
	return &Service{registry: registry, daily: daily}
}

func (s *Service) Series(ctx context.Context, request Request) (metrics.Series, error) {
	definition, err := s.registry.Get(request.MetricID)
	if err != nil {
		return metrics.Series{}, ErrMetricNotFound
	}
	if !contains(definition.SupportedGrains, request.Grain) || request.Timezone == nil || !request.From.Before(request.To) {
		return metrics.Series{}, ErrInvalidRequest
	}
	periods, err := metrics.BuildPeriods(request.From, request.To, request.Grain, request.Timezone)
	if err != nil {
		return metrics.Series{}, ErrInvalidRequest
	}
	if request.MetricID != "health.steps" || s.daily == nil {
		return metrics.Series{}, ErrInvalidRequest
	}
	values, err := s.daily.MetricValues(ctx, "steps", request.From, request.To)
	if err != nil {
		return metrics.Series{}, err
	}
	points := rollupDailyValues(periods, values, definition.Rollup, request.Grain, request.Timezone)
	observedPeriods := 0
	sources := map[string]struct{}{}
	for _, point := range points {
		if point.ObservedDays > 0 {
			observedPeriods++
		}
	}
	for _, value := range values {
		if value.Source != "" {
			sources[value.Source] = struct{}{}
		}
	}
	sourceKinds := make([]string, 0, len(sources))
	for source := range sources {
		sourceKinds = append(sourceKinds, source)
	}
	sort.Strings(sourceKinds)
	return metrics.Series{
		Schema:    "metric-series.v1",
		MetricID:  definition.ID,
		Label:     definition.Label,
		Unit:      definition.Unit,
		ValueType: definition.ValueType,
		Period: metrics.Period{
			Grain:    request.Grain,
			From:     request.From.In(request.Timezone).Format("2006-01-02"),
			To:       request.To.In(request.Timezone).Format("2006-01-02"),
			Timezone: request.Timezone.String(),
		},
		Series: []metrics.DimensionSeries{{
			Dimensions: map[string]string{},
			Points:     points,
			Coverage: metrics.Coverage{
				ExpectedPeriods: len(points),
				ObservedPeriods: observedPeriods,
			},
			Source: metrics.Source{
				Kind:        definition.Kind,
				Method:      definition.AggregationVersion,
				SourceKinds: sourceKinds,
			},
		}},
	}, nil
}

func rollupDailyValues(periods []string, values []DailyMetricValue, rollup, grain string, location *time.Location) []metrics.Point {
	type bucket struct {
		count int
		sum   float64
	}
	buckets := make(map[string]*bucket, len(periods))
	for _, period := range periods {
		buckets[period] = &bucket{}
	}
	for _, value := range values {
		period := dateInLocation(value.Day, location).Format("2006-01-02")
		switch grain {
		case "month":
			period = dateInLocation(value.Day, location).Format("2006-01")
		case "year":
			period = dateInLocation(value.Day, location).Format("2006")
		}
		current, ok := buckets[period]
		if !ok {
			continue
		}
		current.sum += value.Value
		current.count++
	}
	points := make([]metrics.Point, 0, len(periods))
	for _, period := range periods {
		current := buckets[period]
		point := metrics.Point{Period: period, ObservedDays: current.count}
		if current.count > 0 {
			value := current.sum
			if rollup == "average" {
				value /= float64(current.count)
			}
			point.Value = &value
		}
		points = append(points, point)
	}
	return points
}

func dateInLocation(value time.Time, location *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, location)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
