package imports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/i18n"
	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) persistMediaTx(tx *gorm.DB, rawFile models.RawFile, parsed []observations.Media, snapshot models.ImportSnapshot) error {
	if err := tx.Create(&snapshot).Error; err != nil {
		return err
	}
	for _, media := range parsed {
		if err := persistMediaObservation(tx, rawFile, snapshot, media, s.mediaBridge); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) persistMediaHistoryTx(tx *gorm.DB, rawFile models.RawFile, parsed []observations.MediaHistory, snapshot models.ImportSnapshot) error {
	if err := tx.Create(&snapshot).Error; err != nil {
		return err
	}
	for _, history := range parsed {
		itemID, err := ensureMediaItem(tx, history.Media, s.mediaBridge)
		if err != nil {
			return err
		}
		if err := persistMediaMetadata(tx, itemID, history.Media); err != nil {
			return err
		}
		if err := persistMediaCreators(tx, itemID, history.Media); err != nil {
			return err
		}
		if err := persistMediaRelations(tx, itemID, history.Media); err != nil {
			return err
		}
		for _, update := range history.Updates {
			if err := persistMediaStateUpdate(tx, rawFile, snapshot, itemID, update); err != nil {
				return err
			}
		}
	}
	return nil
}

func persistMediaStateUpdate(tx *gorm.DB, rawFile models.RawFile, snapshot models.ImportSnapshot, itemID uuid.UUID, update observations.MediaStateUpdate) error {
	if update.SourceEventID == "" {
		return errors.New("media provider activity missing source event id")
	}
	if update.EffectiveAt.IsZero() {
		return fmt.Errorf("media provider activity %q is missing effective_at", update.SourceEventID)
	}
	update.EffectiveAt = update.EffectiveAt.UTC()
	fingerprintBytes, err := json.Marshal(update)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(fingerprintBytes)
	fingerprint := hex.EncodeToString(digest[:])
	const sourceKind = coreimports.KindAniList
	var existing models.MediaStateHistory
	result := tx.Where("media_item_id = ? and source_kind = ? and source_event_id = ?", itemID, sourceKind, update.SourceEventID).
		Order("observed_at desc, created_at desc, id desc").First(&existing)
	if result.Error == nil && existing.StateFingerprint == fingerprint {
		return nil
	}
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	observedAt := rawFile.CreatedAt
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	if rawFile.ObservedAt != nil {
		observedAt = *rawFile.ObservedAt
	}
	if snapshot.TakenAt != nil && !snapshot.TakenAt.IsZero() {
		observedAt = *snapshot.TakenAt
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.MediaStateHistory{
		ID:               id,
		MediaItemID:      itemID,
		SourceKind:       sourceKind,
		SourceEventID:    update.SourceEventID,
		ObservedAt:       observedAt.UTC(),
		EffectiveAt:      &update.EffectiveAt,
		TimeBasis:        "provider_activity",
		ChangeKind:       "provider_activity",
		StateFingerprint: fingerprint,
		Status:           update.Status,
		Unit:             update.Unit,
		Position:         update.Position,
		Total:            update.Total,
		ProgressPercent:  update.ProgressPercent,
		Rating:           update.Rating,
		RatingScale:      update.RatingScale,
		Note:             update.Note,
		RepeatCount:      update.RepeatCount,
		RawFileID:        &rawFile.ID,
		CreatedAt:        time.Now().UTC(),
	}).Error
}

func persistMediaObservation(tx *gorm.DB, rawFile models.RawFile, snapshot models.ImportSnapshot, media observations.Media, bridge MediaRefBridge) error {
	itemID, err := ensureMediaItem(tx, media, bridge)
	if err != nil {
		return err
	}

	if err := persistMediaMetadata(tx, itemID, media); err != nil {
		return err
	}
	if err := persistMediaCreators(tx, itemID, media); err != nil {
		return err
	}
	listName := media.Provider + " library"
	var list models.MediaList
	result := tx.Where("source_kind = ? and name = ?", media.Provider, listName).First(&list)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		listID, err := ids.New()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		list = models.MediaList{
			ID:         listID,
			Name:       listName,
			ListKind:   mediaListKind,
			SourceKind: media.Provider,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(&list).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	}
	var listItem models.MediaListItem
	result = tx.Where("list_id = ? and media_item_id = ?", list.ID, itemID).First(&listItem)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		listItemID, err := ids.New()
		if err != nil {
			return err
		}
		if err := tx.Create(&models.MediaListItem{ID: listItemID, ListID: list.ID, MediaItemID: itemID, CreatedAt: time.Now().UTC()}).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	}

	if err := persistMediaRelations(tx, itemID, media); err != nil {
		return err
	}
	if err := persistMediaEvents(tx, rawFile, itemID, media); err != nil {
		return err
	}
	if err := persistMediaStateHistory(tx, rawFile, snapshot, itemID, media); err != nil {
		return err
	}
	return upsertMediaProgress(tx, rawFile, itemID, media)
}

