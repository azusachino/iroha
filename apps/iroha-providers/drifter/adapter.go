package drifter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const (
	// AdapterVersion versions the mapping semantics, not the wire schema.
	AdapterVersion = "drifter-2026-11-1"

	// sourceProviderWeRead is the provider that actually owns the shelf. Drifter
	// is a gateway, so an item's identity stays with its source provider and a
	// later direct WeRead adapter would claim the same item rather than a copy.
	sourceProviderWeRead = "weread"
	// sourceProviderISBN is the shared catalog identity of an edition.
	sourceProviderISBN = "isbn"
	// sourceProviderHardcover is the catalog Drifter confirms mappings to.
	sourceProviderHardcover = "hardcover"

	mediaTypeBook   = "book"
	itemRoleBook    = "book"
	languageChinese = "zh"

	// customUploadDescription marks a book the reader uploaded themselves; it
	// has no source catalog page or ISBN.
	customUploadDescription = "Custom upload"
	// finishEventPrefix namespaces the deterministic finish event per book so a
	// replay updates instead of appending a duplicate.
	finishEventPrefix = "weread-finish-"
)

// Adapter maps Drifter reading snapshots into provider-neutral media
// observations. It reads only the stored snapshot bytes.
type Adapter struct{}

func NewAdapter() Adapter { return Adapter{} }

func (Adapter) Descriptor() provider.Descriptor {
	return provider.Descriptor{
		ID:             ProviderID,
		DisplayName:    "Drifter",
		AdapterVersion: AdapterVersion,
		SourceKinds:    []string{SourceKind},
		Domains:        []provider.Domain{provider.DomainMedia},
		Capabilities: []provider.Capability{
			provider.CapabilityMediaLibrary,
			provider.CapabilityMediaProgress,
		},
	}
}

func (Adapter) ImportMedia(ctx context.Context, source provider.Source, options provider.ImportOptions) ([]observations.Media, error) {
	reader, err := source.Open(ctx)
	if err != nil {
		return nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "open_snapshot", Err: err}
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "read_snapshot", Err: err}
	}
	return ParseSnapshot(body, importLocation(options.Timezone))
}

// ParseSnapshot converts one snapshot page into media observations. A book
// without an id or title fails the page instead of being silently dropped: an
// unreadable entry must stay visible, because skipping it would quietly lose
// reading history.
func ParseSnapshot(body []byte, location *time.Location) ([]observations.Media, error) {
	var response ReadingSnapshot
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode drifter snapshot: %w", err)
	}
	entries := make([]observations.Media, 0, len(response.Books))
	for _, book := range response.Books {
		media, err := mapBook(book, location)
		if err != nil {
			return nil, err
		}
		entries = append(entries, media)
	}
	return entries, nil
}

