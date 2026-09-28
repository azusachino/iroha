//go:build integration

package intakecredential

import (
	"context"
	"errors"
	"os"
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
	if err := db.Exec("delete from tb_health_intake_credential").Error; err != nil {
		t.Fatalf("clear credential: %v", err)
	}
	return db
}

func TestCredentialLifecycle(t *testing.T) {
	ctx := context.Background()
	service := NewService(openIntegrationDB(t))

	if err := service.Verify(ctx, "anything"); !errors.Is(err, ErrNotProvisioned) {
		t.Fatalf("unprovisioned verify = %v, want ErrNotProvisioned", err)
	}

	first, err := service.Rotate(ctx)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if len(first) < 43 {
		t.Fatalf("token %q shorter than 32 bytes of entropy", first)
	}
	if err := service.Verify(ctx, first); err != nil {
		t.Fatalf("verify issued token: %v", err)
	}
	for _, bad := range []string{"", "wrong", first + "x"} {
		if err := service.Verify(ctx, bad); !errors.Is(err, ErrInvalid) {
			t.Fatalf("verify %q = %v, want ErrInvalid", bad, err)
		}
	}

	second, err := service.Rotate(ctx)
	if err != nil {
		t.Fatalf("second rotate: %v", err)
	}
	if err := service.Verify(ctx, first); !errors.Is(err, ErrInvalid) {
		t.Fatalf("rotated-out token = %v, want ErrInvalid", err)
	}
	if err := service.Verify(ctx, second); err != nil {
		t.Fatalf("verify rotated token: %v", err)
	}

	var stored []struct{ TokenSHA256 string }
	if err := service.db.Raw("select token_sha256 from tb_health_intake_credential").Scan(&stored).Error; err != nil {
		t.Fatalf("read verifier: %v", err)
	}
	if len(stored) != 1 || stored[0].TokenSHA256 == second {
		t.Fatalf("stored = %+v, want exactly one hashed verifier", stored)
	}
}
