package i18n

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Canonical activity keys identify one catalog entry. A key is stable domain
// vocabulary: it is never localized and never derived from a provider label.
const (
	ActivityKeyRunOutdoor       = "run.outdoor"
	ActivityKeyRunIndoor        = "run.indoor"
	ActivityKeyRunFamily        = "run.family"
	ActivityKeyWalkOutdoor      = "walk.outdoor"
	ActivityKeyWalkIndoor       = "walk.indoor"
	ActivityKeyWalkFamily       = "walk.family"
	ActivityKeyRideOutdoor      = "ride.outdoor"
	ActivityKeyRideIndoor       = "ride.indoor"
	ActivityKeyRideFamily       = "ride.family"
	ActivityKeySwimPool         = "swim.pool"
	ActivityKeySwimOpenWater    = "swim.open_water"
	ActivityKeyHike             = "hike"
	ActivityKeyStrengthTraining = "traditional_strength_training"
	ActivityKeyHIIT             = "hiit"
	ActivityKeyYoga             = "yoga"
	ActivityKeyFitnessGaming    = "fitness_gaming"
	ActivityKeyWorkoutGeneric   = "workout.generic"
	ActivityKeyCustom           = "custom"
)

// Canonical sport slugs. These are the only values activity persistence and
// dedupe may treat as sport identities; observations.NormalizeSport derives
// from this vocabulary instead of keeping a parallel list.
const (
	SportRun                         = "run"
	SportWalk                        = "walk"
	SportRide                        = "ride"
	SportSwim                        = "swim"
	SportHike                        = "hike"
	SportTraditionalStrengthTraining = "traditional_strength_training"
	SportHIIT                        = "hiit"
	SportYoga                        = "yoga"
	SportFitnessGaming               = "fitness_gaming"
	SportOther                       = "other"
)

// Canonical activity titles. System taxonomy titles are universal English:
// they are stored as-is and localized only at render time.
const (
	ActivityTitleOutdoorRun       = "Outdoor Run"
	ActivityTitleIndoorRun        = "Indoor Run"
	ActivityTitleRun              = "Run"
	ActivityTitleOutdoorWalk      = "Outdoor Walk"
	ActivityTitleIndoorWalk       = "Indoor Walk"
	ActivityTitleWalk             = "Walk"
	ActivityTitleOutdoorCycling   = "Outdoor Cycling"
	ActivityTitleIndoorCycling    = "Indoor Cycling"
	ActivityTitleRide             = "Ride"
	ActivityTitlePoolSwim         = "Pool Swim"
	ActivityTitleOpenWaterSwim    = "Open Water Swim"
	ActivityTitleHiking           = "Hiking"
	ActivityTitleStrengthTraining = "Functional Strength Training"
	ActivityTitleHIIT             = "High Intensity Interval Training"
	ActivityTitleYoga             = "Yoga"
	ActivityTitleFitnessGaming    = "Fitness Gaming"
	ActivityTitleWorkout          = "Workout"
)

// Activity locations describe where an entry sits in the taxonomy. A family
// entry is the honest, location-unknown canonical for one sport; its indoor and
// outdoor siblings are reachable only through ActivityContext evidence.
const (
	ActivityLocationFamily    = "family"
	ActivityLocationIndoor    = "indoor"
	ActivityLocationOutdoor   = "outdoor"
	ActivityLocationPool      = "pool"
	ActivityLocationOpenWater = "open_water"
	ActivityLocationGeneric   = "generic"
)

// AppleHealthActivityTypePrefix is the HealthKit enum prefix carried by native
// Apple Health export titles (for example "HKWorkoutActivityTypeRunning").
const AppleHealthActivityTypePrefix = "HKWorkoutActivityType"

