package imports

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	mediaMatchProviderID    = "provider_id"
	mediaMatchBridge        = "bridge_ref"
	mediaMatchTitleYear     = "title_year"
	mediaResolutionOpen     = "open"
	mediaResolutionResolved = "resolved"
	mediaResolutionDedupe   = "dedupe_candidate"
	mediaResolutionConflict = "progress_conflict"

	// titleYearToleranceDays widens the release-date match from "same
	// calendar year" to a symmetric window: different providers frequently
	// anchor a work's release date to different events (first chapter vs
	// first volume vs anime adaptation air date), so the same real work can
	// legitimately show up with release dates a year apart across providers.
	titleYearToleranceDays = 400

	// mediaTitleYearConfidence marks a title/date match as heuristic --
	// lower than the implicit 1.0 of an exact provider ref or bridge hit.
	mediaTitleYearConfidence = 0.7
)

// MediaRefBridge contains the two locally cached hops used by the media
// resolver: provider ref -> MAL, then MAL -> AniList. Keeping the cache behind
// this small interface makes refresh policy independent from persistence.
type MediaRefBridge interface {
	Lookup(provider, externalID string) (observations.MediaExternalRef, bool)
}

// StaticMediaRefBridge is populated by a cache refresh job or a test fixture.
// It deliberately returns only exact mappings; fuzzy matching belongs in the
// resolution inbox, never in this bridge.
type StaticMediaRefBridge map[string]observations.MediaExternalRef

func (b StaticMediaRefBridge) Lookup(provider, externalID string) (observations.MediaExternalRef, bool) {
	ref, ok := b[provider+"/"+externalID]
	return ref, ok
}

// TwoHopMediaRefBridge makes the concrete Bangumi -> MAL -> AniList chain
// explicit. The maps are refreshed outside the importer and can be swapped
// atomically by the worker when a newer cache is available.
type TwoHopMediaRefBridge struct {
	BangumiToMAL map[string]string
	MALToAniList map[string]string
}

// LoadTwoHopMediaRefBridge loads the two provider-maintained maps from local
// JSON cache files. Each file is a JSON object keyed by the source ID. Keeping
// refresh/download policy outside the importer avoids network access in jobs.
func LoadTwoHopMediaRefBridge(bangumiPath, malAniListPath string) (TwoHopMediaRefBridge, error) {
	var bridge TwoHopMediaRefBridge
	if bangumiPath != "" {
		if err := loadStringMap(bangumiPath, &bridge.BangumiToMAL); err != nil {
			return TwoHopMediaRefBridge{}, fmt.Errorf("load Bangumi bridge: %w", err)
		}
	}
	if malAniListPath != "" {
		if err := loadStringMap(malAniListPath, &bridge.MALToAniList); err != nil {
			return TwoHopMediaRefBridge{}, fmt.Errorf("load MAL/AniList bridge: %w", err)
		}
	}
	return bridge, nil
}

func loadStringMap(path string, target *map[string]string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var values map[string]string
	if err := json.Unmarshal(body, &values); err != nil {
		return err
	}
	*target = values
	return nil
}

func (b TwoHopMediaRefBridge) Lookup(provider, externalID string) (observations.MediaExternalRef, bool) {
	if provider == "bangumi" {
		malID, ok := b.BangumiToMAL[externalID]
		if !ok {
			return observations.MediaExternalRef{}, false
		}
		anilistID, ok := b.MALToAniList[malID]
		if !ok {
			return observations.MediaExternalRef{Provider: "mal", ExternalID: malID, MatchedBy: mediaMatchBridge}, true
		}
		return observations.MediaExternalRef{Provider: "anilist", ExternalID: anilistID, MatchedBy: mediaMatchBridge}, true
	}
	if provider == "mal" {
		anilistID, ok := b.MALToAniList[externalID]
		if !ok {
			return observations.MediaExternalRef{}, false
		}
		return observations.MediaExternalRef{Provider: "anilist", ExternalID: anilistID, MatchedBy: mediaMatchBridge}, true
	}
	return observations.MediaExternalRef{}, false
}

