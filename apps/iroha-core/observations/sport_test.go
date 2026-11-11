package observations

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-core/i18n"
)

func TestNormalizeSport(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HKWorkoutActivityTypeRunning", "run"},
		{"running", "run"},
		{"run", "run"},
		{"Run", "run"},
		{"HKWorkoutActivityTypeWalking", "walk"},
		{"walking", "walk"},
		{"walk", "walk"},
		{"HKWorkoutActivityTypeCycling", "ride"},
		{"cycling", "ride"},
		{"ride", "ride"},
		{"bike", "ride"},
		{"HKWorkoutActivityTypeHiking", "hike"},
		{"hiking", "hike"},
		{"hike", "hike"},
		{"HKWorkoutActivityTypeSwimming", "swim"},
		{"swimming", "swim"},
		{"swim", "swim"},
		{"HKWorkoutActivityTypeFitnessGaming", "fitness_gaming"},
		{"FitnessGaming", "fitness_gaming"},
		{"Fitness Gaming", "fitness_gaming"},
		{"fitness_gaming", "fitness_gaming"},
		{"Outdoor Run", "run"},
		{"Indoor Run", "run"},
		{"outdoor_run", "run"},
		{"Outdoor Walk", "walk"},
		{"Outdoor Cycling", "ride"},
		{"Pool Swim", "swim"},
		{"HKWorkoutActivityTypeYoga", "yoga"},
		{"Yoga", "yoga"},
		{"TraditionalStrengthTraining", "traditional_strength_training"},
		{"屋外 ラン", "run"},
		{"屋外ラン", "run"},
		{"屋内ラン", "run"},
		{"屋外ウォーキング", "walk"},
		{"屋外サイクリング", "ride"},
		{"ハイキング", "hike"},
		{"水泳", "swim"},
		{"ヨガ", "yoga"},

		// Japanese localized labels that previously fell through to a slug.
		{"ラン", "run"},
		{"ランニング", "run"},
		{"室内ラン", "run"},
		{"トレッドミル", "run"},
		{"屋内 ラン", "run"},
		{"徒歩", "walk"},
		{"屋内ウォーキング", "walk"},
		{"屋内サイクリング", "ride"},
		{"スピンバイク", "ride"},
		{"自転車", "ride"},
		{"スイム", "swim"},
		{"プールスイミング", "swim"},
		{"登山", "hike"},
		{"筋トレ", "traditional_strength_training"},
		{"筋力トレーニング", "traditional_strength_training"},
		{"機能的筋力トレーニング", "traditional_strength_training"},
		{"高強度インターバルトレーニング", "hiit"},
		{"フィットネスゲーム", "fitness_gaming"},

		// Chinese localized labels; these used to normalize to an empty slug.
		{"户外跑步", "run"},
		{"户外跑", "run"},
		{"跑步", "run"},
		{"室内跑步", "run"},
		{"室内跑", "run"},
		{"跑步机", "run"},
		{"路跑", "run"},
		{"户外步行", "walk"},
		{"室内步行", "walk"},
		{"步行", "walk"},
		{"散步", "walk"},
		{"健走", "walk"},
		{"户外骑行", "ride"},
		{"室内骑行", "ride"},
		{"室内单车", "ride"},
		{"动感单车", "ride"},
		{"骑行", "ride"},
		{"自行车", "ride"},
		{"泳池游泳", "swim"},
		{"游泳", "swim"},
		{"公开水域游泳", "swim"},
		{"徒步", "hike"},
		{"远足", "hike"},
		{"爬山", "hike"},
		{"力量训练", "traditional_strength_training"},
		{"功能性力量训练", "traditional_strength_training"},
		{"高强度间歇训练", "hiit"},
		{"瑜伽", "yoga"},
		{"健身游戏", "fitness_gaming"},

		// Sport slugs shared across localized labels and enum aliases.
		{"HIIT", "hiit"},
		{"HighIntensityIntervalTraining", "hiit"},
		{"HKWorkoutActivityTypeHighIntensityIntervalTraining", "hiit"},
		{"Functional Strength Training", "traditional_strength_training"},
		{"HKWorkoutActivityTypeFunctionalStrengthTraining", "traditional_strength_training"},
		{"HKWorkoutActivityTypeTraditionalStrengthTraining", "traditional_strength_training"},

		// Sports outside the taxonomy keep a stable, provider-specific slug.
		{"Rowing Machine", "rowing_machine"},
		{"Rowing", "rowing"},
		{"HKWorkoutActivityTypeRowing", "rowing"},
		{"HKWorkoutActivityTypeElliptical", "elliptical"},
		{"hkworkoutactivitytyperowing", "rowing"},
		{"Other", "other"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		got := NormalizeSport(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeSport(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestNormalizeSportMatchesActivityTaxonomy asserts that every HAE label and
// every native Apple Health enum declared by the shared taxonomy normalizes to
// that taxonomy entry's sport slug. A split here means the two import paths
// would write different sport_type values for the same workout.
func TestNormalizeSportMatchesActivityTaxonomy(t *testing.T) {
	entries := i18n.ActivityEntries()
	if len(entries) == 0 {
		t.Fatal("activity taxonomy is empty")
	}

	for _, entry := range entries {
		t.Run(entry.CanonicalKey, func(t *testing.T) {
			if got := NormalizeSport(entry.CanonicalTitle); got != entry.SportType {
				t.Errorf("NormalizeSport(%q) = %q, want %q", entry.CanonicalTitle, got, entry.SportType)
			}
			for _, lang := range []i18n.Language{i18n.LangEN, i18n.LangZH, i18n.LangJA} {
				for _, synonym := range entry.Synonyms[lang] {
					if synonym == "" {
						continue
					}
					if got := NormalizeSport(synonym); got != entry.SportType {
						t.Errorf("NormalizeSport(%q) = %q, want %q", synonym, got, entry.SportType)
					}
				}
			}
		})
	}
}

// TestNormalizeSportHAELabelsMatchNativeExportEnums pairs representative
// Health Auto Export labels with the native export's HKWorkoutActivityType enum
// for each standard category.
func TestNormalizeSportHAELabelsMatchNativeExportEnums(t *testing.T) {
	tests := []struct {
		category  string
		enum      string
		haeLabels []string
		want      string
	}{
		{
			category:  "run",
			enum:      "HKWorkoutActivityTypeRunning",
			haeLabels: []string{"Running", "Run", "Outdoor Run", "Indoor Run", "Treadmill", "ラン", "屋外ラン", "屋内ラン", "跑步", "户外跑步", "室内跑步"},
			want:      "run",
		},
		{
			category:  "walk",
			enum:      "HKWorkoutActivityTypeWalking",
			haeLabels: []string{"Walking", "Walk", "Outdoor Walk", "Indoor Walk", "ウォーキング", "屋外ウォーキング", "屋内ウォーキング", "步行", "户外步行", "室内步行"},
			want:      "walk",
		},
		{
			category:  "ride",
			enum:      "HKWorkoutActivityTypeCycling",
			haeLabels: []string{"Cycling", "Ride", "Indoor Cycling", "Spinning", "サイクリング", "屋外サイクリング", "屋内サイクリング", "骑行", "户外骑行", "室内骑行"},
			want:      "ride",
		},
		{
			category:  "swim",
			enum:      "HKWorkoutActivityTypeSwimming",
			haeLabels: []string{"Swimming", "Pool Swim", "Open Water Swim", "水泳", "プールスイミング", "游泳", "泳池游泳"},
			want:      "swim",
		},
		{
			category:  "hike",
			enum:      "HKWorkoutActivityTypeHiking",
			haeLabels: []string{"Hiking", "Hike", "ハイキング", "登山", "徒步", "爬山"},
			want:      "hike",
		},
		{
			category:  "strength training",
			enum:      "HKWorkoutActivityTypeTraditionalStrengthTraining",
			haeLabels: []string{"Traditional Strength Training", "Functional Strength Training", "筋力トレーニング", "機能的筋力トレーニング", "力量训练", "功能性力量训练"},
			want:      "traditional_strength_training",
		},
		{
			category:  "hiit",
			enum:      "HKWorkoutActivityTypeHighIntensityIntervalTraining",
			haeLabels: []string{"High Intensity Interval Training", "HIIT", "高強度インターバルトレーニング", "高强度间歇训练"},
			want:      "hiit",
		},
		{
			category:  "yoga",
			enum:      "HKWorkoutActivityTypeYoga",
			haeLabels: []string{"Yoga", "ヨガ", "瑜伽"},
			want:      "yoga",
		},
		{
			category:  "fitness gaming",
			enum:      "HKWorkoutActivityTypeFitnessGaming",
			haeLabels: []string{"Fitness Gaming", "FitnessGaming", "フィットネスゲーム", "健身游戏"},
			want:      "fitness_gaming",
		},
		{
			category:  "other",
			enum:      "HKWorkoutActivityTypeOther",
			haeLabels: []string{"Other"},
			want:      "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			if got := NormalizeSport(tt.enum); got != tt.want {
				t.Fatalf("NormalizeSport(%q) = %q, want %q", tt.enum, got, tt.want)
			}
			for _, label := range tt.haeLabels {
				if got := NormalizeSport(label); got != tt.want {
					t.Errorf("NormalizeSport(%q) = %q, want %q", label, got, tt.want)
				}
			}
		})
	}
}

// TestNormalizeSportIsIdempotent asserts that canonical slugs and titles are
// fixed points, so normalizing an already-canonical stored value is a no-op.
func TestNormalizeSportIsIdempotent(t *testing.T) {
	for _, entry := range i18n.ActivityEntries() {
		for _, value := range []string{entry.SportType, entry.CanonicalTitle} {
			once := NormalizeSport(value)
			if twice := NormalizeSport(once); twice != once {
				t.Errorf("NormalizeSport(%q) = %q then %q, want a fixed point", value, once, twice)
			}
		}
	}
}

func TestSportAliases(t *testing.T) {
	runAliases := SportAliases("running")
	if len(runAliases) != 2 || runAliases[0] != "run" || runAliases[1] != "running" {
		t.Errorf("SportAliases('running') = %v, want ['run', 'running']", runAliases)
	}

	gamingAliases := SportAliases("FitnessGaming")
	if len(gamingAliases) != 2 || gamingAliases[0] != "fitness_gaming" || gamingAliases[1] != "FitnessGaming" {
		t.Errorf("SportAliases('FitnessGaming') = %v, want ['fitness_gaming', 'FitnessGaming']", gamingAliases)
	}

	localizedAliases := SportAliases("户外跑步")
	if len(localizedAliases) != 2 || localizedAliases[0] != "run" || localizedAliases[1] != "running" {
		t.Errorf("SportAliases('户外跑步') = %v, want ['run', 'running']", localizedAliases)
	}
}
