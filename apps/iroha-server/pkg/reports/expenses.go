package reports

import (
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
)

func Expenses(service *expenses.Service, period Period) (*ExpensesData, error) {
	result, err := service.PeriodReport(expenses.PeriodFilters{From: period.FromDate, To: period.ToDateExclusive})
	if err != nil {
		return nil, err
	}
	return expensesData(result), nil
}

func expensesData(result expenses.PeriodReport) *ExpensesData {
	if result.ExpenseCount == 0 {
		return nil
	}

	totalsByCurrency := make([]ExpenseCurrencyTotal, 0, len(result.TotalsByCurrency))
	for _, total := range result.TotalsByCurrency {
		totalsByCurrency = append(totalsByCurrency, ExpenseCurrencyTotal{
			Currency: total.Currency, CurrencyExponent: total.CurrencyExponent,
			AmountMinor: total.AmountMinor, ExpenseCount: total.ExpenseCount,
		})
	}
	byCategory := make([]ExpenseCategoryTotal, 0, len(result.ByCategory))
	for _, total := range result.ByCategory {
		byCategory = append(byCategory, ExpenseCategoryTotal{
			Category: total.Category, Currency: total.Currency, CurrencyExponent: total.CurrencyExponent,
			AmountMinor: total.AmountMinor, ExpenseCount: total.ExpenseCount,
		})
	}
	return &ExpensesData{ExpenseCount: result.ExpenseCount, TotalsByCurrency: totalsByCurrency, ByCategory: byCategory}
}
