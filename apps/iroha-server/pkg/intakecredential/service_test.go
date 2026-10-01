package intakecredential

import (
	"context"
	"strings"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestIntakeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.IntakeCredential{}); err != nil {
		t.Fatalf("migrate models: %v", err)
	}
	return db
}

func TestCredentialNameValidation(t *testing.T) {
	valid := []string{"primary", "iphone-15", "watch-ultra", "dev-1"}
	for _, name := range valid {
		if !validName.MatchString(name) {
			t.Errorf("expected validName to accept %q", name)
		}
	}

	invalid := []string{"Primary", "iphone_15", "-leading", "space here", "", "with.dot"}
	for _, name := range invalid {
		if validName.MatchString(name) {
			t.Errorf("expected validName to reject %q", name)
		}
	}
}

func TestIntakeCredential_Lifecycle(t *testing.T) {
	db := newTestIntakeDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// 1. Verify before any credential is provisioned returns ErrNotProvisioned
	if _, err := svc.Verify(ctx, "iroha_hae_testtoken123"); err != ErrNotProvisioned {
		t.Fatalf("Verify when empty = %v, want ErrNotProvisioned", err)
	}

	// 2. Reject invalid device names
	if _, _, err := svc.Issue(ctx, "Invalid Device"); err == nil {
		t.Fatal("expected error issuing credential with invalid name")
	}

	// 3. Issue valid credential
	row1, token1, err := svc.Issue(ctx, "primary-iphone")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if !strings.HasPrefix(token1, TokenPrefix) {
		t.Fatalf("token %q does not have prefix %q", token1, TokenPrefix)
	}
	if row1.Name != "primary-iphone" || row1.RevokedAt != nil {
		t.Fatalf("unexpected row: %+v", row1)
	}

	// 4. Verify with correct token succeeds
	verified, err := svc.Verify(ctx, token1)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if verified.ID != row1.ID || verified.Name != "primary-iphone" {
		t.Fatalf("unexpected verified credential: %+v", verified)
	}

	// 5. Verify with invalid token returns ErrInvalid
	if _, err := svc.Verify(ctx, token1+"corrupt"); err != ErrInvalid {
		t.Fatalf("Verify(corrupt) = %v, want ErrInvalid", err)
	}
	if _, err := svc.Verify(ctx, "random-token"); err != ErrInvalid {
		t.Fatalf("Verify(random) = %v, want ErrInvalid", err)
	}

	// 6. Issue a second credential
	row2, token2, err := svc.Issue(ctx, "watch-ultra")
	if err != nil {
		t.Fatalf("Issue second failed: %v", err)
	}

	// 7. List returns both credentials
	list, err := svc.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %d items (err=%v), want 2", len(list), err)
	}
	if list[0].ID != row2.ID || list[1].ID != row1.ID {
		t.Fatalf("unexpected order: newest should be first: %+v", list)
	}

	// 8. Revoke first credential
	if err := svc.Revoke(ctx, row1.ID); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	// 9. Revoked token can no longer verify
	if _, err := svc.Verify(ctx, token1); err != ErrInvalid {
		t.Fatalf("Verify(revoked) = %v, want ErrInvalid", err)
	}

	// 10. Second token is still active
	if _, err := svc.Verify(ctx, token2); err != nil {
		t.Fatalf("Verify second token after first revoked failed: %v", err)
	}

	// 11. Revoking non-existent or already revoked returns ErrNotFound
	if err := svc.Revoke(ctx, row1.ID); err != ErrNotFound {
		t.Fatalf("Revoke already revoked = %v, want ErrNotFound", err)
	}
	if err := svc.Revoke(ctx, uuid.New()); err != ErrNotFound {
		t.Fatalf("Revoke unknown = %v, want ErrNotFound", err)
	}
}
