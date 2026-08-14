package metrics

import "errors"

var (
	ErrInvalidDefinition = errors.New("invalid metric definition")
	ErrDuplicateMetric   = errors.New("duplicate metric definition")
	ErrMetricNotFound    = errors.New("metric not found")
)

type Dimension struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Values          []string `json:"values"`
	Required        bool     `json:"required"`
	ExpandByDefault bool     `json:"expand_by_default"`
}

type Definition struct {
	ID                 string      `json:"id"`
	Domain             string      `json:"domain"`
	Label              string      `json:"label"`
	Description        string      `json:"description"`
	Kind               string      `json:"kind"`
	ValueType          string      `json:"value_type"`
	Unit               string      `json:"unit"`
	ShortUnit          string      `json:"short_unit"`
	SupportedGrains    []string    `json:"supported_grains"`
	Dimensions         []Dimension `json:"dimensions"`
	Reducer            string      `json:"reducer"`
	Rollup             string      `json:"rollup"`
	AggregationVersion string      `json:"aggregation_version"`
	CoverageKind       string      `json:"coverage_kind"`
	SemanticColorToken string      `json:"semantic_color_token"`
	PreferredView      string      `json:"preferred_view"`
}

func (d Definition) validate() error {
	if d.ID == "" || d.Domain == "" || d.Label == "" || d.Description == "" || d.Unit == "" || d.ShortUnit == "" || d.Reducer == "" || d.Rollup == "" || d.AggregationVersion == "" || d.CoverageKind == "" || d.SemanticColorToken == "" || d.PreferredView == "" {
		return ErrInvalidDefinition
	}
	if d.Kind != "canonical" && d.Kind != "derived" {
		return ErrInvalidDefinition
	}
	if d.Rollup != "sum" && d.Rollup != "average" && d.Rollup != "count" {
		return ErrInvalidDefinition
	}
	if d.ValueType == "" || len(d.SupportedGrains) == 0 {
		return ErrInvalidDefinition
	}

	grains := make(map[string]struct{}, len(d.SupportedGrains))
	for _, grain := range d.SupportedGrains {
		if grain == "" {
			return ErrInvalidDefinition
		}
		if _, exists := grains[grain]; exists {
			return ErrInvalidDefinition
		}
		grains[grain] = struct{}{}
	}

	dimensions := make(map[string]struct{}, len(d.Dimensions))
	for _, dimension := range d.Dimensions {
		if dimension.ID == "" || dimension.Label == "" || len(dimension.Values) == 0 {
			return ErrInvalidDefinition
		}
		if _, exists := dimensions[dimension.ID]; exists {
			return ErrInvalidDefinition
		}
		dimensions[dimension.ID] = struct{}{}
		values := make(map[string]struct{}, len(dimension.Values))
		for _, value := range dimension.Values {
			if value == "" {
				return ErrInvalidDefinition
			}
			if _, exists := values[value]; exists {
				return ErrInvalidDefinition
			}
			values[value] = struct{}{}
		}
	}
	return nil
}
