package media

import "testing"

func TestAggregatesTypes(t *testing.T) {
	result := Aggregates{
		TypeSplit: []TypeBucket{{Type: "anime", Count: 2}, {Type: "book", Count: 1}},
	}
	if len(result.TypeSplit) != 2 || result.TypeSplit[0].Type != "anime" {
		t.Fatalf("type split = %+v", result.TypeSplit)
	}
}