type mediaResolution struct {
	ItemID     uuid.UUID
	MatchedBy  string
	Confidence *float64
}

func resolveMediaItem(tx *gorm.DB, media observations.Media, bridge MediaRefBridge) (mediaResolution, error) {
	// The media's own identity is the primary ref; adapters may or may not
	// also echo it into ExternalRefs, so check it explicitly to avoid
	// re-creating a duplicate item (and tripping the unique ref constraint)
	// on every sync of the same entry.
	refs := append([]observations.MediaExternalRef{{Provider: media.Provider, ExternalID: media.ExternalID}}, media.ExternalRefs...)
	for _, ref := range refs {
		matched, err := findExternalRef(tx, ref.Provider, ref.ExternalID)
		if err != nil {
			return mediaResolution{}, err
		}
		if matched != nil {
			return mediaResolution{ItemID: matched.ScopeID, MatchedBy: mediaMatchProviderID, Confidence: ref.Confidence}, nil
		}
	}
	if bridge != nil {
		for _, ref := range refs {
			bridgeRef, ok := bridge.Lookup(ref.Provider, ref.ExternalID)
			if !ok {
				continue
			}
			matched, err := findExternalRef(tx, bridgeRef.Provider, bridgeRef.ExternalID)
			if err != nil {
				return mediaResolution{}, err
			}
			if matched != nil {
				return mediaResolution{ItemID: matched.ScopeID, MatchedBy: mediaMatchBridge, Confidence: bridgeRef.Confidence}, nil
			}
		}
	}

	candidates, err := titleYearCandidates(tx, media)
	if err != nil {
		return mediaResolution{}, err
	}
	switch len(candidates) {
	case 0:
		// No exact-normalized-title match. One more, lower-confidence check:
		// a provider sometimes omits a work's trailing tilde-delimited
		// subtitle entirely rather than reformatting it (verified: Bangumi's
		// "original" title for a real manga was an exact prefix of AniList's,
		// missing "～世界最強はオレだけど、世界最カワは妹に違いない～"
		// wholesale). That's too risky to auto-attach on alone -- two
		// different works could share a long, specific opening clause and
		// diverge only in the subtitle, which is exactly the collision
		// TestNormalizeMediaTitle_CanonicalKeyCollisionSafety guards against
		// for bracketed content. So a prefix match only ever opens a task for
		// a human; it never attaches automatically, regardless of how many
		// candidates it finds.
		prefixCandidates, err := titlePrefixCandidates(tx, media)
		if err != nil {
			return mediaResolution{}, err
		}
		if len(prefixCandidates) > 0 {
			if err := createResolutionTask(tx, media, prefixCandidates, mediaResolutionOpen, json.RawMessage(`{}`)); err != nil {
				return mediaResolution{}, err
			}
		}
	case 1:
		// Unambiguous: exactly one existing item matches title, media type,
		// and release date within tolerance. Attach to it instead of minting
		// a duplicate; log an already-resolved task purely as an audit trail
		// so this decision stays inspectable without requiring human review.
		if err := createResolutionTask(tx, media, candidates, mediaResolutionResolved, autoMergedResolutionJSON()); err != nil {
			return mediaResolution{}, err
		}
		confidence := mediaTitleYearConfidence
		return mediaResolution{ItemID: candidates[0], MatchedBy: mediaMatchTitleYear, Confidence: &confidence}, nil
	default:
		// Ambiguous: more than one existing item matches. Auto-attaching
		// could silently merge into the wrong one, so this stays a human
		// decision -- leave the task open and let a fresh item get created,
		// same as the no-candidate case.
		if err := createResolutionTask(tx, media, candidates, mediaResolutionOpen, json.RawMessage(`{}`)); err != nil {
			return mediaResolution{}, err
		}
	}
	return mediaResolution{}, nil
}

