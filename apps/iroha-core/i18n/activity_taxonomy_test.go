package i18n

import (
	"testing"
)

func boolPtr(value bool) *bool {
	return &value
}

// expectedSynonymLang mirrors the catalog index's first-declared-language rule
// so a label legitimately shared across languages (for example "HIIT") is not
// reported as the wrong match language.
func expectedSynonymLang(entry ActivityEntry, synonym string) Language {
	key := normalizeActivityLabel(synonym)
	for _, lang := range supportedLanguageOrder {
		for _, candidate := range entry.Synonyms[lang] {
			if normalizeActivityLabel(candidate) == key {
				return lang
			}
		}
	}
	return LangUnknown
}

func TestResolveActivitySynonyms(t *testing.T) {
	for _, entry := range ActivityEntries() {
		t.Run(entry.CanonicalKey, func(t *testing.T) {
			for _, lang := range supportedLanguageOrder {
				for _, synonym := range entry.Synonyms[lang] {
					if synonym == "" {
						continue
					}
					got := ResolveActivity(synonym, ActivityContext{})
					if got.CanonicalKey != entry.CanonicalKey {
						t.Errorf("ResolveActivity(%q).CanonicalKey = %q, want %q", synonym, got.CanonicalKey, entry.CanonicalKey)
					}
					if got.CanonicalTitle != entry.CanonicalTitle {
						t.Errorf("ResolveActivity(%q).CanonicalTitle = %q, want %q", synonym, got.CanonicalTitle, entry.CanonicalTitle)
					}
					if got.SportType != entry.SportType {
						t.Errorf("ResolveActivity(%q).SportType = %q, want %q", synonym, got.SportType, entry.SportType)
					}
					if got.IsIndoor != entry.IsIndoor {
						t.Errorf("ResolveActivity(%q).IsIndoor = %v, want %v", synonym, got.IsIndoor, entry.IsIndoor)
					}
					if !got.IsSystemLabel {
						t.Errorf("ResolveActivity(%q).IsSystemLabel = false, want true", synonym)
					}
					if want := expectedSynonymLang(entry, synonym); got.MatchedLang != want {
						t.Errorf("ResolveActivity(%q).MatchedLang = %q, want %q", synonym, got.MatchedLang, want)
					}
				}
			}
		})
	}
}

func TestActivityEntriesAreWellFormed(t *testing.T) {
	locations := map[string]bool{
		ActivityLocationFamily:    true,
		ActivityLocationIndoor:    true,
		ActivityLocationOutdoor:   true,
		ActivityLocationPool:      true,
		ActivityLocationOpenWater: true,
		ActivityLocationGeneric:   true,
	}
	sports := map[string]bool{
		SportRun:                         true,
		SportWalk:                        true,
		SportRide:                        true,
		SportSwim:                        true,
		SportHike:                        true,
		SportTraditionalStrengthTraining: true,
		SportHIIT:                        true,
		SportYoga:                        true,
		SportFitnessGaming:               true,
		SportOther:                       true,
	}

	keys := make(map[string]bool, len(activityEntries))
	synonymOwner := make(map[string]string, len(activityEntries)*4)
	for _, entry := range activityEntries {
		if entry.CanonicalKey == "" || entry.CanonicalTitle == "" || entry.SportType == "" {
			t.Fatalf("entry %+v has an empty key, title, or sport", entry)
		}
		if keys[entry.CanonicalKey] {
			t.Fatalf("duplicate canonical key %q", entry.CanonicalKey)
		}
		keys[entry.CanonicalKey] = true
		if !locations[entry.Location] {
			t.Errorf("entry %q has unknown location %q", entry.CanonicalKey, entry.Location)
		}
		if !sports[entry.SportType] {
			t.Errorf("entry %q has unknown sport %q", entry.CanonicalKey, entry.SportType)
		}
		if entry.Location == ActivityLocationIndoor && !entry.IsIndoor {
			t.Errorf("entry %q is indoor-located but not marked indoor", entry.CanonicalKey)
		}

		nonEmpty := 0
		for _, lang := range supportedLanguageOrder {
			// Family and generic entries carry no localized provider label of
			// their own: they are promotion targets or honest buckets.
			if len(entry.Synonyms[lang]) == 0 && entry.Location != ActivityLocationFamily && entry.Location != ActivityLocationGeneric {
				t.Errorf("entry %q declares no %q synonyms", entry.CanonicalKey, lang)
			}
			for _, synonym := range entry.Synonyms[lang] {
				if synonym == "" {
					continue
				}
				nonEmpty++
				key := normalizeActivityLabel(synonym)
				if owner, exists := synonymOwner[key]; exists && owner != entry.CanonicalKey {
					t.Errorf("synonym %q is declared by both %q and %q", synonym, owner, entry.CanonicalKey)
				}
				synonymOwner[key] = entry.CanonicalKey
			}
		}
		if nonEmpty == 0 {
			t.Errorf("entry %q declares no usable synonyms", entry.CanonicalKey)
		}
	}
}

