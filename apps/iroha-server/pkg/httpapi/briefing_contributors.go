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
func (mediaBriefingContributor) Schema() string { return "media.day.v1" }
func (c mediaBriefingContributor) Contribute(_ context.Context, day briefing.Day) (briefing.Section, error) {
	page, err := c.service.Events(media.EventListFilters{From: &day.Start, To: &day.End, Limit: briefingSectionLimit})
	if err != nil {
		return briefing.Section{}, fmt.Errorf("list media briefing: %w", err)
	}
	items := make([]mediaHomeEventResponse, 0, len(page.Items))
	for _, event := range page.Items {
		items = append(items, mediaHomeEventResponse{
			ID: mediaEventID(event.ID), MediaID: ids.Encode(ids.MediaPrefix, event.MediaItemID),
			Title: event.Title, NativeTitle: event.NativeTitle, CoverImageURL: event.CoverImageURL, EventType: event.EventType,
			OccurredAt: event.OccurredAt, Unit: event.Unit, Position: event.Position, Total: event.Total,
			ProgressPercent: event.ProgressPercent, Rating: normalizedRating(event.Rating, event.RatingScale),
		})
	}
	return listSection(mediaEventListResponse{Items: items, HasMore: page.HasMore}, len(items)), nil
}

func listSection(data any, itemCount int) briefing.Section {
	state := briefing.StateReady
	if itemCount == 0 {
		state = briefing.StateEmpty
	}
	return briefing.Section{State: state, Data: data}
}
