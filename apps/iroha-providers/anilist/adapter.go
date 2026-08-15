package anilist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const (
	ProviderID         = "anilist"
	SourceKind         = coreimports.KindAniList
	ActivitySourceKind = coreimports.KindAniListActivity
	AdapterVersion     = "anilist-2026-07-1"
)

type Adapter struct{}

func NewAdapter() Adapter { return Adapter{} }

func (Adapter) Descriptor() provider.Descriptor {
	return provider.Descriptor{
		ID:             ProviderID,
		DisplayName:    "AniList",
		AdapterVersion: AdapterVersion,
		SourceKinds:    []string{SourceKind, ActivitySourceKind},
		Domains:        []provider.Domain{provider.DomainMedia},
		Capabilities: []provider.Capability{
			provider.CapabilityMediaLibrary,
			provider.CapabilityMediaProgress,
			provider.CapabilityMediaRating,
			provider.CapabilityMediaActivity,
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
	var scoreScale *float64
	if response.Data.User != nil {
		scoreScale = scoreFormatScale(response.Data.User.MediaListOptions.ScoreFormat)
	}
	entries := make([]observations.Media, 0)
	for _, list := range response.Data.MediaListCollection.Lists {
		for _, entry := range list.Entries {
			if entry.Media.ID == 0 {
				return nil, fmt.Errorf("anilist entry %d has no media id", entry.ID)
			}
			entries = append(entries, mapEntry(entry, scoreScale))
		}
	}
	return entries, nil
}

func (Adapter) ImportMediaHistory(ctx context.Context, source provider.Source, _ provider.ImportOptions) ([]observations.MediaHistory, error) {
	reader, err := source.Open(ctx)
	if err != nil {
		return nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "open_activity_snapshot", Err: err}
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "read_activity_snapshot", Err: err}
	}
	var response activityGraphQLResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode anilist activity snapshot: %w", err)
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("anilist graphql error: %s", response.Errors[0].Message)
	}
	history := make([]observations.MediaHistory, 0, len(response.Data.Page.Activities))
	for _, activity := range response.Data.Page.Activities {
		if activity.ID == 0 || activity.CreatedAt == 0 || activity.Media.ID == 0 {
			continue
		}
		media := mapActivityMedia(activity.Media)
		position, unit := parseActivityProgress(activity.Progress, media.MediaType)
		note := strings.TrimSpace(strings.Join([]string{activity.Status, activity.Progress}, " "))
		effectiveAt := time.Unix(activity.CreatedAt, 0).UTC()
		history = append(history, observations.MediaHistory{
			Media: media,
			Updates: []observations.MediaStateUpdate{{
				SourceEventID: "activity:" + strconv.Itoa(activity.ID),
				EffectiveAt:   effectiveAt,
				Status:        mapActivityStatus(activity.Status),
				Unit:          unit,
				Position:      position,
				Total:         activityTotal(activity.Media, unit),
				Note:          note,
			}},
		})
	}
	return history, nil
}

func mapActivityMedia(media mediaNode) observations.Media {
	mediaType, itemRole := mapMediaType(media.Type, media.Format)
	title := firstNonEmpty(media.Title.English, media.Title.Romaji, media.Title.Native)
	return observations.Media{
		Provider:       ProviderID,
		ExternalID:     strconv.Itoa(media.ID),
		MediaType:      mediaType,
		ItemRole:       itemRole,
		Title:          title,
		WorkExternalID: strconv.Itoa(media.ID),
		WorkKind:       "series",
		WorkTitle:      title,
		EpisodeCount:   media.Episodes,
		ChapterCount:   media.Chapters,
		CoverImageURL:  media.CoverImage.Large,
		Titles:         mapTitles(media.Title),
		ExternalRefs:   mapExternalRefs(media),
	}
}

var (
	activityNumberPattern = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)
	activityUnitPattern   = regexp.MustCompile(`(?i)\b(chapters?|episodes?|volumes?)\b`)
)

