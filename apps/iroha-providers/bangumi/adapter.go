package bangumi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const (
	ProviderID     = "bangumi"
	SourceKind     = coreimports.KindBangumi
	AdapterVersion = "bangumi-2026-07-1"
)

type Adapter struct{}

func NewAdapter() Adapter { return Adapter{} }

func (Adapter) Descriptor() provider.Descriptor {
	return provider.Descriptor{
		ID:             ProviderID,
		DisplayName:    "Bangumi",
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
	var response collectionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode bangumi snapshot: %w", err)
	}
	entries := make([]observations.Media, 0, len(response.Data))
	for _, record := range response.Data {
		if record.Subject.ID == 0 {
			return nil, fmt.Errorf("bangumi collection has no subject id")
		}
		entries = append(entries, mapRecord(record))
	}
	return entries, nil
}

func mapRecord(record collectionRecord) observations.Media {
	subject := record.Subject
	mediaType, itemRole := mapSubjectType(record.SubjectType, subject.Platform)
	status, paused := mapCollectionType(record.Type)
	unit := "episodes"
	position := record.EpStatus
	if record.SubjectType == 1 {
		unit = "volumes"
		position = record.VolStatus
	}
	if position == nil {
		unit = ""
	}
	primaryTitle := firstNonEmpty(subject.NameCN, subject.Name)
	updatedAt := parseTime(record.UpdatedAt)
	result := observations.Media{
		Provider:       ProviderID,
		ExternalID:     strconv.Itoa(subject.ID),
		MediaType:      mediaType,
		ItemRole:       itemRole,
		Title:          primaryTitle,
		WorkExternalID: strconv.Itoa(subject.ID),
		WorkKind:       "series",
		WorkTitle:      primaryTitle,
		ReleaseDate:    parseDate(subject.Date),
		EpisodeCount:   subject.Eps,
		VolumeNumber:   floatPtrFromInt(subject.Volumes),
		CoverImageURL:  subject.Images.Large,
		Status:         status,
		Progress:       position,
		Score:          normalizeRate(record.Rate),
		Titles:         mapTitles(subject),
		ExternalRefs:   []observations.MediaExternalRef{{Provider: ProviderID, ExternalID: strconv.Itoa(subject.ID), MatchedBy: "provider_id"}},
		ProgressState: &observations.MediaProgress{
			Status:             status,
			Unit:               unit,
			Position:           position,
			LastUpdateAt:       updatedAt,
			PlayCount:          0,
			HiddenFromContinue: false,
			Paused:             paused,
		},
	}
	if unit == "" {
		result.ProgressState.Unit = "unknown"
	}
	rating := normalizeRate(record.Rate)
	listEvent := observations.MediaEvent{
		EventType:     "list_state",
		EventAt:       updatedAt,
		SourceEventID: strconv.Itoa(subject.ID),
		Unit:          unit,
		Position:      position,
		Rating:        rating,
		Note:          strings.Join(record.Tags, ", ") + noteSuffix(record.Comment),
	}
	if rating != nil {
		// Bangumi ratings are a fixed 0-10 scale.
		scale := 10.0
		listEvent.RatingScale = &scale
	}
	result.Events = []observations.MediaEvent{listEvent}
	return result
}

func mapSubjectType(subjectType int, platform string) (string, string) {
	switch subjectType {
	case 1:
		if strings.Contains(strings.ToLower(platform), "novel") {
			return "light_novel", "book"
		}
		return "manga", "series"
	case 2:
		return "anime_season", "season"
	case 3:
		return "music", "album"
	case 4:
		return "game", "game"
	case 6:
		return "real", "item"
	default:
		return "unknown", "item"
	}
}

func mapCollectionType(value int) (string, bool) {
	switch value {
	case 1:
		return "planned", false
	case 2:
		return "completed", false
	case 3:
		return "in_progress", false
	case 4:
		return "in_progress", true
	case 5:
		return "abandoned", false
	default:
		return "unknown", false
	}
}

func mapTitles(subject subjectRecord) []observations.MediaTitle {
	titles := make([]observations.MediaTitle, 0, 2)
	if subject.Name != "" {
		titles = append(titles, observations.MediaTitle{Title: subject.Name, TitleKind: "original", Provider: ProviderID, IsPrimary: subject.NameCN == ""})
	}
	if subject.NameCN != "" {
		titles = append(titles, observations.MediaTitle{Title: subject.NameCN, Language: "zh-Hans", TitleKind: "localized", Provider: ProviderID, IsPrimary: true})
	}
	return titles
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeRate(value *float64) *float64 {
	if value == nil || *value == 0 {
		return nil
	}
	return value
}

func parseDate(value string) *time.Time {
	if len(value) < len("2006-01-02") {
		return nil
	}
	result, err := time.Parse("2006-01-02", value[:len("2006-01-02")])
	if err != nil {
		return nil
	}
	return &result
}

func parseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	result, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	result = result.UTC()
	return &result
}

func floatPtrFromInt(value *int) *float64 {
	if value == nil {
		return nil
	}
	result := float64(*value)
	return &result
}

func noteSuffix(comment string) string {
	if comment == "" {
		return ""
	}
	return "; " + comment
}
