package reports

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

const DefaultTimezone = "Asia/Tokyo"

const (
	MonthlyReportSchema       = "monthly-report.v1"
	MonthlyReportSeriesSchema = "monthly-report-series.v2"
	MovementSchema            = "monthly-report.movement.v1"
	SleepSchema               = "monthly-report.sleep.v1"
	DailyHealthSchema         = "monthly-report.daily-health.v1"
	MediaSchema               = "monthly-report.media.v1"
	ExpensesSchema            = "monthly-report.expenses.v1"
	DefaultSeriesMonths       = 12
	MaxSeriesMonths           = 24
	CompletenessComplete      = "complete"
	CompletenessPartial       = "partial"
)

var (
	ErrInvalidMonth        = errors.New("invalid report month")
	ErrInvalidTimezone     = errors.New("invalid report timezone")
	ErrInvalidSeriesMonths = errors.New("invalid report series months")
)

// Period contains the calendar and instant boundaries shared by report
// adapters. Calendar dates are UTC midnight values; instants retain the
// requested IANA location and represent the same local month boundaries.
type Period struct {
	Month              string
	Timezone           string
	FromDate           time.Time
	ToDateExclusive    time.Time
	FromInstant        time.Time
	ToInstantExclusive time.Time
}

func ParseMonth(month, timezone string) (Period, error) {
	if len(month) != len("2006-01") || month[4] != '-' || !allDigits(month[:4]) || !allDigits(month[5:]) {
		return Period{}, ErrInvalidMonth
	}
	year, _ := strconv.Atoi(month[:4])
	monthNumber, _ := strconv.Atoi(month[5:])
	if year == 0 || monthNumber < 1 || monthNumber > 12 {
		return Period{}, ErrInvalidMonth
	}

	location, err := LoadTimezone(timezone)
	if err != nil {
		return Period{}, err
	}
	fromInstant := time.Date(year, time.Month(monthNumber), 1, 0, 0, 0, 0, location)
	toInstant := fromInstant.AddDate(0, 1, 0)
	return Period{
		Month:              month,
		Timezone:           location.String(),
		FromDate:           time.Date(year, time.Month(monthNumber), 1, 0, 0, 0, 0, time.UTC),
		ToDateExclusive:    time.Date(toInstant.Year(), toInstant.Month(), toInstant.Day(), 0, 0, 0, 0, time.UTC),
		FromInstant:        fromInstant,
		ToInstantExclusive: toInstant,
	}, nil
}

func LoadTimezone(timezone string) (*time.Location, error) {
	if timezone == "" {
		timezone = DefaultTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTimezone, timezone)
	}
	return location, nil
}

func (p Period) Wire() ReportMonth {
	return ReportMonth{
		Kind: "month", Month: p.Month, From: p.FromDate.Format("2006-01-02"),
		To: p.ToDateExclusive.Format("2006-01-02"), Timezone: p.Timezone,
	}
}

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}
