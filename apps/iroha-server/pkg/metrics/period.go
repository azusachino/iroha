package metrics

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidPeriod    = errors.New("invalid metric period")
	ErrUnsupportedGrain = errors.New("unsupported metric grain")
)

type Period struct {
	Grain    string `json:"grain"`
	From     string `json:"from"`
	To       string `json:"to"`
	Timezone string `json:"timezone"`
}

func BuildPeriods(from, to time.Time, grain string, location *time.Location) ([]string, error) {
	if location == nil {
		return nil, ErrInvalidPeriod
	}
	if !from.Before(to) {
		return nil, ErrInvalidPeriod
	}
	from = dateInLocation(from, location)
	to = dateInLocation(to, location)
	if !from.Before(to) {
		return nil, ErrInvalidPeriod
	}
	if grain == "month" && from.Day() != 1 {
		return nil, fmt.Errorf("%w: month ranges must start on the first day", ErrInvalidPeriod)
	}
	if grain == "year" && (from.Month() != time.January || from.Day() != 1) {
		return nil, fmt.Errorf("%w: year ranges must start on January 1", ErrInvalidPeriod)
	}

	step := func(value time.Time) time.Time { return value.AddDate(0, 0, 1) }
	format := "2006-01-02"
	if grain == "month" {
		step = func(value time.Time) time.Time { return value.AddDate(0, 1, 0) }
		format = "2006-01"
	} else if grain == "year" {
		step = func(value time.Time) time.Time { return value.AddDate(1, 0, 0) }
		format = "2006"
	} else if grain != "day" {
		return nil, ErrUnsupportedGrain
	}

	periods := make([]string, 0)
	for current := from; current.Before(to); current = step(current) {
		periods = append(periods, current.Format(format))
	}
	return periods, nil
}

func dateInLocation(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}
