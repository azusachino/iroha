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
