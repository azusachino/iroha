package metrics

import "encoding/json"

type Point struct {
	Period       string   `json:"period"`
	Value        *float64 `json:"value,omitempty"`
	ValueMinor   *int64   `json:"value_minor,omitempty"`
	ObservedDays int      `json:"observed_days"`
}

func (p Point) MarshalJSON() ([]byte, error) {
	value := any(nil)
	if p.Value != nil {
		value = *p.Value
	}
	payload := map[string]any{
		"period":        p.Period,
		"value":         value,
		"observed_days": p.ObservedDays,
	}
	if p.ValueMinor != nil {
		delete(payload, "value")
		payload["value_minor"] = *p.ValueMinor
	}
	return json.Marshal(payload)
}

type Coverage struct {
	ExpectedPeriods int `json:"expected_periods"`
	ObservedPeriods int `json:"observed_periods"`
}

type Source struct {
	Kind        string   `json:"kind"`
	Method      string   `json:"method"`
	SourceKinds []string `json:"source_kinds"`
}

type DimensionSeries struct {
	Dimensions map[string]string `json:"dimensions"`
	Points     []Point           `json:"points"`
	Coverage   Coverage          `json:"coverage"`
	Source     Source            `json:"source"`
}

type Series struct {
	Schema    string            `json:"schema"`
	MetricID  string            `json:"metric_id"`
	Label     string            `json:"label"`
	Unit      string            `json:"unit"`
	ValueType string            `json:"value_type"`
	Period    Period            `json:"period"`
	Series    []DimensionSeries `json:"series"`
}
