package imports

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	mediaMatchProviderID    = "provider_id"
	mediaMatchBridge        = "bridge_ref"
	mediaMatchTitleYear     = "title_year"
	mediaResolutionOpen     = "open"
	mediaResolutionDedupe   = "dedupe_candidate"
	mediaResolutionConflict = "progress_conflict"
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
	if len(candidates) > 0 {
		if err := createResolutionTask(tx, media, candidates); err != nil {
			return mediaResolution{}, err
		}
	}
	return mediaResolution{}, nil
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

	var rows []struct{ ScopeID uuid.UUID }
	query := tx.Table("tb_media_titles").Select("tb_media_titles.scope_id").Joins("join tb_media_items on tb_media_items.id = tb_media_titles.scope_id").Where("tb_media_titles.scope_type = ? and extract(year from tb_media_items.release_date) = ?", mediaScopeType, media.ReleaseDate.UTC().Year())
	for _, title := range unique {
		query = query.Or("lower(tb_media_titles.title) = ? and extract(year from tb_media_items.release_date) = ?", strings.ToLower(normalizeMediaTitle(title)), media.ReleaseDate.UTC().Year())
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]uuid.UUID, 0, len(rows))
	seenIDs := make(map[uuid.UUID]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seenIDs[row.ScopeID]; ok {
			continue
		}
		seenIDs[row.ScopeID] = struct{}{}
		result = append(result, row.ScopeID)
	}
	return result, nil
}

func createResolutionTask(tx *gorm.DB, media observations.Media, candidates []uuid.UUID) error {
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
	return tx.Create(&models.MediaResolutionTask{
		ID: id, TaskType: mediaResolutionDedupe, Status: mediaResolutionOpen,
		CandidatesJSON: payload, ResolutionJSON: json.RawMessage(`{}`), CreatedAt: time.Now().UTC(),
	}).Error
}

func normalizeMediaTitle(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(title))), " ")
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
		var existing models.MediaExternalRef
		result := tx.Where("provider = ? and external_id = ?", ref.Provider, ref.ExternalID).First(&existing)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			id, err := ids.New()
			if err != nil {
				return err
			}
			if err := tx.Create(&models.MediaExternalRef{ID: id, ScopeType: mediaScopeType, ScopeID: itemID, Provider: ref.Provider, ExternalID: ref.ExternalID, ExternalURL: ref.ExternalURL, MatchedBy: ref.MatchedBy, Confidence: ref.Confidence, CreatedAt: now}).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		} else if existing.ScopeID != itemID {
			return createExternalRefConflictTask(tx, itemID, ref, existing.ScopeID)
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
