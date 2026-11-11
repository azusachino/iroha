package i18n

import "testing"

func confidencePtr(value float64) *float64 {
	return &value
}

func TestResolveWorkTitles(t *testing.T) {
	tests := []struct {
		name            string
		currentPrimary  string
		currentOriginal string
		declaredLang    string
		candidates      []TitleCandidate
		wantPrimary     string
		wantOriginal    string
		wantLanguage    string
	}{
		{
			name: "bangumi japanese original beats the primary-flagged chinese translation",
			candidates: []TitleCandidate{
				{Title: "君のことが大大大大大好きな100人の彼女", TitleKind: TitleKindOriginal, Provider: "bangumi"},
				{Title: "超超超超超喜欢你的100个女朋友", Language: "zh-Hans", TitleKind: TitleKindLocalized, Provider: "bangumi", IsPrimary: true},
			},
			wantPrimary:  "君のことが大大大大大好きな100人の彼女",
			wantOriginal: "君のことが大大大大大好きな100人の彼女",
		},
		{
			name: "anilist native title wins over its own english primary flag",
			candidates: []TitleCandidate{
				{Title: "スパイファミリー", TitleKind: TitleKindOriginal, Provider: "anilist"},
				{Title: "SPY×FAMILY", TitleKind: TitleKindRomanized, Provider: "anilist"},
				{Title: "SPY x FAMILY", Language: "en", TitleKind: TitleKindLocalized, Provider: "anilist", IsPrimary: true},
			},
			wantPrimary:  "スパイファミリー",
			wantOriginal: "スパイファミリー",
		},
		{
			name:         "chinese novel keeps its native title over english and japanese translations",
			declaredLang: "zh",
			candidates: []TitleCandidate{
				{Title: "挪威的森林", Language: "zh", TitleKind: TitleKindOriginal, Provider: "weread"},
				{Title: "Norwegian Wood", Language: "en", TitleKind: TitleKindLocalized, Provider: "hardcover"},
				{Title: "ノルウェイの森", Language: "ja", TitleKind: TitleKindLocalized, Provider: "bangumi"},
			},
			wantPrimary:  "挪威的森林",
			wantOriginal: "挪威的森林",
			wantLanguage: "zh",
		},
		{
			name:            "declared chinese language protects the original from a japanese claim",
			currentPrimary:  "挪威的森林",
			currentOriginal: "挪威的森林",
			declaredLang:    "zh",
			candidates: []TitleCandidate{
				{Title: "ノルウェイの森", TitleKind: TitleKindOriginal, Provider: "anilist"},
				{Title: "Norwegian Wood", Language: "en", TitleKind: TitleKindLocalized, Provider: "hardcover"},
			},
			wantPrimary:  "挪威的森林",
			wantOriginal: "挪威的森林",
			wantLanguage: "zh",
		},
		{
			name:            "a declared matching original claim corrects a title seeded from a translated edition",
			currentPrimary:  "挪威的森林",
			currentOriginal: "挪威的森林",
			declaredLang:    "ja",
			candidates: []TitleCandidate{
				{Title: "ノルウェイの森", TitleKind: TitleKindOriginal, Provider: "anilist"},
			},
			wantPrimary:  "ノルウェイの森",
			wantOriginal: "ノルウェイの森",
			wantLanguage: "ja",
		},
		{
			name: "english book consumed as a chinese edition keeps the english original",
			candidates: []TitleCandidate{
				{Title: "Designing Data-Intensive Applications", Language: "en", TitleKind: TitleKindOriginal, Provider: "hardcover"},
				{Title: "数据密集型应用系统设计", Language: "zh-Hans", TitleKind: TitleKindLocalized, Provider: "weread"},
			},
			wantPrimary:  "Designing Data-Intensive Applications",
			wantOriginal: "Designing Data-Intensive Applications",
			wantLanguage: "en",
		},
		{
			name: "falls back to the english title when the native title is unknown",
			candidates: []TitleCandidate{
				{Title: "Norwegian Wood", Language: "en", TitleKind: TitleKindLocalized, Provider: "hardcover"},
				{Title: "挪威的森林", Language: "zh-Hans", TitleKind: TitleKindLocalized, Provider: "weread"},
			},
			wantPrimary: "Norwegian Wood",
		},
		{
			name: "falls back to an untagged latin title when no english claim is declared",
			candidates: []TitleCandidate{
				{Title: "Norwegian Wood", TitleKind: TitleKindRomanized, Provider: "bangumi"},
				{Title: "挪威的森林", Language: "zh-Hans", TitleKind: TitleKindLocalized, Provider: "weread"},
			},
			wantPrimary: "Norwegian Wood",
		},
		{
			name: "last resort keeps the first claim offered",
			candidates: []TitleCandidate{
				{Title: "挪威的森林", Language: "zh-Hans", TitleKind: TitleKindLocalized, Provider: "weread"},
			},
			wantPrimary: "挪威的森林",
		},
		{
			name:            "a localized claim never overwrites an existing original",
			currentPrimary:  "Norwegian Wood",
			currentOriginal: "Norwegian Wood",
			declaredLang:    "en",
			candidates: []TitleCandidate{
				{Title: "挪威的森林", Language: "zh-Hans", TitleKind: TitleKindLocalized, Provider: "weread", IsPrimary: true},
			},
			wantPrimary:  "Norwegian Wood",
			wantOriginal: "Norwegian Wood",
			wantLanguage: "en",
		},
		{
			name: "highest confidence original claim wins without a declared language",
			candidates: []TitleCandidate{
				{Title: "Low Confidence Title", TitleKind: TitleKindOriginal, Provider: "bangumi", Confidence: confidencePtr(0.4)},
				{Title: "High Confidence Title", TitleKind: TitleKindOriginal, Provider: "anilist", Confidence: confidencePtr(0.9)},
			},
			wantPrimary:  "High Confidence Title",
			wantOriginal: "High Confidence Title",
		},
		{
			name: "the original language is populated from the chosen claim",
			candidates: []TitleCandidate{
				{Title: "ノルウェイの森", Language: "ja", TitleKind: TitleKindOriginal, Provider: "anilist"},
			},
			wantPrimary:  "ノルウェイの森",
			wantOriginal: "ノルウェイの森",
			wantLanguage: "ja",
		},
		{
			name: "empty claims are ignored",
			candidates: []TitleCandidate{
				{Title: "", TitleKind: TitleKindOriginal, Provider: "bangumi"},
				{Title: "ノルウェイの森", Language: "ja", TitleKind: TitleKindOriginal, Provider: "anilist"},
			},
			wantPrimary:  "ノルウェイの森",
			wantOriginal: "ノルウェイの森",
			wantLanguage: "ja",
		},
		{
			name:            "no candidates leave the recorded titles alone",
			currentPrimary:  "Recorded Primary",
			currentOriginal: "Recorded Original",
			declaredLang:    "ja",
			wantPrimary:     "Recorded Primary",
			wantOriginal:    "Recorded Original",
			wantLanguage:    "ja",
		},
		{
			name:            "native first outranks a longer english primary title",
			currentPrimary:  "English Display Name",
			currentOriginal: "日本語名",
			candidates: []TitleCandidate{
				{Title: "日本語名", Language: "ja", TitleKind: TitleKindOriginal, Provider: "anilist"},
				{Title: "English Display Name", Language: "en", TitleKind: TitleKindLocalized, Provider: "anilist"},
			},
			wantPrimary:  "日本語名",
			wantOriginal: "日本語名",
			wantLanguage: "ja",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveWorkTitles(tt.currentPrimary, tt.currentOriginal, tt.declaredLang, tt.candidates)
			if got.PrimaryTitle != tt.wantPrimary {
				t.Errorf("PrimaryTitle = %q, want %q", got.PrimaryTitle, tt.wantPrimary)
			}
			if got.OriginalTitle != tt.wantOriginal {
				t.Errorf("OriginalTitle = %q, want %q", got.OriginalTitle, tt.wantOriginal)
			}
			if got.OriginalLanguage != tt.wantLanguage {
				t.Errorf("OriginalLanguage = %q, want %q", got.OriginalLanguage, tt.wantLanguage)
			}
		})
	}
}

