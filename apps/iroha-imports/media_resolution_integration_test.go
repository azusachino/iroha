//go:build integration

package imports

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type seededMediaItem struct {
	workID, itemID, titleID uuid.UUID
}

// seedMediaItem creates a work+item+primary-title triple and registers
// cleanup for all three, mirroring the shape a real sync leaves behind.
func seedMediaItem(t *testing.T, db *gorm.DB, title, mediaType, itemRole string, releaseDate time.Time) seededMediaItem {
	t.Helper()
	now := time.Now().UTC()
	seeded := seededMediaItem{workID: uuid.New(), itemID: uuid.New(), titleID: uuid.New()}

	if err := db.Create(&models.MediaWork{
		ID: seeded.workID, WorkKind: mediaWorkKind, PrimaryTitle: title, OriginalTitle: title,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("seed media work: %v", err)
	}
	if err := db.Create(&models.MediaItem{
		ID: seeded.itemID, WorkID: &seeded.workID, MediaType: mediaType, ItemRole: itemRole,
		Title: title, OriginalTitle: title, ReleaseDate: &releaseDate,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("seed media item: %v", err)
	}
	if err := db.Create(&models.MediaTitle{
		ID: seeded.titleID, ScopeType: mediaScopeType, ScopeID: seeded.itemID, Title: title,
		TitleKind: "primary", IsPrimary: true, CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("seed media title: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("delete from tb_media_resolution_tasks where candidates_json->>'external_id' like 'integration-%'").Error
		_ = db.Exec("delete from tb_media_titles where id = ?", seeded.titleID).Error
		_ = db.Exec("delete from tb_media_items where id = ?", seeded.itemID).Error
		_ = db.Exec("delete from tb_media_works where id = ?", seeded.workID).Error
	})
	return seeded
}

// TestTitleYearCandidates_DoesNotMatchOnYearAlone locks down the exact user
// symptom: a same-year, same-media-type item with a completely different
// title must not come back as a dedupe candidate. Reproduces the suspicion
// that the Where(...).Or(...) chain in titleYearCandidates ORs the base scope
// condition into the whole clause instead of ANDing it, so it matches every
// item in scope regardless of title.
func TestTitleYearCandidates_DoesNotMatchOnYearAlone(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2031, time.March, 1, 0, 0, 0, 0, time.UTC)
	seedMediaItem(t, db, "Existing Title Alpha", "anime_season", "season", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-does-not-matter",
		Title: "Totally Different Title Beta", MediaType: "anime_season", ItemRole: "season",
		ReleaseDate: &releaseDate,
	}

	candidates, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("titleYearCandidates returned %d candidates for a non-matching title (%v), want 0 -- base scope is being OR'd instead of AND'd with the title match", len(candidates), candidates)
	}
}

// TestTitleYearCandidates_StillMatchesRealTitleYearCollision guards against
// over-correcting the fix above into never matching: a genuine same-title,
// same-year, same-media-type item must still come back as a candidate.
func TestTitleYearCandidates_StillMatchesRealTitleYearCollision(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2032, time.March, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "Shared Title Gamma", "anime_season", "season", releaseDate)

	incoming := observations.Media{
		Provider: "bangumi", ExternalID: "integration-does-not-matter-2",
		Title: "Shared Title Gamma", MediaType: "anime_season", ItemRole: "season",
		ReleaseDate: &releaseDate,
	}

	candidates, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != seeded.itemID {
		t.Fatalf("titleYearCandidates = %v, want exactly [%v]", candidates, seeded.itemID)
	}
}

// TestTitleYearCandidates_MatchesAcrossDifferingReleaseYears reproduces the
// real-world case found in prod: the same manga had release dates 2023-08-30
// (Bangumi) and 2024-06-26 (AniList) -- ~300 days apart, different calendar
// years. The old exact-year filter made this pair structurally invisible
// regardless of the OR/AND fix; the date-window widening must catch it.
func TestTitleYearCandidates_MatchesAcrossDifferingReleaseYears(t *testing.T) {
	db := openImportsIntegrationDB(t)
	existingRelease := time.Date(2033, time.August, 30, 0, 0, 0, 0, time.UTC)
	incomingRelease := time.Date(2034, time.June, 26, 0, 0, 0, 0, time.UTC) // ~300 days later, next calendar year
	seeded := seedMediaItem(t, db, "Cross Year Delta", "manga", "series", existingRelease)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-cross-year",
		Title: "Cross Year Delta", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &incomingRelease,
	}

	candidates, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != seeded.itemID {
		t.Fatalf("titleYearCandidates = %v, want exactly [%v] -- cross-provider release dates ~300 days apart must still match", candidates, seeded.itemID)
	}
}