func ensureMediaItem(tx *gorm.DB, media observations.Media, bridge MediaRefBridge) (uuid.UUID, error) {
	if media.Provider == "" {
		return uuid.Nil, errors.New("parsed media missing provider")
	}
	if media.ExternalID == "" {
		return uuid.Nil, errors.New("parsed media missing external id")
	}
	if media.Title == "" {
		return uuid.Nil, errors.New("parsed media missing title")
	}

	resolution, err := resolveMediaItem(tx, media, bridge)
	if err != nil {
		return uuid.Nil, err
	}
	var externalRef models.MediaExternalRef
	if resolution.ItemID == uuid.Nil {
		workID, err := ids.New()
		if err != nil {
			return uuid.Nil, err
		}
		itemID, err := ids.New()
		if err != nil {
			return uuid.Nil, err
		}
		now := time.Now().UTC()
		// Native First: the authored title claim wins at creation, with the
		// provider's display title as the fallback when no claim resolves.
		// media.Language is the provider's declaration about the work itself; a
		// provider that only knows the consumed edition's language must leave it
		// empty, because the projection never guesses a language from script.
		titleResolution := i18n.ResolveWorkTitles("", "", media.Language, titleCandidates(media.Titles))
		primaryTitle := titleResolution.PrimaryTitle
		if primaryTitle == "" {
			primaryTitle = media.Title
		}
		originalTitle := titleResolution.OriginalTitle
		if originalTitle == "" {
			originalTitle = media.Title
		}
		work := models.MediaWork{
			ID:               workID,
			WorkKind:         mediaWorkKind,
			PrimaryTitle:     primaryTitle,
			OriginalTitle:    originalTitle,
			OriginalLanguage: titleResolution.OriginalLanguage,
			Description:      media.Description,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := tx.Create(&work).Error; err != nil {
			return uuid.Nil, err
		}
		releaseDate, releasePrecision := canonicalReleaseDate(media)
		item := models.MediaItem{
			ID:                   itemID,
			WorkID:               &workID,
			MediaType:            media.MediaType,
			ItemRole:             itemRoleOrDefault(media.ItemRole),
			Title:                media.Title,
			OriginalTitle:        media.Title,
			ReleaseDate:          releaseDate,
			ReleaseDatePrecision: releasePrecision,
			SeasonNumber:         media.SeasonNumber,
			EpisodeNumber:        media.EpisodeNumber,
			ChapterNumber:        media.ChapterNumber,
			VolumeNumber:         media.VolumeNumber,
			DurationSeconds:      media.DurationSeconds,
			PageCount:            media.PageCount,
			EpisodeCount:         media.EpisodeCount,
			ChapterCount:         media.ChapterCount,
			Language:             media.Language,
			Country:              media.Country,
			Publisher:            media.Publisher,
			CoverImageURL:        media.CoverImageURL,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if item.MediaType == "" {
			item.MediaType = mediaUnknownValue
		}
		if err := tx.Create(&item).Error; err != nil {
			return uuid.Nil, err
		}
		externalRefID, err := ids.New()
		if err != nil {
			return uuid.Nil, err
		}
		externalRef = models.MediaExternalRef{
			ID:         externalRefID,
			ScopeType:  mediaScopeType,
			ScopeID:    itemID,
			Provider:   media.Provider,
			ExternalID: media.ExternalID,
			MatchedBy:  "provider_id",
			CreatedAt:  now,
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "external_id"}},
			DoNothing: true,
		}).Create(&externalRef)
		if result.Error != nil {
			return uuid.Nil, result.Error
		}
		if result.RowsAffected == 0 {
			var existing models.MediaExternalRef
			if err := tx.Where("provider = ? and external_id = ?", media.Provider, media.ExternalID).First(&existing).Error; err != nil {
				return uuid.Nil, err
			}
			if err := tx.Delete(&models.MediaItem{}, itemID).Error; err != nil {
				return uuid.Nil, err
			}
			if err := tx.Delete(&models.MediaWork{}, workID).Error; err != nil {
				return uuid.Nil, err
			}
			resolution.ItemID = existing.ScopeID
		}
		if resolution.ItemID == uuid.Nil {
			resolution.ItemID = itemID
		}
	} else {
		if err := requireMediaResolution(resolution); err != nil {
			return uuid.Nil, err
		}
		itemID := resolution.ItemID
		// The item is "owned" by this provider when its own primary ref already
		// pointed here before this sync: then a fresh parse (e.g. a reprocess
		// after a parser fix) may overwrite the item's core fields. If the item
		// was reached via a bridge/title match from a different provider, only
		// fill empty fields so we don't clobber the owner's values each sync.
		ownedItem := !resolution.Explicit || resolution.OwnsItem
		lookupErr := tx.Where("scope_type = ? and scope_id = ? and provider = ? and external_id = ?", mediaScopeType, itemID, media.Provider, media.ExternalID).First(&externalRef).Error
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			ownedItem = false
			if resolution.Explicit {
				// An explicit decision is stronger than the ref's current scope.
				// Rebind it here so an older replay or a concurrent bridge refresh
				// cannot undo the recorded choice.
				if err := tx.Model(&models.MediaExternalRef{}).
					Where("provider = ? and external_id = ?", media.Provider, media.ExternalID).
					Updates(map[string]any{"scope_id": itemID, "matched_by": mediaMatchDecision, "confidence": nil}).Error; err != nil {
					return uuid.Nil, err
				}
				if err := tx.Where("provider = ? and external_id = ?", media.Provider, media.ExternalID).First(&externalRef).Error; err != nil {
					return uuid.Nil, err
				}
			} else {
				refID, idErr := ids.New()
				if idErr != nil {
					return uuid.Nil, idErr
				}
				// INSERT ... ON CONFLICT DO NOTHING: (provider, external_id) is
				// unique across all items, so a concurrent job may have already
				// claimed this ref for a different item while we were resolving.
				newRef := models.MediaExternalRef{ID: refID, ScopeType: mediaScopeType, ScopeID: itemID, Provider: media.Provider, ExternalID: media.ExternalID, MatchedBy: resolution.MatchedBy, Confidence: resolution.Confidence, CreatedAt: time.Now().UTC()}
				result := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "provider"}, {Name: "external_id"}},
					DoNothing: true,
				}).Create(&newRef)
				if result.Error != nil {
					return uuid.Nil, result.Error
				}
				if result.RowsAffected == 0 {
					var existing models.MediaExternalRef
					if err := tx.Where("provider = ? and external_id = ?", media.Provider, media.ExternalID).First(&existing).Error; err != nil {
						return uuid.Nil, err
					}
					if existing.ScopeID != itemID {
						// The provider ref is already owned elsewhere. Preserve the
						// incoming raw observation, but do not block the import or
						// create a blocking action for a secondary-ref disagreement.
						externalRef = existing
						resolution.ItemID = existing.ScopeID
					} else {
						externalRef = existing
					}
				} else {
					externalRef = newRef
				}
			}
		} else if lookupErr != nil {
			return uuid.Nil, lookupErr
		}
		if err := refreshMediaItemFields(tx, itemID, media, ownedItem); err != nil {
			return uuid.Nil, err
		}
	}
	return resolution.ItemID, nil
}

