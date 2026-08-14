package reports

import (
	"sort"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
)

// Movement aggregates canonical activities directly for the report period.
// A nil result means the period has no activities and should serialize as an
// empty report section rather than a fabricated zero-valued payload.
func Movement(service *activities.Service, period Period) (*MovementData, error) {
	result, err := service.PeriodReport(activities.PeriodFilters{
		From: period.FromInstant, To: period.ToInstantExclusive, Timezone: period.Timezone,
	})
	if err != nil {
		return nil, err
	}
	return movementData(result), nil
}

func movementData(result activities.PeriodReport) *MovementData {
	if result.Totals.ActivityCount == 0 {
		return nil
	}
	bySport := make([]MovementSportTotal, 0, len(result.BySport))
	for _, sport := range result.BySport {
		bySport = append(bySport, MovementSportTotal{
			Sport: sport.Sport, ActivityCount: sport.ActivityCount,
			DistanceM: sport.DistanceM, DistanceActivityCount: sport.DistanceKnownCount,
			DurationS: sport.DurationS,
		})
	}
	sort.Slice(bySport, func(i, j int) bool { return bySport[i].Sport < bySport[j].Sport })
	return &MovementData{
		ActivityCount:         result.Totals.ActivityCount,
		DistanceM:             result.Totals.DistanceM,
		DistanceActivityCount: result.Totals.DistanceKnownCount,
		DurationS:             result.Totals.DurationS,
		BySport:               bySport,
	}
}
