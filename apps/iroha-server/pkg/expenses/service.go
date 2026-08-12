package expenses

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 100
	maxMerchantLen   = 200
	maxNoteLen       = 2000
	maxItemNameLen   = 200
	maxSourceKindLen = 50
	maxSourceRefLen  = 200
)

var (
	ErrInvalidExpense = errors.New("invalid expense")
	ErrNotFound       = gorm.ErrRecordNotFound
	ErrSourceConflict = errors.New("expense source identity conflict")
	ErrDeleted        = errors.New("expense is deleted")
)

// Item is a descriptive receipt item. The top-level amount remains
// authoritative and item amounts do not need to sum to it.
type Item struct {
	Name        string `json:"name"`
	AmountMinor *int64 `json:"amount_minor,omitempty"`
}

// Source identifies the client-side create identity used for safe retries.
type Source struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// CreateInput is the canonical expense create payload before normalization.
type CreateInput struct {
	OccurredOn  time.Time
	Currency    string
	AmountMinor int64
	Category    string
	Merchant    string
	Note        string
	Items       []Item
	Source      Source
}

// ReplaceInput contains every mutable expense field. Source identity and the
// original create fingerprint are deliberately absent.
type ReplaceInput struct {
	OccurredOn  time.Time
	Currency    string
	AmountMinor int64
	Category    string
	Merchant    string
	Note        string
	Items       []Item
}

// CreateResult distinguishes a newly inserted row from an idempotent retry.
type CreateResult struct {
	Expense models.Expense
	Created bool
}

// Page is one deterministic keyset page of active expenses.
type Page struct {
	Items      []models.Expense
	NextCursor *Cursor
	HasMore    bool
}

// ListFilters selects active expenses. From is inclusive and To is exclusive.
type ListFilters struct {
	From     *time.Time
	To       *time.Time
	Currency string
	Category string
	Limit    int
	Cursor   *Cursor
}

// Service owns canonical expense persistence and source-identity semantics.
type Service struct {
	db *gorm.DB
}

type PeriodFilters struct {
	From time.Time
	To   time.Time
}

type PeriodCurrencyTotal struct {
	Currency         string
	CurrencyExponent int
	AmountMinor      int64
	ExpenseCount     int
}

type PeriodCategoryTotal struct {
	Category         string
	Currency         string
	CurrencyExponent int
	AmountMinor      int64
	ExpenseCount     int
}

type PeriodReport struct {
	ExpenseCount     int
	TotalsByCurrency []PeriodCurrencyTotal
	ByCategory       []PeriodCategoryTotal
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) PeriodReport(filters PeriodFilters) (PeriodReport, error) {
	if !filters.From.Before(filters.To) {
		return PeriodReport{}, errors.New("period from must be before to")
	}

	var expenseCountValue int64
	if err := s.db.Model(&models.Expense{}).
		Where("deleted_at is null and occurred_on >= ? and occurred_on < ?", dateOnly(filters.From), dateOnly(filters.To)).
		Count(&expenseCountValue).Error; err != nil {
		return PeriodReport{}, err
	}

	type currencyRow struct {
		Currency     string `gorm:"column:currency"`
		AmountMinor  int64  `gorm:"column:amount_minor"`
		ExpenseCount int    `gorm:"column:expense_count"`
	}
	var currencyRows []currencyRow
	if err := s.db.Model(&models.Expense{}).
		Select("currency, sum(amount_minor)::bigint AS amount_minor, count(*)::int AS expense_count").
		Where("deleted_at is null and occurred_on >= ? and occurred_on < ?", dateOnly(filters.From), dateOnly(filters.To)).
		Group("currency").Order("currency").Scan(&currencyRows).Error; err != nil {
		return PeriodReport{}, err
	}

	type categoryRow struct {
		Category     string `gorm:"column:category"`
		Currency     string `gorm:"column:currency"`
		AmountMinor  int64  `gorm:"column:amount_minor"`
		ExpenseCount int    `gorm:"column:expense_count"`
	}
	var categoryRows []categoryRow
	if err := s.db.Model(&models.Expense{}).
		Select("category, currency, sum(amount_minor)::bigint AS amount_minor, count(*)::int AS expense_count").
		Where("deleted_at is null and occurred_on >= ? and occurred_on < ?", dateOnly(filters.From), dateOnly(filters.To)).
		Group("category, currency").Order("category, currency").Scan(&categoryRows).Error; err != nil {
		return PeriodReport{}, err
	}

	totalsByCurrency := make([]PeriodCurrencyTotal, 0, len(currencyRows))
	for _, row := range currencyRows {
		totalsByCurrency = append(totalsByCurrency, PeriodCurrencyTotal{
			Currency: row.Currency, CurrencyExponent: SupportedCurrencies[row.Currency],
			AmountMinor: row.AmountMinor, ExpenseCount: row.ExpenseCount,
		})
	}
	byCategory := make([]PeriodCategoryTotal, 0, len(categoryRows))
	for _, row := range categoryRows {
		byCategory = append(byCategory, PeriodCategoryTotal{
			Category: row.Category, Currency: row.Currency, CurrencyExponent: SupportedCurrencies[row.Currency],
			AmountMinor: row.AmountMinor, ExpenseCount: row.ExpenseCount,
		})
	}
	return PeriodReport{ExpenseCount: int(expenseCountValue), TotalsByCurrency: totalsByCurrency, ByCategory: byCategory}, nil
}