// persistMediaCreators records the credited creator roles a source reports for
// an item. A creator row is canonical by name; each role is scoped to the item,
// so one edition's translator cannot become another edition's author. Replaying
// the same source snapshot reuses the existing rows instead of duplicating
// them, and a single observation that repeats a credit is collapsed here.
func persistMediaCreators(tx *gorm.DB, itemID uuid.UUID, media observations.Media) error {
	now := time.Now().UTC()
	seen := make(map[string]struct{}, len(media.Creators))
	for _, credit := range media.Creators {
		name := strings.TrimSpace(credit.Name)
		if name == "" {
			continue
		}
		role := strings.TrimSpace(credit.Role)
		if role == "" {
			role = observations.MediaCreatorRoleAuthor
		}
		key := name + "\x00" + role
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}

		var creator models.MediaCreator
		result := tx.Where("name = ?", name).First(&creator)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			creatorID, err := ids.New()
			if err != nil {
				return err
			}
			creator = models.MediaCreator{ID: creatorID, Name: name, SortName: name, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&creator).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		}

		var existing models.MediaCreatorRole
		result = tx.Where("creator_id = ? and scope_type = ? and scope_id = ? and role = ?", creator.ID, mediaScopeType, itemID, role).First(&existing)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			roleID, err := ids.New()
			if err != nil {
				return err
			}
			if err := tx.Create(&models.MediaCreatorRole{
				ID: roleID, CreatorID: creator.ID, ScopeType: mediaScopeType, ScopeID: itemID,
				Role: role, Provider: credit.Provider, CreatedAt: now,
			}).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

