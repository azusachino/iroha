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
	MetricID   string
	From       time.Time
	To         time.Time
	Grain      string
	Timezone   *time.Location
	Dimensions map[string][]string
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

type ActivityMetricValue struct {
	StartedAt time.Time
	Sport     string
	DistanceM *float64
	DurationS *int
	Source    string
}

type ActivityMetricSource interface {
	ActivityValues(time.Time, time.Time, string) ([]ActivityMetricValue, error)
}

type ExpenseMetricValue struct {
	OccurredOn  time.Time
	Currency    string
	Category    string
	AmountMinor int64
	Source      string
}

type ExpenseMetricSource interface {
	ExpenseValues(time.Time, time.Time) ([]ExpenseMetricValue, error)
}

type Service struct {
	registry   *metrics.Registry
	daily      DailyMetricSource
	activities ActivityMetricSource
	expenses   ExpenseMetricSource
}

func NewService(registry *metrics.Registry, daily DailyMetricSource, activities ActivityMetricSource, expenses ExpenseMetricSource) *Service {
	return &Service{registry: registry, daily: daily, activities: activities, expenses: expenses}
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
	selections, err := dimensionSelections(definition, request.Dimensions)
	if err != nil {
		return metrics.Series{}, ErrInvalidRequest
	}
	series := make([]metrics.DimensionSeries, 0, len(selections))
	for _, selection := range selections {
		var dimensionSeries metrics.DimensionSeries
		switch request.MetricID {
		case "health.steps":
			if s.daily == nil || len(selection) != 0 {
				return metrics.Series{}, ErrInvalidRequest
			}
			values, err := s.daily.MetricValues(ctx, "steps", request.From, request.To)
			if err != nil {
				return metrics.Series{}, err
			}
			dimensionSeries = dailyDimensionSeries(periods, values, definition, request)
		case "movement.activity_count", "movement.distance_m", "movement.duration_s":
			if s.activities == nil {
				return metrics.Series{}, ErrInvalidRequest
			}
			values, err := s.activities.ActivityValues(request.From, request.To, request.Timezone.String())
			if err != nil {
				return metrics.Series{}, err
			}
			dimensionSeries = activityDimensionSeries(periods, values, definition, request, selection)
		case "expenses.amount_minor", "expenses.count":
			if s.expenses == nil {
				return metrics.Series{}, ErrInvalidRequest
			}
			values, err := s.expenses.ExpenseValues(request.From, request.To)
			if err != nil {
				return metrics.Series{}, err
			}
			dimensionSeries = expenseDimensionSeries(periods, values, definition, request, selection)
		default:
			return metrics.Series{}, ErrInvalidRequest
		}
		dimensionSeries.Dimensions = selection
		series = append(series, dimensionSeries)
	}
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
		Series: series,
	}, nil
}

