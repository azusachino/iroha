package i18n

import "strings"

// Title kinds a provider may declare for a title claim. A claim is one
// provider's statement about one way of titling a work; it is not itself the
// canonical title.
const (
	TitleKindOriginal  = "original"
	TitleKindRomanized = "romanized"
	TitleKindLocalized = "localized"
)

// TitleCandidate is one provider's title claim for a work.
//
// Language is the provider's declared BCP 47 tag (for example "zh-Hans"); an
// empty Language means the provider did not declare one, and script detection
// (DetectScript) is only used as an auxiliary matching heuristic - never as a
// claim about the work's original language.
type TitleCandidate struct {
	Title      string
	Language   string
	Script     string
	Region     string
	TitleKind  string
	Provider   string
	IsPrimary  bool
	Confidence *float64
}

// WorkTitleResolution is the canonical title projection for a work.
type WorkTitleResolution struct {
	PrimaryTitle     string
	OriginalTitle    string
	OriginalLanguage string
}

// ResolveWorkTitles computes canonical work titles from candidate title claims.
//
// The rules follow ADR-0012's Native First / edition-preserved split:
//
//   - Only a title_kind = "original" claim may set OriginalTitle. A localized
//     translation never overwrites the authored title, and a provider's
//     IsPrimary marker on a translation (Bangumi marks NameCN as primary) is
//     ignored for this decision.
//   - An original claim fills an empty OriginalTitle. When the caller declares
//     an original language, an original claim in that language may also correct
//     an existing OriginalTitle (for example one seeded from a translated
//     edition). Without a declaration, an existing original title wins: the
//     projection cannot tell a provider correction from a different edition's
//     claim, so it preserves the evidence already recorded.
//   - PrimaryTitle is the authored title whenever it is known. Otherwise it
//     falls back to a universal English claim, then to the first claim offered,
//     and finally to the current primary title.
//   - OriginalLanguage is kept when declared, otherwise taken from the chosen
//     original claim when that claim declares one.
//
// The function is pure: it never reads or writes storage, so the same claims
// always produce the same projection.
func ResolveWorkTitles(currentPrimary, currentOriginal, declaredOriginalLang string, candidates []TitleCandidate) WorkTitleResolution {
	resolution := WorkTitleResolution{
		PrimaryTitle:     currentPrimary,
		OriginalTitle:    currentOriginal,
		OriginalLanguage: declaredOriginalLang,
	}

	// A sync that carries no usable claim is not evidence about the work, so it
	// must not rewrite titles the work already has.
	if !hasUsableClaim(candidates) {
		return resolution
	}

	original := selectOriginalCandidate(candidates, declaredOriginalLang)
	if original != nil {
		switch {
		case resolution.OriginalTitle == "":
			resolution.OriginalTitle = original.Title
		case declaredOriginalLang != "" && candidateMatchesLanguage(*original, declaredOriginalLang):
			resolution.OriginalTitle = original.Title
		}
		if resolution.OriginalLanguage == "" {
			resolution.OriginalLanguage = original.Language
		}
	}

	// Native First: the work's primary title is its authored title.
	if resolution.OriginalTitle != "" {
		resolution.PrimaryTitle = resolution.OriginalTitle
		return resolution
	}

	// The authored title is unknown: universal English is the documented
	// fallback, then any title the providers did offer.
	if english := englishTitleCandidate(candidates); english != "" {
		resolution.PrimaryTitle = english
		return resolution
	}
	if first := firstTitleCandidate(candidates); first != "" {
		resolution.PrimaryTitle = first
	}
	return resolution
}

// selectOriginalCandidate picks the authored-title claim to trust.
//
// With a declared original language, only claims that agree with it are
// eligible: a declared language is authority, so a Japanese claim cannot
// displace a work declared Chinese just because its script looks more
// "original". Without a declaration, the highest-confidence claim wins, with
// the provider's own primary marker and declaration order as tie-breakers.
func selectOriginalCandidate(candidates []TitleCandidate, declaredLang string) *TitleCandidate {
	var best *TitleCandidate
	for i := range candidates {
		candidate := &candidates[i]
		if candidate.Title == "" || candidate.TitleKind != TitleKindOriginal {
			continue
		}
		if declaredLang != "" && !candidateMatchesLanguage(*candidate, declaredLang) {
			continue
		}
		if best == nil || outranksOriginal(*candidate, *best) {
			best = candidate
		}
	}
	return best
}

func outranksOriginal(candidate, current TitleCandidate) bool {
	if candidateConfidence(candidate) != candidateConfidence(current) {
		return candidateConfidence(candidate) > candidateConfidence(current)
	}
	return candidate.IsPrimary && !current.IsPrimary
}

func candidateConfidence(candidate TitleCandidate) float64 {
	if candidate.Confidence == nil {
		return 0
	}
	return *candidate.Confidence
}

// candidateMatchesLanguage checks a claim against a declared language using the
// claim's own declaration when present, and script detection only as a
// fallback for providers that omit a language tag.
func candidateMatchesLanguage(candidate TitleCandidate, declaredLang string) bool {
	if candidate.Language != "" {
		return languageMatches(candidate.Language, declaredLang)
	}
	return DetectScript(candidate.Title) == Language(primaryLanguageSubtag(declaredLang))
}

// englishTitleCandidate finds the universal English title: a declared English
// claim first, then any Latin-script title (a provider may leave a romanized or
// English title untagged).
func englishTitleCandidate(candidates []TitleCandidate) string {
	for _, candidate := range candidates {
		if candidate.Title != "" && languageMatches(candidate.Language, string(LangEN)) {
			return candidate.Title
		}
	}
	for _, candidate := range candidates {
		if candidate.Title != "" && DetectScript(candidate.Title) == LangEN {
			return candidate.Title
		}
	}
	return ""
}

func firstTitleCandidate(candidates []TitleCandidate) string {
	for _, candidate := range candidates {
		if candidate.Title != "" {
			return candidate.Title
		}
	}
	return ""
}

func hasUsableClaim(candidates []TitleCandidate) bool {
	return firstTitleCandidate(candidates) != ""
}

// languageMatches compares BCP 47 tags by primary subtag, so "zh" and "zh-Hans"
// agree while "ja" never matches "zh".
func languageMatches(a, b string) bool {
	primary := primaryLanguageSubtag(a)
	return primary != "" && primary == primaryLanguageSubtag(b)
}

func primaryLanguageSubtag(tag string) string {
	normalized := strings.ToLower(strings.TrimSpace(tag))
	if index := strings.IndexAny(normalized, "-_"); index >= 0 {
		normalized = normalized[:index]
	}
	return normalized
}
