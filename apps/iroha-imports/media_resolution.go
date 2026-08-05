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
		// Nothing shares this title in the date window -- a genuinely new item.
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

func titleYearCandidates(tx *gorm.DB, media observations.Media) ([]uuid.UUID, error) {
	if media.ReleaseDate == nil {
		return nil, nil
	}
	titles := make([]string, 0, len(media.Titles)+1)
	if media.Title != "" {
		titles = append(titles, media.Title)
	}
	for _, title := range media.Titles {
		if title.Title != "" {
			titles = append(titles, title.Title)
		}
	}
	seen := make(map[string]struct{}, len(titles))
	unique := titles[:0]
	for _, title := range titles {
		key := normalizeMediaTitle(title)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			unique = append(unique, title)
		}
	}
	if len(unique) == 0 {
		return nil, nil
	}

	normalizedIncoming := make(map[string]struct{}, len(unique))
	for _, title := range unique {
		normalizedIncoming[normalizeMediaTitle(title)] = struct{}{}
	}

	windowStart := media.ReleaseDate.AddDate(0, 0, -titleYearToleranceDays)
	windowEnd := media.ReleaseDate.AddDate(0, 0, titleYearToleranceDays)
	// media_type/item_role must match too: the same franchise legitimately
	// has separate items (an anime season and its manga adaptation, a TV
	// series and its movie) that share a title and a nearby release date but
	// must never be merged into each other. Title filtering happens in Go,
	// not SQL: normalizeMediaTitle's NFKC fold + bracket-annotation strip has
	// no cheap SQL equivalent, and different providers routinely render the
	// same title with different bracket styles or trailing reading glosses
	// (verified against real prod duplicates an exact SQL string match
	// missed). The scope filters below keep this fetch small.
	var rows []struct {
		ScopeID uuid.UUID
		Title   string
	}
	query := tx.Table("tb_media_titles").Select("tb_media_titles.scope_id, tb_media_titles.title").
		Joins("join tb_media_items on tb_media_items.id = tb_media_titles.scope_id").
		Where("tb_media_titles.scope_type = ? and tb_media_items.media_type = ? and tb_media_items.item_role = ? and tb_media_items.release_date between ? and ?",
			mediaScopeType, mediaTypeOrDefault(media.MediaType), itemRoleOrDefault(media.ItemRole), windowStart, windowEnd)
	if err := query.Find(&rows).Error; err != nil {
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

// titleAnnotationPattern strips parenthetical/bracketed reading glosses and
// alt-spelling annotations. Different providers render the same gloss with
// different bracket styles for the same title (e.g. a Bangumi "original"
// title keeping a trailing furigana note in （）that an AniList native title
// omits, or the same in-title gloss rendered in （）on one side and 《》 on the
// other) -- verified against real prod duplicates that survived an exact
// title match. NFKC folds fullwidth parens to ASCII "()" before this pattern
// runs, so it only needs to match one paren style plus the CJK angle-bracket
// style separately.
var titleAnnotationPattern = regexp.MustCompile(`\([^()]*\)|《[^《》]*》`)

func normalizeMediaTitle(title string) string {
	folded := norm.NFKC.String(title)
	stripped := titleAnnotationPattern.ReplaceAllString(folded, "")
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(stripped))), " ")
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
