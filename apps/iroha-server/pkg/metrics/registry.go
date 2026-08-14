package metrics

import (
	"sort"
)

type Registry struct {
	byID map[string]Definition
	ids  []string
}

func NewRegistry(definitions []Definition) (*Registry, error) {
	byID := make(map[string]Definition, len(definitions))
	for _, definition := range definitions {
		if err := definition.validate(); err != nil {
			return nil, err
		}
		if _, exists := byID[definition.ID]; exists {
			return nil, ErrDuplicateMetric
		}
		byID[definition.ID] = cloneDefinition(definition)
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return &Registry{byID: byID, ids: ids}, nil
}

func (r *Registry) List() []Definition {
	definitions := make([]Definition, 0, len(r.ids))
	for _, id := range r.ids {
		definitions = append(definitions, cloneDefinition(r.byID[id]))
	}
	return definitions
}

func (r *Registry) Get(id string) (Definition, error) {
	definition, ok := r.byID[id]
	if !ok {
		return Definition{}, ErrMetricNotFound
	}
	return cloneDefinition(definition), nil
}

func cloneDefinition(definition Definition) Definition {
	definition.SupportedGrains = append([]string(nil), definition.SupportedGrains...)
	definition.Dimensions = append([]Dimension(nil), definition.Dimensions...)
	for index := range definition.Dimensions {
		definition.Dimensions[index].Values = append([]string(nil), definition.Dimensions[index].Values...)
	}
	return definition
}

func DefaultRegistry() (*Registry, error) {
	return NewRegistry([]Definition{
		{ID: "expenses.amount_minor", Domain: "expenses", Label: "Expenses", Description: "Active expense total in the selected currency and optional category.", Kind: "derived", ValueType: "money", Unit: "minor currency unit", ShortUnit: "minor", SupportedGrains: []string{"day", "month", "year"}, Dimensions: []Dimension{{ID: "currency", Label: "Currency", Values: []string{"EUR", "GBP", "JPY", "USD"}, Required: true}, {ID: "category", Label: "Category", Values: []string{"entertainment", "food", "groceries", "health", "housing", "other", "shopping", "subscriptions", "transport", "utilities", "work"}}}, Reducer: "sum_active", Rollup: "sum", AggregationVersion: "expenses.sum_active.v1", CoverageKind: "observed_days", SemanticColorToken: "expense", PreferredView: "bar"},
		{ID: "expenses.count", Domain: "expenses", Label: "Expenses", Description: "Count of active expenses in the selected currency and optional category.", Kind: "derived", ValueType: "count", Unit: "count", ShortUnit: "items", SupportedGrains: []string{"day", "month", "year"}, Dimensions: []Dimension{{ID: "currency", Label: "Currency", Values: []string{"EUR", "GBP", "JPY", "USD"}, Required: true}, {ID: "category", Label: "Category", Values: []string{"entertainment", "food", "groceries", "health", "housing", "other", "shopping", "subscriptions", "transport", "utilities", "work"}}}, Reducer: "count_active", Rollup: "count", AggregationVersion: "expenses.count_active.v1", CoverageKind: "observed_days", SemanticColorToken: "expense", PreferredView: "bar"},
		{ID: "health.distance_km", Domain: "daily", Label: "Distance", Description: "Source-selected daily distance.", Kind: "canonical", ValueType: "number", Unit: "km", ShortUnit: "km", SupportedGrains: []string{"day", "month", "year"}, Reducer: "source_priority", Rollup: "sum", AggregationVersion: "health.distance_km.v1", CoverageKind: "observed_days", SemanticColorToken: "health", PreferredView: "line"},
		{ID: "health.exercise_min", Domain: "daily", Label: "Exercise", Description: "Canonical daily exercise minutes.", Kind: "canonical", ValueType: "duration", Unit: "min", ShortUnit: "min", SupportedGrains: []string{"day", "month", "year"}, Reducer: "source_summary", Rollup: "sum", AggregationVersion: "health.exercise_min.v1", CoverageKind: "observed_days", SemanticColorToken: "health", PreferredView: "line"},
		{ID: "health.flights", Domain: "daily", Label: "Flights", Description: "Source-selected daily flights climbed.", Kind: "canonical", ValueType: "count", Unit: "count", ShortUnit: "flights", SupportedGrains: []string{"day", "month", "year"}, Reducer: "source_priority", Rollup: "sum", AggregationVersion: "health.flights.v1", CoverageKind: "observed_days", SemanticColorToken: "health", PreferredView: "line"},
		{ID: "health.hrv_sdnn", Domain: "daily", Label: "HRV", Description: "Source-selected daily heart-rate variability.", Kind: "canonical", ValueType: "number", Unit: "ms", ShortUnit: "ms", SupportedGrains: []string{"day", "month", "year"}, Reducer: "source_priority", Rollup: "average", AggregationVersion: "health.hrv_sdnn.v1", CoverageKind: "observed_days", SemanticColorToken: "health", PreferredView: "line"},
		{ID: "health.move_kcal", Domain: "daily", Label: "Move", Description: "Canonical daily move energy.", Kind: "canonical", ValueType: "number", Unit: "kcal", ShortUnit: "kcal", SupportedGrains: []string{"day", "month", "year"}, Reducer: "source_summary", Rollup: "sum", AggregationVersion: "health.move_kcal.v1", CoverageKind: "observed_days", SemanticColorToken: "health", PreferredView: "line"},
		{ID: "health.resting_hr", Domain: "daily", Label: "Resting heart rate", Description: "Source-selected daily resting heart rate.", Kind: "canonical", ValueType: "number", Unit: "bpm", ShortUnit: "bpm", SupportedGrains: []string{"day", "month", "year"}, Reducer: "source_priority", Rollup: "average", AggregationVersion: "health.resting_hr.v1", CoverageKind: "observed_days", SemanticColorToken: "health", PreferredView: "line"},
		{ID: "health.stand_hours", Domain: "daily", Label: "Stand", Description: "Canonical daily stand hours.", Kind: "canonical", ValueType: "duration", Unit: "h", ShortUnit: "h", SupportedGrains: []string{"day", "month", "year"}, Reducer: "source_summary", Rollup: "sum", AggregationVersion: "health.stand_hours.v1", CoverageKind: "observed_days", SemanticColorToken: "health", PreferredView: "line"},
		{ID: "health.steps", Domain: "daily", Label: "Steps", Description: "Source-selected daily step count.", Kind: "canonical", ValueType: "count", Unit: "count", ShortUnit: "steps", SupportedGrains: []string{"day", "month", "year"}, Reducer: "source_priority", Rollup: "sum", AggregationVersion: "health.steps.v1", CoverageKind: "observed_days", SemanticColorToken: "health", PreferredView: "line"},
		{ID: "media.completed_count", Domain: "media", Label: "Completed media", Description: "Count of completed media items.", Kind: "derived", ValueType: "count", Unit: "count", ShortUnit: "items", SupportedGrains: []string{"month"}, Dimensions: []Dimension{{ID: "media_kind", Label: "Media kind", Values: []string{"anime", "book", "game", "manga"}, ExpandByDefault: true}}, Reducer: "count_completed", Rollup: "count", AggregationVersion: "media.count_completed.v1", CoverageKind: "observed_periods", SemanticColorToken: "media", PreferredView: "bar"},
		{ID: "movement.activity_count", Domain: "movement", Label: "Activities", Description: "Count of canonical activity sessions.", Kind: "derived", ValueType: "count", Unit: "count", ShortUnit: "activities", SupportedGrains: []string{"day", "month", "year"}, Dimensions: []Dimension{{ID: "sport", Label: "Sport", Values: []string{"hike", "ride", "run", "swim", "walk", "other"}, ExpandByDefault: true}}, Reducer: "count_activity", Rollup: "count", AggregationVersion: "movement.count_activity.v1", CoverageKind: "observed_periods", SemanticColorToken: "movement", PreferredView: "bar"},
		{ID: "movement.distance_m", Domain: "movement", Label: "Distance", Description: "Sum of non-null canonical activity distances.", Kind: "derived", ValueType: "number", Unit: "m", ShortUnit: "m", SupportedGrains: []string{"day", "month", "year"}, Dimensions: []Dimension{{ID: "sport", Label: "Sport", Values: []string{"hike", "ride", "run", "swim", "walk", "other"}, ExpandByDefault: true}}, Reducer: "sum_distance", Rollup: "sum", AggregationVersion: "movement.sum_distance.v1", CoverageKind: "observed_periods", SemanticColorToken: "movement", PreferredView: "bar"},
		{ID: "movement.duration_s", Domain: "movement", Label: "Duration", Description: "Sum of non-null canonical activity durations.", Kind: "derived", ValueType: "duration", Unit: "s", ShortUnit: "s", SupportedGrains: []string{"day", "month", "year"}, Dimensions: []Dimension{{ID: "sport", Label: "Sport", Values: []string{"hike", "ride", "run", "swim", "walk", "other"}, ExpandByDefault: true}}, Reducer: "sum_duration", Rollup: "sum", AggregationVersion: "movement.sum_duration.v1", CoverageKind: "observed_periods", SemanticColorToken: "movement", PreferredView: "bar"},
		{ID: "sleep.asleep_s", Domain: "sleep", Label: "Asleep time", Description: "Average asleep duration by sleep kind.", Kind: "derived", ValueType: "duration", Unit: "s", ShortUnit: "s", SupportedGrains: []string{"day", "month", "year"}, Dimensions: []Dimension{{ID: "sleep_kind", Label: "Sleep kind", Values: []string{"main", "nap"}, ExpandByDefault: true}}, Reducer: "average_asleep", Rollup: "average", AggregationVersion: "sleep.average_asleep.v1", CoverageKind: "observed_periods", SemanticColorToken: "sleep", PreferredView: "line"},
		{ID: "sleep.efficiency", Domain: "sleep", Label: "Sleep efficiency", Description: "Average main-sleep efficiency.", Kind: "derived", ValueType: "percentage", Unit: "%", ShortUnit: "%", SupportedGrains: []string{"day", "month", "year"}, Dimensions: []Dimension{{ID: "sleep_kind", Label: "Sleep kind", Values: []string{"main"}, ExpandByDefault: true}}, Reducer: "average_efficiency", Rollup: "average", AggregationVersion: "sleep.average_efficiency.v1", CoverageKind: "observed_periods", SemanticColorToken: "sleep", PreferredView: "line"},
	})
}