// persistMediaRelations writes the provider relation graph (adaptation/sequel/
// etc.) into tb_media_relations. Both endpoints must already resolve to items
// via external refs; edges to items not (yet) in the collection are skipped
// rather than materialized as stub items. The unique constraint dedupes edges
// re-observed across syncs.
func persistMediaRelations(tx *gorm.DB, itemID uuid.UUID, media observations.Media) error {
	for _, rel := range media.Relations {
		if rel.RelationType == "" || rel.ToExternalID == "" {
			continue
		}
		toRef, err := findExternalRef(tx, rel.Provider, rel.ToExternalID)
		if err != nil {
			return err
		}
		if toRef == nil {
			continue
		}
		fromID := itemID
		if rel.FromExternalID != "" && rel.FromExternalID != media.ExternalID {
			fromRef, err := findExternalRef(tx, rel.Provider, rel.FromExternalID)
			if err != nil {
				return err
			}
			if fromRef == nil {
				continue
			}
			fromID = fromRef.ScopeID
		}
		id, err := ids.New()
		if err != nil {
			return err
		}
		relation := models.MediaRelation{
			ID: id, FromType: mediaScopeType, FromID: fromID, ToType: mediaScopeType, ToID: toRef.ScopeID,
			RelationType: rel.RelationType, Provider: rel.Provider, Confidence: rel.Confidence, CreatedAt: time.Now().UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relation).Error; err != nil {
			return err
		}
	}
	return nil
}