// TestTitleYearCandidates_DoesNotMatchDifferentMediaType reproduces the
// prod case where an anime_season and its manga adaptation share an exact
// title and a nearby release date but must never be treated as the same
// item -- media_type (and item_role) must scope the match.
func TestTitleYearCandidates_DoesNotMatchDifferentMediaType(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2035, time.January, 12, 0, 0, 0, 0, time.UTC)
	seedMediaItem(t, db, "Cross Media Epsilon", "anime_season", "season", releaseDate)

	incoming := observations.Media{
		Provider: "bangumi", ExternalID: "integration-cross-media",
		Title: "Cross Media Epsilon", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &releaseDate,
	}

	candidates, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("titleYearCandidates matched a manga item against an incoming anime_season (%v), want 0 -- media_type must scope the match", candidates)
	}
}

// The following four cases are the exact title strings pulled from
// production (kubectl exec into iroha-postgis-0, tb_media_titles) during the
// 2026-08-06 dedup investigation -- not approximations. Each one is a
// distinct way two providers rendered "the same title" that survived every
// prior fix until normalizeMediaTitle accounted for it.

// TestTitleYearCandidates_MatchesAcrossBracketedAnnotation: Bangumi's
// "original" title for this manga kept a trailing furigana gloss in
// fullwidth parens that AniList's native title omitted entirely.
func TestTitleYearCandidates_MatchesAcrossBracketedAnnotation(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2038, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "拝啓、天国の姉さん、勇者になった姪がエロすぎてーー 叔父さん、保護者とかそろそろ無理です＋（ぷらす）", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-bracket-annotation",
		Title: "拝啓、天国の姉さん、勇者になった姪がエロすぎてーー 叔父さん、保護者とかそろそろ無理です＋", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &releaseDate,
	}

	candidates, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != seeded.itemID {
		t.Fatalf("titleYearCandidates = %v, want exactly [%v] -- a trailing bracketed gloss must not block the match", candidates, seeded.itemID)
	}
}

// TestTitleYearCandidates_MatchesAcrossBracketStyle: the same in-title gloss
// rendered in fullwidth round brackets on one provider (AniList's native
// title) and CJK angle brackets on the other (Bangumi's original title).
func TestTitleYearCandidates_MatchesAcrossBracketStyle(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2039, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "追放された転生王子、『自動製作《オートクラフト》』スキルで領地を爆速で開拓し最強の村を作ってしまう〜最強クラフトスキルで始める、楽々領地開拓スローライフ〜", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-bracket-style",
		Title: "追放された転生王子、『自動製作（オートクラフト）』スキルで領地を爆速で開拓し最強の村を作ってしまう　〜最強クラフトスキルで始める、楽々領地開拓スローライフ〜", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &releaseDate,
	}

	candidates, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != seeded.itemID {
		t.Fatalf("titleYearCandidates = %v, want exactly [%v] -- differing bracket styles around the same gloss must not block the match", candidates, seeded.itemID)
	}
}

// TestTitleYearCandidates_MatchesAcrossFullwidthTilde: the same title with a
// fullwidth tilde (～) plus a single space on one provider, and an ASCII
// tilde (~) plus two spaces on the other.
func TestTitleYearCandidates_MatchesAcrossFullwidthTilde(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2040, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "史上最強の魔法剣士、Fランク冒険者に転生する  ~剣聖と魔帝、2つの前世を持った男の英雄譚~", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-fullwidth-tilde",
		Title: "史上最強の魔法剣士、Fランク冒険者に転生する ～剣聖と魔帝、2つの前世を持った男の英雄譚～", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &releaseDate,
	}

	candidates, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != seeded.itemID {
		t.Fatalf("titleYearCandidates = %v, want exactly [%v] -- a fullwidth-vs-ASCII tilde and spacing difference must not block the match", candidates, seeded.itemID)
	}
}

