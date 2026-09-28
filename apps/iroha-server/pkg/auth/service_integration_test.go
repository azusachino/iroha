//go:build integration

package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"

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
	if err := db.Exec("delete from tb_users").Error; err != nil {
		t.Fatalf("clear users: %v", err)
	}
	return db
}

func TestOwnerAuthLifecycle(t *testing.T) {
	ctx := context.Background()
	service := NewService(openIntegrationDB(t))
	const password = "correct horse battery"

	if required, err := service.SetupRequired(ctx); err != nil || !required {
		t.Fatalf("SetupRequired = %v, %v; want true", required, err)
	}
	if _, _, err := service.Login(ctx, "owner", password); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login before setup = %v, want ErrInvalidCredentials", err)
	}
	if _, _, err := service.Setup(ctx, "owner", "short"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("short password = %v, want ErrInvalidInput", err)
	}
	if _, _, err := service.Setup(ctx, "Owner Name", password); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("bad username = %v, want ErrInvalidInput", err)
	}

	setupToken, who, err := service.Setup(ctx, "owner", password)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if who.Username != "owner" || who.CSRFToken == "" {
		t.Fatalf("setup principal = %+v", who)
	}
	if _, _, err := service.Setup(ctx, "other", password); !errors.Is(err, ErrAlreadySetUp) {
		t.Fatalf("second setup = %v, want ErrAlreadySetUp", err)
	}
	var stored []string
	service.db.Raw("select password_hash from tb_user_passwords").Scan(&stored)
	if len(stored) != 1 || !strings.HasPrefix(stored[0], "$argon2id$") || strings.Contains(stored[0], password) {
		t.Fatalf("stored hash = %v", stored)
	}

	if got, err := service.Authenticate(ctx, setupToken); err != nil || got.Username != "owner" {
		t.Fatalf("authenticate setup session = %+v, %v", got, err)
	}
	for _, bad := range [][2]string{{"owner", "wrong password!!"}, {"nobody", password}} {
		if _, _, err := service.Login(ctx, bad[0], bad[1]); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("login %v = %v, want ErrInvalidCredentials", bad, err)
		}
	}
	uid, err := ids.Decode(ids.UserPrefix, who.UserID)
	if err != nil {
		t.Fatalf("decode user id: %v", err)
	}
	if _, err := service.SetDisplayName(ctx, uid, strings.Repeat("x", 65)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("65-char display name = %v, want ErrInvalidInput", err)
	}
	if _, err := service.SetDisplayName(ctx, uid, "  Haru  "); err != nil {
		t.Fatalf("set display name: %v", err)
	}
	if got, _ := service.Authenticate(ctx, setupToken); got.DisplayName != "Haru" {
		t.Fatalf("display name = %q, want Haru", got.DisplayName)
	}
	if _, err := service.SetDisplayName(ctx, uid, ""); err != nil {
		t.Fatalf("clear display name: %v", err)
	}
	if got, _ := service.Authenticate(ctx, setupToken); got.DisplayName != "" {
		t.Fatalf("cleared display name = %q", got.DisplayName)
	}

	loginToken, _, err := service.Login(ctx, "owner", password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if err := service.Logout(ctx, loginToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.Authenticate(ctx, loginToken); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("logged-out session = %v, want ErrUnauthenticated", err)
	}

	service.now = func() time.Time { return time.Now().UTC().Add(SessionTTL + time.Minute) }
	if _, err := service.Authenticate(ctx, setupToken); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expired session = %v, want ErrUnauthenticated", err)
	}
	service.now = func() time.Time { return time.Now().UTC() }

	if _, err := service.ResetPassword(ctx, "a brand new password"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := service.Authenticate(ctx, setupToken); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("session after reset = %v, want ErrUnauthenticated", err)
	}
	if _, _, err := service.Login(ctx, "owner", password); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password after reset = %v", err)
	}
	if _, _, err := service.Login(ctx, "owner", "a brand new password"); err != nil {
		t.Fatalf("new password after reset: %v", err)
	}
}
