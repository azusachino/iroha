package observations

import (
	"regexp"
	"strings"

	"github.com/azusachino/iroha/apps/iroha-core/i18n"
)

var (
	matchFirstCap = regexp.MustCompile("([A-Z])([A-Z][a-z])")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// NormalizeSport maps raw provider sport names, Apple Health activity types,
// localized activity labels, and legacy aliases into the canonical sport
// vocabulary owned by i18n's activity taxonomy:
//   - "run", "running", "HKWorkoutActivityTypeRunning", "ラン", "跑步" -> "run"
//   - "walk", "walking", "HKWorkoutActivityTypeWalking", "ウォーキング", "步行" -> "walk"
//   - "ride", "cycling", "bike", "HKWorkoutActivityTypeCycling", "サイクリング", "骑行" -> "ride"
//   - "hike", "hiking", "HKWorkoutActivityTypeHiking", "ハイキング", "徒步" -> "hike"
//   - "swim", "swimming", "HKWorkoutActivityTypeSwimming", "水泳", "游泳" -> "swim"
//   - "FitnessGaming", "HKWorkoutActivityTypeFitnessGaming", "健身游戏" -> "fitness_gaming"
//   - "HIIT", "HighIntensityIntervalTraining", "高強度インターバルトレーニング" -> "hiit"
//   - "Traditional Strength Training", "機能的筋力トレーニング" -> "traditional_strength_training"
//
// The taxonomy's exact synonym table is the primary source, so a localized
// Health Auto Export label and the native export's enum resolve to the same slug
// (this is what keeps cross-source dedupe on (started_at, sport_type) intact).
// Labels outside the taxonomy are classified by keyword and, failing that,
// converted to a clean snake_case slug, so provider-specific sports such as
// "HKWorkoutActivityTypeRowing" keep a stable identity ("rowing") instead of
// collapsing into "other". An empty value stays empty: absence of a sport is not
// a classification.
func NormalizeSport(value string) string {
	label := strings.TrimSpace(value)
	if label == "" {
		return ""
	}
	label = i18n.TrimAppleHealthActivityTypePrefix(label)
	if classification := i18n.ResolveActivity(label, i18n.ActivityContext{}); classification.IsSystemLabel {
		return classification.SportType
	}
	if sport := i18n.InferSportType(label); sport != "" {
		return sport
	}
	return sportSlug(label)
}

// sportSlug converts an unrecognized sport name into a clean snake_case slug.
func sportSlug(value string) string {
	snake := matchFirstCap.ReplaceAllString(value, "${1}_${2}")
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
	case i18n.SportRun:
		return []string{i18n.SportRun, "running"}
	case i18n.SportWalk:
		return []string{i18n.SportWalk, "walking"}
	case i18n.SportRide:
		return []string{i18n.SportRide, "cycling", "bike"}
	case i18n.SportHike:
		return []string{i18n.SportHike, "hiking"}
	case i18n.SportSwim:
		return []string{i18n.SportSwim, "swimming"}
	case i18n.SportFitnessGaming:
		return []string{i18n.SportFitnessGaming, "FitnessGaming"}
	default:
		return []string{canon}
	}
}