// TrimAppleHealthActivityTypePrefix removes a leading HealthKit enum prefix and
// returns the remaining label verbatim. The comparison is case-insensitive so a
// provider that lower-cases the enum still yields a readable label.
func TrimAppleHealthActivityTypePrefix(label string) string {
	if len(label) >= len(AppleHealthActivityTypePrefix) &&
		strings.EqualFold(label[:len(AppleHealthActivityTypePrefix)], AppleHealthActivityTypePrefix) {
		return label[len(AppleHealthActivityTypePrefix):]
	}
	return label
}

// supportedLanguageOrder is the deterministic order synonyms are indexed and
// documented in.
var supportedLanguageOrder = []Language{LangEN, LangZH, LangJA}

// ActivityContext carries the workout evidence that disambiguates a bare
// activity label. Location is never present in HealthKit's activity enum: it is
// carried by HKMetadataKeyIndoorWorkout (Indoor) and GPS route presence
// (HasRoute), so both stay pointers/booleans. SourceKind and Provider describe
// where the evidence came from; DistanceM is available for callers that need
// distance-aware presentation.
type ActivityContext struct {
	Provider   string
	SourceKind string
	Indoor     *bool
	HasRoute   bool
	DistanceM  *float64
}

// ActivityClassification is the resolved, evidence-backed reading of one raw
// activity label. CanonicalTitle is universal English for system labels and the
// caller's verbatim text for custom titles.
type ActivityClassification struct {
	CanonicalKey   string
	SportType      string
	IsIndoor       bool
	CanonicalTitle string
	IsSystemLabel  bool
	MatchedLang    Language
}

// ActivityEntry is one catalog row: a canonical key, its canonical sport and
// title, its taxonomy location, and every known provider label per language.
type ActivityEntry struct {
	CanonicalKey   string
	SportType      string
	Location       string
	IsIndoor       bool
	CanonicalTitle string
	Synonyms       map[Language][]string
}

