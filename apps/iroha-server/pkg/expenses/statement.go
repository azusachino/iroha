package expenses

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/revisions"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatementCompletenessPartial  = "partial"
	StatementCompletenessComplete = "complete"
	statementDateFormat           = "2006-01-02"
)

var (
	ErrInvalidStatement   = errors.New("invalid expense statement")
	ErrStatementConflict  = errors.New("expense statement revision conflict")
	ErrStatementOutOfDate = errors.New("expense statement revision is older than the latest revision")
)

type StatementManifest struct {
	AccountKey   string    `json:"account_key"`
	SourceKind   string    `json:"source_kind"`
	StatementRef string    `json:"statement_ref"`
	PeriodFrom   time.Time `json:"period_from"`
	PeriodTo     time.Time `json:"period_to"`
	Completeness string    `json:"completeness"`
	Revision     int64     `json:"revision"`
}

type StatementRow struct {
	TransactionID          string
	OccurredOn             time.Time
	Currency               string
	AmountMinor            int64
	Kind                   string
	Category               string
	Merchant               string
	Note                   string
	OriginalTransactionRef string
}

type StatementValidationError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type StatementPreview struct {
	Manifest StatementManifest          `json:"manifest"`
	SHA256   string                     `json:"sha256"`
	RowCount int                        `json:"row_count"`
	Valid    bool                       `json:"valid"`
	Errors   []StatementValidationError `json:"errors"`
}

type StatementImportResult struct {
	Statement  models.ExpenseStatement `json:"statement"`
	Created    bool                    `json:"created"`
	Idempotent bool                    `json:"idempotent"`
	Changed    int                     `json:"changed"`
	Tombstoned int                     `json:"tombstoned"`
}

var requiredStatementColumns = []string{"transaction_id", "occurred_on", "currency", "amount_minor", "kind", "category", "merchant"}

// PreviewStatement validates the complete CSV without writing anything.
func (s *Service) PreviewStatement(manifest StatementManifest, data []byte) StatementPreview {
	return parseStatementCSV(manifest, data)
}

func parseStatementCSV(manifest StatementManifest, data []byte) StatementPreview {
	preview := StatementPreview{Manifest: manifest, Errors: []StatementValidationError{}}
	sum := sha256.Sum256(data)
	preview.SHA256 = hex.EncodeToString(sum[:])
	if err := validateStatementManifest(&manifest); err != nil {
		preview.Errors = append(preview.Errors, StatementValidationError{Row: 0, Field: "manifest", Message: err.Error()})
		preview.Manifest = manifest
		return preview
	}
	preview.Manifest = manifest

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		preview.Errors = append(preview.Errors, StatementValidationError{Row: 1, Field: "header", Message: "missing CSV header"})
		return preview
	}
	if err != nil {
		preview.Errors = append(preview.Errors, StatementValidationError{Row: 1, Field: "header", Message: err.Error()})
		return preview
	}
	columns, headerErrors := statementColumns(header)
	preview.Errors = append(preview.Errors, headerErrors...)
	if len(headerErrors) > 0 {
		return preview
	}

	rowNumber := 1
	seen := map[string]struct{}{}
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		rowNumber++
		if readErr != nil {
			preview.Errors = append(preview.Errors, StatementValidationError{Row: rowNumber, Field: "csv", Message: readErr.Error()})
			continue
		}
		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			continue
		}
		row, rowErrors := parseStatementRow(columns, record, manifest, rowNumber)
		preview.Errors = append(preview.Errors, rowErrors...)
		if len(rowErrors) > 0 {
			continue
		}
		if _, exists := seen[row.TransactionID]; exists {
			preview.Errors = append(preview.Errors, StatementValidationError{Row: rowNumber, Field: "transaction_id", Message: "duplicate transaction ID"})
			continue
		}
		seen[row.TransactionID] = struct{}{}
		preview.RowCount++
	}
	preview.Valid = len(preview.Errors) == 0
	return preview
}

