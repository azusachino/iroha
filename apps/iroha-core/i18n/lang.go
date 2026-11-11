// Package i18n owns Iroha's pure multilingual domain primitives: the supported
// language set, an auxiliary script heuristic, and the canonical activity
// taxonomy. It has no database, transport, provider, or UI dependencies.
package i18n

import "unicode"

// Language is a BCP 47 primary language subtag restricted to the languages
// this cockpit represents. LangUnknown is the zero value and never a claim.
type Language string

// Supported languages. Localized presentation is rendered from these codes;
// canonical system taxonomy values (activity titles, sport slugs) stay in
// universal English.
const (
	LangEN      Language = "en"
	LangZH      Language = "zh"
	LangJA      Language = "ja"
	LangUnknown Language = ""
)

// IsSupported reports whether lang is one of the three supported languages.
func IsSupported(lang Language) bool {
	switch lang {
	case LangEN, LangZH, LangJA:
		return true
	default:
		return false
	}
}

// DetectScript classifies the writing system a string appears to use and
// returns the language that script most strongly implies.
//
// This is strictly an auxiliary matching heuristic. Script is not language:
// Japanese titles are frequently Latin ("BLEACH") and Chinese and Japanese
// share thousands of ideographs. Callers must never use this to infer a work's
// original_language or an edition's language; authoritative provider
// declarations own those facts.
//
// Kana (Hiragana, Katakana, or Katakana phonetic extensions) implies Japanese
// because Han ideographs alone cannot distinguish Japanese from Chinese.
// Han ideographs without kana imply Chinese. Latin letters alone fall back to
// English. Anything else (digits, punctuation, emoji) is LangUnknown.
func DetectScript(s string) Language {
	hasHan := false
	hasLatin := false
	for _, r := range s {
		switch {
		case unicode.In(r, unicode.Hiragana, unicode.Katakana):
			return LangJA
		case unicode.Is(unicode.Han, r):
			hasHan = true
		case unicode.Is(unicode.Latin, r):
			hasLatin = true
		}
	}
	switch {
	case hasHan:
		return LangZH
	case hasLatin:
		return LangEN
	default:
		return LangUnknown
	}
}
