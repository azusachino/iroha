package httpapi

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/reports"
)

const (
	DefaultReadScopeTimezone = reports.DefaultTimezone
	MaxReadScopeDays         = 3660
	dateLayout               = "2006-01-02"
	monthLayout              = "2006-01"
	yearLayout               = "2006"
)

var (
	ErrInvalidReadScope  = errors.New("invalid read scope")
	ErrInvalidScopeDate  = errors.New("invalid read scope date")
	ErrInvalidScopeRange = errors.New("invalid read scope range")
	ErrInvalidScopeTZ    = errors.New("invalid read scope timezone")
	ErrConflictingScope  = errors.New("conflicting read scope")
	ErrFutureReadScope   = errors.New("future read scope")
	ErrExcessiveScope    = errors.New("excessive read scope")
	ErrMissingScopeNow   = errors.New("read scope now is required")
)

type ScopeKind string

const (
	ScopeLifetime ScopeKind = "lifetime"
	ScopeYear     ScopeKind = "year"
	ScopeMonth    ScopeKind = "month"
	ScopeDay      ScopeKind = "day"
	ScopeRange    ScopeKind = "range"
)

// ReadScopeInput is the transport-neutral input a handler can map from its
// query parameters. Lifetime is explicit so no invalid date syntax is given a
// special meaning.
type ReadScopeInput struct {
	Date     string
	From     string
	To       string
	Timezone string
	Lifetime bool
}

type ReadScope struct {
	Kind     ScopeKind
	Timezone string

	// Calendar bounds are UTC-midnight date values for date-keyed tables.
	Calendar CalendarBounds
	// Instant bounds are local-midnight boundaries represented in the selected
	// IANA location for timestamp-keyed tables.
	Instant InstantBounds
}

type CalendarBounds struct {
	From        time.Time
	ToExclusive time.Time
}

type InstantBounds struct {
	From        time.Time
	ToExclusive time.Time
}

// ResolveReadScope validates and resolves a scalar date or explicit date
// range. now must be supplied by the caller so request validation is
// deterministic and testable.
func ResolveReadScope(input ReadScopeInput, now time.Time) (ReadScope, error) {
	if now.IsZero() {
		return ReadScope{}, ErrMissingScopeNow
	}
	location, err := loadReadScopeTimezone(input.Timezone)
	if err != nil {
		return ReadScope{}, err
	}

	hasDate := input.Date != ""
	hasFrom := input.From != ""
	hasTo := input.To != ""
	hasRange := hasFrom || hasTo
	if input.Lifetime && (hasDate || hasRange) {
		return ReadScope{}, ErrConflictingScope
	}
	if hasDate && hasRange {
		return ReadScope{}, ErrConflictingScope
	}
	if !input.Lifetime && !hasDate && !hasRange {
		return ReadScope{}, ErrInvalidReadScope
	}

	var from, to time.Time
	kind := ScopeRange
	switch {
	case input.Lifetime:
		return ReadScope{Kind: ScopeLifetime, Timezone: location.String()}, nil
	case hasDate:
		from, to, kind, err = scalarReadScope(input.Date, location)
	case hasRange:
		if !hasFrom || !hasTo {
			return ReadScope{}, ErrInvalidScopeRange
		}
		from, to, err = dateRange(input.From, input.To, location)
	default:
		return ReadScope{}, ErrInvalidReadScope
	}
	if err != nil {
		return ReadScope{}, err
	}

	nowInLocation := now.In(location)
	if futureCalendarDate(from, nowInLocation) {
		return ReadScope{}, ErrFutureReadScope
	}
	if utcDate(to).Sub(utcDate(from)) > MaxReadScopeDays*24*time.Hour {
		return ReadScope{}, ErrExcessiveScope
	}

	return ReadScope{
		Kind:     kind,
		Timezone: location.String(),
		Calendar: CalendarBounds{
			From:        utcDate(from),
			ToExclusive: utcDate(to),
		},
		Instant: InstantBounds{From: from, ToExclusive: to},
	}, nil
}

func futureCalendarDate(value, now time.Time) bool {
	if value.Year() != now.Year() {
		return value.Year() > now.Year()
	}
	if value.Month() != now.Month() {
		return value.Month() > now.Month()
	}
	return value.Day() > now.Day()
}

func scalarReadScope(value string, location *time.Location) (time.Time, time.Time, ScopeKind, error) {
	switch len(value) {
	case len(yearLayout):
		year, err := parseYear(value)
		if err != nil {
			return time.Time{}, time.Time{}, "", err
		}
		from := time.Date(year, time.January, 1, 0, 0, 0, 0, location)
		return from, from.AddDate(1, 0, 0), ScopeYear, nil
	case len(monthLayout):
		if value[4] != '-' {
			return time.Time{}, time.Time{}, "", ErrInvalidScopeDate
		}
		year, err := parseYear(value[:4])
		if err != nil {
			return time.Time{}, time.Time{}, "", err
		}
		month, err := parsePart(value[5:], 1, 12)
		if err != nil {
			return time.Time{}, time.Time{}, "", ErrInvalidScopeDate
		}
		from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, location)
		return from, from.AddDate(0, 1, 0), ScopeMonth, nil
	case len(dateLayout):
		if value[4] != '-' || value[7] != '-' {
			return time.Time{}, time.Time{}, "", ErrInvalidScopeDate
		}
		from, err := parseDate(value, location)
		if err != nil {
			return time.Time{}, time.Time{}, "", err
		}
		return from, from.AddDate(0, 0, 1), ScopeDay, nil
	default:
		return time.Time{}, time.Time{}, "", ErrInvalidScopeDate
	}
}

func dateRange(fromValue, toValue string, location *time.Location) (time.Time, time.Time, error) {
	from, err := parseDate(fromValue, location)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := parseDate(toValue, location)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !to.After(from) {
		return time.Time{}, time.Time{}, ErrInvalidScopeRange
	}
	return from, to, nil
}

func parseDate(value string, location *time.Location) (time.Time, error) {
	parsed, err := time.ParseInLocation(dateLayout, value, location)
	if err != nil || parsed.Format(dateLayout) != value {
		return time.Time{}, ErrInvalidScopeDate
	}
	return parsed, nil
}

func parseYear(value string) (int, error) {
	if len(value) != 4 || !allReadScopeDigits(value) {
		return 0, ErrInvalidScopeDate
	}
	year, err := strconv.Atoi(value)
	if err != nil || year == 0 {
		return 0, ErrInvalidScopeDate
	}
	return year, nil
}

func parsePart(value string, min, max int) (int, error) {
	if !allReadScopeDigits(value) {
		return 0, ErrInvalidScopeDate
	}
	part, err := strconv.Atoi(value)
	if err != nil || part < min || part > max {
		return 0, ErrInvalidScopeDate
	}
	return part, nil
}

func loadReadScopeTimezone(value string) (*time.Location, error) {
	location, err := reports.LoadTimezone(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidScopeTZ, valueOrDefaultTimezone(value))
	}
	return location, nil
}

func valueOrDefaultTimezone(value string) string {
	if value == "" {
		return DefaultReadScopeTimezone
	}
	return value
}

func utcDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func allReadScopeDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
