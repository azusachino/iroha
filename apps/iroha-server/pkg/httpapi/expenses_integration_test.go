//go:build integration

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
)

func TestIntegrationExpenseEndpoints(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	server := newIntegrationServer(t, db)

	createBody := `{"occurred_on":"2026-08-12","currency":"jpy","amount_minor":1300,"category":"FOOD","merchant":" Ramen Shop ","note":" lunch ","items":[{"name":"Ramen","amount_minor":800},{"name":"Tea"}],"source":{"kind":"local_agent","ref":"receipt-001"}}`
	created := requestJSON(t, server, http.MethodPost, "/api/v1/expenses", createBody, http.StatusCreated, func(body map[string]any) {
		if body["id"] == nil || body["occurred_on"] != "2026-08-12" || body["currency"] != "JPY" || body["currency_exponent"] != float64(0) || body["amount_minor"] != float64(1300) || body["category"] != "food" || body["merchant"] != "Ramen Shop" || body["note"] != "lunch" {
			t.Fatalf("created expense = %#v", body)
		}
		items := body["items"].([]any)
		if len(items) != 2 || items[0].(map[string]any)["amount_minor"] != float64(800) || items[1].(map[string]any)["amount_minor"] != nil {
			t.Fatalf("created items = %#v", items)
		}
		source := body["source"].(map[string]any)
		if source["kind"] != "local_agent" || source["ref"] != "receipt-001" {
			t.Fatalf("created source = %#v", source)
		}
	})
	id := stringValue(t, created, "id")

	retry := requestJSON(t, server, http.MethodPost, "/api/v1/expenses", createBody, http.StatusOK, nil)
	if stringValue(t, retry, "id") != id {
		t.Fatalf("retry id = %s, want %s", stringValue(t, retry, "id"), id)
	}
	requestJSON(t, server, http.MethodPost, "/api/v1/expenses", strings.Replace(createBody, "1300", "1400", 1), http.StatusConflict, assertErrorResponse(t, "expense_source_conflict", "expense source identity conflict"))

	requestJSON(t, server, http.MethodGet, "/api/v1/expenses/"+id, "", http.StatusOK, func(body map[string]any) {
		if body["id"] != id {
			t.Fatalf("expense detail = %#v", body)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/expenses/expense_bad", "", http.StatusBadRequest, assertErrorResponse(t, "invalid_expense_id", "invalid expense id"))

	requestJSON(t, server, http.MethodPut, "/api/v1/expenses/"+id, `{"occurred_on":"2026-08-13","currency":"JPY","amount_minor":1500,"category":"food","merchant":"Updated","note":"updated","items":[]}`, http.StatusOK, func(body map[string]any) {
		if body["occurred_on"] != "2026-08-13" || body["amount_minor"] != float64(1500) || body["merchant"] != "Updated" {
			t.Fatalf("replaced expense = %#v", body)
		}
		if body["source"].(map[string]any)["ref"] != "receipt-001" {
			t.Fatalf("replace lost source = %#v", body["source"])
		}
	})

	second := requestJSON(t, server, http.MethodPost, "/api/v1/expenses", `{"occurred_on":"2026-08-12","currency":"USD","amount_minor":2500,"category":"transport","source":{"kind":"local_agent","ref":"receipt-002"}}`, http.StatusCreated, nil)
	secondID := stringValue(t, second, "id")
	firstPage := requestJSON(t, server, http.MethodGet, "/api/v1/expenses?limit=1&from=2026-08-12&to=2026-08-14&currency=JPY", "", http.StatusOK, func(body map[string]any) {
		items := body["items"].([]any)
		if len(items) != 1 || items[0].(map[string]any)["id"] != id || body["has_more"] != false {
			t.Fatalf("filtered expense page = %#v", body)
		}
	})
	if firstPage["next_cursor"] != nil {
		t.Fatalf("filtered page cursor = %#v, want null", firstPage["next_cursor"])
	}
	pagedFirst := requestJSON(t, server, http.MethodGet, "/api/v1/expenses?limit=1", "", http.StatusOK, func(body map[string]any) {
		if len(body["items"].([]any)) != 1 || body["items"].([]any)[0].(map[string]any)["id"] != id || body["has_more"] != true {
			t.Fatalf("first expense page = %#v", body)
		}
		if body["next_cursor"] == nil {
			t.Fatal("first expense page has no cursor")
		}
	})
	cursor := stringValue(t, pagedFirst, "next_cursor")
	requestJSON(t, server, http.MethodGet, "/api/v1/expenses?limit=1&cursor="+cursor, "", http.StatusOK, func(body map[string]any) {
		items := body["items"].([]any)
		if len(items) != 1 || items[0].(map[string]any)["id"] != secondID || body["has_more"] != false {
			t.Fatalf("second expense page = %#v", body)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/expenses?from=2026-08-14&to=2026-08-12", "", http.StatusBadRequest, assertErrorResponse(t, "invalid_date_range", "invalid date range"))
	requestJSON(t, server, http.MethodGet, "/api/v1/expenses?category=travel", "", http.StatusBadRequest, assertErrorResponse(t, "invalid_category", "invalid category"))

	requestStatus(t, server, http.MethodDelete, "/api/v1/expenses/"+id, http.StatusNoContent)
	requestStatus(t, server, http.MethodDelete, "/api/v1/expenses/"+id, http.StatusNoContent)
	requestJSON(t, server, http.MethodGet, "/api/v1/expenses/"+id, "", http.StatusGone, assertErrorResponse(t, "expense_deleted", "expense is deleted"))
	requestJSON(t, server, http.MethodGet, "/api/v1/expenses", "", http.StatusOK, func(body map[string]any) {
		for _, item := range body["items"].([]any) {
			if item.(map[string]any)["id"] == id {
				t.Fatalf("deleted expense returned in list: %#v", body)
			}
		}
	})
}

func TestExpensePeriodReportIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	svc := expenses.NewService(db)

	create := func(ref string, date time.Time, currency string, amount int64, category string) expenses.CreateResult {
		t.Helper()
		result, err := svc.Create(expenses.CreateInput{
			OccurredOn: date, Currency: currency, AmountMinor: amount, Category: category,
			Source: expenses.Source{Kind: "test", Ref: ref},
		})
		if err != nil {
			t.Fatalf("create %s: %v", ref, err)
		}
		return result
	}
	create("aug-jpy-food-1", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), "JPY", 1300, "food")
	create("aug-jpy-food-2", time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC), "JPY", 500, "food")
	create("aug-usd-food", time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC), "USD", 2500, "food")
	create("aug-usd-transport", time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), "USD", 1000, "transport")
	create("sep-boundary", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), "JPY", 9999, "shopping")
	deleted := create("aug-deleted", time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC), "JPY", 700, "shopping")
	if err := svc.Delete(deleted.Expense.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	result, err := svc.PeriodReport(expenses.PeriodFilters{
		From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("period report: %v", err)
	}
	if result.ExpenseCount != 4 || len(result.TotalsByCurrency) != 2 || len(result.ByCategory) != 3 {
		t.Fatalf("result = %+v", result)
	}
	if result.TotalsByCurrency[0].Currency != "JPY" || result.TotalsByCurrency[0].AmountMinor != 1800 || result.TotalsByCurrency[0].ExpenseCount != 2 {
		t.Fatalf("JPY total = %+v", result.TotalsByCurrency[0])
	}
	if result.TotalsByCurrency[1].Currency != "USD" || result.TotalsByCurrency[1].AmountMinor != 3500 || result.TotalsByCurrency[1].ExpenseCount != 2 {
		t.Fatalf("USD total = %+v", result.TotalsByCurrency[1])
	}
	if result.ByCategory[0].Category != "food" || result.ByCategory[0].Currency != "JPY" || result.ByCategory[0].AmountMinor != 1800 || result.ByCategory[1].Category != "food" || result.ByCategory[1].Currency != "USD" || result.ByCategory[2].Category != "transport" {
		t.Fatalf("category totals = %+v", result.ByCategory)
	}
}

func requestStatus(t *testing.T, handler http.Handler, method, path string, wantStatus int) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d body = %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("%s %s body = %q, want empty", method, path, rec.Body.String())
	}
}
