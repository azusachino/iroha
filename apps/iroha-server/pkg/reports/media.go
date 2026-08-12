package reports

import "github.com/azusachino/iroha/apps/iroha-server/pkg/media"

func Media(service *media.Service, period Period) (*MediaData, error) {
	result, err := service.PeriodReport(media.PeriodFilters{From: period.FromInstant, To: period.ToInstantExclusive})
	if err != nil {
		return nil, err
	}
	return mediaData(result), nil
}

func mediaData(result media.PeriodReport) *MediaData {
	if result.EventCount == 0 && result.CompletedCount == 0 {
		return nil
	}

	byKind := make([]MediaKindTotal, 0, len(result.ByKind))
	for _, kind := range result.ByKind {
		byKind = append(byKind, MediaKindTotal{
			Kind: kind.Kind, EventCount: kind.EventCount, CompletedCount: kind.CompletedCount,
		})
	}
	completedItems := make([]MediaCompleted, 0, len(result.CompletedItems))
	for _, item := range result.CompletedItems {
		completedItems = append(completedItems, MediaCompleted{
			ID: item.ID.String(), Title: item.Title, MediaType: item.MediaType, CompletedAt: item.CompletedAt,
		})
	}
	return &MediaData{
		EventCount: result.EventCount, CompletedCount: result.CompletedCount, RatedCount: result.RatedCount,
		AverageRating: result.AverageRating, ByKind: byKind, CompletedItems: completedItems,
	}
}
