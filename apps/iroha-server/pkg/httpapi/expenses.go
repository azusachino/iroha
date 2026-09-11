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
	OccurredOn             string               `json:"occurred_on"`
	AccountKey             string               `json:"account_key"`
	Kind                   string               `json:"kind"`
	RefundOf               string               `json:"refund_of,omitempty"`
	OriginalTransactionRef string               `json:"original_transaction_ref,omitempty"`
	Currency               string               `json:"currency"`
	AmountMinor            int64                `json:"amount_minor"`
	Category               string               `json:"category"`
	Merchant               string               `json:"merchant"`
	Note                   string               `json:"note"`
	Items                  []expenseItemRequest `json:"items"`
	Source                 expenseSourceRequest `json:"source"`
}

type replaceExpenseRequest struct {
	OccurredOn             string               `json:"occurred_on"`
	AccountKey             string               `json:"account_key"`
	Kind                   string               `json:"kind"`
	RefundOf               string               `json:"refund_of,omitempty"`
	OriginalTransactionRef string               `json:"original_transaction_ref,omitempty"`
	Currency               string               `json:"currency"`
	AmountMinor            int64                `json:"amount_minor"`
	Category               string               `json:"category"`
	Merchant               string               `json:"merchant"`
	Note                   string               `json:"note"`
	Items                  []expenseItemRequest `json:"items"`
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
	ID                     string                `json:"id"`
	OccurredOn             string                `json:"occurred_on"`
	AccountKey             string                `json:"account_key"`
	Kind                   string                `json:"kind"`
	RefundOf               string                `json:"refund_of,omitempty"`
	OriginalTransactionRef string                `json:"original_transaction_ref,omitempty"`
	Currency               string                `json:"currency"`
	CurrencyExponent       int                   `json:"currency_exponent"`
	AmountMinor            int64                 `json:"amount_minor"`
	Category               string                `json:"category"`
	Merchant               string                `json:"merchant"`
	Note                   string                `json:"note"`
	Items                  []expenseItemResponse `json:"items"`
	Source                 expenseSourceResponse `json:"source"`
	CreatedAt              time.Time             `json:"created_at"`
	UpdatedAt              time.Time             `json:"updated_at"`
}

type statementManifestRequest struct {
	AccountKey   string `json:"account_key"`
	SourceKind   string `json:"source_kind"`
	StatementRef string `json:"statement_ref"`
	PeriodFrom   string `json:"period_from"`
	PeriodTo     string `json:"period_to"`
	Completeness string `json:"completeness"`
	Revision     int64  `json:"revision"`
}

type statementRequest struct {
	Manifest statementManifestRequest `json:"manifest"`
	CSV      string                   `json:"csv"`
}

type statementPreviewResponse struct {
	Manifest statementManifestRequest            `json:"manifest"`
	SHA256   string                              `json:"sha256"`
	RowCount int                                 `json:"row_count"`
	Valid    bool                                `json:"valid"`
	Errors   []expenses.StatementValidationError `json:"errors"`
}