// TestTitleYearCandidates_MatchesAcrossSubtitleSpacing: Bangumi ran the main
// title straight into its tilde-delimited subtitle with no space; AniList
// inserted one. Collapsing whitespace isn't enough here -- there's no
// redundant whitespace to collapse, the two sides simply disagree on whether
// a separator space exists at all, so the comparison key must drop
// whitespace entirely rather than just normalize runs of it.
func TestTitleYearCandidates_MatchesAcrossSubtitleSpacing(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2041, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "その悪役貴族、ママヒロインが好きすぎる～真摯な努力で最強となり不遇な推しキャラ助けまくる～", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-subtitle-spacing",
		Title: "その悪役貴族、ママヒロインが好きすぎる ～真摯な努力で最強となり不遇な推しキャラ助けまくる～", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &releaseDate,
	}

	candidates, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != seeded.itemID {
		t.Fatalf("titleYearCandidates = %v, want exactly [%v] -- an inserted separator space must not block the match", candidates, seeded.itemID)
	}
}

// TestTitlePrefixCandidates_MatchesOmittedSubtitle reproduces a real prod
// duplicate that exact matching (even with bracket/tilde/spacing
// normalization) can never catch: Bangumi's title for this manga ran
// straight to the end where AniList's had an entire additional trailing
// subtitle. This isn't a formatting difference to normalize away -- one
// side's content is genuinely a superset of the other's.
func TestTitlePrefixCandidates_MatchesOmittedSubtitle(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2042, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "死ぬ運命にある悪役令嬢の兄に転生したので、妹を育てて未来を変えたいと思います", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-omitted-subtitle",
		Title: "死ぬ運命にある悪役令嬢の兄に転生したので、妹を育てて未来を変えたいと思います ～世界最強はオレだけど、世界最カワは妹に違いない～", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &releaseDate,
	}

	// The exact matcher must not find this -- it's a genuinely different
	// string, not a formatting variant.
	exact, err := titleYearCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titleYearCandidates: %v", err)
	}
	if len(exact) != 0 {
		t.Fatalf("titleYearCandidates = %v, want 0 -- an omitted subtitle is not an exact-match case", exact)
	}

	prefix, err := titlePrefixCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titlePrefixCandidates: %v", err)
	}
	if len(prefix) != 1 || prefix[0] != seeded.itemID {
		t.Fatalf("titlePrefixCandidates = %v, want exactly [%v]", prefix, seeded.itemID)
	}
}

// TestTitlePrefixCandidates_MatchesShortOmittedSubtitle is a second real prod
// case showing 25 runes was too conservative a floor: Bangumi's title for
// this manga was AniList's with the entire "～山に追放されたので、..."
// subtitle dropped, sharing only a 14-rune prefix ("異世界グルメで成り上が
// り無双") -- short enough that the original threshold missed it entirely
// (no task, no visibility, same as before titlePrefixCandidates existed).
func TestTitlePrefixCandidates_MatchesShortOmittedSubtitle(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2044, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "異世界グルメで成り上がり無双", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-short-omitted-subtitle",
		Title: "異世界グルメで成り上がり無双～山に追放されたので、のんびりキャンプを楽しんでいたらいつの間にか強くなっていて、王侯貴族や実力者たちが俺を放っておいてくれません。一方、俺を追放した貴族たちは破滅が始まる～", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &releaseDate,
	}

	prefix, err := titlePrefixCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titlePrefixCandidates: %v", err)
	}
	if len(prefix) != 1 || prefix[0] != seeded.itemID {
		t.Fatalf("titlePrefixCandidates = %v, want exactly [%v] -- a 14-rune shared prefix must be caught", prefix, seeded.itemID)
	}
}

