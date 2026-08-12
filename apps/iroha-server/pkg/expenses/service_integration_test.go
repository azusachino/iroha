//go:build integration

package expenses

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestServiceCreateIsIdempotentAndTombstonesIdentity(t *testing.T) {
	db := openIntegrationDB(t)
	clearExpenses(t, db)
	t.Cleanup(func() { clearExpenses(t, db) })
	svc := NewService(db)

	input := testCreateInput("receipt-idempotent", time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC))
	first, err := svc.Create(input)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if !first.Created {
		t.Fatal("first create was not marked Created")
	}

	retry, err := svc.Create(input)
	if err != nil {
		t.Fatalf("identical retry: %v", err)
	}
	if retry.Created || retry.Expense.ID != first.Expense.ID {
		t.Fatalf("retry = %+v, want same existing row", retry)
	}

	conflictInput := input
	conflictInput.Merchant = "Different merchant"
	if _, err := svc.Create(conflictInput); !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("conflicting retry error = %v, want ErrSourceConflict", err)
	}

	replaced, err := svc.Replace(first.Expense.ID, ReplaceInput{
		OccurredOn: input.OccurredOn, Currency: "JPY", AmountMinor: 1500,
		Category: "food", Merchant: "Updated merchant", Note: "updated", Items: []Item{{Name: "ramen"}},
	})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if replaced.AmountMinor != 1500 || replaced.Merchant != "Updated merchant" || replaced.SourceRef != input.Source.Ref || replaced.CreateFingerprint != first.Expense.CreateFingerprint {
		t.Fatalf("replaced expense = %+v", replaced)
	}

	current, err := svc.Create(input)
	if err != nil {
		t.Fatalf("retry after replace: %v", err)
	}
	if current.Expense.AmountMinor != 1500 {
		t.Fatalf("retry returned stale expense = %+v", current.Expense)
	}

	if err := svc.Delete(first.Expense.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := svc.Delete(first.Expense.ID); err != nil {
		t.Fatalf("repeated delete: %v", err)
	}
	if _, err := svc.Get(first.Expense.ID); !errors.Is(err, ErrDeleted) {
		t.Fatalf("get deleted error = %v, want ErrDeleted", err)
	}
	if _, err := svc.Replace(first.Expense.ID, ReplaceInput{
		OccurredOn: input.OccurredOn, Currency: "JPY", AmountMinor: 1, Category: "food",
	}); !errors.Is(err, ErrDeleted) {
		t.Fatalf("replace deleted error = %v, want ErrDeleted", err)
	}
	if _, err := svc.Create(input); !errors.Is(err, ErrDeleted) {
		t.Fatalf("create deleted identity error = %v, want ErrDeleted", err)
	}
}

func TestServiceListUsesHalfOpenDatesAndStableCursorOrder(t *testing.T) {
	db := openIntegrationDB(t)
	clearExpenses(t, db)
	t.Cleanup(func() { clearExpenses(t, db) })
	svc := NewService(db)

	if _, err := svc.Create(testCreateInput("receipt-old", time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(testCreateInput("receipt-new", time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.August, 2, 0, 0, 0, 0, time.UTC)
	page, err := svc.List(ListFilters{From: &from, To: &to, Limit: 1})
	if err != nil {
		t.Fatalf("first list: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].SourceRef != "receipt-old" || page.HasMore {
		t.Fatalf("filtered page = %+v, want only the inclusive from date", page)
	}

	all, err := svc.List(ListFilters{Limit: 1})
	if err != nil {
		t.Fatalf("first paged list: %v", err)
	}
	if len(all.Items) != 1 || all.Items[0].SourceRef != "receipt-new" || !all.HasMore || all.NextCursor == nil {
		t.Fatalf("first page = %+v", all)
	}
	next, err := svc.List(ListFilters{Limit: 1, Cursor: all.NextCursor})
	if err != nil {
		t.Fatalf("second paged list: %v", err)
	}
	if len(next.Items) != 1 || next.Items[0].SourceRef != "receipt-old" || next.HasMore {
		t.Fatalf("second page = %+v", next)
	}

	deletedID := next.Items[0].ID
	if err := svc.Delete(deletedID); err != nil {
		t.Fatalf("delete listed expense: %v", err)
	}
	remaining, err := svc.List(ListFilters{})
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(remaining.Items) != 1 || remaining.Items[0].SourceRef != "receipt-new" {
		t.Fatalf("remaining active expenses = %+v", remaining.Items)
	}
}

func TestServiceConcurrentCreateHasOneCanonicalRow(t *testing.T) {
	db := openIntegrationDB(t)
	clearExpenses(t, db)
	t.Cleanup(func() { clearExpenses(t, db) })

	input := testCreateInput("receipt-concurrent", time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC))
	results := make(chan CreateResult, 2)
	errorsCh := make(chan error, 2)
	var wait sync.WaitGroup
	for i := 0; i < 2; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := NewService(db).Create(input)
			results <- result
			errorsCh <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsCh)

	var rows []CreateResult
	for result := range results {
		rows = append(rows, result)
	}
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent create: %v", err)
		}
	}
	if len(rows) != 2 || rows[0].Expense.ID != rows[1].Expense.ID {
		t.Fatalf("concurrent results = %+v, want one canonical ID", rows)
	}
	createdCount := 0
	for _, result := range rows {
		if result.Created {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("created count = %d, want 1", createdCount)
	}
}

func testCreateInput(sourceRef string, occurredOn time.Time) CreateInput {
	return CreateInput{
		OccurredOn: occurredOn, Currency: "JPY", AmountMinor: 1300, Category: "food",
		Merchant: "Ramen Shop", Source: Source{Kind: "local_agent", Ref: sourceRef},
	}
}