func dailyDimensionSeries(periods []string, values []DailyMetricValue, definition metrics.Definition, request Request) metrics.DimensionSeries {
	points := rollupDailyValues(periods, values, definition.Rollup, request.Grain, request.Timezone)
	return dimensionSeries(points, valuesToSources(values, func(value DailyMetricValue) string { return value.Source }), definition)
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

func activityDimensionSeries(periods []string, values []ActivityMetricValue, definition metrics.Definition, request Request, selection map[string]string) metrics.DimensionSeries {
	type bucket struct {
		sum   float64
		count int
		days  map[string]struct{}
	}
	buckets := make(map[string]*bucket, len(periods))
	for _, period := range periods {
		buckets[period] = &bucket{days: map[string]struct{}{}}
	}
	for _, activity := range values {
		if sport, ok := selection["sport"]; ok && activity.Sport != sport {
			continue
		}
		value, ok := activityValue(definition.ID, activity)
		if !ok {
			continue
		}
		period := dateInLocation(activity.StartedAt, request.Timezone).Format("2006-01-02")
		switch request.Grain {
		case "month":
			period = dateInLocation(activity.StartedAt, request.Timezone).Format("2006-01")
		case "year":
			period = dateInLocation(activity.StartedAt, request.Timezone).Format("2006")
		}
		bucket, ok := buckets[period]
		if !ok {
			continue
		}
		bucket.sum += value
		bucket.count++
		bucket.days[dateInLocation(activity.StartedAt, request.Timezone).Format("2006-01-02")] = struct{}{}
	}
	points := make([]metrics.Point, 0, len(periods))
	for _, period := range periods {
		bucket := buckets[period]
		point := metrics.Point{Period: period, ObservedDays: len(bucket.days)}
		if bucket.count > 0 {
			value := bucket.sum
			point.Value = &value
		}
		points = append(points, point)
	}
	return dimensionSeries(points, valuesToSources(values, func(value ActivityMetricValue) string {
		if sport, ok := selection["sport"]; ok && value.Sport != sport {
			return ""
		}
		if _, ok := activityValue(definition.ID, value); !ok {
			return ""
		}
		return value.Source
	}), definition)
}

func activityValue(metricID string, activity ActivityMetricValue) (float64, bool) {
	switch metricID {
	case "movement.activity_count":
		return 1, true
	case "movement.distance_m":
		if activity.DistanceM == nil {
			return 0, false
		}
		return *activity.DistanceM, true
	case "movement.duration_s":
		if activity.DurationS == nil {
			return 0, false
		}
		return float64(*activity.DurationS), true
	default:
		return 0, false
	}
}

func expenseDimensionSeries(periods []string, values []ExpenseMetricValue, definition metrics.Definition, request Request, selection map[string]string) metrics.DimensionSeries {
	type bucket struct {
		amount int64
		count  int
		days   map[string]struct{}
	}
	buckets := make(map[string]*bucket, len(periods))
	for _, period := range periods {
		buckets[period] = &bucket{days: map[string]struct{}{}}
	}
	for _, expense := range values {
		if currency, ok := selection["currency"]; !ok || expense.Currency != currency {
			continue
		}
		if category, ok := selection["category"]; ok && expense.Category != category {
			continue
		}
		period := dateInLocation(expense.OccurredOn, request.Timezone).Format("2006-01-02")
		switch request.Grain {
		case "month":
			period = dateInLocation(expense.OccurredOn, request.Timezone).Format("2006-01")
		case "year":
			period = dateInLocation(expense.OccurredOn, request.Timezone).Format("2006")
		}
		bucket, ok := buckets[period]
		if !ok {
			continue
		}
		bucket.amount += expense.AmountMinor
		bucket.count++
		bucket.days[dateInLocation(expense.OccurredOn, request.Timezone).Format("2006-01-02")] = struct{}{}
	}
	points := make([]metrics.Point, 0, len(periods))
	for _, period := range periods {
		bucket := buckets[period]
		point := metrics.Point{Period: period, ObservedDays: len(bucket.days)}
		if definition.ID == "expenses.amount_minor" {
			point.Minor = true
			if bucket.count > 0 {
				value := bucket.amount
				point.ValueMinor = &value
			}
		} else if bucket.count > 0 {
			value := float64(bucket.count)
			point.Value = &value
		}
		points = append(points, point)
	}
	return dimensionSeries(points, valuesToSources(values, func(value ExpenseMetricValue) string {
		if currency, ok := selection["currency"]; !ok || value.Currency != currency {
			return ""
		}
		if category, ok := selection["category"]; ok && value.Category != category {
			return ""
		}
		return value.Source
	}), definition)
}

func dimensionSeries(points []metrics.Point, sourceKinds []string, definition metrics.Definition) metrics.DimensionSeries {
	observedPeriods := 0
	for _, point := range points {
		if point.ObservedDays > 0 {
			observedPeriods++
		}
	}
	return metrics.DimensionSeries{
		Points: points,
		Coverage: metrics.Coverage{
			ExpectedPeriods: len(points),
			ObservedPeriods: observedPeriods,
		},
		Source: metrics.Source{
			Kind:        definition.Kind,
			Method:      definition.AggregationVersion,
			SourceKinds: sourceKinds,
		},
	}
}

func valuesToSources[T any](values []T, source func(T) string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		if kind := source(value); kind != "" {
			seen[kind] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for kind := range seen {
		result = append(result, kind)
	}
	sort.Strings(result)
	return result
}

func dimensionSelections(definition metrics.Definition, requested map[string][]string) ([]map[string]string, error) {
	selections := []map[string]string{{}}
	for _, dimension := range definition.Dimensions {
		values := dimension.Values
		if selected, ok := requested[dimension.ID]; ok {
			values = selected
		} else if dimension.Required {
			return nil, ErrInvalidRequest
		} else if !dimension.ExpandByDefault {
			continue
		}
		if len(values) == 0 {
			return nil, ErrInvalidRequest
		}
		for _, value := range values {
			if !contains(dimension.Values, value) {
				return nil, ErrInvalidRequest
			}
		}
		next := make([]map[string]string, 0, len(selections)*len(values))
		for _, selection := range selections {
			for _, value := range values {
				copySelection := cloneSelection(selection)
				copySelection[dimension.ID] = value
				next = append(next, copySelection)
			}
		}
		selections = next
	}
	for name := range requested {
		found := false
		for _, dimension := range definition.Dimensions {
			if dimension.ID == name {
				found = true
				break
			}
		}
		if !found {
			return nil, ErrInvalidRequest
		}
	}
	if len(selections) > 32 {
		return nil, ErrInvalidRequest
	}
	return selections, nil
}

func cloneSelection(selection map[string]string) map[string]string {
	clone := make(map[string]string, len(selection)+1)
	for key, value := range selection {
		clone[key] = value
	}
	return clone
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