// TestTitlePrefixCandidates_DoesNotMatchDifferentSeason is the end-to-end
// version of the real prod false positive: "My Hero Academia" (season 1)
// must not surface as a prefix candidate for an incoming "My Hero Academia
// Season 2" sync, even though it passes the length check and the date
// window (real seasons of an annual franchise often release within a year
// of each other).
func TestTitlePrefixCandidates_DoesNotMatchDifferentSeason(t *testing.T) {
	db := openImportsIntegrationDB(t)
	season1Date := time.Date(2045, time.April, 3, 0, 0, 0, 0, time.UTC)
	season2Date := time.Date(2046, time.April, 1, 0, 0, 0, 0, time.UTC)
	seedMediaItem(t, db, "My Hero Academia", "anime_season", "season", season1Date)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-different-season",
		Title: "My Hero Academia Season 2", MediaType: "anime_season", ItemRole: "season",
		ReleaseDate: &season2Date,
	}

	prefix, err := titlePrefixCandidates(db, incoming)
	if err != nil {
		t.Fatalf("titlePrefixCandidates: %v", err)
	}
	if len(prefix) != 0 {
		t.Fatalf("titlePrefixCandidates = %v, want 0 -- season 1 and season 2 are not duplicates", prefix)
	}
}

// TestTitlePrefixMatch_RequiresAMinimumSharedLength guards the collision
// risk a prefix heuristic introduces: two different works with a short
// generic shared opening (a common trope phrase, not a specific plot
// clause) must not register as a prefix match.
func TestTitlePrefixMatch_RequiresAMinimumSharedLength(t *testing.T) {
	if titlePrefixMatch("異世界転生した", "異世界転生した俺はチート能力で無双する") {
		t.Fatal("titlePrefixMatch matched on a short generic shared opening, want false")
	}
}

// TestTitlePrefixMatch_RejectsSeasonAndPartMarkers guards against a
// systematic false-positive class found in real prod data at
// titlePrefixMinRunes=12: "My Hero Academia" vs "My Hero Academia Season 2"
// (and equivalents) both pass the length check, but a missing season/part
// marker means "a different installment of the same franchise" -- the
// opposite of a duplicate. No rune-count threshold can distinguish this
// from a genuinely omitted subtitle; the trailing content itself must be
// inspected.
func TestTitlePrefixMatch_RejectsSeasonAndPartMarkers(t *testing.T) {
	cases := []struct{ name, shorter, longer string }{
		{"English Season N", "My Hero Academia", "My Hero Academia Season 2"},
		{"English Part N", "Komi Can't Communicate", "Komi Can't Communicate Part 2"},
		{"English Cour N", "Mushoku Tensei: Jobless Reincarnation", "Mushoku Tensei: Jobless Reincarnation Cour 2"},
		{"Japanese numbered season + part", "進撃の巨人 第三季", "進撃の巨人 第三季 Part.2"},
		{"Chinese final season + part", "进击的巨人 最终季", "进击的巨人 最终季 Part.2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := normalizeMediaTitle(tc.shorter), normalizeMediaTitle(tc.longer)
			if titlePrefixMatch(a, b) {
				t.Fatalf("titlePrefixMatch(%q, %q) = true, want false -- a season/part marker must not register as a duplicate", a, b)
			}
		})
	}
}

// TestResolveMediaItem_PrefixMatchOpensTaskButDoesNotAutoAttach is the
// safety-critical case: a prefix match must never auto-attach, even when
// it's the only candidate. TestNormalizeMediaTitle_CanonicalKeyCollisionSafety
// already shows two different works can share a long, specific opening and
// diverge only in a trailing subtitle -- unlike an exact normalized-title
// match, a prefix relationship alone isn't strong enough evidence to merge
// automatically, so this always stays a human decision.
func TestResolveMediaItem_PrefixMatchOpensTaskButDoesNotAutoAttach(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2043, time.June, 1, 0, 0, 0, 0, time.UTC)
	seedMediaItem(t, db, "死ぬ運命にある悪役令嬢の兄に転生したので、妹を育てて未来を変えたいと思います", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-prefix-no-autoattach",
		Title: "死ぬ運命にある悪役令嬢の兄に転生したので、妹を育てて未来を変えたいと思います ～世界最強はオレだけど、世界最カワは妹に違いない～", MediaType: "manga", ItemRole: "series",
		ReleaseDate: &releaseDate,
	}

	resolution, err := resolveMediaItem(db, incoming, nil)
	if err != nil {
		t.Fatalf("resolveMediaItem: %v", err)
	}
	if resolution.ItemID != uuid.Nil {
		t.Fatalf("resolveMediaItem.ItemID = %v, want uuid.Nil -- a prefix match must never auto-attach", resolution.ItemID)
	}

	var task models.MediaResolutionTask
	if err := db.Where("candidates_json->>'external_id' = ?", incoming.ExternalID).First(&task).Error; err != nil {
		t.Fatalf("expected an open task for the prefix match: %v", err)
	}
	if task.Status != mediaResolutionOpen {
		t.Fatalf("prefix-match task status = %q, want %q -- it must require human review, not get auto-resolved", task.Status, mediaResolutionOpen)
	}
}