func validateStatementManifest(manifest *StatementManifest) error {
	manifest.AccountKey = normalizeAccountKey(manifest.AccountKey)
	manifest.SourceKind = strings.ToLower(strings.TrimSpace(manifest.SourceKind))
	manifest.StatementRef = strings.TrimSpace(manifest.StatementRef)
	manifest.PeriodFrom = dateOnly(manifest.PeriodFrom)
	manifest.PeriodTo = dateOnly(manifest.PeriodTo)
	manifest.Completeness = strings.ToLower(strings.TrimSpace(manifest.Completeness))
	if manifest.AccountKey == "" || len(manifest.AccountKey) > maxAccountKeyLen || strings.ContainsAny(manifest.AccountKey, "/\\") {
		return fmt.Errorf("%w: account_key", ErrInvalidStatement)
	}
	if manifest.SourceKind == "" || len(manifest.SourceKind) > maxSourceKindLen {
		return fmt.Errorf("%w: source_kind", ErrInvalidStatement)
	}
	if manifest.StatementRef == "" || len(manifest.StatementRef) > maxSourceRefLen || strings.ContainsAny(manifest.StatementRef, "/\\") {
		return fmt.Errorf("%w: statement_ref", ErrInvalidStatement)
	}
	if !manifest.PeriodFrom.Before(manifest.PeriodTo) {
		return fmt.Errorf("%w: period", ErrInvalidStatement)
	}
	if manifest.Completeness != StatementCompletenessPartial && manifest.Completeness != StatementCompletenessComplete {
		return fmt.Errorf("%w: completeness", ErrInvalidStatement)
	}
	if manifest.Revision <= 0 {
		return fmt.Errorf("%w: revision", ErrInvalidStatement)
	}
	return nil
}

func statementColumns(header []string) (map[string]int, []StatementValidationError) {
	columns := make(map[string]int, len(header))
	errorsFound := []StatementValidationError{}
	known := map[string]bool{}
	for _, name := range append(requiredStatementColumns, "note", "original_transaction_ref") {
		known[name] = true
	}
	for index, value := range header {
		name := strings.TrimSpace(value)
		if name == "" || !known[name] {
			errorsFound = append(errorsFound, StatementValidationError{Row: 1, Field: "header", Message: "unknown or empty column " + strconv.Quote(name)})
			continue
		}
		if _, exists := columns[name]; exists {
			errorsFound = append(errorsFound, StatementValidationError{Row: 1, Field: "header", Message: "duplicate column " + name})
			continue
		}
		columns[name] = index
	}
	for _, required := range requiredStatementColumns {
		if _, exists := columns[required]; !exists {
			errorsFound = append(errorsFound, StatementValidationError{Row: 1, Field: "header", Message: "missing column " + required})
		}
	}
	return columns, errorsFound
}

func parseStatementRow(columns map[string]int, record []string, manifest StatementManifest, rowNumber int) (StatementRow, []StatementValidationError) {
	value := func(name string) string {
		index, exists := columns[name]
		if !exists || index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}
	row := StatementRow{TransactionID: value("transaction_id"), Currency: strings.ToUpper(value("currency")), Kind: strings.ToLower(value("kind")), Category: strings.ToLower(value("category")), Merchant: value("merchant"), Note: value("note"), OriginalTransactionRef: value("original_transaction_ref")}
	errorsFound := []StatementValidationError{}
	if row.TransactionID == "" || len(row.TransactionID) > maxSourceRefLen || strings.ContainsAny(row.TransactionID, "/\\") {
		errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "transaction_id", Message: "must be a non-empty stable identifier"})
	}
	parsedDate, err := time.Parse(statementDateFormat, value("occurred_on"))
	if err != nil {
		errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "occurred_on", Message: "must use YYYY-MM-DD"})
	} else {
		row.OccurredOn = parsedDate
		if parsedDate.Before(manifest.PeriodFrom) || !parsedDate.Before(manifest.PeriodTo) {
			errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "occurred_on", Message: "must be inside the manifest period"})
		}
	}
	amount, err := strconv.ParseInt(value("amount_minor"), 10, 64)
	if err != nil || amount <= 0 {
		errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "amount_minor", Message: "must be a positive integer"})
	} else {
		row.AmountMinor = amount
	}
	if _, exists := SupportedCurrencies[row.Currency]; !exists {
		errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "currency", Message: "unsupported currency"})
	}
	if row.Kind == "transfer" {
		errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "kind", Message: "transfers are not spending rows"})
	} else if row.Kind != KindExpense && row.Kind != KindRefund {
		errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "kind", Message: "must be expense or refund"})
	}
	if _, exists := SupportedCategories[row.Category]; !exists {
		errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "category", Message: "unsupported category"})
	}
	if utf8RuneCount(row.Merchant) > maxMerchantLen || utf8RuneCount(row.Note) > maxNoteLen {
		errorsFound = append(errorsFound, StatementValidationError{Row: rowNumber, Field: "merchant/note", Message: "text exceeds the supported length"})
	}
	return row, errorsFound
}