func autoMergedResolutionJSON() json.RawMessage {
	return json.RawMessage(`{"decision":"auto_merged","matched_by":"title_year"}`)
}

func findExternalRef(tx *gorm.DB, provider, externalID string) (*models.MediaExternalRef, error) {
	if provider == "" || externalID == "" {
		return nil, nil
	}
	var ref models.MediaExternalRef
	result := tx.Where("provider = ? and external_id = ?", provider, externalID).First(&ref)
	if errorsIsNotFound(result.Error) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &ref, nil
}

// scopedTitleCandidateRows fetches every (scope_id, title) row in the same
// media_type/item_role/release-date-window scope as media -- the shared
// fetch behind both titleYearCandidates (exact match) and
// titlePrefixCandidates (prefix match). See titleYearCandidates for why
// title filtering happens in Go instead of SQL.
func scopedTitleCandidateRows(tx *gorm.DB, media observations.Media) ([]struct {
	ScopeID uuid.UUID
	Title   string
}, error,
) {
	var rows []struct {
		ScopeID uuid.UUID
		Title   string
	}
	if media.ReleaseDate == nil {
		return rows, nil
	}
	windowStart := media.ReleaseDate.AddDate(0, 0, -titleYearToleranceDays)
	windowEnd := media.ReleaseDate.AddDate(0, 0, titleYearToleranceDays)
	// media_type/item_role must match too: the same franchise legitimately
	// has separate items (an anime season and its manga adaptation, a TV
	// series and its movie) that share a title and a nearby release date but
	// must never be merged into each other.
	err := tx.Table("tb_media_titles").Select("tb_media_titles.scope_id, tb_media_titles.title").
		Joins("join tb_media_items on tb_media_items.id = tb_media_titles.scope_id").
		Where("tb_media_titles.scope_type = ? and tb_media_items.media_type = ? and tb_media_items.item_role = ? and tb_media_items.release_date between ? and ?",
			mediaScopeType, mediaTypeOrDefault(media.MediaType), itemRoleOrDefault(media.ItemRole), windowStart, windowEnd).
		Find(&rows).Error
	return rows, err
}

// incomingTitleSet dedupes and normalizes media's own title + alternate
// titles into a lookup set, keyed by their normalized form.
func incomingTitleSet(media observations.Media) map[string]struct{} {
	titles := make([]string, 0, len(media.Titles)+1)
	if media.Title != "" {
		titles = append(titles, media.Title)
	}
	for _, title := range media.Titles {
		if title.Title != "" {
			titles = append(titles, title.Title)
		}
	}
	normalized := make(map[string]struct{}, len(titles))
	for _, title := range titles {
		if key := normalizeMediaTitle(title); key != "" {
			normalized[key] = struct{}{}
		}
	}
	return normalized
}

func titleYearCandidates(tx *gorm.DB, media observations.Media) ([]uuid.UUID, error) {
	normalizedIncoming := incomingTitleSet(media)
	if len(normalizedIncoming) == 0 {
		return nil, nil
	}
	rows, err := scopedTitleCandidateRows(tx, media)
	if err != nil {
		return nil, err
	}
	result := make([]uuid.UUID, 0, len(rows))
	seenIDs := make(map[uuid.UUID]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := normalizedIncoming[normalizeMediaTitle(row.Title)]; !ok {
			continue
		}
		if _, ok := seenIDs[row.ScopeID]; ok {
			continue
		}
		seenIDs[row.ScopeID] = struct{}{}
		result = append(result, row.ScopeID)
	}
	return result, nil
}

// titlePrefixMinRunes bounds how short a shared prefix can be before two
// titles are considered a prefix match. It is a hardcoded heuristic, not a
// guarantee -- tuned against two real prod pairs (needed to accept a
// 14-rune shared prefix, needed to reject a 7-rune one) and nothing more
// rigorous than that. This mechanism only ever opens a review task, never
// auto-attaches (see resolveMediaItem), so the cost of setting it too low is
// bounded (a dismissible false-positive task) rather than unbounded (a bad
// merge) -- but it is still just a tuned number, and titleSeasonMarkerSuffix
// below exists precisely because the number alone was verified insufficient:
// at this threshold "My Hero Academia" vs "My Hero Academia Season 2" also
// passes the length check, and those are not duplicates.
const titlePrefixMinRunes = 12

