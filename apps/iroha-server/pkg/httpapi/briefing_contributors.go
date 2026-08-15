package httpapi

import (
	"context"
	"fmt"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/briefing"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
)

const briefingSectionLimit = 20

func NewBriefingRegistry(
	dailyService *daily.Service,
	sleepService *sleep.Service,
	activityService *activities.Service,
	mediaService *media.Service,
) (*briefing.Registry, error) {
	return briefing.NewRegistry(
		dailyBriefingContributor{service: dailyService},
		sleepBriefingContributor{service: sleepService},
		activityBriefingContributor{service: activityService},
		mediaBriefingContributor{service: mediaService},
	)
}

type dailyBriefingContributor struct{ service *daily.Service }

func (dailyBriefingContributor) Key() string    { return "daily" }
func (dailyBriefingContributor) Schema() string { return "daily.day.v1" }
func (c dailyBriefingContributor) Contribute(_ context.Context, day briefing.Day) (briefing.Section, error) {
	page, err := c.service.List(daily.ListFilters{From: &day.Date, To: &day.End, Limit: briefingSectionLimit})
	if err != nil {
		return briefing.Section{}, fmt.Errorf("list daily briefing: %w", err)
	}
	items := make([]dailyResponse, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, toDailyResponse(row))
	}
	return listSection(dailyListResponse{Items: items, HasMore: page.HasMore}, len(items)), nil
}

type sleepBriefingContributor struct{ service *sleep.Service }

func (sleepBriefingContributor) Key() string    { return "sleep" }
func (sleepBriefingContributor) Schema() string { return "sleep.day.v1" }
func (c sleepBriefingContributor) Contribute(_ context.Context, day briefing.Day) (briefing.Section, error) {
	page, err := c.service.List(sleep.ListFilters{From: &day.Date, To: &day.End, Limit: briefingSectionLimit})
	if err != nil {
		return briefing.Section{}, fmt.Errorf("list sleep briefing: %w", err)
	}
	items := make([]sleepResponse, 0, len(page.Items))
	for _, session := range page.Items {
		items = append(items, toSleepResponse(session))
	}
	return listSection(sleepListResponse{Items: items, HasMore: page.HasMore}, len(items)), nil
}

type activityBriefingContributor struct{ service *activities.Service }

func (activityBriefingContributor) Key() string    { return "activities" }
func (activityBriefingContributor) Schema() string { return "activities.day.v1" }
func (c activityBriefingContributor) Contribute(_ context.Context, day briefing.Day) (briefing.Section, error) {
	page, err := c.service.List(activities.ListFilters{
		StartedFrom:   &day.Start,
		StartedBefore: &day.End,
		Limit:         briefingSectionLimit,
	})
	if err != nil {
		return briefing.Section{}, fmt.Errorf("list activities briefing: %w", err)
	}
	items := make([]activityResponse, 0, len(page.Items))
	for _, activity := range page.Items {
		items = append(items, toActivityResponse(activity))
	}
	return listSection(activityListResponse{Items: items, HasMore: page.HasMore}, len(items)), nil
}

type mediaBriefingContributor struct{ service *media.Service }

func (mediaBriefingContributor) Key() string    { return "media" }
func (mediaBriefingContributor) Schema() string { return "media.day.v2" }
func (c mediaBriefingContributor) Contribute(_ context.Context, day briefing.Day) (briefing.Section, error) {
	eventPage, err := c.service.Events(media.EventListFilters{From: &day.Start, To: &day.End, Limit: briefingSectionLimit})
	if err != nil {
		return briefing.Section{}, fmt.Errorf("list media sessions briefing: %w", err)
	}
	changePage, err := c.service.DatedChanges(day.Start, day.End, briefingSectionLimit)
	if err != nil {
		return briefing.Section{}, fmt.Errorf("list media dated updates briefing: %w", err)
	}
	sessions := make([]mediaHomeEventResponse, 0, len(eventPage.Items))
	for _, event := range eventPage.Items {
		sessions = append(sessions, mediaHomeEventResponse{
			ID: mediaEventID(event.ID), MediaID: ids.Encode(ids.MediaPrefix, event.MediaItemID),
			Title: event.Title, NativeTitle: event.NativeTitle, CoverImageURL: event.CoverImageURL, EventType: event.EventType,
			OccurredAt: event.OccurredAt, Unit: event.Unit, Position: event.Position, Total: event.Total,
			ProgressPercent: event.ProgressPercent, Rating: normalizedRating(event.Rating, event.RatingScale),
		})
	}
	updates := make([]mediaChangeResponse, 0, len(changePage.Items))
	for _, change := range changePage.Items {
		updates = append(updates, toMediaChangeResponse(change))
	}
	timezone := day.Timezone
	if timezone == "" {
		timezone = "UTC"
	}
	state := briefing.StateEmpty
	if len(sessions)+len(updates) > 0 {
		state = briefing.StateReady
	}
	return briefing.Section{
		State: state,
		Data: mediaDayResponse{
			Sessions: mediaDayList[mediaHomeEventResponse]{
				State: briefingListState(len(sessions)), Items: sessions, Count: len(sessions), HasMore: eventPage.HasMore,
			},
			DatedUpdates: mediaDayList[mediaChangeResponse]{
				State: briefingListState(len(updates)), Items: updates, Count: len(updates), HasMore: changePage.HasMore,
			},
			Coverage: mediaDayCoverage{Timezone: timezone, Date: day.Date.Format("2006-01-02")},
		},
	}, nil
}

type mediaDayList[T any] struct {
	State   briefing.SectionState `json:"state"`
	Items   []T                   `json:"items"`
	Count   int                   `json:"count"`
	HasMore bool                  `json:"has_more"`
}

type mediaDayCoverage struct {
	Timezone string `json:"timezone"`
	Date     string `json:"date"`
}

type mediaDayResponse struct {
	Sessions     mediaDayList[mediaHomeEventResponse] `json:"sessions"`
	DatedUpdates mediaDayList[mediaChangeResponse]    `json:"dated_updates"`
	Coverage     mediaDayCoverage                     `json:"coverage"`
}

func briefingListState(itemCount int) briefing.SectionState {
	if itemCount == 0 {
		return briefing.StateEmpty
	}
	return briefing.StateReady
}

func listSection(data any, itemCount int) briefing.Section {
	state := briefing.StateReady
	if itemCount == 0 {
		state = briefing.StateEmpty
	}
	return briefing.Section{State: state, Data: data}
}
