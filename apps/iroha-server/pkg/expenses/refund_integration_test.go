//go:build integration

package expenses

import (
	"testing"
	"time"
)

func TestRefundsKeepGrossRefundNetAndAccountsExplicit(t *testing.T) {
	db := openIntegrationDB(t)
	clearExpenses(t, db)
	t.Cleanup(func() { clearExpenses(t, db) })
	svc := NewService(db)

	purchase, err := svc.Create(CreateInput{
		OccurredOn: time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC),
		AccountKey: "card-main", Kind: KindExpense, Currency: "JPY", AmountMinor: 1000,
		Category: "food", Source: Source{Kind: "statement", Ref: "same-row"},
	})
	if err != nil {
		t.Fatalf("create purchase: %v", err)
	}
	if _, err := svc.Create(CreateInput{
		OccurredOn: time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC),
		AccountKey: "card-main", Kind: KindRefund, RefundOfExpenseID: &purchase.Expense.ID,
		Currency: "JPY", AmountMinor: 250, Category: "food", Source: Source{Kind: "statement", Ref: "refund-row"},
	}); err != nil {
		t.Fatalf("create linked refund: %v", err)
	}
	if _, err := svc.Create(CreateInput{
		OccurredOn: time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC),
		AccountKey: "cash", Kind: KindRefund, Currency: "USD", AmountMinor: 500,
		Category: "shopping", Source: Source{Kind: "statement", Ref: "same-row"},
	}); err != nil {
		t.Fatalf("create unlinked refund on another account: %v", err)
	}

	august, err := svc.PeriodReport(PeriodFilters{
		From: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("august report: %v", err)
	}
	if august.PurchaseCount != 1 || august.RefundCount != 0 || len(august.TotalsByCurrency) != 1 || august.TotalsByCurrency[0].GrossAmountMinor != 1000 || august.TotalsByCurrency[0].RefundAmountMinor != 0 || august.TotalsByCurrency[0].NetAmountMinor != 1000 {
		t.Fatalf("august report = %+v, want purchase gross 1000 and no refund", august)
	}

	september, err := svc.PeriodReport(PeriodFilters{
		From: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("september report: %v", err)
	}
	if september.PurchaseCount != 0 || september.RefundCount != 2 || len(september.TotalsByCurrency) != 2 {
		t.Fatalf("september report = %+v, want two refunds in separate currencies", september)
	}
	if september.TotalsByCurrency[0].NetAmountMinor >= 0 || september.TotalsByCurrency[0].RefundCount != 1 || september.TotalsByCurrency[1].NetAmountMinor >= 0 {
		t.Fatalf("september currency totals = %+v, want negative refund-only nets", september.TotalsByCurrency)
	}
}
