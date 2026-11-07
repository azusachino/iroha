package observations

import (
	"regexp"
	"strings"
)

var (
	matchFirstCap = regexp.MustCompile("([A-Z])([A-Z][a-z])")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// NormalizeSport maps raw provider sport names, Apple Health activity types, and legacy aliases
// into a single canonical sport vocabulary:
//   - "run", "running", "HKWorkoutActivityTypeRunning" -> "run"
//   - "walk", "walking", "HKWorkoutActivityTypeWalking" -> "walk"
//   - "ride", "cycling", "bike", "HKWorkoutActivityTypeCycling" -> "ride"
//   - "hike", "hiking", "HKWorkoutActivityTypeHiking" -> "hike"
//   - "swim", "swimming", "HKWorkoutActivityTypeSwimming" -> "swim"
//   - "fitness_gaming", "FitnessGaming", "HKWorkoutActivityTypeFitnessGaming" -> "fitness_gaming"
//
// Other sports are converted to clean snake_case slugs.
func NormalizeSport(value string) string {
	val := strings.TrimSpace(value)
	val = strings.TrimPrefix(val, "HKWorkoutActivityType")
	normalized := strings.ToLower(val)

	switch {
	case strings.Contains(normalized, "fitnessgaming") || strings.Contains(normalized, "fitness_gaming") || strings.Contains(normalized, "fitness gaming"):
		return "fitness_gaming"
	case strings.Contains(normalized, "hike") || strings.Contains(normalized, "hiking") || strings.Contains(val, "ハイキング"):
		return "hike"
	case strings.Contains(normalized, "run") || strings.Contains(normalized, "running") || strings.Contains(val, "ラン"):
		return "run"
	case strings.Contains(normalized, "walk") || strings.Contains(normalized, "walking") || strings.Contains(val, "ウォーキング") || strings.Contains(val, "徒歩"):
		return "walk"
	case strings.Contains(normalized, "cycle") || strings.Contains(normalized, "cycling") || strings.Contains(normalized, "bike") || strings.Contains(normalized, "biking") || normalized == "ride" || strings.Contains(val, "サイクリング") || strings.Contains(val, "自転車"):
		return "ride"
	case strings.Contains(normalized, "swim") || strings.Contains(normalized, "swimming") || strings.Contains(val, "スイム") || strings.Contains(val, "水泳"):
		return "swim"
	case strings.Contains(val, "筋トレ") || strings.Contains(val, "筋力トレーニング"):
		return "traditional_strength_training"
	case strings.Contains(val, "ヨガ"):
		return "yoga"
	}

	// For PascalCase or spaced strings, convert to snake_case
	snake := matchFirstCap.ReplaceAllString(val, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	snake = strings.ToLower(snake)
	snake = strings.ReplaceAll(snake, " ", "_")
	snake = strings.ReplaceAll(snake, "-", "_")

	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, snake)
}

// SportAliases returns all known legacy aliases for a canonical sport type,
// allowing queries with filters to match both canonical and legacy stored rows.
func SportAliases(sport string) []string {
	canon := NormalizeSport(sport)
	switch canon {
	case "run":
		return []string{"run", "running"}
	case "walk":
		return []string{"walk", "walking"}
	case "ride":
		return []string{"ride", "cycling", "bike"}
	case "hike":
		return []string{"hike", "hiking"}
	case "swim":
		return []string{"swim", "swimming"}
	case "fitness_gaming":
		return []string{"fitness_gaming", "FitnessGaming"}
	default:
		return []string{canon}
	}
}