func utf8RuneCount(value string) int {
	return len([]rune(value))
}

func statementRowFingerprint(row StatementRow) string {
	payload, _ := json.Marshal(row)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// ImportStatement applies one validated statement atomically. A partial
// statement never deletes rows; only a complete revision can create
// tombstones, and only inside its declared account/source/period scope.
func (s *Service) ImportStatement(manifest StatementManifest, data []byte) (StatementImportResult, error) {
	preview := parseStatementCSV(manifest, data)
	if !preview.Valid {
		return StatementImportResult{}, fmt.Errorf("%w: %d validation error(s)", ErrInvalidStatement, len(preview.Errors))
	}
	manifest = preview.Manifest
	var result StatementImportResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var latest models.ExpenseStatement
		latestErr := tx.Where("account_key = ? and source_kind = ? and statement_ref = ?", manifest.AccountKey, manifest.SourceKind, manifest.StatementRef).Order("revision desc").First(&latest).Error
		if latestErr == nil {
			switch {
			case manifest.Revision < latest.Revision:
				return ErrStatementOutOfDate
			case manifest.Revision == latest.Revision && latest.SHA256 == preview.SHA256:
				result = StatementImportResult{Statement: latest, Idempotent: true}
				return nil
			case manifest.Revision == latest.Revision:
				return ErrStatementConflict
			}
		} else if !errors.Is(latestErr, gorm.ErrRecordNotFound) {
			return latestErr
		}

		now := time.Now().UTC()
		statement := models.ExpenseStatement{ID: uuid.New(), AccountKey: manifest.AccountKey, SourceKind: manifest.SourceKind, StatementRef: manifest.StatementRef, PeriodFrom: manifest.PeriodFrom, PeriodTo: manifest.PeriodTo, Completeness: manifest.Completeness, Revision: manifest.Revision, SHA256: preview.SHA256, RowCount: preview.RowCount, CreatedAt: now, UpdatedAt: now}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&statement).Error; err != nil {
			return err
		}
		changed := 0
		incoming := make(map[string]struct{}, len(preview.Errors))
		idsByTransaction := make(map[string]uuid.UUID, len(preview.Errors))
		rows := parseValidStatementRows(manifest, data)
		for _, row := range rows {
			incoming[row.TransactionID] = struct{}{}
			expenseID, didChange, err := s.applyStatementRow(tx, statement.ID, manifest, row, now)
			if err != nil {
				return err
			}
			if didChange {
				changed++
			}
			idsByTransaction[row.TransactionID] = expenseID
		}
		for _, row := range rows {
			if row.Kind != KindRefund || row.OriginalTransactionRef == "" {
				continue
			}
			targetID := idsByTransaction[row.OriginalTransactionRef]
			if targetID == uuid.Nil {
				var target models.Expense
				if err := tx.Where("account_key = ? and source_kind = ? and source_ref = ?", manifest.AccountKey, manifest.SourceKind, row.OriginalTransactionRef).First(&target).Error; err == nil {
					targetID = target.ID
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			if targetID != uuid.Nil {
				resultChanged, err := s.linkStatementRefund(tx, manifest, row, targetID, now)
				if err != nil {
					return err
				}
				if resultChanged {
					changed++
				}
			}
		}

		tombstoned, err := s.tombstoneMissingStatementRows(tx, statement.ID, manifest, incoming, now)
		if err != nil {
			return err
		}
		result = StatementImportResult{Statement: statement, Created: true, Changed: changed, Tombstoned: tombstoned}
		namespaces := []string{revisions.NamespaceCoverage}
		if changed > 0 || tombstoned > 0 {
			namespaces = append(namespaces, revisions.NamespaceExpenses, revisions.NamespaceMetrics, revisions.NamespaceReports)
		}
		return revisions.Bump(tx, namespaces...)
	})
	return result, err
}

