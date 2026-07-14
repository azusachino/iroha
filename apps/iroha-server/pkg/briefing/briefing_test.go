package briefing

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type testContributor struct {
	key   string
	state SectionState
	err   error
	data  any
}

func (c testContributor) Key() string    { return c.key }
func (c testContributor) Schema() string { return c.key + ".day.v1" }
func (c testContributor) Contribute(context.Context, Day) (Section, error) {
	return Section{State: c.state, Data: c.data}, c.err
}

func TestRegistryBuildsOrderedSectionsAndIsolatesErrors(t *testing.T) {
	registry, err := NewRegistry(
		testContributor{key: "daily", data: map[string]int{"steps": 3}},
		testContributor{key: "sleep", err: errors.New("database unavailable")},
		testContributor{key: "media", state: StateEmpty, data: map[string]any{}},
	)
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}

	day, err := ParseDay("2026-07-14")
	if err != nil {
		t.Fatalf("parse day: %v", err)
	}
	response := registry.Build(context.Background(), day)
	if response.Date != "2026-07-14" {
		t.Fatalf("date = %q", response.Date)
	}
	if response.PreviousDate != "2026-07-13" || response.NextDate != "2026-07-15" {
		t.Fatalf("navigation = %q -> %q", response.PreviousDate, response.NextDate)
	}
	if got := []string{response.Sections[0].Key, response.Sections[1].Key, response.Sections[2].Key}; !reflect.DeepEqual(got, []string{"daily", "sleep", "media"}) {
		t.Fatalf("section order = %v", got)
	}
	if response.Sections[0].State != StateReady {
		t.Fatalf("default state = %q", response.Sections[0].State)
	}
	if response.Sections[1].State != StateUnavailable {
		t.Fatalf("failed state = %q", response.Sections[1].State)
	}
	if response.Sections[2].State != StateEmpty {
		t.Fatalf("empty state = %q", response.Sections[2].State)
	}
}

func TestRegistryRejectsDuplicateKeys(t *testing.T) {
	_, err := NewRegistry(testContributor{key: "daily"}, testContributor{key: "daily"})
	if !errors.Is(err, ErrDuplicateContributor) {
		t.Fatalf("error = %v, want duplicate contributor", err)
	}
}

func TestParseDayUsesHalfOpenUTCWindow(t *testing.T) {
	day, err := ParseDay("2026-07-14")
	if err != nil {
		t.Fatalf("parse day: %v", err)
	}
	if day.Start.UTC().Format("2006-01-02T15:04:05Z07:00") != "2026-07-14T00:00:00Z" {
		t.Fatalf("start = %s", day.Start)
	}
	if day.End.Sub(day.Start).Hours() != 24 {
		t.Fatalf("window = %s", day.End.Sub(day.Start))
	}
}