// activityEntries is the canonical catalog. Every standard category lists the
// Health Auto Export (localized by the device's iOS locale) and native Apple
// Health (HKWorkoutActivityType enum) labels that must resolve to the same
// entry, so cross-source dedupe on (started_at, sport_type) does not split.
//
// "Ride" is listed at family level even though Strava uses it for every
// cycling workout including virtual rides: promoting it to "Outdoor Cycling"
// would fabricate location the way the ADR rejects for bare "Running".
var activityEntries = []ActivityEntry{
	{
		CanonicalKey:   ActivityKeyRunOutdoor,
		SportType:      SportRun,
		Location:       ActivityLocationOutdoor,
		CanonicalTitle: ActivityTitleOutdoorRun,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleOutdoorRun, "Outdoor Running"},
			LangJA: {"屋外ラン", "屋外 ラン", "屋外ランニング"},
			LangZH: {"户外跑步", "户外跑", "路跑"},
		},
	},
	{
		CanonicalKey:   ActivityKeyRunIndoor,
		SportType:      SportRun,
		Location:       ActivityLocationIndoor,
		IsIndoor:       true,
		CanonicalTitle: ActivityTitleIndoorRun,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleIndoorRun, "Treadmill", "Treadmill Run"},
			LangJA: {"屋内ラン", "屋内 ラン", "トレッドミル", "室内ラン"},
			LangZH: {"室内跑步", "跑步机", "室内跑"},
		},
	},
	{
		CanonicalKey:   ActivityKeyRunFamily,
		SportType:      SportRun,
		Location:       ActivityLocationFamily,
		CanonicalTitle: ActivityTitleRun,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleRun, "Running", "HKWorkoutActivityTypeRunning"},
			LangJA: {"ラン"},
			LangZH: {"跑步"},
		},
	},
	{
		CanonicalKey:   ActivityKeyWalkOutdoor,
		SportType:      SportWalk,
		Location:       ActivityLocationOutdoor,
		CanonicalTitle: ActivityTitleOutdoorWalk,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleOutdoorWalk, "Outdoor Walking"},
			LangJA: {"屋外ウォーキング", "屋外 ウォーキング"},
			LangZH: {"户外步行", "户外散步", "健走"},
		},
	},
	{
		CanonicalKey:   ActivityKeyWalkIndoor,
		SportType:      SportWalk,
		Location:       ActivityLocationIndoor,
		IsIndoor:       true,
		CanonicalTitle: ActivityTitleIndoorWalk,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleIndoorWalk},
			LangJA: {"屋内ウォーキング", "屋内 ウォーキング"},
			LangZH: {"室内步行", "室内散步"},
		},
	},
	{
		CanonicalKey:   ActivityKeyWalkFamily,
		SportType:      SportWalk,
		Location:       ActivityLocationFamily,
		CanonicalTitle: ActivityTitleWalk,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleWalk, "Walking", "HKWorkoutActivityTypeWalking"},
			LangJA: {"ウォーキング", "徒歩"},
			LangZH: {"步行", "散步"},
		},
	},
	{
		CanonicalKey:   ActivityKeyRideOutdoor,
		SportType:      SportRide,
		Location:       ActivityLocationOutdoor,
		CanonicalTitle: ActivityTitleOutdoorCycling,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleOutdoorCycling, "Cycling", "Biking", "HKWorkoutActivityTypeCycling"},
			LangJA: {"屋外サイクリング", "屋外 サイクリング", "サイクリング", "自転車"},
			LangZH: {"户外骑行", "骑行", "自行车"},
		},
	},
	{
		CanonicalKey:   ActivityKeyRideIndoor,
		SportType:      SportRide,
		Location:       ActivityLocationIndoor,
		IsIndoor:       true,
		CanonicalTitle: ActivityTitleIndoorCycling,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleIndoorCycling, "Spinning"},
			LangJA: {"屋内サイクリング", "屋内 サイクリング", "スピンバイク"},
			LangZH: {"室内骑行", "室内单车", "动感单车"},
		},
	},
	{
		CanonicalKey:   ActivityKeyRideFamily,
		SportType:      SportRide,
		Location:       ActivityLocationFamily,
		CanonicalTitle: ActivityTitleRide,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleRide},
		},
	},
	{
		CanonicalKey:   ActivityKeySwimPool,
		SportType:      SportSwim,
		Location:       ActivityLocationPool,
		IsIndoor:       true,
		CanonicalTitle: ActivityTitlePoolSwim,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitlePoolSwim, "Swimming", "Pool Swimming", "HKWorkoutActivityTypeSwimming"},
			LangJA: {"プールスイミング", "スイム", "水泳"},
			LangZH: {"泳池游泳", "游泳"},
		},
	},
	{
		CanonicalKey:   ActivityKeySwimOpenWater,
		SportType:      SportSwim,
		Location:       ActivityLocationOpenWater,
		CanonicalTitle: ActivityTitleOpenWaterSwim,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleOpenWaterSwim},
			LangJA: {"オープンウォータースイミング"},
			LangZH: {"公开水域游泳"},
		},
	},
	{
		CanonicalKey:   ActivityKeyHike,
		SportType:      SportHike,
		Location:       ActivityLocationOutdoor,
		CanonicalTitle: ActivityTitleHiking,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleHiking, "Hike", "HKWorkoutActivityTypeHiking"},
			LangJA: {"ハイキング", "登山"},
			LangZH: {"徒步", "远足", "爬山"},
		},
	},
	{
		CanonicalKey:   ActivityKeyStrengthTraining,
		SportType:      SportTraditionalStrengthTraining,
		Location:       ActivityLocationIndoor,
		IsIndoor:       true,
		CanonicalTitle: ActivityTitleStrengthTraining,
		Synonyms: map[Language][]string{
			LangEN: {
				ActivityTitleStrengthTraining,
				"Traditional Strength Training",
				"Strength Training",
				"HKWorkoutActivityTypeTraditionalStrengthTraining",
				"HKWorkoutActivityTypeFunctionalStrengthTraining",
			},
			LangJA: {"機能的筋力トレーニング", "筋トレ", "筋力トレーニング"},
			LangZH: {"功能性力量训练", "力量训练", "健身"},
		},
	},
	{
		CanonicalKey:   ActivityKeyHIIT,
		SportType:      SportHIIT,
		Location:       ActivityLocationIndoor,
		IsIndoor:       true,
		CanonicalTitle: ActivityTitleHIIT,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleHIIT, "HIIT", "HKWorkoutActivityTypeHighIntensityIntervalTraining"},
			LangJA: {"高強度インターバルトレーニング", "HIIT"},
			LangZH: {"高强度间歇训练", "HIIT"},
		},
	},
	{
		CanonicalKey:   ActivityKeyYoga,
		SportType:      SportYoga,
		Location:       ActivityLocationIndoor,
		IsIndoor:       true,
		CanonicalTitle: ActivityTitleYoga,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleYoga, "HKWorkoutActivityTypeYoga"},
			LangJA: {"ヨガ"},
			LangZH: {"瑜伽"},
		},
	},
	{
		CanonicalKey:   ActivityKeyFitnessGaming,
		SportType:      SportFitnessGaming,
		Location:       ActivityLocationIndoor,
		IsIndoor:       true,
		CanonicalTitle: ActivityTitleFitnessGaming,
		Synonyms: map[Language][]string{
			LangEN: {ActivityTitleFitnessGaming, "FitnessGaming", "HKWorkoutActivityTypeFitnessGaming"},
			LangJA: {"フィットネスゲーム"},
			LangZH: {"健身游戏"},
		},
	},
	{
		CanonicalKey:   ActivityKeyWorkoutGeneric,
		SportType:      SportOther,
		Location:       ActivityLocationGeneric,
		CanonicalTitle: ActivityTitleWorkout,
		Synonyms: map[Language][]string{
			LangEN: {"", ActivityTitleWorkout, "Other", "HKWorkoutActivityTypeOther"},
		},
	},
}