// NormalizeCreate validates and canonicalizes a create payload and returns
// the fingerprint used for idempotent source retries.
func NormalizeCreate(input CreateInput) (CreateInput, string, error) {
	normalized := CreateInput{
		OccurredOn:  dateOnly(input.OccurredOn),
		Currency:    strings.ToUpper(strings.TrimSpace(input.Currency)),
		AmountMinor: input.AmountMinor,
		Category:    strings.ToLower(strings.TrimSpace(input.Category)),
		Merchant:    strings.TrimSpace(input.Merchant),
		Note:        strings.TrimSpace(input.Note),
		Source: Source{
			Kind: strings.ToLower(strings.TrimSpace(input.Source.Kind)),
			Ref:  strings.TrimSpace(input.Source.Ref),
		},
	}
	normalized.Items = normalizeItems(input.Items)
	if err := validateCreate(normalized); err != nil {
		return CreateInput{}, "", err
	}
	fingerprint, err := fingerprintCreate(normalized)
	if err != nil {
		return CreateInput{}, "", err
	}
	return normalized, fingerprint, nil
}

// NormalizeReplace validates and canonicalizes mutable fields.
func NormalizeReplace(input ReplaceInput) (ReplaceInput, error) {
	normalized := ReplaceInput{
		OccurredOn:  dateOnly(input.OccurredOn),
		Currency:    strings.ToUpper(strings.TrimSpace(input.Currency)),
		AmountMinor: input.AmountMinor,
		Category:    strings.ToLower(strings.TrimSpace(input.Category)),
		Merchant:    strings.TrimSpace(input.Merchant),
		Note:        strings.TrimSpace(input.Note),
		Items:       normalizeItems(input.Items),
	}
	if err := validateMutable(normalized.OccurredOn, normalized.Currency, normalized.AmountMinor, normalized.Category, normalized.Merchant, normalized.Note, normalized.Items); err != nil {
		return ReplaceInput{}, err
	}
	return normalized, nil
}

func normalizeItems(items []Item) []Item {
	normalized := make([]Item, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, Item{Name: strings.TrimSpace(item.Name), AmountMinor: item.AmountMinor})
	}
	return normalized
}

func validateCreate(input CreateInput) error {
	if err := validateMutable(input.OccurredOn, input.Currency, input.AmountMinor, input.Category, input.Merchant, input.Note, input.Items); err != nil {
		return err
	}
	if input.Source.Kind == "" || len(input.Source.Kind) > maxSourceKindLen {
		return fmt.Errorf("%w: source.kind", ErrInvalidExpense)
	}
	if input.Source.Ref == "" || len(input.Source.Ref) > maxSourceRefLen || strings.ContainsAny(input.Source.Ref, `/\\`) {
		return fmt.Errorf("%w: source.ref", ErrInvalidExpense)
	}
	return nil
}

func validateMutable(occurredOn time.Time, currency string, amountMinor int64, category, merchant, note string, items []Item) error {
	if occurredOn.IsZero() {
		return fmt.Errorf("%w: occurred_on", ErrInvalidExpense)
	}
	if _, ok := SupportedCurrencies[currency]; !ok {
		return fmt.Errorf("%w: currency", ErrInvalidExpense)
	}
	if amountMinor <= 0 {
		return fmt.Errorf("%w: amount_minor", ErrInvalidExpense)
	}
	if _, ok := SupportedCategories[category]; !ok {
		return fmt.Errorf("%w: category", ErrInvalidExpense)
	}
	if utf8.RuneCountInString(merchant) > maxMerchantLen {
		return fmt.Errorf("%w: merchant", ErrInvalidExpense)
	}
	if utf8.RuneCountInString(note) > maxNoteLen {
		return fmt.Errorf("%w: note", ErrInvalidExpense)
	}
	if len(items) > MaxItems {
		return fmt.Errorf("%w: items", ErrInvalidExpense)
	}
	for _, item := range items {
		if item.Name == "" || utf8.RuneCountInString(item.Name) > maxItemNameLen {
			return fmt.Errorf("%w: item.name", ErrInvalidExpense)
		}
		if item.AmountMinor != nil && *item.AmountMinor < 0 {
			return fmt.Errorf("%w: item.amount_minor", ErrInvalidExpense)
		}
	}
	return nil
}

type fingerprintPayload struct {
	OccurredOn  string `json:"occurred_on"`
	Currency    string `json:"currency"`
	AmountMinor int64  `json:"amount_minor"`
	Category    string `json:"category"`
	Merchant    string `json:"merchant"`
	Note        string `json:"note"`
	Items       []Item `json:"items"`
	Source      Source `json:"source"`
}

