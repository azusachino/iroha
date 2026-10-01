package metricseries

import (
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
)

type ActivityServiceSource struct {
	Service *activities.Service
}

func (s ActivityServiceSource) ActivityValues(from, to time.Time, timezone string) ([]ActivityMetricValue, error) {
	rows, err := s.Service.PeriodActivities(activities.PeriodFilters{From: from, To: to, Timezone: timezone})
	if err != nil {
		return nil, err
	}
	values := make([]ActivityMetricValue, len(rows))
	for index, row := range rows {
		values[index] = ActivityMetricValue{
			StartedAt: row.StartedAt,
			Sport:     metricSport(row.SportType),
			DistanceM: row.DistanceM,
			DurationS: row.DurationS,
			Source:    row.SourceKind,
		}
	}
	return values, nil
}

func metricSport(value string) string {
	switch observations.NormalizeSport(value) {
	case "hike":
		return "hike"
	case "ride":
		return "ride"
	case "run":
		return "run"
	case "swim":
		return "swim"
	case "walk":
		return "walk"
	default:
		return "other"
	}
}
