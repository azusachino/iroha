package mediaresolution

import (
	"testing"

	"github.com/google/uuid"
)

func TestStatuses(t *testing.T) {
	if StatusOpen == StatusResolved || StatusOpen == StatusDismissed || StatusResolved == StatusDismissed {
		t.Fatal("media resolution statuses must be distinct")
	}
}

func TestResolve_RejectsInvalidStatus(t *testing.T) {
	svc := NewService(nil)
	if _, err := svc.Resolve(uuid.Nil, "bogus", nil); err == nil {
		t.Fatal("expected error for a status other than resolved/dismissed")
	}
}