func parseActivityProgress(progress, mediaType string) (*float64, string) {
	unit := "episodes"
	if mediaType == "manga" || mediaType == "light_novel" || mediaType == "one_shot" {
		unit = "chapters"
	}
	if match := activityUnitPattern.FindString(progress); match != "" {
		switch strings.ToLower(match[:1]) {
		case "c":
			unit = "chapters"
		case "v":
			unit = "volumes"
		}
	}
	numbers := activityNumberPattern.FindAllString(progress, -1)
	if len(numbers) == 0 {
		return nil, unit
	}
	value, err := strconv.ParseFloat(numbers[len(numbers)-1], 64)
	if err != nil {
		return nil, unit
	}
	return &value, unit
}

func activityTotal(media mediaNode, unit string) *float64 {
	var value *int
	switch unit {
	case "chapters":
		value = media.Chapters
	case "episodes":
		value = media.Episodes
	case "volumes":
		value = media.Volumes
	}
	if value == nil {
		return nil
	}
	result := float64(*value)
	return &result
}

func mapActivityStatus(status string) string {
	value := strings.ToLower(strings.TrimSpace(status))
	switch value {
	case "completed":
		return "completed"
	case "dropped", "abandoned":
		return "abandoned"
	case "paused":
		return "in_progress"
	case "read", "watched", "listened", "started", "re-watched", "rewatched":
		return "in_progress"
	}
	if strings.Contains(value, "read") || strings.Contains(value, "watch") || strings.Contains(value, "listen") {
		return "in_progress"
	}
	return "unknown"
}

// scoreFormatScale converts AniList's per-user scoreFormat into the numeric
// rating scale, so a stored score (e.g. 8.5) is not ambiguous between a /10
// and a /100 scale.
func scoreFormatScale(format string) *float64 {
	var scale float64
	switch format {
	case "POINT_100":
		scale = 100
	case "POINT_10", "POINT_10_DECIMAL":
		scale = 10
	case "POINT_5":
		scale = 5
	case "POINT_3":
		scale = 3
	default:
		return nil
	}
	return &scale
}

func mapEntry(entry mediaListEntry, scoreScale *float64) observations.Media {
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
	startedOn := entry.StartedAt.Partial()
	completedOn := entry.CompletedAt.Partial()
	releaseOn := media.StartDate.Partial()
	lastUpdateAt := unixTime(entry.UpdatedAt)

	result := observations.Media{
		Provider:         ProviderID,
		ExternalID:       strconv.Itoa(media.ID),
		MediaType:        mediaType,
		ItemRole:         itemRole,
		Title:            primaryTitle,
		WorkExternalID:   strconv.Itoa(media.ID),
		WorkKind:         "series",
		WorkTitle:        primaryTitle,
		ReleaseDate:      media.StartDate.Time(),
		ReleaseDateOn:    releaseOn,
		DurationSeconds:  nil,
		EpisodeCount:     media.Episodes,
		ChapterCount:     media.Chapters,
		CoverImageURL:    media.CoverImage.Large,
		Description:      strings.TrimSpace(media.Description),
		Status:           status,
		Progress:         position,
		Score:            score,
		StartedOn:        startedOn,
		CompletedOn:      completedOn,
		StateSourceID:    strconv.Itoa(entry.ID),
		StateNote:        entry.Notes,
		StateRatingScale: scoreScale,
		Titles:           mapTitles(media.Title),
		ExternalRefs:     mapExternalRefs(media),
		Relations:        mapRelations(media),
		ProgressState: &observations.MediaProgress{
			Status:             status,
			Unit:               unit,
			Position:           position,
			LastUpdateAt:       lastUpdateAt,
			StartedOn:          startedOn,
			CompletedOn:        completedOn,
			PlayCount:          entry.Repeat,
			HiddenFromContinue: false,
			Paused:             paused,
		},
	}
	// A MediaList row is current provider state, not a dated consumption
	// event. The entry ID, score, and note are retained by state history.
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