// titleRemainderMinRunes rejects a prefix match when the "extra" content on
// the longer title is too short to plausibly be an omitted subtitle clause.
// This is the general-purpose backstop titleSeasonMarkerPattern's keyword
// list can't be: a franchise doesn't have to spell "Season 2" to mean it --
// Gintama's sequels are literally the base title plus a single mark (銀魂
// -> 銀魂゜, then 銀魂°), which is 1 rune and matches no keyword pattern.
// Measured directly against every case found in prod: every false-positive
// remainder (season/part markers) was 1-7 runes; every genuine omitted
// subtitle was 25+ runes. 10 sits in the gap between them.
const titleRemainderMinRunes = 10

// titleSeasonMarkerPattern matches season/part/cour markers so a prefix
// match can be rejected when the "extra" content on the longer title is one
// of these -- verified necessary against real prod false positives ("My
// Hero Academia" vs "My Hero Academia Season 2", "進撃の巨人 第三季" vs
// "... 第三季 Part.2", "Komi Can't Communicate" vs "... Part 2"): a missing
// season/part marker means "different installment of the same franchise,"
// the opposite of a duplicate, and no rune-count threshold can distinguish
// that from a genuinely omitted subtitle -- the trailing content itself has
// to be inspected. Matched against the remainder after the shared prefix,
// which has already been through normalizeMediaTitle (NFKC-folded, lowered,
// whitespace-stripped), so "Season 2" arrives as "season2".
var titleSeasonMarkerPattern = regexp.MustCompile(
	`^(season|part|cour|ova|movie|special|finalseason)[.\-:]?\d*` +
		`|^第[0-9〇一二三四五六七八九十百]+[期部季話话弾巻篇]` +
		`|^(最終季|最终季|劇場版|完結編|完结篇|終章|终章)`,
)