type statementResponse struct {
	ID           string    `json:"id"`
	AccountKey   string    `json:"account_key"`
	SourceKind   string    `json:"source_kind"`
	StatementRef string    `json:"statement_ref"`
	PeriodFrom   string    `json:"period_from"`
	PeriodTo     string    `json:"period_to"`
	Completeness string    `json:"completeness"`
	Revision     int64     `json:"revision"`
	SHA256       string    `json:"sha256"`
	RowCount     int       `json:"row_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type statementImportResponse struct {
	Statement  statementResponse `json:"statement"`
	Created    bool              `json:"created"`
	Idempotent bool              `json:"idempotent"`
	Changed    int               `json:"changed"`
	Tombstoned int               `json:"tombstoned"`
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

func (s *Server) handlePreviewExpenseStatement(w http.ResponseWriter, r *http.Request) {
	if s.deps.ExpenseService == nil {
		writeError(w, http.StatusServiceUnavailable, "expense service unavailable")
		return
	}
	var request statementRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	manifest, err := request.Manifest.input()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview := s.deps.ExpenseService.PreviewStatement(manifest, []byte(request.CSV))
	writeJSON(w, http.StatusOK, statementPreviewResponse{Manifest: statementManifestResponse(preview.Manifest), SHA256: preview.SHA256, RowCount: preview.RowCount, Valid: preview.Valid, Errors: preview.Errors})
}

func (s *Server) handleImportExpenseStatement(w http.ResponseWriter, r *http.Request) {
	if s.deps.ExpenseService == nil {
		writeError(w, http.StatusServiceUnavailable, "expense service unavailable")
		return
	}
	var request statementRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	manifest, err := request.Manifest.input()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.deps.ExpenseService.ImportStatement(manifest, []byte(request.CSV))
	if err != nil {
		switch {
		case errors.Is(err, expenses.ErrStatementConflict), errors.Is(err, expenses.ErrStatementOutOfDate), errors.Is(err, expenses.ErrSourceConflict):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, expenses.ErrInvalidStatement):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			s.deps.Logger.Error("import expense statement", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to import expense statement")
		}
		return
	}
	if result.Created && (result.Changed > 0 || result.Tombstoned > 0) {
		if err := s.invalidateExpenseCaches(r); err != nil {
			s.deps.Logger.Error("invalidate caches after expense statement import", "error", err)
		}
	}
	status := http.StatusCreated
	if result.Idempotent {
		status = http.StatusOK
	}
	writeJSON(w, status, statementImportResponse{Statement: toStatementResponse(result.Statement), Created: result.Created, Idempotent: result.Idempotent, Changed: result.Changed, Tombstoned: result.Tombstoned})
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

func (s *Server) handleExpenseBounds(w http.ResponseWriter, r *http.Request) {
	if s.deps.ExpenseService == nil {
		writeError(w, http.StatusServiceUnavailable, "expense service unavailable")
		return
	}
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = s.deps.Config.Server.Timezone
	}
	minDate, maxDate, ok, err := s.deps.ExpenseService.Bounds(s.clockNow(), timezone)
	if err != nil {
		if strings.Contains(err.Error(), "load timezone") {
			writeError(w, http.StatusBadRequest, "invalid timezone")
			return
		}
		s.deps.Logger.Error("expense bounds", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load expense bounds")
		return
	}
	writeBounds(w, minDate, maxDate, ok)
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
	refundOf, err := parseOptionalExpenseID(r.RefundOf)
	if err != nil {
		return expenses.CreateInput{}, err
	}
	return expenses.CreateInput{
		OccurredOn: occurredOn, AccountKey: r.AccountKey, Kind: r.Kind, RefundOfExpenseID: refundOf, OriginalTransactionRef: r.OriginalTransactionRef,
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
	refundOf, err := parseOptionalExpenseID(r.RefundOf)
	if err != nil {
		return expenses.ReplaceInput{}, err
	}
	return expenses.ReplaceInput{
		OccurredOn: occurredOn, AccountKey: r.AccountKey, Kind: r.Kind, RefundOfExpenseID: refundOf, OriginalTransactionRef: r.OriginalTransactionRef,
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
	if value := query.Get("account_key"); value != "" {
		filters.AccountKey = strings.TrimSpace(value)
	}
	if value := query.Get("kind"); value != "" {
		filters.Kind = strings.ToLower(strings.TrimSpace(value))
		if filters.Kind != expenses.KindExpense && filters.Kind != expenses.KindRefund {
			writeError(w, http.StatusBadRequest, "invalid kind")
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

func parseOptionalExpenseID(value string) (*uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	id, err := ids.Decode(ids.ExpensePrefix, strings.TrimSpace(value))
	if err != nil {
		return nil, errors.Join(expenses.ErrInvalidExpense, errors.New("refund_of"))
	}
	return &id, nil
}

func (r statementManifestRequest) input() (expenses.StatementManifest, error) {
	from, err := time.Parse("2006-01-02", strings.TrimSpace(r.PeriodFrom))
	if err != nil {
		return expenses.StatementManifest{}, errors.New("invalid period_from")
	}
	to, err := time.Parse("2006-01-02", strings.TrimSpace(r.PeriodTo))
	if err != nil {
		return expenses.StatementManifest{}, errors.New("invalid period_to")
	}
	return expenses.StatementManifest{AccountKey: r.AccountKey, SourceKind: r.SourceKind, StatementRef: r.StatementRef, PeriodFrom: from, PeriodTo: to, Completeness: r.Completeness, Revision: r.Revision}, nil
}

func statementManifestResponse(manifest expenses.StatementManifest) statementManifestRequest {
	return statementManifestRequest{AccountKey: manifest.AccountKey, SourceKind: manifest.SourceKind, StatementRef: manifest.StatementRef, PeriodFrom: manifest.PeriodFrom.Format("2006-01-02"), PeriodTo: manifest.PeriodTo.Format("2006-01-02"), Completeness: manifest.Completeness, Revision: manifest.Revision}
}

func toStatementResponse(statement models.ExpenseStatement) statementResponse {
	return statementResponse{ID: statement.ID.String(), AccountKey: statement.AccountKey, SourceKind: statement.SourceKind, StatementRef: statement.StatementRef, PeriodFrom: statement.PeriodFrom.Format("2006-01-02"), PeriodTo: statement.PeriodTo.Format("2006-01-02"), Completeness: statement.Completeness, Revision: statement.Revision, SHA256: statement.SHA256, RowCount: statement.RowCount, CreatedAt: statement.CreatedAt, UpdatedAt: statement.UpdatedAt}
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
	refundOf := ""
	if row.RefundOfExpenseID != nil {
		refundOf = ids.Encode(ids.ExpensePrefix, *row.RefundOfExpenseID)
	}
	return expenseResponse{
		ID:         ids.Encode(ids.ExpensePrefix, row.ID),
		OccurredOn: row.OccurredOn.Format("2006-01-02"),
		AccountKey: row.AccountKey, Kind: row.Kind, RefundOf: refundOf, OriginalTransactionRef: row.OriginalTransactionRef,
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