func parseValidStatementRows(manifest StatementManifest, data []byte) []StatementRow {
	preview := parseStatementCSV(manifest, data)
	if !preview.Valid {
		return nil
	}
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	header, _ := reader.Read()
	columns, _ := statementColumns(header)
	rows := []StatementRow{}
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			continue
		}
		row, rowErrors := parseStatementRow(columns, record, manifest, 0)
		if len(rowErrors) == 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func (s *Service) applyStatementRow(tx *gorm.DB, statementID uuid.UUID, manifest StatementManifest, row StatementRow, now time.Time) (uuid.UUID, bool, error) {
	var previous models.ExpenseStatementRow
	previousErr := tx.Table("tb_expense_statement_rows as row").Joins("join tb_expense_statements as statement on statement.id = row.statement_id").Where("row.account_key = ? and row.source_kind = ? and row.transaction_id = ?", manifest.AccountKey, manifest.SourceKind, row.TransactionID).Order("statement.revision desc, statement.created_at desc").First(&previous).Error
	if previousErr != nil && !errors.Is(previousErr, gorm.ErrRecordNotFound) {
		return uuid.Nil, false, previousErr
	}
	if previousErr == nil && previous.RowFingerprint == statementRowFingerprint(row) && previous.TombstonedAt == nil {
		current := models.ExpenseStatementRow{ID: uuid.New(), StatementID: statementID, AccountKey: manifest.AccountKey, SourceKind: manifest.SourceKind, TransactionID: row.TransactionID, ExpenseID: previous.ExpenseID, RowFingerprint: previous.RowFingerprint, OccurredOn: row.OccurredOn, CreatedAt: now, UpdatedAt: now}
		return previous.ExpenseID, false, tx.Create(&current).Error
	}

	input := statementCreateInput(manifest, row, nil)
	normalized, fingerprint, err := NormalizeCreate(input)
	if err != nil {
		return uuid.Nil, false, err
	}
	var expense models.Expense
	if previousErr == nil {
		if err := tx.First(&expense, "id = ?", previous.ExpenseID).Error; err != nil {
			return uuid.Nil, false, err
		}
	} else {
		existingErr := tx.Where("account_key = ? and source_kind = ? and source_ref = ?", normalized.AccountKey, normalized.Source.Kind, normalized.Source.Ref).First(&expense).Error
		if existingErr == nil {
			return uuid.Nil, false, fmt.Errorf("%w: transaction %s already exists outside statement lineage", ErrSourceConflict, row.TransactionID)
		}
		if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			return uuid.Nil, false, existingErr
		}
		expenseID, err := ids.New()
		if err != nil {
			return uuid.Nil, false, err
		}
		expense = models.Expense{ID: expenseID, CreatedAt: now}
	}
	itemsJSON, err := json.Marshal([]Item{})
	if err != nil {
		return uuid.Nil, false, err
	}
	expense.OccurredOn, expense.AccountKey, expense.Kind, expense.Currency = normalized.OccurredOn, normalized.AccountKey, normalized.Kind, normalized.Currency
	expense.AmountMinor, expense.Category, expense.Merchant, expense.Note = normalized.AmountMinor, normalized.Category, normalized.Merchant, normalized.Note
	expense.ItemsJSON, expense.SourceKind, expense.SourceRef, expense.CreateFingerprint = itemsJSON, normalized.Source.Kind, normalized.Source.Ref, fingerprint
	expense.OriginalTransactionRef, expense.RefundOfExpenseID, expense.UpdatedAt, expense.DeletedAt = normalized.OriginalTransactionRef, normalized.RefundOfExpenseID, now, nil
	if err := tx.Save(&expense).Error; err != nil {
		return uuid.Nil, false, err
	}
	statementRow := models.ExpenseStatementRow{ID: uuid.New(), StatementID: statementID, AccountKey: manifest.AccountKey, SourceKind: manifest.SourceKind, TransactionID: row.TransactionID, ExpenseID: expense.ID, RowFingerprint: statementRowFingerprint(row), OccurredOn: row.OccurredOn, CreatedAt: now, UpdatedAt: now}
	return expense.ID, true, tx.Create(&statementRow).Error
}

