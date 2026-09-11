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
		gross, refunds         int64
		purchases, refundCount int
	}
	currencies := map[string]currencyTotal{}
	type categoryKey struct{ category, currency string }
	categories := map[categoryKey]currencyTotal{}
	for _, value := range values {
		magnitude := value.MagnitudeMinor
		if magnitude == 0 {
			magnitude = value.AmountMinor
			if magnitude < 0 {
				magnitude = -magnitude
			}
		}
		currency := currencies[value.Currency]
		if value.Kind == expenses.KindRefund || (value.Kind == "" && value.AmountMinor < 0) {
			currency.refunds += magnitude
			currency.refundCount++
		} else {
			currency.gross += magnitude
			currency.purchases++
		}
		currencies[value.Currency] = currency
		key := categoryKey{category: value.Category, currency: value.Currency}
		category := categories[key]
		if value.Kind == expenses.KindRefund || (value.Kind == "" && value.AmountMinor < 0) {
			category.refunds += magnitude
			category.refundCount++
		} else {
			category.gross += magnitude
			category.purchases++
		}
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
		totalsByCurrency = append(totalsByCurrency, ExpenseCurrencyTotal{Currency: currency, CurrencyExponent: expenses.SupportedCurrencies[currency], AmountMinor: total.gross - total.refunds, GrossAmountMinor: total.gross, RefundAmountMinor: total.refunds, NetAmountMinor: total.gross - total.refunds, ExpenseCount: total.purchases + total.refundCount, PurchaseCount: total.purchases, RefundCount: total.refundCount})
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
		byCategory = append(byCategory, ExpenseCategoryTotal{Category: key.category, Currency: key.currency, CurrencyExponent: expenses.SupportedCurrencies[key.currency], AmountMinor: total.gross - total.refunds, GrossAmountMinor: total.gross, RefundAmountMinor: total.refunds, NetAmountMinor: total.gross - total.refunds, ExpenseCount: total.purchases + total.refundCount, PurchaseCount: total.purchases, RefundCount: total.refundCount})
	}
	purchases, refunds := 0, 0
	for _, total := range currencies {
		purchases += total.purchases
		refunds += total.refundCount
	}
	return &ExpensesData{ExpenseCount: len(values), PurchaseCount: purchases, RefundCount: refunds, TotalsByCurrency: totalsByCurrency, ByCategory: byCategory}
}
