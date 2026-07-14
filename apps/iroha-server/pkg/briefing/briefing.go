package briefing

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

var ErrDuplicateContributor = errors.New("duplicate briefing contributor")

type Day struct {
	Date  time.Time
	Start time.Time
	End   time.Time
}

func ParseDay(value string) (Day, error) {
	date, err := time.ParseInLocation(dateLayout, value, time.UTC)
	if err != nil {
		return Day{}, fmt.Errorf("invalid briefing date: %w", err)
	}
	return Day{Date: date, Start: date, End: date.AddDate(0, 0, 1)}, nil
}

type SectionState string

const (
	StateReady       SectionState = "ready"
	StateEmpty       SectionState = "empty"
	StateUnavailable SectionState = "unavailable"
)

type Section struct {
	Key    string       `json:"key"`
	Schema string       `json:"schema"`
	State  SectionState `json:"state"`
	Data   any          `json:"data"`
}

type Response struct {
	Date         string    `json:"date"`
	PreviousDate string    `json:"previous_date"`
	NextDate     string    `json:"next_date"`
	Sections     []Section `json:"sections"`
}

type Contributor interface {
	Key() string
	Schema() string
	Contribute(context.Context, Day) (Section, error)
}

type Registry struct {
	contributors []Contributor
}

func NewRegistry(contributors ...Contributor) (*Registry, error) {
	registry := &Registry{contributors: make([]Contributor, 0, len(contributors))}
	seen := make(map[string]struct{}, len(contributors))
	for _, contributor := range contributors {
		if contributor == nil {
			continue
		}
		if _, exists := seen[contributor.Key()]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateContributor, contributor.Key())
		}
		seen[contributor.Key()] = struct{}{}
		registry.contributors = append(registry.contributors, contributor)
	}
	return registry, nil
}

func (r *Registry) Build(ctx context.Context, day Day) Response {
	sections := make([]Section, 0, len(r.contributors))
	for _, contributor := range r.contributors {
		section, err := contributor.Contribute(ctx, day)
		if err != nil {
			sections = append(sections, Section{
				Key:    contributor.Key(),
				Schema: contributor.Schema(),
				State:  StateUnavailable,
				Data:   map[string]string{},
			})
			continue
		}
		section.Key = contributor.Key()
		section.Schema = contributor.Schema()
		if section.State == "" {
			section.State = StateReady
		}
		if section.Data == nil {
			section.Data = map[string]any{}
		}
		sections = append(sections, section)
	}
	return Response{
		Date:         day.Date.Format(dateLayout),
		PreviousDate: day.Date.AddDate(0, 0, -1).Format(dateLayout),
		NextDate:     day.Date.AddDate(0, 0, 1).Format(dateLayout),
		Sections:     sections,
	}
}
