package anilist

import "time"

type graphQLResponse struct {
	Data struct {
		MediaListCollection *mediaListCollection `json:"MediaListCollection"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
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
	ID        int         `json:"id"`
	IDMal     *int        `json:"idMal"`
	Type      string      `json:"type"`
	Format    string      `json:"format"`
	Episodes  *int        `json:"episodes"`
	Chapters  *int        `json:"chapters"`
	Volumes   *int        `json:"volumes"`
	Title     mediaTitle  `json:"title"`
	StartDate anilistDate `json:"startDate"`
	Relations struct {
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

type anilistDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

func (d anilistDate) Time() *time.Time {
	if d.Year == 0 {
		return nil
	}
	month := time.Month(d.Month)
	if month == 0 {
		month = time.January
	}
	result := time.Date(d.Year, month, d.Day, 0, 0, 0, 0, time.UTC)
	return &result
}
