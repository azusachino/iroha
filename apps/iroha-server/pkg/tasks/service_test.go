package tasks

import "testing"

func TestNormalizeTitle(t *testing.T) {
	if got := NormalizeTitle("  review sleep  "); got != "review sleep" {
		t.Fatalf("NormalizeTitle() = %q, want %q", got, "review sleep")
	}
}

func TestStatuses(t *testing.T) {
	if StatusOpen == StatusCompleted || StatusOpen == StatusCanceled {
		t.Fatal("task statuses must be distinct")
	}
}