func statementCreateInput(manifest StatementManifest, row StatementRow, refundOf *uuid.UUID) CreateInput {
	return CreateInput{OccurredOn: row.OccurredOn, AccountKey: manifest.AccountKey, Kind: row.Kind, RefundOfExpenseID: refundOf, OriginalTransactionRef: row.OriginalTransactionRef, Currency: row.Currency, AmountMinor: row.AmountMinor, Category: row.Category, Merchant: row.Merchant, Note: row.Note, Source: Source{Kind: manifest.SourceKind, Ref: row.TransactionID}}
}

func (s *Service) linkStatementRefund(tx *gorm.DB, manifest StatementManifest, row StatementRow, targetID uuid.UUID, now time.Time) (bool, error) {
	var expense models.Expense
	if err := tx.Where("account_key = ? and source_kind = ? and source_ref = ?", manifest.AccountKey, manifest.SourceKind, row.TransactionID).First(&expense).Error; err != nil {
		return false, err
	}
	if expense.RefundOfExpenseID != nil && *expense.RefundOfExpenseID == targetID {
		return false, nil
	}
	input := statementCreateInput(manifest, row, &targetID)
	_, fingerprint, err := NormalizeCreate(input)
	if err != nil {
		return false, err
	}
	result := tx.Model(&models.Expense{}).Where("id = ?", expense.ID).Updates(map[string]any{"refund_of_expense_id": targetID, "create_fingerprint": fingerprint, "updated_at": now})
	return result.RowsAffected == 1, result.Error
}

func (s *Service) tombstoneMissingStatementRows(tx *gorm.DB, statementID uuid.UUID, manifest StatementManifest, incoming map[string]struct{}, now time.Time) (int, error) {
	if manifest.Completeness != StatementCompletenessComplete {
		return 0, nil
	}
	type priorRow struct {
		ID            uuid.UUID  `gorm:"column:id"`
		TransactionID string     `gorm:"column:transaction_id"`
		ExpenseID     uuid.UUID  `gorm:"column:expense_id"`
		Fingerprint   string     `gorm:"column:row_fingerprint"`
		OccurredOn    time.Time  `gorm:"column:occurred_on"`
		TombstonedAt  *time.Time `gorm:"column:tombstoned_at"`
	}
	var candidates []priorRow
	if err := tx.Table("tb_expense_statement_rows as row").Select("row.id, row.transaction_id, row.expense_id, row.row_fingerprint, row.occurred_on, row.tombstoned_at").Joins("join tb_expense_statements as statement on statement.id = row.statement_id").Where("row.account_key = ? and row.source_kind = ? and row.occurred_on >= ? and row.occurred_on < ?", manifest.AccountKey, manifest.SourceKind, manifest.PeriodFrom, manifest.PeriodTo).Order("row.transaction_id, statement.revision desc, statement.created_at desc").Scan(&candidates).Error; err != nil {
		return 0, err
	}
	seen := map[string]struct{}{}
	tombstoned := 0
	for _, candidate := range candidates {
		if _, exists := seen[candidate.TransactionID]; exists {
			continue
		}
		seen[candidate.TransactionID] = struct{}{}
		if _, exists := incoming[candidate.TransactionID]; exists || candidate.TombstonedAt != nil {
			continue
		}
		result := tx.Model(&models.Expense{}).Where("id = ? and deleted_at is null", candidate.ExpenseID).Updates(map[string]any{"deleted_at": now, "updated_at": now})
		if result.Error != nil {
			return tombstoned, result.Error
		}
		if result.RowsAffected == 1 {
			tombstoned++
		}
		tombstone := models.ExpenseStatementRow{ID: uuid.New(), StatementID: statementID, AccountKey: manifest.AccountKey, SourceKind: manifest.SourceKind, TransactionID: candidate.TransactionID, ExpenseID: candidate.ExpenseID, RowFingerprint: candidate.Fingerprint, OccurredOn: candidate.OccurredOn, TombstonedAt: &now, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&tombstone).Error; err != nil {
			return tombstoned, err
		}
	}
	return tombstoned, nil
}
