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
	Date     time.Time
	Start    time.Time
	End      time.Time
	Timezone string
}

func ParseDay(value string) (Day, error) {
	return ParseDayInLocation(value, "UTC")
}

func ParseDayInLocation(value, timezone string) (Day, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Day{}, fmt.Errorf("invalid briefing timezone: %w", err)
	}
	date, err := time.ParseInLocation(dateLayout, value, location)
	if err != nil {
		return Day{}, fmt.Errorf("invalid briefing date: %w", err)
	}
	return Day{Date: date, Start: date, End: date.AddDate(0, 0, 1), Timezone: location.String()}, nil
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
	Status SourceStatus `json:"status"`
	Data   any          `json:"data"`
}

// SourceStatus intentionally keeps transport, collection and freshness
// independent. The briefing can report canonical content without claiming
// that the source covered the whole requested day.
type SourceStatus struct {
	Availability string `json:"availability"`
	Collection   string `json:"collection"`
	Operation    string `json:"operation"`
	Freshness    string `json:"freshness"`
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
				Status: SourceStatus{Availability: "supported", Collection: "unknown", Operation: "failed", Freshness: "unknown"},
				Data:   map[string]string{},
			})
			continue
		}
		section.Key = contributor.Key()
		section.Schema = contributor.Schema()
		if section.State == "" {
			section.State = StateReady
		}
		if section.Status.Availability == "" {
			section.Status.Availability = "supported"
		}
		if section.Status.Collection == "" {
			section.Status.Collection = "unknown"
		}
		if section.Status.Operation == "" {
			section.Status.Operation = "idle"
		}
		if section.Status.Freshness == "" {
			section.Status.Freshness = "unknown"
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