// TestResolveMediaItem_AutoAttachesOnUnambiguousTitleYearMatch exercises the
// full resolver: a single unambiguous title/date match must attach to the
// existing item (not mint a duplicate) and leave an audit trail as an
// already-resolved task, requiring no human action.
func TestResolveMediaItem_AutoAttachesOnUnambiguousTitleYearMatch(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2036, time.May, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "Auto Attach Zeta", "anime_season", "season", releaseDate)

	incoming := observations.Media{
		Provider: "bangumi", ExternalID: "integration-auto-attach",
		Title: "Auto Attach Zeta", MediaType: "anime_season", ItemRole: "season",
		ReleaseDate: &releaseDate,
	}

	resolution, err := resolveMediaItem(db, incoming, nil)
	if err != nil {
		t.Fatalf("resolveMediaItem: %v", err)
	}
	if resolution.ItemID != seeded.itemID {
		t.Fatalf("resolveMediaItem attached to %v, want the existing item %v", resolution.ItemID, seeded.itemID)
	}
	if resolution.MatchedBy != mediaMatchTitleYear {
		t.Fatalf("resolveMediaItem.MatchedBy = %q, want %q", resolution.MatchedBy, mediaMatchTitleYear)
	}

	var task models.MediaResolutionTask
	if err := db.Where("candidates_json->>'external_id' = ?", incoming.ExternalID).First(&task).Error; err != nil {
		t.Fatalf("expected an audit task for the auto-attach: %v", err)
	}
	if task.Status != mediaResolutionResolved {
		t.Fatalf("audit task status = %q, want %q -- an unambiguous match must not require human review", task.Status, mediaResolutionResolved)
	}
}

// TestResolveMediaItem_AmbiguousTitleYearStaysOpen exercises the other side:
// when two existing items both match, resolveMediaItem must not guess -- it
// leaves ItemID unset (so the caller mints a fresh item) and opens a task for
// a human instead of silently picking one.
func TestResolveMediaItem_AmbiguousTitleYearStaysOpen(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2037, time.May, 1, 0, 0, 0, 0, time.UTC)
	seedMediaItem(t, db, "Ambiguous Eta", "anime_season", "season", releaseDate)
	seedMediaItem(t, db, "Ambiguous Eta", "anime_season", "season", releaseDate)

	incoming := observations.Media{
		Provider: "bangumi", ExternalID: "integration-ambiguous",
		Title: "Ambiguous Eta", MediaType: "anime_season", ItemRole: "season",
		ReleaseDate: &releaseDate,
	}

	resolution, err := resolveMediaItem(db, incoming, nil)
	if err != nil {
		t.Fatalf("resolveMediaItem: %v", err)
	}
	if resolution.ItemID != uuid.Nil {
		t.Fatalf("resolveMediaItem.ItemID = %v, want uuid.Nil for an ambiguous match", resolution.ItemID)
	}

	var task models.MediaResolutionTask
	if err := db.Where("candidates_json->>'external_id' = ?", incoming.ExternalID).First(&task).Error; err != nil {
		t.Fatalf("expected an open task for the ambiguous match: %v", err)
	}
	if task.Status != mediaResolutionOpen {
		t.Fatalf("ambiguous task status = %q, want %q", task.Status, mediaResolutionOpen)
	}
}
