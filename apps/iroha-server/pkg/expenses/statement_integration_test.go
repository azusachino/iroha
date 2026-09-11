//go:build integration

package expenses

import (
	"reflect"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/gorm"
)

func TestImportStatementReplayCorrectionAndScopedCompleteness(t *testing.T) {
	db := openIntegrationDB(t)
	clearStatementData(t, db)
	t.Cleanup(func() { clearStatementData(t, db) })
	svc := NewService(db)
	manifest := StatementManifest{AccountKey: "card-main", SourceKind: "bank_csv", StatementRef: "aug-2026", PeriodFrom: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), PeriodTo: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Completeness: StatementCompletenessComplete, Revision: 1}
	firstCSV := []byte("transaction_id,occurred_on,currency,amount_minor,kind,category,merchant,note,original_transaction_ref\ntx-1,2026-08-02,JPY,1200,expense,food,Cafe,,\ntx-2,2026-08-03,JPY,800,expense,transport,Train,,\n")

	if preview := svc.PreviewStatement(manifest, firstCSV); !preview.Valid || preview.RowCount != 2 {
		t.Fatalf("first preview = %+v", preview)
	}
	first, err := svc.ImportStatement(manifest, firstCSV)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if !first.Created || first.Changed != 2 || first.Tombstoned != 0 {
		t.Fatalf("first import = %+v", first)
	}
	augustBefore, err := svc.PeriodReport(PeriodFilters{From: manifest.PeriodFrom, To: manifest.PeriodTo})
	if err != nil {
		t.Fatalf("august report before September import: %v", err)
	}
	augustValuesBefore, err := svc.PeriodExpenses(PeriodFilters{From: manifest.PeriodFrom, To: manifest.PeriodTo})
	if err != nil {
		t.Fatalf("august series before September import: %v", err)
	}
	september := StatementManifest{AccountKey: manifest.AccountKey, SourceKind: manifest.SourceKind, StatementRef: "sep-2026", PeriodFrom: manifest.PeriodTo, PeriodTo: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Completeness: StatementCompletenessComplete, Revision: 1}
	if _, err := svc.ImportStatement(september, []byte("transaction_id,occurred_on,currency,amount_minor,kind,category,merchant\nsep-1,2026-09-02,JPY,500,expense,food,Cafe\n")); err != nil {
		t.Fatalf("September import: %v", err)
	}
	augustAfter, err := svc.PeriodReport(PeriodFilters{From: manifest.PeriodFrom, To: manifest.PeriodTo})
	if err != nil {
		t.Fatalf("august report after September import: %v", err)
	}
	augustValuesAfter, err := svc.PeriodExpenses(PeriodFilters{From: manifest.PeriodFrom, To: manifest.PeriodTo})
	if err != nil {
		t.Fatalf("august series after September import: %v", err)
	}
	if !reflect.DeepEqual(augustBefore, augustAfter) || !reflect.DeepEqual(augustValuesBefore, augustValuesAfter) {
		t.Fatalf("September import changed August reads: before report=%+v after report=%+v before series=%+v after series=%+v", augustBefore, augustAfter, augustValuesBefore, augustValuesAfter)
	}
	initialRevision := readExpenseRevision(t, db)

	replay, err := svc.ImportStatement(manifest, firstCSV)
	if err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	if !replay.Idempotent || readExpenseRevision(t, db) != initialRevision {
		t.Fatalf("replay = %+v, revision = %d; want idempotent without canonical revision advance", replay, readExpenseRevision(t, db))
	}

	partial := manifest
	partial.Revision = 2
	partial.Completeness = StatementCompletenessPartial
	partialCSV := []byte("transaction_id,occurred_on,currency,amount_minor,kind,category,merchant\ntx-1,2026-08-02,JPY,1500,expense,food,Cafe updated\n")
	partialResult, err := svc.ImportStatement(partial, partialCSV)
	if err != nil {
		t.Fatalf("partial correction: %v", err)
	}
	if partialResult.Tombstoned != 0 || partialResult.Changed != 1 {
		t.Fatalf("partial correction = %+v, want no tombstones and one correction", partialResult)
	}
	var tx2 models.Expense
	if err := db.Where("source_kind = ? and source_ref = ?", manifest.SourceKind, "tx-2").First(&tx2).Error; err != nil {
		t.Fatalf("load tx-2 after partial statement: %v", err)
	}
	if tx2.DeletedAt != nil {
		t.Fatal("partial statement tombstoned an omitted row")
	}

	complete := partial
	complete.Revision = 3
	complete.Completeness = StatementCompletenessComplete
	completeResult, err := svc.ImportStatement(complete, partialCSV)
	if err != nil {
		t.Fatalf("complete correction: %v", err)
	}
	if completeResult.Tombstoned != 1 {
		t.Fatalf("complete correction = %+v, want one tombstone", completeResult)
	}
	if err := db.First(&tx2, "id = ?", tx2.ID).Error; err != nil {
		t.Fatalf("reload tx-2: %v", err)
	}
	if tx2.DeletedAt == nil {
		t.Fatal("complete statement did not tombstone omitted row")
	}

	otherAccount := complete
	otherAccount.AccountKey = "cash"
	otherAccount.StatementRef = "aug-2026-cash"
	otherAccount.Revision = 1
	if _, err := svc.ImportStatement(otherAccount, []byte("transaction_id,occurred_on,currency,amount_minor,kind,category,merchant\ntx-2,2026-08-03,JPY,800,expense,transport,Train\n")); err != nil {
		t.Fatalf("same transaction ID in another account: %v", err)
	}
	if _, err := svc.ImportStatement(manifest, []byte("transaction_id,occurred_on,currency,amount_minor,kind,category,merchant\ntx-bad,2026-08-03,JPY,800,transfer,transport,Move\n")); err == nil {
		t.Fatal("invalid transfer import unexpectedly reached persistence")
	}
}

func clearStatementData(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("delete from tb_expense_statement_rows").Error; err != nil {
		t.Fatalf("clear statement rows: %v", err)
	}
	if err := db.Exec("delete from tb_expense_statements").Error; err != nil {
		t.Fatalf("clear statements: %v", err)
	}
	clearExpenses(t, db)
}

func readExpenseRevision(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var revision int64
	if err := db.Raw("select coalesce((select revision from tb_read_revisions where namespace = 'read_expenses'), 0)").Scan(&revision).Error; err != nil {
		t.Fatalf("read expense revision: %v", err)
	}
	return revision
}
