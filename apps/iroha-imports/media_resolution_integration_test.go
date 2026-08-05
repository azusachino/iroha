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

// TestTitleYearCandidates_MatchesAcrossBracketedAnnotation reproduces a real
// prod duplicate: Bangumi's "original" title kept a trailing furigana gloss
// in fullwidth parens that AniList's native title for the same manga
// omitted entirely. An exact string match (even case/whitespace-normalized)
// never catches this; stripping bracketed annotations must.
func TestTitleYearCandidates_MatchesAcrossBracketedAnnotation(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2038, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "Haikei Tengoku no Nee-san Plus（ぷらす）", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-bracket-annotation",
		Title: "Haikei Tengoku no Nee-san Plus", MediaType: "manga", ItemRole: "series",
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

// TestTitleYearCandidates_MatchesAcrossBracketStyle reproduces a second real
// prod duplicate: the same in-title gloss rendered in fullwidth round
// brackets on one provider and CJK angle brackets on the other.
func TestTitleYearCandidates_MatchesAcrossBracketStyle(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2039, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "Exiled Prince『Auto-Craft（AC）』Village Life", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-bracket-style",
		Title: "Exiled Prince『Auto-Craft《AC》』Village Life", MediaType: "manga", ItemRole: "series",
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

// TestTitleYearCandidates_MatchesAcrossFullwidthTilde reproduces a third real
// prod duplicate: the same title with a fullwidth tilde (～) on one provider
// and an ASCII tilde (~) plus different spacing on the other.
func TestTitleYearCandidates_MatchesAcrossFullwidthTilde(t *testing.T) {
	db := openImportsIntegrationDB(t)
	releaseDate := time.Date(2040, time.June, 1, 0, 0, 0, 0, time.UTC)
	seeded := seedMediaItem(t, db, "Strongest Mage  ~Two Past Lives~", "manga", "series", releaseDate)

	incoming := observations.Media{
		Provider: "anilist", ExternalID: "integration-fullwidth-tilde",
		Title: "Strongest Mage ～Two Past Lives～", MediaType: "manga", ItemRole: "series",
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
