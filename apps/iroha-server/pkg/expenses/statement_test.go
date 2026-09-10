package expenses

import (
	"strings"
	"testing"
	"time"
)

func TestPreviewStatementRejectsTransfersAndDuplicateRows(t *testing.T) {
	manifest := StatementManifest{AccountKey: "card-main", SourceKind: "bank_csv", StatementRef: "aug-2026", PeriodFrom: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), PeriodTo: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Completeness: StatementCompletenessComplete, Revision: 1}
	data := strings.TrimSpace(`transaction_id,occurred_on,currency,amount_minor,kind,category,merchant,note,original_transaction_ref
tx-1,2026-08-02,JPY,1200,expense,food,Cafe,,
tx-1,2026-08-03,JPY,100,refund,food,Cafe,,tx-1
tx-transfer,2026-08-04,JPY,500,transfer,other,Move,,`) + "\n"

	preview := (&Service{}).PreviewStatement(manifest, []byte(data))
	if preview.Valid || preview.RowCount != 1 {
		t.Fatalf("preview = %+v, want one valid row and validation errors", preview)
	}
	if len(preview.Errors) != 2 {
		t.Fatalf("validation errors = %+v, want duplicate and transfer errors", preview.Errors)
	}
}

func TestPreviewStatementAcceptsOptionalColumnsAndNormalizesManifest(t *testing.T) {
	manifest := StatementManifest{SourceKind: " BANK_CSV ", StatementRef: "aug-2026", PeriodFrom: time.Date(2026, 8, 1, 0, 0, 0, 0, time.FixedZone("JST", 9*60*60)), PeriodTo: time.Date(2026, 9, 1, 0, 0, 0, 0, time.FixedZone("JST", 9*60*60)), Completeness: " COMPLETE ", Revision: 1}
	data := []byte("transaction_id,occurred_on,currency,amount_minor,kind,category,merchant\ntx-1,2026-08-02,jpy,1200,expense,food,Cafe\n")

	preview := (&Service{}).PreviewStatement(manifest, data)
	if !preview.Valid || preview.RowCount != 1 || preview.SHA256 == "" {
		t.Fatalf("preview = %+v, want valid normalized statement", preview)
	}
	if preview.Manifest.AccountKey != "default" || preview.Manifest.SourceKind != "bank_csv" || preview.Manifest.Completeness != StatementCompletenessComplete {
		t.Fatalf("normalized manifest = %+v", preview.Manifest)
	}
}
