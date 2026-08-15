package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type expenseItemRequest struct {
	Name        string `json:"name"`
	AmountMinor *int64 `json:"amount_minor"`
}

type expenseSourceRequest struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

type createExpenseRequest struct {
	OccurredOn  string               `json:"occurred_on"`
	Currency    string               `json:"currency"`
	AmountMinor int64                `json:"amount_minor"`
	Category    string               `json:"category"`
	Merchant    string               `json:"merchant"`
	Note        string               `json:"note"`
	Items       []expenseItemRequest `json:"items"`
	Source      expenseSourceRequest `json:"source"`
}

type replaceExpenseRequest struct {
	OccurredOn  string               `json:"occurred_on"`
	Currency    string               `json:"currency"`
	AmountMinor int64                `json:"amount_minor"`
	Category    string               `json:"category"`
	Merchant    string               `json:"merchant"`
	Note        string               `json:"note"`
	Items       []expenseItemRequest `json:"items"`
}

type expenseItemResponse struct {
	Name        string `json:"name"`
	AmountMinor *int64 `json:"amount_minor,omitempty"`
}

type expenseSourceResponse struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

type expenseResponse struct {
	ID               string                `json:"id"`
	OccurredOn       string                `json:"occurred_on"`
	Currency         string                `json:"currency"`
	CurrencyExponent int                   `json:"currency_exponent"`
	AmountMinor      int64                 `json:"amount_minor"`
	Category         string                `json:"category"`
	Merchant         string                `json:"merchant"`
	Note             string                `json:"note"`
	Items            []expenseItemResponse `json:"items"`
	Source           expenseSourceResponse `json:"source"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

type expensePageResponse struct {
	Items      []expenseResponse `json:"items"`
	NextCursor *string           `json:"next_cursor"`
	HasMore    bool              `json:"has_more"`
}

func (s *Server) handleCreateExpense(w http.ResponseWriter, r *http.Request) {
	if s.deps.ExpenseService == nil {
		writeError(w, http.StatusServiceUnavailable, "expense service unavailable")
		return
	}
	var request createExpenseRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	input, err := request.createInput()
	if err != nil {
		writeExpenseError(w, err)
		return
	}
	result, err := s.deps.ExpenseService.Create(input)
	if err != nil {
		writeExpenseError(w, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
		if err := s.invalidateExpenseCaches(r); err != nil {
			s.deps.Logger.Error("invalidate caches after expense creation", "error", err)
		}
	}
	writeJSON(w, status, toExpenseResponse(result.Expense))
}

func (s *Server) handleListExpenses(w http.ResponseWriter, r *http.Request) {
	if s.deps.ExpenseService == nil {
		writeError(w, http.StatusServiceUnavailable, "expense service unavailable")
		return
	}
	filters, ok := s.parseExpenseFilters(w, r)
	if !ok {
		return
	}
	page, err := s.deps.ExpenseService.List(filters)
	if err != nil {
		s.deps.Logger.Error("list expenses", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list expenses")
		return
	}
	response := expensePageResponse{Items: make([]expenseResponse, 0, len(page.Items)), HasMore: page.HasMore}
	for _, row := range page.Items {
		response.Items = append(response.Items, toExpenseResponse(row))
	}
	if page.NextCursor != nil {
		cursor := expenses.EncodeCursor(*page.NextCursor)
		response.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetExpense(w http.ResponseWriter, r *http.Request) {
	if s.deps.ExpenseService == nil {
		writeError(w, http.StatusServiceUnavailable, "expense service unavailable")
		return
	}
	id, ok := parseExpenseID(w, r)
	if !ok {
		return
	}
	row, err := s.deps.ExpenseService.Get(id)
	if err != nil {
		writeExpenseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toExpenseResponse(row))
}

func (s *Server) handleReplaceExpense(w http.ResponseWriter, r *http.Request) {
	if s.deps.ExpenseService == nil {
		writeError(w, http.StatusServiceUnavailable, "expense service unavailable")
		return
	}
	id, ok := parseExpenseID(w, r)
	if !ok {
		return
	}
	var request replaceExpenseRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	input, err := request.replaceInput()
	if err != nil {
		writeExpenseError(w, err)
		return
	}
	row, err := s.deps.ExpenseService.Replace(id, input)
	if err != nil {
		writeExpenseError(w, err)
		return
	}
	if err := s.invalidateExpenseCaches(r); err != nil {
		s.deps.Logger.Error("invalidate caches after expense replacement", "error", err)
	}
	writeJSON(w, http.StatusOK, toExpenseResponse(row))
}

func (s *Server) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	if s.deps.ExpenseService == nil {
		writeError(w, http.StatusServiceUnavailable, "expense service unavailable")
		return
	}
	id, ok := parseExpenseID(w, r)
	if !ok {
		return
	}
	if err := s.deps.ExpenseService.Delete(id); err != nil {
		writeExpenseError(w, err)
		return
	}
	if err := s.invalidateExpenseCaches(r); err != nil {
		s.deps.Logger.Error("invalidate caches after expense deletion", "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) invalidateExpenseCaches(r *http.Request) error {
	if s.deps.Cache == nil {
		return nil
	}
	return s.deps.Cache.InvalidateChange(r.Context(), cache.ChangeExpense)
}

func (r createExpenseRequest) createInput() (expenses.CreateInput, error) {
	occurredOn, err := time.Parse("2006-01-02", strings.TrimSpace(r.OccurredOn))
	if err != nil {
		return expenses.CreateInput{}, errors.Join(expenses.ErrInvalidExpense, errors.New("occurred_on"))
	}
	return expenses.CreateInput{
		OccurredOn:  occurredOn,
		Currency:    r.Currency,
		AmountMinor: r.AmountMinor,
		Category:    r.Category,
		Merchant:    r.Merchant,
		Note:        r.Note,
		Items:       toExpenseItems(r.Items),
		Source:      expenses.Source{Kind: r.Source.Kind, Ref: r.Source.Ref},
	}, nil
}

func (r replaceExpenseRequest) replaceInput() (expenses.ReplaceInput, error) {
	occurredOn, err := time.Parse("2006-01-02", strings.TrimSpace(r.OccurredOn))
	if err != nil {
		return expenses.ReplaceInput{}, errors.Join(expenses.ErrInvalidExpense, errors.New("occurred_on"))
	}
	return expenses.ReplaceInput{
		OccurredOn:  occurredOn,
		Currency:    r.Currency,
		AmountMinor: r.AmountMinor,
		Category:    r.Category,
		Merchant:    r.Merchant,
		Note:        r.Note,
		Items:       toExpenseItems(r.Items),
	}, nil
}

func toExpenseItems(items []expenseItemRequest) []expenses.Item {
	result := make([]expenses.Item, 0, len(items))
	for _, item := range items {
		result = append(result, expenses.Item{Name: item.Name, AmountMinor: item.AmountMinor})
	}
	return result
}

func parseExpenseFilters(w http.ResponseWriter, r *http.Request) (expenses.ListFilters, bool) {
	limit, ok := parsePageLimit(w, r)
	if !ok {
		return expenses.ListFilters{}, false
	}
	query := r.URL.Query()
	filters := expenses.ListFilters{Limit: limit}
	for key, destination := range map[string]**time.Time{"from": &filters.From, "to": &filters.To} {
		if value := query.Get(key); value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid "+key)
				return expenses.ListFilters{}, false
			}
			*destination = &parsed
		}
	}
	if filters.From != nil && filters.To != nil && filters.From.After(*filters.To) {
		writeError(w, http.StatusBadRequest, "invalid date range")
		return expenses.ListFilters{}, false
	}
	if value := query.Get("currency"); value != "" {
		filters.Currency = strings.ToUpper(strings.TrimSpace(value))
		if _, ok := expenses.SupportedCurrencies[filters.Currency]; !ok {
			writeError(w, http.StatusBadRequest, "invalid currency")
			return expenses.ListFilters{}, false
		}
	}
	if value := query.Get("category"); value != "" {
		filters.Category = strings.ToLower(strings.TrimSpace(value))
		if _, ok := expenses.SupportedCategories[filters.Category]; !ok {
			writeError(w, http.StatusBadRequest, "invalid category")
			return expenses.ListFilters{}, false
		}
	}
	if value := query.Get("cursor"); value != "" {
		cursor, err := expenses.DecodeCursor(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return expenses.ListFilters{}, false
		}
		filters.Cursor = &cursor
	}
	return filters, true
}

func parseExpenseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := ids.Decode(ids.ExpensePrefix, chi.URLParam(r, "expenseId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid expense id")
		return uuid.Nil, false
	}
	return id, true
}

func writeExpenseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, expenses.ErrInvalidExpense):
		writeError(w, http.StatusBadRequest, "invalid expense")
	case errors.Is(err, expenses.ErrSourceConflict):
		writeContractError(w, http.StatusConflict, "expense_source_conflict", err.Error())
	case errors.Is(err, expenses.ErrDeleted):
		writeContractError(w, http.StatusGone, "expense_deleted", err.Error())
	case errors.Is(err, expenses.ErrNotFound):
		writeError(w, http.StatusNotFound, "expense not found")
	default:
		writeError(w, http.StatusInternalServerError, "failed to access expense")
	}
}

func toExpenseResponse(row models.Expense) expenseResponse {
	items := make([]expenses.Item, 0)
	if len(row.ItemsJSON) > 0 {
		_ = json.Unmarshal(row.ItemsJSON, &items)
	}
	responseItems := make([]expenseItemResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, expenseItemResponse{Name: item.Name, AmountMinor: item.AmountMinor})
	}
	return expenseResponse{
		ID:               ids.Encode(ids.ExpensePrefix, row.ID),
		OccurredOn:       row.OccurredOn.Format("2006-01-02"),
		Currency:         row.Currency,
		CurrencyExponent: expenses.SupportedCurrencies[row.Currency],
		AmountMinor:      row.AmountMinor,
		Category:         row.Category,
		Merchant:         row.Merchant,
		Note:             row.Note,
		Items:            responseItems,
		Source:           expenseSourceResponse{Kind: row.SourceKind, Ref: row.SourceRef},
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}
