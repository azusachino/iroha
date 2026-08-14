//go:build integration

package expenses

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestExpenseMigrationConstraints(t *testing.T) {
	db := openIntegrationDB(t)
	clearExpenses(t, db)
	t.Cleanup(func() { clearExpenses(t, db) })

	createdAt := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	valid := models.Expense{
		ID:                uuid.Must(uuid.NewV7()),
		OccurredOn:        time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC),
		Currency:          "JPY",
		AmountMinor:       1300,
		Category:          "food",
		ItemsJSON:         json.RawMessage(`[{"name":"ramen","amount_minor":1300}]`),
		SourceKind:        "local_agent",
		SourceRef:         "receipt-1",
		CreateFingerprint: "fingerprint-1",
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
	}
	if err := db.Create(&valid).Error; err != nil {
		t.Fatalf("create valid expense: %v", err)
	}

	assertRejected := func(name string, mutate func(*models.Expense)) {
		t.Helper()
		candidate := valid
		candidate.ID = uuid.Must(uuid.NewV7())
		candidate.SourceRef = "receipt-" + name
		candidate.CreateFingerprint = "fingerprint-" + name
		mutate(&candidate)
		if err := db.Create(&candidate).Error; err == nil {
			t.Fatalf("%s: invalid expense was accepted", name)
		}
	}

	assertRejected("zero-amount", func(expense *models.Expense) { expense.AmountMinor = 0 })
	assertRejected("currency", func(expense *models.Expense) { expense.Currency = "CNY" })
	assertRejected("category", func(expense *models.Expense) { expense.Category = "travel" })
	assertRejected("object-items", func(expense *models.Expense) { expense.ItemsJSON = json.RawMessage(`{}`) })
	assertRejected("timestamp", func(expense *models.Expense) { expense.UpdatedAt = createdAt.Add(-time.Second) })
	assertRejected("deleted-timestamp", func(expense *models.Expense) { deleted := createdAt.Add(-time.Second); expense.DeletedAt = &deleted })

	var tooManyItems []map[string]any
	for i := 0; i < MaxItems+1; i++ {
		tooManyItems = append(tooManyItems, map[string]any{"name": "item"})
	}
	itemsJSON, err := json.Marshal(tooManyItems)
	if err != nil {
		t.Fatalf("marshal too many items: %v", err)
	}
	assertRejected("item-limit", func(expense *models.Expense) { expense.ItemsJSON = itemsJSON })

	duplicate := valid
	duplicate.ID = uuid.Must(uuid.NewV7())
	duplicate.CreateFingerprint = "fingerprint-duplicate"
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate source identity was accepted")
	}
}

func openIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func clearExpenses(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("delete from tb_expenses").Error; err != nil {
		t.Fatalf("clear expenses: %v", err)
	}
}
