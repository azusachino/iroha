package metricseries

import (
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
)

type ExpenseServiceSource struct {
	Service *expenses.Service
}

func (s ExpenseServiceSource) ExpenseValues(from, to time.Time) ([]ExpenseMetricValue, error) {
	rows, err := s.Service.PeriodExpenses(expenses.PeriodFilters{From: from, To: to})
	if err != nil {
		return nil, err
	}
	values := make([]ExpenseMetricValue, len(rows))
	for index, row := range rows {
		values[index] = ExpenseMetricValue{
			OccurredOn:  row.OccurredOn,
			Currency:    row.Currency,
			Category:    row.Category,
			AmountMinor: row.AmountMinor,
			Source:      row.Source,
		}
	}
	return values, nil
}
