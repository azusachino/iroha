//go:build integration

package intakecredential

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
	if err := db.Exec("delete from tb_intake_credentials").Error; err != nil {
		t.Fatalf("clear credentials: %v", err)
	}
	return db
}

func TestCredentialLifecycle(t *testing.T) {
	ctx := context.Background()
	service := NewService(openIntegrationDB(t))

	if _, err := service.Verify(ctx, TokenPrefix+"anything"); !errors.Is(err, ErrNotProvisioned) {
		t.Fatalf("unprovisioned verify = %v, want ErrNotProvisioned", err)
	}
	if _, _, err := service.Issue(ctx, "Bad Name"); err == nil {
		t.Fatal("issue with invalid name succeeded")
	}

	old, first, err := service.Issue(ctx, "primary")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !strings.HasPrefix(first, TokenPrefix) || len(first) < len(TokenPrefix)+43 {
		t.Fatalf("token %q lacks prefix or 32 bytes of entropy", first)
	}
	got, err := service.Verify(ctx, first)
	if err != nil || got.ID != old.ID || got.Name != "primary" || got.LastUsedAt == nil {
		t.Fatalf("verify issued token = %+v, %v", got, err)
	}
	for _, bad := range []string{"", "wrong", first + "x", strings.TrimPrefix(first, TokenPrefix)} {
		if _, err := service.Verify(ctx, bad); !errors.Is(err, ErrInvalid) {
			t.Fatalf("verify %q = %v, want ErrInvalid", bad, err)
		}
	}

	// Rotation overlaps: both tokens work until the old one is revoked.
	_, second, err := service.Issue(ctx, "primary")
	if err != nil {
		t.Fatalf("second issue: %v", err)
	}
	for _, token := range []string{first, second} {
		if _, err := service.Verify(ctx, token); err != nil {
			t.Fatalf("overlapping token rejected: %v", err)
		}
	}
	if err := service.Revoke(ctx, old.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := service.Revoke(ctx, old.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second revoke = %v, want ErrNotFound", err)
	}
	if _, err := service.Verify(ctx, first); !errors.Is(err, ErrInvalid) {
		t.Fatalf("revoked token = %v, want ErrInvalid", err)
	}
	if _, err := service.Verify(ctx, second); err != nil {
		t.Fatalf("verify rotated token: %v", err)
	}

	rows, err := service.List(ctx)
	if err != nil || len(rows) != 2 {
		t.Fatalf("list = %d rows, %v; want 2", len(rows), err)
	}
	for _, row := range rows {
		if row.TokenSHA256 == first || row.TokenSHA256 == second {
			t.Fatal("plaintext token stored")
		}
	}
}
