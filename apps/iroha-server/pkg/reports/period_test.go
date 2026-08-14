package reports

import (
	"errors"
	"testing"
	"time"
)

func TestParseMonthTokyoBoundary(t *testing.T) {
	period, err := ParseMonth("2026-02", "Asia/Tokyo")
	if err != nil {
		t.Fatalf("parse month: %v", err)
	}
	if !period.FromDate.Equal(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)) || !period.ToDateExclusive.Equal(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("date bounds = %v..%v", period.FromDate, period.ToDateExclusive)
	}
	if got := period.Wire(); got.Kind != "month" || got.Month != "2026-02" || got.From != "2026-02-01" || got.To != "2026-03-01" || got.Timezone != "Asia/Tokyo" {
		t.Fatalf("wire period = %+v", got)
	}
	if !period.FromInstant.Equal(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.FixedZone("+09", 9*60*60))) || !period.ToInstantExclusive.Equal(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.FixedZone("+09", 9*60*60))) {
		t.Fatalf("instant bounds = %v..%v", period.FromInstant, period.ToInstantExclusive)
	}
}

func TestParseMonthLeapAndYearRollover(t *testing.T) {
	leap, err := ParseMonth("2024-02", "UTC")
	if err != nil {
		t.Fatalf("parse leap month: %v", err)
	}
	if !leap.ToDateExclusive.Equal(time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("leap month end = %v", leap.ToDateExclusive)
	}
	rollover, err := ParseMonth("2026-12", "UTC")
	if err != nil {
		t.Fatalf("parse rollover month: %v", err)
	}
	if !rollover.ToInstantExclusive.Equal(time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("rollover end = %v", rollover.ToInstantExclusive)
	}
}

func TestParseMonthRejectsMalformedValuesAndTimezone(t *testing.T) {
	for _, month := range []string{"2026-1", "2026/01", "2026-00", "2026-13", "0000-01", "2026-01-01"} {
		if _, err := ParseMonth(month, "UTC"); !errors.Is(err, ErrInvalidMonth) {
			t.Errorf("ParseMonth(%q) error = %v, want ErrInvalidMonth", month, err)
		}
	}
	if _, err := ParseMonth("2026-01", "Not/A/Timezone"); !errors.Is(err, ErrInvalidTimezone) {
		t.Fatalf("invalid timezone error = %v, want ErrInvalidTimezone", err)
	}
	period, err := ParseMonth("2026-01", "")
	if err != nil || period.Timezone != DefaultTimezone {
		t.Fatalf("default timezone period = %+v, err = %v", period, err)
	}
}

func TestNewSectionDistinguishesEmptyFromAvailable(t *testing.T) {
	if section := NewSection[MovementData]("monthly-report.movement.v1", nil); section.State != SectionEmpty || section.Data != nil {
		t.Fatalf("empty section = %+v", section)
	}
	data := &MovementData{ActivityCount: 1}
	if section := NewSection("monthly-report.movement.v1", data); section.State != SectionAvailable || section.Data != data {
		t.Fatalf("available section = %+v", section)
	}
}
