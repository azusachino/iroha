package metricseries

import (
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
)

type SleepServiceSource struct {
	Service *sleep.Service
}

func (s SleepServiceSource) SleepValues(from, to time.Time) ([]SleepMetricValue, error) {
	rows, err := s.Service.PeriodSessions(sleep.PeriodFilters{From: from, To: to})
	if err != nil {
		return nil, err
	}
	values := make([]SleepMetricValue, len(rows))
	for index, row := range rows {
		values[index] = SleepMetricValue{WakeDate: row.WakeDate, SleepKind: row.SleepKind, AsleepS: row.AsleepS, Efficiency: row.Efficiency, Source: row.Source}
	}
	return values, nil
}
