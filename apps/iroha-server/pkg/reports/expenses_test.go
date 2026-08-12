package reports

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
)

func TestExpensesDataMapsSeparateCurrencyTotals(t *testing.T) {
	data := expensesData(expenses.PeriodReport{
		ExpenseCount: 3,
		TotalsByCurrency: []expenses.PeriodCurrencyTotal{
			{Currency: "JPY", CurrencyExponent: 0, AmountMinor: 1800, ExpenseCount: 2},
			{Currency: "USD", CurrencyExponent: 2, AmountMinor: 2500, ExpenseCount: 1},
		},
		ByCategory: []expenses.PeriodCategoryTotal{{Category: "food", Currency: "JPY", CurrencyExponent: 0, AmountMinor: 1800, ExpenseCount: 2}},
	})

	if data.ExpenseCount != 3 || len(data.TotalsByCurrency) != 2 || data.TotalsByCurrency[0].Currency != "JPY" || data.TotalsByCurrency[1].Currency != "USD" {
		t.Fatalf("data = %+v", data)
	}
	if data.TotalsByCurrency[1].AmountMinor != 2500 || data.TotalsByCurrency[1].CurrencyExponent != 2 {
		t.Fatalf("USD total = %+v", data.TotalsByCurrency[1])
	}
}

func TestExpensesDataReturnsNilForEmptyReport(t *testing.T) {
	if expensesData(expenses.PeriodReport{}) != nil {
		t.Fatal("empty expense report should map to nil")
	}
}