// synonymMatch records which catalog entry and which declared language a
// normalized label matched.
type synonymMatch struct {
	entryIndex int
	lang       Language
}

var (
	activityEntryIndex   = buildActivityEntryIndex()
	activitySynonymIndex = buildActivitySynonymIndex()
)

// ActivityEntries returns a copy of the canonical catalog in declaration order.
// Callers must treat the returned entries as read-only.
func ActivityEntries() []ActivityEntry {
	entries := make([]ActivityEntry, len(activityEntries))
	copy(entries, activityEntries)
	return entries
}

// ResolveActivity reads a raw activity label in the context of the workout it
// came from and returns the canonical classification.
//
// Matching is exact against the catalog: labels are trimmed, case-folded, have
// the HealthKit enum prefix removed, and have separator characters dropped, but
// a label must equal a whole synonym. Substring matching is deliberately absent
// so user-authored titles such as "朝ラン 10km" stay verbatim.
//
// A bare family label ("Run", "ラン", "跑步", "Walk", "Ride") is promoted to its
// indoor sibling only when Indoor is explicitly true, and to its outdoor sibling
// only when Indoor is explicitly false or a GPS route exists. Without that
// evidence the family canonical is returned unchanged: no location is inferred.
func ResolveActivity(raw string, ctx ActivityContext) ActivityClassification {
	label := strings.TrimSpace(raw)
	if label == "" {
		return classificationFromEntry(entryIndexForKey(ActivityKeyWorkoutGeneric), true, LangUnknown)
	}
	if match, ok := activitySynonymIndex[normalizeActivityLabel(label)]; ok {
		entry := activityEntries[match.entryIndex]
		if entry.Location == ActivityLocationFamily {
			return classificationFromEntry(promoteFamilyEntry(match.entryIndex, ctx), true, match.lang)
		}
		return classificationFromEntry(match.entryIndex, true, match.lang)
	}

	sport := InferSportType(label)
	if sport == "" {
		sport = SportOther
	}
	return ActivityClassification{
		CanonicalKey:   ActivityKeyCustom,
		SportType:      sport,
		IsIndoor:       ctx.Indoor != nil && *ctx.Indoor,
		CanonicalTitle: raw,
		IsSystemLabel:  false,
		MatchedLang:    DetectScript(label),
	}
}