// persistMediaEvents appends only exact events explicitly supplied by an
// adapter. Provider list snapshots do not enter this table, and there is no
// fallback from a fuzzy completion date or import time.
func persistMediaEvents(tx *gorm.DB, rawFile models.RawFile, itemID uuid.UUID, media observations.Media) error {
	for _, event := range media.Events {
		if !observations.ValidMediaEventType(event.EventType) {
			return fmt.Errorf("media exact event has invalid event type %q", event.EventType)
		}
		if event.EventAt.IsZero() {
			return fmt.Errorf("media exact event %q is missing event_at", event.EventType)
		}
		event.EventAt = event.EventAt.UTC()
		unchanged, err := latestEventUnchanged(tx, itemID, rawFile.SourceKind, event)
		if err != nil {
			return err
		}
		if unchanged {
			continue
		}
		id, err := ids.New()
		if err != nil {
			return err
		}
		if err := tx.Create(&models.MediaConsumptionEvent{
			ID:              id,
			MediaItemID:     itemID,
			EventType:       event.EventType,
			EventAt:         event.EventAt,
			SourceKind:      rawFile.SourceKind,
			SourceEventID:   event.SourceEventID,
			Unit:            event.Unit,
			Position:        event.Position,
			Total:           event.Total,
			ProgressPercent: event.ProgressPercent,
			Rating:          event.Rating,
			RatingScale:     event.RatingScale,
			Note:            event.Note,
			RawFileID:       &rawFile.ID,
			CreatedAt:       time.Now().UTC(),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// persistMediaStateHistory records the provider's typed current state. The
// fingerprint is calculated from canonical fields, so repeated snapshots and
// reprocessing do not create a second observation. A provider record timestamp
// remains ordering/provenance only; observed_at is when Iroha saw the state.
func persistMediaStateHistory(tx *gorm.DB, rawFile models.RawFile, snapshot models.ImportSnapshot, itemID uuid.UUID, media observations.Media) error {
	observedAt := rawFile.CreatedAt
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	if rawFile.ObservedAt != nil {
		observedAt = *rawFile.ObservedAt
	}
	if snapshot.TakenAt != nil && !snapshot.TakenAt.IsZero() {
		observedAt = *snapshot.TakenAt
	}
	observedAt = observedAt.UTC()
	sourceEventID := media.StateSourceID
	if sourceEventID == "" {
		sourceEventID = media.ExternalID
	}
	startedValue, startedPrecision := partialDateFields(media.StartedOn)
	completedValue, completedPrecision := partialDateFields(media.CompletedOn)
	state := struct {
		Status               string
		Unit                 string
		Position             *float64
		Total                *float64
		ProgressPercent      *float64
		Rating               *float64
		RatingScale          *float64
		Note                 string
		RepeatCount          int
		StartedOnValue       *time.Time
		StartedOnPrecision   string
		CompletedOnValue     *time.Time
		CompletedOnPrecision string
	}{
		Status: media.Status, Position: media.Progress, Rating: media.Score, Note: media.StateNote,
		StartedOnValue: startedValue, StartedOnPrecision: startedPrecision,
		CompletedOnValue: completedValue, CompletedOnPrecision: completedPrecision,
	}
	if media.ProgressState != nil {
		state.Status = media.ProgressState.Status
		state.Unit = media.ProgressState.Unit
		state.Position = media.ProgressState.Position
		state.Total = media.ProgressState.Total
		state.ProgressPercent = media.ProgressState.ProgressPercent
		state.RepeatCount = media.ProgressState.PlayCount
		if state.StartedOnValue == nil {
			state.StartedOnValue, state.StartedOnPrecision = partialDateFields(media.ProgressState.StartedOn)
		}
		if state.CompletedOnValue == nil {
			state.CompletedOnValue, state.CompletedOnPrecision = partialDateFields(media.ProgressState.CompletedOn)
		}
	}
	state.RatingScale = media.StateRatingScale
	effectiveOnValue, effectiveOnPrecision := (*time.Time)(nil), ""
	if state.CompletedOnPrecision == string(observations.DatePrecisionDay) {
		effectiveOnValue, effectiveOnPrecision = state.CompletedOnValue, state.CompletedOnPrecision
	}
	fingerprintBytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(fingerprintBytes)
	fingerprint := hex.EncodeToString(digest[:])

	var existing models.MediaStateHistory
	result := tx.Where("media_item_id = ? and source_kind = ? and source_event_id = ?", itemID, rawFile.SourceKind, sourceEventID).
		Order("observed_at desc, created_at desc, id desc").First(&existing)
	if result.Error == nil && existing.StateFingerprint == fingerprint {
		return nil
	}
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	changeKind := "snapshot"
	if result.Error == nil {
		changeKind = "changed"
	}
	// Provider timestamps describe when the upstream record changed, but a
	// current list snapshot is not an activity event. Keep that timestamp as
	// provenance only; the state observation itself is still Iroha-observed
	// until a connector supplies a real provider activity record.
	timeBasis := "iroha_observed"
	var providerRecordedAt *time.Time
	if media.ProgressState != nil && media.ProgressState.LastUpdateAt != nil {
		providerRecordedAt = media.ProgressState.LastUpdateAt
	}
	if effectiveOnPrecision == string(observations.DatePrecisionDay) {
		timeBasis = "source_date"
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	result = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.MediaStateHistory{
		ID: id, MediaItemID: itemID, SourceKind: rawFile.SourceKind, SourceEventID: sourceEventID,
		ObservedAt: observedAt, TimeBasis: timeBasis, ChangeKind: changeKind, StateFingerprint: fingerprint,
		Status: state.Status, Unit: state.Unit, Position: state.Position, Total: state.Total,
		ProgressPercent: state.ProgressPercent, Rating: state.Rating, RatingScale: state.RatingScale,
		Note: state.Note, RepeatCount: state.RepeatCount, StartedOnValue: state.StartedOnValue,
		StartedOnPrecision: state.StartedOnPrecision, CompletedOnValue: state.CompletedOnValue,
		CompletedOnPrecision: state.CompletedOnPrecision, EffectiveOnValue: effectiveOnValue,
		EffectiveOnPrecision: effectiveOnPrecision, ProviderRecordedAt: providerRecordedAt,
		RawFileID: &rawFile.ID, CreatedAt: time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	// The raw-file-scoped fingerprint index makes concurrent retries of the
	// same snapshot idempotent without preventing a legitimate A -> B -> A
	// state transition across later observations.
	return nil
}

func partialDateFields(value *observations.PartialDate) (*time.Time, string) {
	if value == nil || !value.Valid() {
		return nil, ""
	}
	date := value.Value
	return &date, string(value.Precision)
}

// latestEventUnchanged reports whether the most recent event for the same
// source entry already records identical progress/rating/note, meaning a fresh
// sync observed no change. Events without a stable source id always append.
func latestEventUnchanged(tx *gorm.DB, itemID uuid.UUID, sourceKind string, event observations.MediaEvent) (bool, error) {
	if event.SourceEventID == "" {
		return false, nil
	}
	var existing models.MediaConsumptionEvent
	err := tx.Where("media_item_id = ? and source_kind = ? and source_event_id = ? and event_type = ?",
		itemID, sourceKind, event.SourceEventID, event.EventType).
		Order("event_at desc, created_at desc").
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return existing.EventAt.Equal(event.EventAt) &&
		existing.Unit == event.Unit &&
		floatPtrEqual(existing.Position, event.Position) &&
		floatPtrEqual(existing.Total, event.Total) &&
		floatPtrEqual(existing.Rating, event.Rating) &&
		floatPtrEqual(existing.ProgressPercent, event.ProgressPercent) &&
		floatPtrEqual(existing.RatingScale, event.RatingScale) &&
		existing.Note == event.Note, nil
}

// upsertMediaProgress recomputes the current-progress projection, preferring
// the adapter's rich ProgressState (unit/play_count/last_update/hidden) and
// falling back to flat fields. A cross-source status disagreement is routed to
// the source history instead of silently overwriting.
func upsertMediaProgress(tx *gorm.DB, rawFile models.RawFile, itemID uuid.UUID, media observations.Media) error {
	progress := models.MediaProgress{
		MediaItemID: itemID,
		Status:      media.Status,
		Position:    media.Progress,
		SourceKind:  rawFile.SourceKind,
		UpdatedAt:   time.Now().UTC(),
	}
	progress.StartedOnValue, progress.StartedOnPrecision = partialDateFields(media.StartedOn)
	progress.CompletedOnValue, progress.CompletedOnPrecision = partialDateFields(media.CompletedOn)
	if ps := media.ProgressState; ps != nil {
		progress.Status = ps.Status
		progress.Unit = ps.Unit
		progress.Position = ps.Position
		progress.Total = ps.Total
		progress.ProgressPercent = ps.ProgressPercent
		progress.StartedOnValue, progress.StartedOnPrecision = partialDateFields(ps.StartedOn)
		progress.LastUpdateAt = ps.LastUpdateAt
		progress.CompletedOnValue, progress.CompletedOnPrecision = partialDateFields(ps.CompletedOn)
		progress.PlayCount = ps.PlayCount
		// Paused/on-hold entries stay status=in_progress but must not surface in
		// the "continue" strip; fold the adapter's Paused flag into the column.
		progress.HiddenFromContinue = ps.HiddenFromContinue || ps.Paused
	}
	if progress.Status == "" {
		progress.Status = mediaUnknownValue
	}
	var existing models.MediaProgress
	result := tx.Where("media_item_id = ?", itemID).First(&existing)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return tx.Create(&progress).Error
	}
	if result.Error != nil {
		return result.Error
	}
	if existing.SourceKind != "" && existing.SourceKind != rawFile.SourceKind && !mediaProgressSourceWins(rawFile.SourceKind, existing.SourceKind) {
		return nil
	}
	if mediaProgressIsOlder(existing, progress) {
		return nil
	}
	return tx.Model(&existing).Updates(map[string]any{
		"status":                 progress.Status,
		"unit":                   progress.Unit,
		"position":               progress.Position,
		"total":                  progress.Total,
		"progress_percent":       progress.ProgressPercent,
		"started_on_value":       progress.StartedOnValue,
		"started_on_precision":   progress.StartedOnPrecision,
		"last_update_at":         progress.LastUpdateAt,
		"completed_on_value":     progress.CompletedOnValue,
		"completed_on_precision": progress.CompletedOnPrecision,
		"play_count":             progress.PlayCount,
		"hidden_from_continue":   progress.HiddenFromContinue,
		"source_kind":            progress.SourceKind,
		"updated_at":             progress.UpdatedAt,
	}).Error
}

func mediaProgressIsOlder(existing, incoming models.MediaProgress) bool {
	if existing.SourceKind != incoming.SourceKind || existing.LastUpdateAt == nil {
		return false
	}
	if incoming.LastUpdateAt == nil {
		return true
	}
	return incoming.LastUpdateAt.Before(*existing.LastUpdateAt)
}

func floatPtrEqual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func itemRoleOrDefault(role string) string {
	if role == "" {
		return mediaItemRole
	}
	return role
}

func canonicalReleaseDate(media observations.Media) (*time.Time, string) {
	if media.ReleaseDateOn != nil && media.ReleaseDateOn.Valid() {
		date := media.ReleaseDateOn.Value
		return &date, string(media.ReleaseDateOn.Precision)
	}
	if media.ReleaseDate != nil {
		return media.ReleaseDate, string(observations.DatePrecisionDay)
	}
	return nil, ""
}

// titleCandidates adapts provider title claims to the pure projection contract
// in i18n, which owns the title policy and deliberately depends on no provider
// or persistence package.
func titleCandidates(titles []observations.MediaTitle) []i18n.TitleCandidate {
	candidates := make([]i18n.TitleCandidate, 0, len(titles))
	for _, title := range titles {
		if title.Title == "" {
			continue
		}
		candidates = append(candidates, i18n.TitleCandidate{
			Title:      title.Title,
			Language:   title.Language,
			Script:     title.Script,
			Region:     title.Region,
			TitleKind:  title.TitleKind,
			Provider:   title.Provider,
			IsPrimary:  title.IsPrimary,
			Confidence: title.Confidence,
		})
	}
	return candidates
}

// refreshMediaItemFields updates the core columns of an already-existing item
// from a fresh observation. When owned (the item's own provider is syncing) a
// non-empty incoming value overwrites, so a reprocess after a parser fix
// re-applies; otherwise only empty columns are filled, so a bridge/title match
// from another provider never clobbers the owner's data.
func refreshMediaItemFields(tx *gorm.DB, itemID uuid.UUID, media observations.Media, owned bool) error {
	var item models.MediaItem
	if err := tx.First(&item, "id = ?", itemID).Error; err != nil {
		return err
	}
	updates := map[string]any{}
	setStr := func(col, existing, incoming string) {
		if incoming == "" || incoming == existing {
			return
		}
		if owned || existing == "" {
			updates[col] = incoming
		}
	}
	setInt := func(col string, existing, incoming *int) {
		if incoming == nil {
			return
		}
		if owned || existing == nil {
			updates[col] = *incoming
		}
	}
	setFloat := func(col string, existing, incoming *float64) {
		if incoming == nil {
			return
		}
		if owned || existing == nil {
			updates[col] = *incoming
		}
	}

	// Edition preserved: an item is one consumed edition (a Chinese WeRead book,
	// an English Hardcover edition), so its title follows the same ownership rule
	// as every other column. Script ranking used to let any Japanese or English
	// sync replace a Chinese edition's title.
	if media.Title != "" && media.Title != item.Title && (owned || item.Title == "") {
		updates["title"] = media.Title
		if item.OriginalTitle == "" {
			updates["original_title"] = media.Title
		}
	}

	if media.MediaType != "" && media.MediaType != mediaUnknownValue {
		setStr("media_type", item.MediaType, media.MediaType)
	}
	if media.ItemRole != "" {
		setStr("item_role", item.ItemRole, media.ItemRole)
	}
	if releaseDate, precision := canonicalReleaseDate(media); releaseDate != nil && (owned || item.ReleaseDate == nil) {
		updates["release_date"] = releaseDate
		updates["release_date_precision"] = precision
	}
	setInt("season_number", item.SeasonNumber, media.SeasonNumber)
	setInt("episode_number", item.EpisodeNumber, media.EpisodeNumber)
	setFloat("chapter_number", item.ChapterNumber, media.ChapterNumber)
	setFloat("volume_number", item.VolumeNumber, media.VolumeNumber)
	setInt("duration_seconds", item.DurationSeconds, media.DurationSeconds)
	setInt("page_count", item.PageCount, media.PageCount)
	setInt("episode_count", item.EpisodeCount, media.EpisodeCount)
	setInt("chapter_count", item.ChapterCount, media.ChapterCount)
	setStr("language", item.Language, media.Language)
	setStr("country", item.Country, media.Country)
	setStr("publisher", item.Publisher, media.Publisher)
	setStr("cover_image_url", item.CoverImageURL, media.CoverImageURL)

	if len(updates) > 0 {
		updates["updated_at"] = time.Now().UTC()
		if err := tx.Model(&models.MediaItem{}).Where("id = ?", itemID).Updates(updates).Error; err != nil {
			return err
		}
	}

	// Work titles are a pure projection over title claims (ADR-0012): Native
	// First, and a localized translation never overwrites the authored title.
	// Description follows the same owned/existing-empty rule as the item fields.
	if item.WorkID != nil && (media.Description != "" || len(media.Titles) > 0 || media.Title != "") {
		var work models.MediaWork
		if err := tx.First(&work, "id = ?", *item.WorkID).Error; err != nil {
			return err
		}
		workUpdates := map[string]any{}
		if media.Description != "" && (owned || work.Description == "") && work.Description != media.Description {
			workUpdates["description"] = media.Description
		}
		candidates := titleCandidates(media.Titles)
		if len(candidates) == 0 && media.Title != "" {
			// The provider gave no title claims; its display title is still a
			// localized claim about the work, never an authored one.
			candidates = []i18n.TitleCandidate{{
				Title:     media.Title,
				Language:  media.Language,
				TitleKind: i18n.TitleKindLocalized,
				Provider:  media.Provider,
			}}
		}
		resolution := i18n.ResolveWorkTitles(work.PrimaryTitle, work.OriginalTitle, work.OriginalLanguage, candidates)
		if resolution.PrimaryTitle != "" && resolution.PrimaryTitle != work.PrimaryTitle {
			workUpdates["primary_title"] = resolution.PrimaryTitle
		}
		if resolution.OriginalTitle != "" && resolution.OriginalTitle != work.OriginalTitle {
			workUpdates["original_title"] = resolution.OriginalTitle
		}
		if resolution.OriginalLanguage != "" && resolution.OriginalLanguage != work.OriginalLanguage {
			workUpdates["original_language"] = resolution.OriginalLanguage
		}
		if len(workUpdates) > 0 {
			workUpdates["updated_at"] = time.Now().UTC()
			if err := tx.Model(&models.MediaWork{}).Where("id = ?", *item.WorkID).Updates(workUpdates).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
