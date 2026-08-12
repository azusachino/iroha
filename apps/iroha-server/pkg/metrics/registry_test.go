package metrics

import (
	"errors"
	"testing"
)

func TestDefaultRegistryIsValidAndDeterministic(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	definitions := registry.List()
	if len(definitions) != 16 {
		t.Fatalf("definition count = %d, want 16", len(definitions))
	}
	for index := 1; index < len(definitions); index++ {
		if definitions[index-1].ID >= definitions[index].ID {
			t.Fatalf("definitions are not sorted: %q before %q", definitions[index-1].ID, definitions[index].ID)
		}
	}

	definition, err := registry.Get("expenses.amount_minor")
	if err != nil {
		t.Fatalf("get expense metric: %v", err)
	}
	if definition.ValueType != "money" || len(definition.Dimensions) != 2 {
		t.Fatalf("expense definition = %+v", definition)
	}
}

func TestRegistryRejectsDuplicateAndInvalidDefinitions(t *testing.T) {
	valid := Definition{
		ID: "health.steps", Domain: "daily", Label: "Steps", Description: "Daily steps.", Kind: "canonical",
		ValueType: "count", Unit: "count", ShortUnit: "steps", SupportedGrains: []string{"day"},
		Reducer: "source_priority", AggregationVersion: "health.steps.v1", CoverageKind: "observed_days",
		SemanticColorToken: "health", PreferredView: "line",
	}
	if _, err := NewRegistry([]Definition{valid, valid}); !errors.Is(err, ErrDuplicateMetric) {
		t.Fatalf("duplicate error = %v, want %v", err, ErrDuplicateMetric)
	}
	invalid := valid
	invalid.SupportedGrains = []string{"day", "day"}
	if _, err := NewRegistry([]Definition{invalid}); !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("invalid error = %v, want %v", err, ErrInvalidDefinition)
	}
}

func TestRegistryReturnsCopies(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	definitions := registry.List()
	definitions[0].SupportedGrains[0] = "corrupted"
	definitions[0].Dimensions = append(definitions[0].Dimensions, Dimension{ID: "corrupted"})

	again, err := registry.Get(definitions[0].ID)
	if err != nil {
		t.Fatalf("get copied definition: %v", err)
	}
	if again.SupportedGrains[0] == "corrupted" {
		t.Fatal("registry grain slice was mutated through List")
	}
	for _, dimension := range again.Dimensions {
		if dimension.ID == "corrupted" {
			t.Fatal("registry dimension slice was mutated through List")
		}
	}
}

func TestRegistryReportsMissingMetric(t *testing.T) {
	registry, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("default registry: %v", err)
	}
	if _, err := registry.Get("missing.metric"); !errors.Is(err, ErrMetricNotFound) {
		t.Fatalf("missing error = %v, want %v", err, ErrMetricNotFound)
	}
}
