//go:build integration

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
