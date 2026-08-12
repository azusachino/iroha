package reports

import (
	"sort"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
)

func Expenses(service *expenses.Service, period Period) (*ExpensesData, error) {
	values, err := service.PeriodExpenses(expenses.PeriodFilters{From: period.FromDate, To: period.ToDateExclusive})
	if err != nil {
		return nil, err
	}
	return expensesDataValues(values), nil
}

func expensesDataValues(values []expenses.MetricValue) *ExpensesData {
	if len(values) == 0 {
		return nil
	}

	type currencyTotal struct {
		amount int64
		count  int
	}
	currencies := map[string]currencyTotal{}
	type categoryKey struct{ category, currency string }
	categories := map[categoryKey]currencyTotal{}
	for _, value := range values {
		currency := currencies[value.Currency]
		currency.amount += value.AmountMinor
		currency.count++
		currencies[value.Currency] = currency
		key := categoryKey{category: value.Category, currency: value.Currency}
		category := categories[key]
		category.amount += value.AmountMinor
		category.count++
		categories[key] = category
	}
	currencyKeys := make([]string, 0, len(currencies))
	for currency := range currencies {
		currencyKeys = append(currencyKeys, currency)
	}
	sort.Strings(currencyKeys)
	totalsByCurrency := make([]ExpenseCurrencyTotal, 0, len(currencyKeys))
	for _, currency := range currencyKeys {
		total := currencies[currency]
		totalsByCurrency = append(totalsByCurrency, ExpenseCurrencyTotal{Currency: currency, CurrencyExponent: expenses.SupportedCurrencies[currency], AmountMinor: total.amount, ExpenseCount: total.count})
	}
	categoryKeys := make([]categoryKey, 0, len(categories))
	for key := range categories {
		categoryKeys = append(categoryKeys, key)
	}
	sort.Slice(categoryKeys, func(i, j int) bool {
		if categoryKeys[i].category == categoryKeys[j].category {
			return categoryKeys[i].currency < categoryKeys[j].currency
		}
		return categoryKeys[i].category < categoryKeys[j].category
	})
	byCategory := make([]ExpenseCategoryTotal, 0, len(categoryKeys))
	for _, key := range categoryKeys {
		total := categories[key]
		byCategory = append(byCategory, ExpenseCategoryTotal{Category: key.category, Currency: key.currency, CurrencyExponent: expenses.SupportedCurrencies[key.currency], AmountMinor: total.amount, ExpenseCount: total.count})
	}
	return &ExpensesData{ExpenseCount: len(values), TotalsByCurrency: totalsByCurrency, ByCategory: byCategory}
}