// InferSportType classifies a free-form label by keyword, returning a canonical
// sport slug or "" when nothing matches. It is the fallback for labels outside
// the catalog (for example "Trail Running 12km"); callers that need a concrete
// bucket should map "" to SportOther themselves.
//
// Latin keywords match whole tokens, so "Brunch" is not a run and "15km" is not
// "5k". Han and kana keywords have no word boundaries and match as substrings,
// except katakana, which must not touch other katakana ("ランチ" is lunch, not
// a run). A bare distance is the weakest evidence and is consulted only when no
// sport keyword matches.
func InferSportType(raw string) string {
	lower := strings.ToLower(raw)
	tokens := sportTokens(raw)
	for _, rule := range sportKeywordRules {
		for _, keyword := range rule.keywords {
			if matchSportKeyword(lower, tokens, keyword) {
				return rule.sport
			}
		}
	}
	for _, distance := range runDistanceTokens {
		if strings.Contains(tokens, sportTokens(distance)) {
			return SportRun
		}
	}
	return ""
}

// matchSportKeyword reports whether keyword occurs in lower. tokens is lower
// reduced to space-separated letter/digit runs, padded with spaces.
func matchSportKeyword(lower, tokens, keyword string) bool {
	if keyword == "" {
		return false
	}
	if !isLatinKeyword(keyword) {
		return containsBounded(lower, keyword)
	}
	return strings.Contains(tokens, sportTokens(keyword))
}

// sportTokens reduces text to its lowercase words, separated and surrounded by
// single spaces. Words split at any non-alphanumeric rune and at a lower-to-upper
// camel-case step, so "FunctionalStrengthTraining" and "fitness_gaming" yield
// the same words as their spaced forms.
func sportTokens(text string) string {
	var b strings.Builder
	b.WriteByte(' ')
	var prev rune
	for _, r := range text {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			if !strings.HasSuffix(b.String(), " ") {
				b.WriteByte(' ')
			}
		default:
			if unicode.IsUpper(r) && unicode.IsLower(prev) {
				b.WriteByte(' ')
			}
			b.WriteRune(unicode.ToLower(r))
		}
		prev = r
	}
	if !strings.HasSuffix(b.String(), " ") {
		b.WriteByte(' ')
	}
	return b.String()
}

