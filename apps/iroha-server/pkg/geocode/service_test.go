package geocode

import (
	"errors"
	"testing"
	"time"
)

func TestRetryAfterParsesSecondsAndFallsBack(t *testing.T) {
	if got := retryAfter("37"); got != 37*time.Second {
		t.Fatalf("retryAfter seconds = %s, want 37s", got)
	}
	if got := retryAfter("not-a-date"); got != defaultRetryAfter {
		t.Fatalf("retryAfter fallback = %s, want %s", got, defaultRetryAfter)
	}
}

func TestTooManyRequestsErrorCarriesRetryDelay(t *testing.T) {
	cause := errors.New("geocoder returned HTTP 429")
	err := &RetryAfterError{Cause: cause, Delay: 5 * time.Minute}
	if !errors.Is(err, cause) {
		t.Fatal("retry error does not unwrap to cause")
	}
	if delay, ok := err.RetryAfterDuration(); !ok || delay != 5*time.Minute {
		t.Fatalf("retry delay = %s, ok = %t", delay, ok)
	}
}