func fingerprintCreate(input CreateInput) (string, error) {
	payload, err := json.Marshal(fingerprintPayload{
		OccurredOn: input.OccurredOn.Format("2006-01-02"), Currency: input.Currency,
		AmountMinor: input.AmountMinor, Category: input.Category, Merchant: input.Merchant,
		Note: input.Note, Items: input.Items, Source: input.Source,
	})
	if err != nil {
		return "", fmt.Errorf("fingerprint expense: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func dateOnly(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func (s *Service) Create(input CreateInput) (CreateResult, error) {
	normalized, fingerprint, err := NormalizeCreate(input)
	if err != nil {
		return CreateResult{}, err
	}
	id, err := ids.New()
	if err != nil {
		return CreateResult{}, err
	}
	now := time.Now().UTC()
	itemsJSON, err := json.Marshal(normalized.Items)
	if err != nil {
		return CreateResult{}, fmt.Errorf("marshal expense items: %w", err)
	}
	row := models.Expense{
		ID: id, OccurredOn: normalized.OccurredOn, Currency: normalized.Currency,
		AmountMinor: normalized.AmountMinor, Category: normalized.Category,
		Merchant: normalized.Merchant, Note: normalized.Note, ItemsJSON: itemsJSON,
		SourceKind: normalized.Source.Kind, SourceRef: normalized.Source.Ref,
		CreateFingerprint: fingerprint, CreatedAt: now, UpdatedAt: now,
	}
	result := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if result.Error != nil {
		return CreateResult{}, result.Error
	}
	if result.RowsAffected == 1 {
		return CreateResult{Expense: row, Created: true}, nil
	}

	var existing models.Expense
	if err := s.db.Where("source_kind = ? and source_ref = ?", row.SourceKind, row.SourceRef).First(&existing).Error; err != nil {
		return CreateResult{}, err
	}
	if existing.DeletedAt != nil {
		return CreateResult{}, ErrDeleted
	}
	if existing.CreateFingerprint != fingerprint {
		return CreateResult{}, ErrSourceConflict
	}
	return CreateResult{Expense: existing}, nil
}

func (s *Service) List(filters ListFilters) (Page, error) {
	limit := filters.Limit
	if limit <= 0 || limit > maxPageLimit {
		limit = defaultPageLimit
	}
	query := s.db.Model(&models.Expense{}).Where("deleted_at is null")
	if filters.From != nil {
		query = query.Where("occurred_on >= ?", dateOnly(*filters.From))
	}
	if filters.To != nil {
		query = query.Where("occurred_on < ?", dateOnly(*filters.To))
	}
	if filters.Currency != "" {
		query = query.Where("currency = ?", strings.ToUpper(strings.TrimSpace(filters.Currency)))
	}
	if filters.Category != "" {
		query = query.Where("category = ?", strings.ToLower(strings.TrimSpace(filters.Category)))
	}
	if filters.Cursor != nil {
		query = query.Where("(occurred_on, id) < (?, ?)", dateOnly(filters.Cursor.OccurredOn), filters.Cursor.ID)
	}
	var rows []models.Expense
	if err := query.Order("occurred_on desc, id desc").Limit(limit + 1).Find(&rows).Error; err != nil {
		return Page{}, err
	}
	page := Page{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = &Cursor{OccurredOn: last.OccurredOn, ID: last.ID}
	}
	return page, nil
}

func (s *Service) Get(id uuid.UUID) (models.Expense, error) {
	var row models.Expense
	if err := s.db.First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Expense{}, ErrNotFound
		}
		return models.Expense{}, err
	}
	if row.DeletedAt != nil {
		return models.Expense{}, ErrDeleted
	}
	return row, nil
}

func (s *Service) Replace(id uuid.UUID, input ReplaceInput) (models.Expense, error) {
	normalized, err := NormalizeReplace(input)
	if err != nil {
		return models.Expense{}, err
	}
	now := time.Now().UTC()
	itemsJSON, err := json.Marshal(normalized.Items)
	if err != nil {
		return models.Expense{}, fmt.Errorf("marshal expense items: %w", err)
	}
	result := s.db.Model(&models.Expense{}).Where("id = ? and deleted_at is null", id).Updates(map[string]any{
		"occurred_on": normalized.OccurredOn, "currency": normalized.Currency,
		"amount_minor": normalized.AmountMinor, "category": normalized.Category,
		"merchant": normalized.Merchant, "note": normalized.Note,
		"items_json": itemsJSON, "updated_at": now,
	})
	if result.Error != nil {
		return models.Expense{}, result.Error
	}
	if result.RowsAffected == 0 {
		return models.Expense{}, s.lookupMissingOrDeleted(id)
	}
	return s.Get(id)
}

func (s *Service) Delete(id uuid.UUID) error {
	now := time.Now().UTC()
	result := s.db.Model(&models.Expense{}).Where("id = ? and deleted_at is null", id).Updates(map[string]any{
		"deleted_at": now, "updated_at": now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var row models.Expense
	if err := s.db.First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) lookupMissingOrDeleted(id uuid.UUID) error {
	var row models.Expense
	if err := s.db.First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if row.DeletedAt != nil {
		return ErrDeleted
	}
	return ErrNotFound
}
