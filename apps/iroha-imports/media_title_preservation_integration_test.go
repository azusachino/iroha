//go:build integration

package imports

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/i18n"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/gorm"
)

// TestIntegrationMediaEditionTitlePreservation locks down the two ADR-0012
// guarantees the old script ranking violated:
//
//   - an item is one consumed edition, so a sync from another provider never
//     replaces its title;
//   - a work keeps its authored title, so a localized translation never
//     overwrites original_title.
func TestIntegrationMediaEditionTitlePreservation(t *testing.T) {
	db := openImportsIntegrationDB(t)
	seeded := seedMediaItem(t, db, "挪威的森林", "book", "book", time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))

	refresh := func(media observations.Media, owned bool) {
		t.Helper()
		if err := db.Transaction(func(tx *gorm.DB) error {
			return refreshMediaItemFields(tx, seeded.itemID, media, owned)
		}); err != nil {
			t.Fatalf("refreshMediaItemFields(%s, owned=%t): %v", media.Provider, owned, err)
		}
	}
	loadItem := func() models.MediaItem {
		t.Helper()
		var item models.MediaItem
		if err := db.First(&item, "id = ?", seeded.itemID).Error; err != nil {
			t.Fatalf("load item: %v", err)
		}
		return item
	}
	loadWork := func() models.MediaWork {
		t.Helper()
		var work models.MediaWork
		if err := db.First(&work, "id = ?", seeded.workID).Error; err != nil {
			t.Fatalf("load work: %v", err)
		}
		return work
	}

	// A different provider (a bridge/title match, so owned=false) reports the
	// work under its English title. It must not rewrite the consumed Chinese
	// edition, and it must not touch the authored title either.
	refresh(observations.Media{
		Provider:   "hardcover",
		ExternalID: "hardcover-" + seeded.itemID.String(),
		MediaType:  "book",
		ItemRole:   "book",
		Title:      "Norwegian Wood",
		Language:   "en",
		Titles: []observations.MediaTitle{
			{Title: "Norwegian Wood", Language: "en", TitleKind: i18n.TitleKindLocalized, Provider: "hardcover"},
		},
	}, false)

	if item := loadItem(); item.Title != "挪威的森林" {
		t.Errorf("item title = %q, want the preserved Chinese edition title", item.Title)
	}
	if work := loadWork(); work.OriginalTitle != "挪威的森林" || work.PrimaryTitle != "挪威的森林" {
		t.Errorf("work = %q/%q, want the Chinese original preserved", work.PrimaryTitle, work.OriginalTitle)
	}

	// The owning provider may refresh its own edition (a parser fix, a renamed
	// edition), but its localized title still cannot become the work's authored
	// title.
	refresh(observations.Media{
		Provider:   "weread",
		ExternalID: "weread-" + seeded.itemID.String(),
		MediaType:  "book",
		ItemRole:   "book",
		Title:      "挪威的森林（新版）",
		Language:   "zh",
	}, true)

	if item := loadItem(); item.Title != "挪威的森林（新版）" {
		t.Errorf("item title = %q, want the owning provider's refreshed title", item.Title)
	}
	if work := loadWork(); work.OriginalTitle != "挪威的森林" {
		t.Errorf("work original title = %q, want it untouched by a localized claim", work.OriginalTitle)
	}
}

// TestIntegrationMediaWorkAdoptsAuthoredTitle covers the Native First half: a
// work created from a translated edition title adopts the authored title once a
// provider supplies an original claim, even when that provider also marks its
// translation as primary.
func TestIntegrationMediaWorkAdoptsAuthoredTitle(t *testing.T) {
	db := openImportsIntegrationDB(t)
	const (
		chineseTitle  = "超超超超超喜欢你的100个女朋友"
		japaneseTitle = "君のことが大大大大大好きな100人の彼女"
	)
	seeded := seedMediaItem(t, db, chineseTitle, "anime", "season", time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err := db.Model(&models.MediaWork{}).Where("id = ?", seeded.workID).Update("original_title", "").Error; err != nil {
		t.Fatalf("clear seeded original title: %v", err)
	}

	media := observations.Media{
		Provider:   "bangumi",
		ExternalID: "bangumi-" + seeded.itemID.String(),
		MediaType:  "anime",
		ItemRole:   "season",
		Title:      chineseTitle,
		Titles: []observations.MediaTitle{
			{Title: japaneseTitle, TitleKind: i18n.TitleKindOriginal, Provider: "bangumi"},
			{Title: chineseTitle, Language: "zh-Hans", TitleKind: i18n.TitleKindLocalized, Provider: "bangumi", IsPrimary: true},
		},
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return refreshMediaItemFields(tx, seeded.itemID, media, false)
	}); err != nil {
		t.Fatalf("refreshMediaItemFields(bangumi): %v", err)
	}

	var work models.MediaWork
	if err := db.First(&work, "id = ?", seeded.workID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if work.OriginalTitle != japaneseTitle {
		t.Errorf("original title = %q, want the authored Japanese title", work.OriginalTitle)
	}
	if work.PrimaryTitle != japaneseTitle {
		t.Errorf("primary title = %q, want the authored title (Bangumi's NameCN is only primary by its own flag)", work.PrimaryTitle)
	}

	// The consumed edition keeps its own Chinese title.
	var item models.MediaItem
	if err := db.First(&item, "id = ?", seeded.itemID).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if item.Title != chineseTitle {
		t.Errorf("item title = %q, want the consumed Chinese edition title", item.Title)
	}
}