func TestActivityTaxonomyCoversStandardCategories(t *testing.T) {
	want := []string{
		ActivityKeyRunOutdoor,
		ActivityKeyRunIndoor,
		ActivityKeyRunFamily,
		ActivityKeyWalkOutdoor,
		ActivityKeyWalkIndoor,
		ActivityKeyWalkFamily,
		ActivityKeyRideOutdoor,
		ActivityKeyRideIndoor,
		ActivityKeyRideFamily,
		ActivityKeySwimPool,
		ActivityKeySwimOpenWater,
		ActivityKeyHike,
		ActivityKeyStrengthTraining,
		ActivityKeyHIIT,
		ActivityKeyYoga,
		ActivityKeyFitnessGaming,
		ActivityKeyWorkoutGeneric,
	}
	got := make(map[string]bool, len(activityEntries))
	for _, entry := range activityEntries {
		got[entry.CanonicalKey] = true
	}
	for _, key := range want {
		if !got[key] {
			t.Errorf("taxonomy is missing canonical key %q", key)
		}
	}
	if len(got) != len(want) {
		t.Errorf("taxonomy has %d entries, want %d", len(got), len(want))
	}
}

func TestResolveActivityContextPromotion(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		ctx        ActivityContext
		wantKey    string
		wantTitle  string
		wantIndoor bool
	}{
		{
			name:      "bare Japanese run with a route is outdoor",
			raw:       "ラン",
			ctx:       ActivityContext{HasRoute: true},
			wantKey:   ActivityKeyRunOutdoor,
			wantTitle: ActivityTitleOutdoorRun,
		},
		{
			name:       "bare Japanese run with indoor metadata is indoor",
			raw:        "ラン",
			ctx:        ActivityContext{Indoor: boolPtr(true)},
			wantKey:    ActivityKeyRunIndoor,
			wantTitle:  ActivityTitleIndoorRun,
			wantIndoor: true,
		},
		{
			name:      "bare Japanese run with explicit outdoor metadata is outdoor",
			raw:       "ラン",
			ctx:       ActivityContext{Indoor: boolPtr(false)},
			wantKey:   ActivityKeyRunOutdoor,
			wantTitle: ActivityTitleOutdoorRun,
		},
		{
			name:       "indoor evidence outranks a route",
			raw:        "ラン",
			ctx:        ActivityContext{Indoor: boolPtr(true), HasRoute: true},
			wantKey:    ActivityKeyRunIndoor,
			wantTitle:  ActivityTitleIndoorRun,
			wantIndoor: true,
		},
		{
			name:      "English family label with a route is outdoor",
			raw:       "Running",
			ctx:       ActivityContext{HasRoute: true},
			wantKey:   ActivityKeyRunOutdoor,
			wantTitle: ActivityTitleOutdoorRun,
		},
		{
			name:       "Chinese family label with indoor metadata is indoor",
			raw:        "跑步",
			ctx:        ActivityContext{Indoor: boolPtr(true)},
			wantKey:    ActivityKeyRunIndoor,
			wantTitle:  ActivityTitleIndoorRun,
			wantIndoor: true,
		},
		{
			name:      "bare walk with a route is outdoor",
			raw:       "ウォーキング",
			ctx:       ActivityContext{HasRoute: true},
			wantKey:   ActivityKeyWalkOutdoor,
			wantTitle: ActivityTitleOutdoorWalk,
		},
		{
			name:       "bare walk with indoor metadata is indoor",
			raw:        "Walk",
			ctx:        ActivityContext{Indoor: boolPtr(true)},
			wantKey:    ActivityKeyWalkIndoor,
			wantTitle:  ActivityTitleIndoorWalk,
			wantIndoor: true,
		},
		{
			name:      "bare ride with a route is outdoor cycling",
			raw:       "Ride",
			ctx:       ActivityContext{HasRoute: true},
			wantKey:   ActivityKeyRideOutdoor,
			wantTitle: ActivityTitleOutdoorCycling,
		},
		{
			name:       "bare ride with indoor metadata is indoor cycling",
			raw:        "Ride",
			ctx:        ActivityContext{Indoor: boolPtr(true)},
			wantKey:    ActivityKeyRideIndoor,
			wantTitle:  ActivityTitleIndoorCycling,
			wantIndoor: true,
		},
		{
			name:      "explicit outdoor label is never demoted",
			raw:       "Outdoor Run",
			ctx:       ActivityContext{Indoor: boolPtr(true)},
			wantKey:   ActivityKeyRunOutdoor,
			wantTitle: ActivityTitleOutdoorRun,
		},
		{
			name:       "explicit indoor label is never promoted",
			raw:        "屋内ラン",
			ctx:        ActivityContext{HasRoute: true},
			wantKey:    ActivityKeyRunIndoor,
			wantTitle:  ActivityTitleIndoorRun,
			wantIndoor: true,
		},
		{
			name:       "swimming has no family entry, so route evidence cannot claim open water",
			raw:        "Swimming",
			ctx:        ActivityContext{HasRoute: true},
			wantKey:    ActivityKeySwimPool,
			wantTitle:  ActivityTitlePoolSwim,
			wantIndoor: true,
		},
		{
			name:      "hiking is inherently outdoor",
			raw:       "ハイキング",
			ctx:       ActivityContext{Indoor: boolPtr(false)},
			wantKey:   ActivityKeyHike,
			wantTitle: ActivityTitleHiking,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveActivity(tt.raw, tt.ctx)
			if got.CanonicalKey != tt.wantKey {
				t.Errorf("CanonicalKey = %q, want %q", got.CanonicalKey, tt.wantKey)
			}
			if got.CanonicalTitle != tt.wantTitle {
				t.Errorf("CanonicalTitle = %q, want %q", got.CanonicalTitle, tt.wantTitle)
			}
			if got.IsIndoor != tt.wantIndoor {
				t.Errorf("IsIndoor = %v, want %v", got.IsIndoor, tt.wantIndoor)
			}
			if !got.IsSystemLabel {
				t.Error("IsSystemLabel = false, want true")
			}
		})
	}
}

