package reports

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
)

func TestMovementDataIsTypedAndDeterministicallyOrdered(t *testing.T) {
	data := movementData(activities.PeriodReport{
		Totals: activities.SummaryTotals{ActivityCount: 2, DistanceM: 2500, DistanceKnownCount: 1, DurationS: 900},
		BySport: []activities.PeriodSportTotal{
			{Sport: "run", ActivityCount: 1, DistanceM: 2500, DistanceKnownCount: 1, DurationS: 600},
			{Sport: "bike", ActivityCount: 1, DurationS: 300},
		},
	})
	if data == nil || data.ActivityCount != 2 || data.DistanceActivityCount != 1 || len(data.BySport) != 2 {
		t.Fatalf("movement data = %+v", data)
	}
	if data.BySport[0].Sport != "bike" || data.BySport[1].Sport != "run" {
		t.Fatalf("sport order = %+v", data.BySport)
	}
	if movementData(activities.PeriodReport{}) != nil {
		t.Fatal("empty movement report returned a zero-valued payload")
	}
}
