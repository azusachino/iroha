package revisions

import "testing"

func TestOrderedNamespacesDeduplicatesAndSorts(t *testing.T) {
	got := orderedNamespaces([]string{"read_reports", "", "read_daily", "read_reports", "read_activities"})
	want := []string{"read_activities", "read_daily", "read_reports"}
	if len(got) != len(want) {
		t.Fatalf("ordered namespaces = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("ordered namespaces = %#v, want %#v", got, want)
		}
	}
}
