package reports

import (
	"errors"
	"testing"
	"time"
)

func TestGenerateMonthlyValidatesPeriodBeforeServices(t *testing.T) {
	_, err := GenerateMonthly("2026-13", "UTC", Services{}, time.Time{})
	if !errors.Is(err, ErrInvalidMonth) {
		t.Fatalf("error = %v, want invalid month", err)
	}
	_, err = GenerateMonthly("2026-08", "Not/ATimezone", Services{}, time.Time{})
	if !errors.Is(err, ErrInvalidTimezone) {
		t.Fatalf("error = %v, want invalid timezone", err)
	}
}

func TestGenerateMonthlyRejectsMissingServices(t *testing.T) {
	_, err := GenerateMonthly("2026-08", "UTC", Services{}, time.Time{})
	if !errors.Is(err, ErrMissingService) {
		t.Fatalf("error = %v, want missing service", err)
	}
}

func TestGenerateMonthlySeriesValidatesWindow(t *testing.T) {
	for _, months := range []int{0, -1, MaxSeriesMonths + 1} {
		_, err := GenerateMonthlySeries("2026-08", "UTC", months, Services{}, time.Time{})
		if !errors.Is(err, ErrInvalidSeriesMonths) {
			t.Errorf("months=%d error = %v, want invalid series months", months, err)
		}
	}
	_, err := GenerateMonthlySeries("2026-13", "UTC", DefaultSeriesMonths, Services{}, time.Time{})
	if !errors.Is(err, ErrInvalidMonth) {
		t.Fatalf("invalid end month error = %v, want invalid month", err)
	}
}

func TestMonthlySeriesCompletenessUsesCanonicalPeriodBoundary(t *testing.T) {
	location, err := LoadTimezone("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	period, err := ParseMonth("2026-08", "Asia/Tokyo")
	if err != nil {
		t.Fatalf("parse period: %v", err)
	}
	if got := monthCompleteness(period.Wire(), time.Date(2026, 8, 31, 13, 0, 0, 0, time.UTC), location); got != CompletenessPartial {
		t.Fatalf("partial month completeness = %q", got)
	}
	if got := monthCompleteness(period.Wire(), time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), location); got != CompletenessComplete {
		t.Fatalf("complete month completeness = %q", got)
	}
}

func TestReportHasDataOnlyForAvailableSections(t *testing.T) {
	if reportHasData(MonthlyReport{}) {
		t.Fatal("empty report has data")
	}
	report := MonthlyReport{}
	report.Sections.Movement.Data = &MovementData{}
	if !reportHasData(report) {
		t.Fatal("available movement section was not detected")
	}
}
