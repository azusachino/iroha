package anilist

import (
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
)

type graphQLResponse struct {
	Data struct {
		User                *anilistUser         `json:"User"`
		MediaListCollection *mediaListCollection `json:"MediaListCollection"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type anilistUser struct {
	MediaListOptions struct {
		ScoreFormat string `json:"scoreFormat"`
	} `json:"mediaListOptions"`
}

type mediaListCollection struct {
	Lists []mediaList `json:"lists"`
}

type mediaList struct {
	Entries []mediaListEntry `json:"entries"`
}

type mediaListEntry struct {
	ID              int         `json:"id"`
	Status          string      `json:"status"`
	Score           *float64    `json:"score"`
	Progress        *float64    `json:"progress"`
	ProgressVolumes *float64    `json:"progressVolumes"`
	Repeat          int         `json:"repeat"`
	Notes           string      `json:"notes"`
	StartedAt       anilistDate `json:"startedAt"`
	CompletedAt     anilistDate `json:"completedAt"`
	UpdatedAt       int64       `json:"updatedAt"`
	Media           mediaNode   `json:"media"`
}

type mediaNode struct {
	ID          int             `json:"id"`
	IDMal       *int            `json:"idMal"`
	Type        string          `json:"type"`
	Format      string          `json:"format"`
	Episodes    *int            `json:"episodes"`
	Chapters    *int            `json:"chapters"`
	Volumes     *int            `json:"volumes"`
	Title       mediaTitle      `json:"title"`
	StartDate   anilistDate     `json:"startDate"`
	CoverImage  mediaCoverImage `json:"coverImage"`
	Description string          `json:"description"`
	Relations   struct {
		Edges []struct {
			RelationType string    `json:"relationType"`
			Node         mediaNode `json:"node"`
		} `json:"edges"`
	} `json:"relations"`
}

type mediaTitle struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
	Native  string `json:"native"`
}

type mediaCoverImage struct {
	Large string `json:"large"`
}

type anilistDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

func (d anilistDate) Time() *time.Time {
	partial := d.Partial()
	if partial == nil || partial.Precision != observations.DatePrecisionDay {
		return nil
	}
	result := partial.Value
	return &result
}

func (d anilistDate) Partial() *observations.PartialDate {
	if d.Year == 0 {
		return nil
	}
	partial, err := observations.NewPartialDate(d.Year, d.Month, d.Day)
	if err != nil {
		return nil
	}
	return partial
}
