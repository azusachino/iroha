package reports

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
)

func TestExpensesDataMapsSeparateCurrencyTotals(t *testing.T) {
	data := expensesDataValues([]expenses.MetricValue{
		{Currency: "JPY", Category: "food", AmountMinor: 900},
		{Currency: "JPY", Category: "food", AmountMinor: 900},
		{Currency: "USD", Category: "transport", AmountMinor: 2500},
	})

	if data.ExpenseCount != 3 || len(data.TotalsByCurrency) != 2 || data.TotalsByCurrency[0].Currency != "JPY" || data.TotalsByCurrency[1].Currency != "USD" {
		t.Fatalf("data = %+v", data)
	}
	if data.TotalsByCurrency[1].AmountMinor != 2500 || data.TotalsByCurrency[1].CurrencyExponent != 2 {
		t.Fatalf("USD total = %+v", data.TotalsByCurrency[1])
	}
}

func TestExpensesDataReturnsNilForEmptyReport(t *testing.T) {
	if expensesDataValues(nil) != nil {
		t.Fatal("empty expense report should map to nil")
	}
}
