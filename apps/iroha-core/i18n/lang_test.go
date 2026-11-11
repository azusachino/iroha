package i18n

import "testing"

func TestIsSupported(t *testing.T) {
	tests := []struct {
		lang Language
		want bool
	}{
		{LangEN, true},
		{LangZH, true},
		{LangJA, true},
		{LangUnknown, false},
		{Language("ko"), false},
		{Language("EN"), false},
	}

	for _, tt := range tests {
		if got := IsSupported(tt.lang); got != tt.want {
			t.Errorf("IsSupported(%q) = %v, want %v", tt.lang, got, tt.want)
		}
	}
}

func TestDetectScript(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Language
	}{
		{"empty", "", LangUnknown},
		{"hiragana", "あいうえお", LangJA},
		{"katakana", "アイウエオ", LangJA},
		{"katakana phonetic extensions", "ㇰㇱㇲ", LangJA},
		{"halfwidth katakana", "ｱｲｳ", LangJA},
		{"kanji only", "東京", LangZH},
		{"simplified chinese", "户外跑步", LangZH},
		{"han extension a", "㐀㐁", LangZH},
		{"latin", "Outdoor Run", LangEN},
		{"latin lowercase", "runner", LangEN},
		{"kana with han", "屋外ラン", LangJA},
		{"kana with latin", "Tokyo ラン", LangJA},
		{"han with latin", "iPhone 跑步", LangZH},
		{"digits and punctuation", "2026-11-11 10:00", LangUnknown},
		{"punctuation only", "!?。,.:;-_", LangUnknown},
		{"emoji only", "🏃‍♂️", LangUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectScript(tt.input); got != tt.want {
				t.Errorf("DetectScript(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