func TestResolveActivityHonestFamilyFallback(t *testing.T) {
	tests := []struct {
		raw       string
		wantKey   string
		wantTitle string
		wantSport string
	}{
		{"ラン", ActivityKeyRunFamily, ActivityTitleRun, SportRun},
		{"Running", ActivityKeyRunFamily, ActivityTitleRun, SportRun},
		{"跑步", ActivityKeyRunFamily, ActivityTitleRun, SportRun},
		{"Walking", ActivityKeyWalkFamily, ActivityTitleWalk, SportWalk},
		{"Ride", ActivityKeyRideFamily, ActivityTitleRide, SportRide},
		{"サイクリング", ActivityKeyRideOutdoor, ActivityTitleOutdoorCycling, SportRide},
		{"Swimming", ActivityKeySwimPool, ActivityTitlePoolSwim, SportSwim},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got := ResolveActivity(tt.raw, ActivityContext{})
			if got.CanonicalKey != tt.wantKey {
				t.Errorf("CanonicalKey = %q, want %q", got.CanonicalKey, tt.wantKey)
			}
			if got.CanonicalTitle != tt.wantTitle {
				t.Errorf("CanonicalTitle = %q, want %q", got.CanonicalTitle, tt.wantTitle)
			}
			if got.SportType != tt.wantSport {
				t.Errorf("SportType = %q, want %q", got.SportType, tt.wantSport)
			}
			if got.IsIndoor && tt.wantKey == ActivityKeyRunFamily {
				t.Error("family-level run claimed indoor without evidence")
			}
			if !got.IsSystemLabel {
				t.Error("IsSystemLabel = false, want true")
			}
		})
	}
}