// titlePrefixCandidates finds items in scope whose title is a strict prefix
// or superset of one of media's own titles -- the case exact matching can't
// cover: one provider's title running straight to the end where the other
// has an additional trailing tilde-delimited subtitle (verified real case:
// Bangumi's title for a manga was AniList's title with
// "～世界最強はオレだけど、世界最カワは妹に違いない～" entirely absent,
// not reformatted). Deliberately never used for auto-attach -- see the
// call site in resolveMediaItem.
func titlePrefixCandidates(tx *gorm.DB, media observations.Media) ([]uuid.UUID, error) {
	normalizedIncoming := incomingTitleSet(media)
	if len(normalizedIncoming) == 0 {
		return nil, nil
	}
	rows, err := scopedTitleCandidateRows(tx, media)
	if err != nil {
		return nil, err
	}
	result := make([]uuid.UUID, 0, len(rows))
	seenIDs := make(map[uuid.UUID]struct{}, len(rows))
	for _, row := range rows {
		rowNorm := normalizeMediaTitle(row.Title)
		matched := false
		for incoming := range normalizedIncoming {
			if titlePrefixMatch(incoming, rowNorm) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if _, ok := seenIDs[row.ScopeID]; ok {
			continue
		}
		seenIDs[row.ScopeID] = struct{}{}
		result = append(result, row.ScopeID)
	}
	return result, nil
}

// titlePrefixMatch reports whether a and b are the same string up to the
// point where the shorter one ends -- i.e. one is a strict prefix of the
// other -- and that shared prefix meets titlePrefixMinRunes. Equal strings
// return false: that's an exact match, already handled by
// titleYearCandidates, and treating it as a "prefix" here would just
// duplicate that candidate into the lower-confidence pool.
func titlePrefixMatch(a, b string) bool {
	if a == b {
		return false
	}
	shorter, longer := a, b
	if len([]rune(a)) > len([]rune(b)) {
		shorter, longer = b, a
	}
	if len([]rune(shorter)) < titlePrefixMinRunes {
		return false
	}
	if !strings.HasPrefix(longer, shorter) {
		return false
	}
	remainder := longer[len(shorter):]
	if len([]rune(remainder)) < titleRemainderMinRunes {
		return false
	}
	return !titleSeasonMarkerPattern.MatchString(remainder)
}

func mediaTypeOrDefault(mediaType string) string {
	if mediaType == "" {
		return mediaUnknownValue
	}
	return mediaType
}

func createResolutionTask(tx *gorm.DB, media observations.Media, candidates []uuid.UUID, status string, resolutionJSON json.RawMessage) error {
	payload, err := json.Marshal(map[string]any{
		"provider":     media.Provider,
		"external_id":  media.ExternalID,
		"title":        media.Title,
		"release_date": releaseDate(media.ReleaseDate),
		"candidates":   candidates,
	})
	if err != nil {
		return err
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	task := models.MediaResolutionTask{
		ID: id, TaskType: mediaResolutionDedupe, Status: status,
		CandidatesJSON: payload, ResolutionJSON: resolutionJSON, CreatedAt: time.Now().UTC(),
	}
	if status != mediaResolutionOpen {
		now := time.Now().UTC()
		task.ResolvedAt = &now
	}
	return tx.Create(&task).Error
}

// parenAnnotationPattern and angleAnnotationPattern locate bracketed spans
// that MIGHT be reading-gloss annotations. NFKC folds fullwidth parens to
// ASCII "()" before either pattern runs, so parenAnnotationPattern only
// needs to match one paren style; the CJK angle-bracket style is separate.
var (
	parenAnnotationPattern = regexp.MustCompile(`\(([^()]*)\)`)
	angleAnnotationPattern = regexp.MustCompile(`《([^《》]*)》`)
	// pureKanaPattern is what actually decides whether a bracketed span gets
	// stripped: only if its content is entirely hiragana/katakana (plus the
	// katakana long-vowel mark ー and middle dot ・), i.e. a phonetic reading
	// gloss with no identity-bearing content. A blanket "strip anything
	// bracketed" is unsafe -- verified: it collapsed "薬屋のひとりごと（第二期）"
	// (a real Season 2 marker in kanji) onto "薬屋のひとりごと" (Season 1), and
	// "Fullmetal Alchemist (2003)" onto "... (2009)", a different remake.
	// Kanji, digits, and Latin letters all fail this pattern and are left in
	// place, keeping such disambiguators distinct.
	pureKanaPattern = regexp.MustCompile(`^[\p{Hiragana}\p{Katakana}ー・]*$`)
)

func stripKanaOnlyAnnotations(title string) string {
	strip := func(re *regexp.Regexp, s string) string {
		return re.ReplaceAllStringFunc(s, func(match string) string {
			if sub := re.FindStringSubmatch(match); len(sub) == 2 && pureKanaPattern.MatchString(sub[1]) {
				return ""
			}
			return match
		})
	}
	title = strip(parenAnnotationPattern, title)
	title = strip(angleAnnotationPattern, title)
	return title
}

// normalizeMediaTitle builds a comparison-only key, never a display value:
// it removes whitespace entirely rather than collapsing it, because
// providers disagree on whether a space separates a title from a
// tilde/dash-delimited subtitle (verified against a real prod pair --
// "...好きすぎる～真摯..." vs "...好きすぎる ～真摯..." -- the extra space
// isn't redundant on either side, so collapsing runs of whitespace to one
// space each still leaves them unequal; only removing whitespace altogether
// makes them comparable).
func normalizeMediaTitle(title string) string {
	folded := norm.NFKC.String(title)
	stripped := stripKanaOnlyAnnotations(folded)
	return strings.Join(strings.Fields(strings.ToLower(stripped)), "")
}

func releaseDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

func errorsIsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func requireMediaResolution(resolution mediaResolution) error {
	if resolution.ItemID == uuid.Nil {
		return fmt.Errorf("media resolution has no item")
	}
	return nil
}

func persistMediaMetadata(tx *gorm.DB, itemID uuid.UUID, media observations.Media) error {
	now := time.Now().UTC()
	for _, title := range media.Titles {
		if strings.TrimSpace(title.Title) == "" {
			continue
		}
		var existing models.MediaTitle
		result := tx.Where("scope_type = ? and scope_id = ? and title = ? and provider = ?", mediaScopeType, itemID, title.Title, title.Provider).First(&existing)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			id, err := ids.New()
			if err != nil {
				return err
			}
			if err := tx.Create(&models.MediaTitle{ID: id, ScopeType: mediaScopeType, ScopeID: itemID, Title: title.Title, Language: title.Language, Script: title.Script, Region: title.Region, TitleKind: title.TitleKind, Provider: title.Provider, IsPrimary: title.IsPrimary, Confidence: title.Confidence, CreatedAt: now}).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		}
	}
	for _, ref := range media.ExternalRefs {
		if ref.Provider == "" || ref.ExternalID == "" {
			continue
		}
		id, err := ids.New()
		if err != nil {
			return err
		}
		// INSERT ... ON CONFLICT DO NOTHING, not lookup-then-insert: concurrent
		// workers can resolve the same (provider, external_id) at once, and a
		// plain check-then-create races the unique constraint.
		newRef := models.MediaExternalRef{ID: id, ScopeType: mediaScopeType, ScopeID: itemID, Provider: ref.Provider, ExternalID: ref.ExternalID, ExternalURL: ref.ExternalURL, MatchedBy: ref.MatchedBy, Confidence: ref.Confidence, CreatedAt: now}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "external_id"}},
			DoNothing: true,
		}).Create(&newRef)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			var existing models.MediaExternalRef
			if err := tx.Where("provider = ? and external_id = ?", ref.Provider, ref.ExternalID).First(&existing).Error; err != nil {
				return err
			}
			if existing.ScopeID != itemID {
				return createExternalRefConflictTask(tx, itemID, ref, existing.ScopeID)
			}
		}
	}
	return nil
}

