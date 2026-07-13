package httpapi

import "testing"

func TestNormalizedRating(t *testing.T) {
	rating, scale := 4.2, 5.0
	got := normalizedRating(&rating, &scale)
	if got == nil || *got != 8.4 {
		t.Fatalf("normalized rating = %v, want 8.4", got)
	}
}

func TestNormalizedRatingWithoutScale(t *testing.T) {
	rating := 8.0
	if got := normalizedRating(&rating, nil); got != nil {
		t.Fatalf("normalized rating = %v, want nil", got)
	}
}