func mapBook(book readingSnapshotBook, location *time.Location) (observations.Media, error) {
	externalID := strings.TrimSpace(book.SourceBookID)
	if externalID == "" {
		return observations.Media{}, fmt.Errorf("drifter book %q is missing source_book_id", book.Title)
	}
	title := strings.TrimSpace(book.Title)
	if title == "" {
		return observations.Media{}, fmt.Errorf("drifter book %s is missing title", externalID)
	}
	if location == nil {
		location = time.UTC
	}

	media := observations.Media{
		Provider:      sourceProviderWeRead,
		ExternalID:    externalID,
		MediaType:     mediaTypeBook,
		ItemRole:      itemRoleBook,
		Title:         title,
		Language:      languageChinese,
		Publisher:     strings.TrimSpace(book.Publisher),
		CoverImageURL: strings.TrimSpace(book.CoverURL),
		Titles: []observations.MediaTitle{{
			Title: title, Language: languageChinese, TitleKind: "original", Provider: sourceProviderWeRead, IsPrimary: true,
		}},
		ExternalRefs: []observations.MediaExternalRef{{
			Provider: sourceProviderWeRead, ExternalID: externalID, MatchedBy: "provider_id",
		}},
	}
	if book.IsCustom {
		media.Description = customUploadDescription
	}
	if author := strings.TrimSpace(book.Author); author != "" {
		media.Creators = append(media.Creators, observations.MediaCreatorRole{Name: author, Role: observations.MediaCreatorRoleAuthor, Provider: sourceProviderWeRead})
	}
	if translator := strings.TrimSpace(book.Translator); translator != "" {
		media.Creators = append(media.Creators, observations.MediaCreatorRole{Name: translator, Role: observations.MediaCreatorRoleTranslator, Provider: sourceProviderWeRead})
	}
	if isbn := strings.TrimSpace(book.ISBN); isbn != "" {
		media.ExternalRefs = append(media.ExternalRefs, observations.MediaExternalRef{Provider: sourceProviderISBN, ExternalID: isbn})
	}
	if target := book.MatchedTarget; target != nil {
		if target.EditionID > 0 {
			media.ExternalRefs = append(media.ExternalRefs, observations.MediaExternalRef{
				Provider: sourceProviderHardcover, ExternalID: strconv.FormatInt(target.EditionID, 10),
			})
		}
		if target.WorkID > 0 {
			media.WorkExternalID = strconv.FormatInt(target.WorkID, 10)
			media.WorkTitle = target.OriginalTitle
		}
	}

	status := mapReadingStatus(book.Status)
	percent := float64(book.ProgressPercent)
	media.Status = status
	var readingTime *int
	if book.ReadingTimeSeconds > 0 {
		rt := book.ReadingTimeSeconds
		readingTime = &rt
	}
	media.ProgressState = &observations.MediaProgress{
		Status:             status,
		ProgressPercent:    &percent,
		LastUpdateAt:       book.LastReadAt,
		ReadingTimeSeconds: readingTime,
	}
	if book.StartedOn != nil {
		if startedOn, err := parseDay(*book.StartedOn); err == nil {
			media.StartedOn = startedOn
			media.ProgressState.StartedOn = startedOn
		}
	}
	if book.FinishedAt != nil && !book.FinishedAt.IsZero() {
		finishedAt := book.FinishedAt.UTC()
		// The completion date is a calendar fact for the reader, so it uses the
		// import timezone rather than UTC's date.
		if completedOn, err := parseDay(finishedAt.In(location).Format(time.DateOnly)); err == nil {
			media.CompletedOn = completedOn
			media.ProgressState.CompletedOn = completedOn
		}
		var note string
		if readingTime != nil && *readingTime > 0 {
			note = formatReadingDuration(*readingTime)
		}
		media.Events = append(media.Events, observations.MediaEvent{
			EventType:       observations.MediaEventFinished,
			EventAt:         finishedAt,
			SourceEventID:   finishEventPrefix + externalID,
			ProgressPercent: &percent,
			Note:            note,
		})
	}
	return media, nil
}

func formatReadingDuration(seconds int) string {
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("Read for %dh %dm", hours, mins)
	}
	return fmt.Sprintf("Read for %dm", mins)
}

// mapReadingStatus is the gateway's status vocabulary. "abandoned" is a known
// state, so it stays distinct instead of collapsing into "unknown".
func mapReadingStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return "completed"
	case "reading", "in_progress":
		return "in_progress"
	case "planned":
		return "planned"
	case "abandoned":
		return "abandoned"
	default:
		return "unknown"
	}
}

func parseDay(value string) (*observations.PartialDate, error) {
	day, err := time.Parse(time.DateOnly, strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	return observations.NewPartialDate(day.Year(), int(day.Month()), day.Day())
}

// importLocation resolves the import timezone, falling back to UTC when the
// configured zone is empty or unknown.
func importLocation(timezone string) *time.Location {
	if timezone == "" {
		return time.UTC
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}
	return location
}