// TestResolveWorkTitlesNeverPromotesATranslation pins the invariant the old
// script ranking violated: no combination of localized or romanized claims can
// install an original title.
func TestResolveWorkTitlesNeverPromotesATranslation(t *testing.T) {
	candidates := []TitleCandidate{
		{Title: "Norwegian Wood", Language: "en", TitleKind: TitleKindLocalized, Provider: "hardcover", IsPrimary: true},
		{Title: "ノルウェイの森", Language: "ja", TitleKind: TitleKindLocalized, Provider: "bangumi", IsPrimary: true},
		{Title: "Noruwee no Mori", TitleKind: TitleKindRomanized, Provider: "anilist", IsPrimary: true},
	}
	got := ResolveWorkTitles("", "", "", candidates)
	if got.OriginalTitle != "" {
		t.Errorf("OriginalTitle = %q, want empty: translations are not authored titles", got.OriginalTitle)
	}
	if got.OriginalLanguage != "" {
		t.Errorf("OriginalLanguage = %q, want empty: no original claim declared one", got.OriginalLanguage)
	}
	if got.PrimaryTitle != "Norwegian Wood" {
		t.Errorf("PrimaryTitle = %q, want the English fallback", got.PrimaryTitle)
	}
}

func TestLanguageMatches(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"zh", "zh-Hans", true},
		{"zh-Hans", "zh", true},
		{"ZH", "zh", true},
		{"ja", "zh", false},
		{"en", "ja", false},
		{"", "zh", false},
		{"zh", "", false},
	}

	for _, tt := range tests {
		if got := languageMatches(tt.a, tt.b); got != tt.want {
			t.Errorf("languageMatches(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