func TestResolveActivityCustomTitles(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantSport string
		wantLang  Language
	}{
		{"Japanese custom title", "朝ラン 10km", SportRun, LangJA},
		{"English custom title", "Sub-40 10k Tempo", SportRun, LangEN},
		{"Chinese custom title", "皇居2周", SportOther, LangZH},
		{"Latin custom title by keyword", "Tokyo Marathon 2026", SportRun, LangEN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveActivity(tt.raw, ActivityContext{})
			if got.IsSystemLabel {
				t.Fatalf("IsSystemLabel = true, want false for %q", tt.raw)
			}
			if got.CanonicalKey != ActivityKeyCustom {
				t.Errorf("CanonicalKey = %q, want %q", got.CanonicalKey, ActivityKeyCustom)
			}
			if got.CanonicalTitle != tt.raw {
				t.Errorf("CanonicalTitle = %q, want verbatim %q", got.CanonicalTitle, tt.raw)
			}
			if got.SportType != tt.wantSport {
				t.Errorf("SportType = %q, want %q", got.SportType, tt.wantSport)
			}
			if got.MatchedLang != tt.wantLang {
				t.Errorf("MatchedLang = %q, want %q", got.MatchedLang, tt.wantLang)
			}
			if got.IsIndoor {
				t.Error("IsIndoor = true without indoor evidence")
			}
		})
	}

	verbatim := "  Tokyo Marathon 2026  "
	if got := ResolveActivity(verbatim, ActivityContext{}); got.CanonicalTitle != verbatim {
		t.Errorf("CanonicalTitle = %q, want verbatim %q", got.CanonicalTitle, verbatim)
	}

	indoor := ResolveActivity("Treadmill Intervals", ActivityContext{Indoor: boolPtr(true)})
	if !indoor.IsIndoor {
		t.Error("IsIndoor = false, want true when the context carries indoor metadata")
	}
}

func TestResolveActivitySubstringTitlesAreNotSystemLabels(t *testing.T) {
	// A title that merely contains a synonym must stay a custom title: this is
	// the boundary that protects user-authored workout names.
	tests := []string{"Morning Run Club", "Run 10K", "Walk&Talk", "Morning Cycle"}
	for _, raw := range tests {
		if got := ResolveActivity(raw, ActivityContext{}); got.IsSystemLabel || got.CanonicalTitle != raw {
			t.Errorf("ResolveActivity(%q) = %+v, want a verbatim custom title", raw, got)
		}
	}
}

func TestResolveActivityEmptyLabel(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t"} {
		got := ResolveActivity(raw, ActivityContext{Indoor: boolPtr(true), HasRoute: true})
		if got.CanonicalKey != ActivityKeyWorkoutGeneric {
			t.Errorf("ResolveActivity(%q).CanonicalKey = %q, want %q", raw, got.CanonicalKey, ActivityKeyWorkoutGeneric)
		}
		if got.CanonicalTitle != ActivityTitleWorkout || got.SportType != SportOther {
			t.Errorf("ResolveActivity(%q) = %+v, want generic Workout/other", raw, got)
		}
		if !got.IsSystemLabel {
			t.Errorf("ResolveActivity(%q).IsSystemLabel = false, want true", raw)
		}
		if got.MatchedLang != LangUnknown {
			t.Errorf("ResolveActivity(%q).MatchedLang = %q, want %q", raw, got.MatchedLang, LangUnknown)
		}
	}
}

