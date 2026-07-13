package anilist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const (
	ProviderID     = "anilist"
	SourceKind     = coreimports.KindAniList
	AdapterVersion = "anilist-2026-07-1"
)

type Adapter struct{}

func NewAdapter() Adapter { return Adapter{} }

func (Adapter) Descriptor() provider.Descriptor {
	return provider.Descriptor{
		ID:             ProviderID,
		DisplayName:    "AniList",
		AdapterVersion: AdapterVersion,
		SourceKinds:    []string{SourceKind},
		Domains:        []provider.Domain{provider.DomainMedia},
		Capabilities: []provider.Capability{
			provider.CapabilityMediaLibrary,
			provider.CapabilityMediaProgress,
			provider.CapabilityMediaRating,
		},
	}
}

func (Adapter) ImportMedia(ctx context.Context, source provider.Source, _ provider.ImportOptions) ([]observations.Media, error) {
	reader, err := source.Open(ctx)
	if err != nil {
		return nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "open_snapshot", Err: err}
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "read_snapshot", Err: err}
	}
	return ParseSnapshot(body)
}

func ParseSnapshot(body []byte) ([]observations.Media, error) {
	var response graphQLResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode anilist snapshot: %w", err)
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("anilist graphql error: %s", response.Errors[0].Message)
	}
	if response.Data.MediaListCollection == nil {
		return nil, nil
	}
	entries := make([]observations.Media, 0)
	for _, list := range response.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			if entry.Media.ID == 0 {
				return nil, fmt.Errorf("anilist entry %d has no media id", entry.ID)
			}
			entries = append(entries, mapEntry(entry))
		}
	}
	return entries, nil
}

func mapEntry(entry mediaListEntry) observations.Media {
	media := entry.Media
	primaryTitle := firstNonEmpty(media.Title.English, media.Title.Romaji, media.Title.Native)
	mediaType, itemRole := mapMediaType(media.Type, media.Format)
	unit := "episodes"
	position := entry.Progress
	if media.Type == "MANGA" || media.Type == "NOVEL" {
		unit = "chapters"
		position = entry.Progress
		if position == nil {
			position = entry.ProgressVolumes
			unit = "volumes"
		}
	}
	status, paused := mapStatus(entry.Status)
	score := normalizeScore(entry.Score)
	startedAt := entry.StartedAt.Time()
	completedAt := entry.CompletedAt.Time()
	lastUpdateAt := unixTime(entry.UpdatedAt)

	result := observations.Media{
		Provider:        ProviderID,
		ExternalID:      strconv.Itoa(media.ID),
		MediaType:       mediaType,
		ItemRole:        itemRole,
		Title:           primaryTitle,
		WorkExternalID:  strconv.Itoa(media.ID),
		WorkKind:        "series",
		WorkTitle:       primaryTitle,
		ReleaseDate:     media.StartDate.Time(),
		DurationSeconds: nil,
		EpisodeCount:    media.Episodes,
		ChapterCount:    media.Chapters,
		Status:          status,
		Progress:        position,
		Score:           score,
		StartedAt:       startedAt,
		CompletedAt:     completedAt,
		Titles:          mapTitles(media.Title),
		ExternalRefs:    mapExternalRefs(media),
		Relations:       mapRelations(media),
		ProgressState: &observations.MediaProgress{
			Status:             status,
			Unit:               unit,
			Position:           position,
			StartedAt:          startedAt,
			LastUpdateAt:       lastUpdateAt,
			FinishedAt:         completedAt,
			PlayCount:          entry.Repeat,
			HiddenFromContinue: false,
			Paused:             paused,
		},
	}
	listEvent := observations.MediaEvent{
		EventType:     "list_state",
		EventAt:       completedAt,
		SourceEventID: strconv.Itoa(entry.ID),
		Unit:          unit,
		Position:      position,
		Rating:        score,
		Note:          entry.Notes,
	}
	if listEvent.EventAt == nil {
		listEvent.EventAt = startedAt
	}
	if listEvent.EventAt == nil {
		listEvent.EventAt = lastUpdateAt
	}
	result.Events = append(result.Events, listEvent)
	for i := 0; i < entry.Repeat; i++ {
		result.Events = append(result.Events, observations.MediaEvent{
			EventType:     "rewatch",
			EventAt:       completedAt,
			SourceEventID: strconv.Itoa(entry.ID) + ":repeat:" + strconv.Itoa(i+1),
			Unit:          unit,
		})
	}
	return result
}

func normalizeScore(value *float64) *float64 {
	if value == nil || *value == 0 {
		return nil
	}
	return value
}

func mapMediaType(mediaType, format string) (string, string) {
	if mediaType == "MANGA" {
		switch format {
		case "NOVEL":
			return "light_novel", "book"
		case "ONE_SHOT":
			return "one_shot", "one_shot"
		default:
			return "manga", "series"
		}
	}
	switch format {
	case "MOVIE":
		return "movie", "movie"
	case "OVA":
		return "ova", "special"
	case "ONA":
		return "ona", "special"
	case "SPECIAL":
		return "special", "special"
	default:
		return "anime_season", "season"
	}
}

func mapStatus(status string) (string, bool) {
	switch status {
	case "CURRENT", "REPEATING":
		return "in_progress", false
	case "COMPLETED":
		return "completed", false
	case "PLANNING":
		return "planned", false
	case "DROPPED":
		return "abandoned", false
	case "PAUSED":
		return "in_progress", true
	default:
		return "unknown", false
	}
}

func mapTitles(title mediaTitle) []observations.MediaTitle {
	titles := make([]observations.MediaTitle, 0, 3)
	appendTitle := func(value, kind string, primary bool) {
		if value == "" {
			return
		}
		titles = append(titles, observations.MediaTitle{Title: value, TitleKind: kind, Provider: ProviderID, IsPrimary: primary})
	}
	primary := firstNonEmpty(title.English, title.Romaji, title.Native)
	appendTitle(title.Native, "original", title.Native == primary)
	appendTitle(title.Romaji, "romanized", title.Romaji == primary)
	appendTitle(title.English, "localized", title.English == primary)
	return titles
}

func mapExternalRefs(media mediaNode) []observations.MediaExternalRef {
	refs := []observations.MediaExternalRef{{Provider: ProviderID, ExternalID: strconv.Itoa(media.ID), MatchedBy: "provider_id"}}
	if media.IDMal != nil && *media.IDMal > 0 {
		refs = append(refs, observations.MediaExternalRef{Provider: "mal", ExternalID: strconv.Itoa(*media.IDMal), MatchedBy: "provider_id"})
	}
	return refs
}

func mapRelations(media mediaNode) []observations.MediaRelation {
	relations := make([]observations.MediaRelation, 0, len(media.Relations.Edges))
	for _, edge := range media.Relations.Edges {
		if edge.Node.ID == 0 {
			continue
		}
		relations = append(relations, observations.MediaRelation{
			FromType:       "item",
			FromExternalID: strconv.Itoa(media.ID),
			ToType:         "item",
			ToExternalID:   strconv.Itoa(edge.Node.ID),
			RelationType:   edge.RelationType,
			Provider:       ProviderID,
		})
	}
	return relations
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func unixTime(value int64) *time.Time {
	if value == 0 {
		return nil
	}
	result := time.Unix(value, 0).UTC()
	return &result
}