func createProgressConflictTask(tx *gorm.DB, media observations.Media, itemID uuid.UUID, localStatus, remoteStatus string) error {
	var open int64
	if err := tx.Model(&models.MediaResolutionTask{}).
		Where("task_type = ? and status = ? and candidates_json->>'media_item_id' = ?", mediaResolutionConflict, mediaResolutionOpen, itemID.String()).
		Count(&open).Error; err != nil {
		return err
	}
	if open > 0 {
		return nil
	}
	payload, err := json.Marshal(map[string]any{
		"provider": media.Provider, "external_id": media.ExternalID, "media_item_id": itemID,
		"local_status": localStatus, "remote_status": remoteStatus,
	})
	if err != nil {
		return err
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	return tx.Create(&models.MediaResolutionTask{ID: id, TaskType: mediaResolutionConflict, Status: mediaResolutionOpen, CandidatesJSON: payload, ResolutionJSON: json.RawMessage(`{}`), CreatedAt: time.Now().UTC()}).Error
}

func createExternalRefConflictTask(tx *gorm.DB, itemID uuid.UUID, ref observations.MediaExternalRef, existingItemID uuid.UUID) error {
	payload, err := json.Marshal(map[string]any{
		"provider": ref.Provider, "external_id": ref.ExternalID,
		"incoming_item_id": itemID, "existing_item_id": existingItemID,
	})
	if err != nil {
		return err
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	return tx.Create(&models.MediaResolutionTask{ID: id, TaskType: mediaResolutionConflict, Status: mediaResolutionOpen, CandidatesJSON: payload, ResolutionJSON: json.RawMessage(`{}`), CreatedAt: time.Now().UTC()}).Error
}