func isLatinKeyword(keyword string) bool {
	for _, r := range keyword {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// containsBounded finds keyword in text. A katakana keyword only matches when
// the neighboring runes are not katakana; any other script matches anywhere.
func containsBounded(text, keyword string) bool {
	if first, _ := utf8.DecodeRuneInString(keyword); !unicode.Is(unicode.Katakana, first) {
		return strings.Contains(text, keyword)
	}
	for offset := 0; ; {
		i := strings.Index(text[offset:], keyword)
		if i < 0 {
			return false
		}
		start, end := offset+i, offset+i+len(keyword)
		before, _ := utf8.DecodeLastRuneInString(text[:start])
		after, _ := utf8.DecodeRuneInString(text[end:])
		if !unicode.Is(unicode.Katakana, before) && !unicode.Is(unicode.Katakana, after) {
			return true
		}
		_, size := utf8.DecodeRuneInString(keyword)
		offset = start + size
	}
}

// runDistanceTokens are race distances that suggest a run only when the label
// names no sport.
var runDistanceTokens = []string{"5k", "10k"}

// sportKeywordRules is ordered: earlier rules win, so "HIIT Run" reads as HIIT
// rather than run and "Fitness Gaming" is not mistaken for generic training.
// Latin keywords are whole words, so inflections are listed explicitly.
var sportKeywordRules = []struct {
	sport    string
	keywords []string
}{
	{SportFitnessGaming, []string{"fitnessgaming", "fitness gaming", "フィットネスゲーム", "健身游戏"}},
	{SportHIIT, []string{"hiit", "highintensityinterval", "high intensity interval", "高強度インターバル", "高强度间歇"}},
	{SportTraditionalStrengthTraining, []string{"strength", "筋トレ", "筋力トレーニング", "力量训练", "健身"}},
	{SportYoga, []string{"yoga", "ヨガ", "瑜伽"}},
	{SportHike, []string{"hike", "hikes", "hiking", "ハイキング", "登山", "徒步", "远足", "爬山"}},
	{SportRun, []string{"run", "runs", "running", "jog", "jogging", "treadmill", "marathon", "ラン", "ランニング", "跑步"}},
	{SportWalk, []string{"walk", "walks", "walking", "ウォーキング", "徒歩", "歩行", "步行", "散步"}},
	{SportRide, []string{"cycle", "cycling", "bike", "biking", "ride", "rides", "サイクリング", "自転車", "骑行", "自行车", "单车"}},
	{SportSwim, []string{"swim", "swimming", "スイム", "水泳", "游泳"}},
}

func buildActivityEntryIndex() map[string]int {
	index := make(map[string]int, len(activityEntries))
	for i, entry := range activityEntries {
		index[entry.CanonicalKey] = i
	}
	return index
}

func buildActivitySynonymIndex() map[string]synonymMatch {
	index := make(map[string]synonymMatch, len(activityEntries)*3)
	for i, entry := range activityEntries {
		for _, lang := range supportedLanguageOrder {
			for _, synonym := range entry.Synonyms[lang] {
				key := normalizeActivityLabel(synonym)
				if key == "" {
					continue
				}
				if _, exists := index[key]; exists {
					continue
				}
				index[key] = synonymMatch{entryIndex: i, lang: lang}
			}
		}
	}
	return index
}

func entryIndexForKey(key string) int {
	if index, ok := activityEntryIndex[key]; ok {
		return index
	}
	return activityEntryIndex[ActivityKeyWorkoutGeneric]
}

// familyVariantIndex returns the indoor or outdoor sibling of a family entry.
func familyVariantIndex(sport string, location string) (int, bool) {
	for i, entry := range activityEntries {
		if entry.SportType == sport && entry.Location == location {
			return i, true
		}
	}
	return 0, false
}

// promoteFamilyEntry upgrades a family-level match only when the workout
// carries real location evidence.
func promoteFamilyEntry(index int, ctx ActivityContext) int {
	entry := activityEntries[index]
	switch {
	case ctx.Indoor != nil && *ctx.Indoor:
		if indoor, ok := familyVariantIndex(entry.SportType, ActivityLocationIndoor); ok {
			return indoor
		}
	case (ctx.Indoor != nil && !*ctx.Indoor) || ctx.HasRoute:
		if outdoor, ok := familyVariantIndex(entry.SportType, ActivityLocationOutdoor); ok {
			return outdoor
		}
	}
	return index
}

func classificationFromEntry(index int, isSystemLabel bool, lang Language) ActivityClassification {
	entry := activityEntries[index]
	return ActivityClassification{
		CanonicalKey:   entry.CanonicalKey,
		SportType:      entry.SportType,
		IsIndoor:       entry.IsIndoor,
		CanonicalTitle: entry.CanonicalTitle,
		IsSystemLabel:  isSystemLabel,
		MatchedLang:    lang,
	}
}

// normalizeActivityLabel builds the exact-match comparison key for a label:
// trimmed, enum prefix removed, case-folded, and separator-free. Removing
// separators makes "Outdoor Run", "outdoor_run", and "Outdoor-Run" one key
// without ever matching a substring of a longer user title.
func normalizeActivityLabel(raw string) string {
	label := TrimAppleHealthActivityTypePrefix(strings.TrimSpace(raw))

	var key strings.Builder
	key.Grow(len(label))
	for _, r := range label {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			continue
		}
		key.WriteRune(unicode.ToLower(r))
	}
	return key.String()
}