func TestResolveActivityNormalization(t *testing.T) {
	tests := []struct {
		raw       string
		wantKey   string
		wantTitle string
	}{
		{"  outdoor   run  ", ActivityKeyRunOutdoor, ActivityTitleOutdoorRun},
		{"OUTDOOR RUN", ActivityKeyRunOutdoor, ActivityTitleOutdoorRun},
		{"outdoor_run", ActivityKeyRunOutdoor, ActivityTitleOutdoorRun},
		{"Outdoor-Run", ActivityKeyRunOutdoor, ActivityTitleOutdoorRun},
		{"屋外 ラン", ActivityKeyRunOutdoor, ActivityTitleOutdoorRun},
		{"HKWorkoutActivityTypeRunning", ActivityKeyRunFamily, ActivityTitleRun},
		{"hkworkoutactivitytyperunning", ActivityKeyRunFamily, ActivityTitleRun},
		{"TraditionalStrengthTraining", ActivityKeyStrengthTraining, ActivityTitleStrengthTraining},
		{"HKWorkoutActivityTypeTraditionalStrengthTraining", ActivityKeyStrengthTraining, ActivityTitleStrengthTraining},
		{"HKWorkoutActivityTypeHighIntensityIntervalTraining", ActivityKeyHIIT, ActivityTitleHIIT},
		{"HighIntensityIntervalTraining", ActivityKeyHIIT, ActivityTitleHIIT},
		{"高強度インターバルトレーニング", ActivityKeyHIIT, ActivityTitleHIIT},
		{"機能的筋力トレーニング", ActivityKeyStrengthTraining, ActivityTitleStrengthTraining},
		{"FunctionalStrengthTraining", ActivityKeyStrengthTraining, ActivityTitleStrengthTraining},
		{"FitnessGaming", ActivityKeyFitnessGaming, ActivityTitleFitnessGaming},
		{"户外跑步", ActivityKeyRunOutdoor, ActivityTitleOutdoorRun},
		{"室内跑步", ActivityKeyRunIndoor, ActivityTitleIndoorRun},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got := ResolveActivity(tt.raw, ActivityContext{})
			if got.CanonicalKey != tt.wantKey || got.CanonicalTitle != tt.wantTitle {
				t.Errorf("ResolveActivity(%q) = %q/%q, want %q/%q", tt.raw, got.CanonicalKey, got.CanonicalTitle, tt.wantKey, tt.wantTitle)
			}
			if !got.IsSystemLabel {
				t.Errorf("ResolveActivity(%q).IsSystemLabel = false, want true", tt.raw)
			}
		})
	}
}

func TestInferSportType(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"Trail Running 12km", SportRun},
		{"treadmill", SportRun},
		{"Morning Jog", SportRun},
		{"Easy 5k", SportRun},
		{"Sub-40 10k Tempo", SportRun},
		{"Cycling Intervals", SportRide},
		{"Bike commute", SportRide},
		{"Indoor Swim Drills", SportSwim},
		{"Evening Yoga Flow", SportYoga},
		{"Strength Circuit", SportTraditionalStrengthTraining},
		{"HIIT Session", SportHIIT},
		{"Fitness Gaming", SportFitnessGaming},
		{"朝ラン 10km", SportRun},
		{"筋トレ", SportTraditionalStrengthTraining},
		{"健身 45分", SportTraditionalStrengthTraining},
		{"骑行", SportRide},
		{"游泳", SportSwim},
		{"Rowing Machine", ""},
		{"", ""},
		// Distances are not sport evidence, and keywords are whole tokens.
		{"Morning Ride 15km", SportRide},
		{"Cycling 25km", SportRide},
		{"Walk 15km", SportWalk},
		{"Brunch ride", SportRide},
		{"Morning 5k", SportRun},
		{"10K race", SportRun},
		{"Brunch", ""},
		{"Evening 15km", ""},
		{"FunctionalStrengthTraining", SportTraditionalStrengthTraining},
		{"HighIntensityIntervalTraining", SportHIIT},
		{"ランチ", ""},
		{"プランク", ""},
		{"朝ランニング", SportRun},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := InferSportType(tt.raw); got != tt.want {
				t.Errorf("InferSportType(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
