package reports

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/google/uuid"
)

func TestMediaDataMapsPeriodReport(t *testing.T) {
	firstID := uuid.New()
	secondID := uuid.New()
	firstAt := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	secondAt := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	average := 8.5
	data := mediaData(media.PeriodReport{
		EventCount: 3, CompletedCount: 2, RatedCount: 1, AverageRating: &average,
		ByKind: []media.PeriodKindTotal{{Kind: "anime_season", EventCount: 3, CompletedCount: 2}},
		CompletedItems: []media.PeriodCompletedItem{
			{ID: firstID, Title: "First", MediaType: "anime_season", CompletedAt: firstAt},
			{ID: secondID, Title: "Second", MediaType: "anime_season", CompletedAt: secondAt},
		},
	})

	if data.EventCount != 3 || data.CompletedCount != 2 || data.RatedCount != 1 || data.AverageRating == nil || *data.AverageRating != average {
		t.Fatalf("data = %+v", data)
	}
	if len(data.CompletedItems) != 2 || data.CompletedItems[0].ID != firstID.String() || data.CompletedItems[1].ID != secondID.String() {
		t.Fatalf("completed_items = %+v", data.CompletedItems)
	}
}

func TestMediaDataReturnsNilForEmptyReport(t *testing.T) {
	if mediaData(media.PeriodReport{}) != nil {
		t.Fatal("empty media report should map to nil")
	}
}
