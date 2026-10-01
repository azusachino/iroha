package auth

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

func newTestAuthDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserPassword{}, &models.Session{}); err != nil {
		t.Fatalf("migrate models: %v", err)
	}
	return db
}

func TestPasswordValidation(t *testing.T) {
	if err := checkPassword("short"); err == nil {
		t.Error("expected error for short password")
	}
	if err := checkPassword(strings.Repeat("a", MinPasswordLength)); err != nil {
		t.Errorf("unexpected error for min length password: %v", err)
	}
	if err := checkPassword(strings.Repeat("a", MaxPasswordLength+1)); err == nil {
		t.Error("expected error for overly long password")
	}
}

func TestPasswordHashingAndVerification(t *testing.T) {
	pw := "correct horse battery staple"
	hash, err := hashPassword(pw)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	ok, err := verifyPassword(hash, pw)
	if err != nil || !ok {
		t.Fatalf("verifyPassword(correct) = %v, %v", ok, err)
	}
	ok, err = verifyPassword(hash, "wrong-password")
	if err != nil || ok {
		t.Fatalf("verifyPassword(wrong) = %v, %v", ok, err)
	}
}

func TestDigestIsDeterministicAndHashed(t *testing.T) {
	d1 := digest("token-123")
	d2 := digest("token-123")
	if d1 != d2 {
		t.Fatalf("digest is not deterministic: %s vs %s", d1, d2)
	}
	if d1 == "token-123" {
		t.Fatal("digest should not equal raw token")
	}
}

func TestAuthService_SetupAndLoginLifecycle(t *testing.T) {
	db := newTestAuthDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// 1. Initially setup is required
	req, err := svc.SetupRequired(ctx)
	if err != nil || !req {
		t.Fatalf("SetupRequired initially = %v, err=%v; want true", req, err)
	}

	// 2. Reject invalid username
	if _, _, err := svc.Setup(ctx, "Bad User", "correct-password-long"); err == nil {
		t.Fatal("expected error for invalid username")
	}

	// 3. Reject short password
	if _, _, err := svc.Setup(ctx, "owner", "short"); err == nil {
		t.Fatal("expected error for short password")
	}

	// 4. Initial setup succeeds
	token, principal, err := svc.Setup(ctx, "owner", "correct-password-long")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if principal.Username != "owner" || token == "" || principal.SessionID == "" {
		t.Fatalf("unexpected principal: %+v, token=%s", principal, token)
	}

	// 5. Setup is no longer required
	req, err = svc.SetupRequired(ctx)
	if err != nil || req {
		t.Fatalf("SetupRequired after setup = %v, err=%v; want false", req, err)
	}

	// 6. Duplicate setup fails
	if _, _, err := svc.Setup(ctx, "second_owner", "correct-password-long"); err == nil {
		t.Fatal("expected error on duplicate setup")
	}

	// 7. Authenticate with valid session token
	authed, err := svc.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if authed.Username != "owner" || authed.UserID != principal.UserID {
		t.Fatalf("unexpected authed principal: %+v", authed)
	}

	// 8. Authenticate with empty or unknown token
	if _, err := svc.Authenticate(ctx, ""); err != ErrUnauthenticated {
		t.Fatalf("Authenticate(empty) = %v, want ErrUnauthenticated", err)
	}
	if _, err := svc.Authenticate(ctx, "unknown-token"); err != ErrUnauthenticated {
		t.Fatalf("Authenticate(unknown) = %v, want ErrUnauthenticated", err)
	}

	// 9. Login with invalid password fails
	if _, _, err := svc.Login(ctx, "owner", "wrong-password-here"); err != ErrInvalidCredentials {
		t.Fatalf("Login(wrong pw) = %v, want ErrInvalidCredentials", err)
	}

	// 10. Login with unknown username fails
	if _, _, err := svc.Login(ctx, "ghost", "correct-password-long"); err != ErrInvalidCredentials {
		t.Fatalf("Login(unknown user) = %v, want ErrInvalidCredentials", err)
	}

	// 11. Login with correct password succeeds and issues new session
	newToken, newPrincipal, err := svc.Login(ctx, "owner", "correct-password-long")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if newPrincipal.Username != "owner" || newToken == "" {
		t.Fatalf("unexpected new principal: %+v", newPrincipal)
	}

	// 12. Logout revokes session
	if err := svc.Logout(ctx, token); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
	if _, err := svc.Authenticate(ctx, token); err != ErrUnauthenticated {
		t.Fatalf("Authenticate after logout = %v, want ErrUnauthenticated", err)
	}
	// Newer session remains valid
	if _, err := svc.Authenticate(ctx, newToken); err != nil {
		t.Fatalf("Newer session invalidated by logout: %v", err)
	}

	// 13. ResetPassword updates password and revokes all active sessions
	uname, err := svc.ResetPassword(ctx, "brand-new-password-long")
	if err != nil || uname != "owner" {
		t.Fatalf("ResetPassword failed: uname=%s err=%v", uname, err)
	}
	if _, err := svc.Authenticate(ctx, newToken); err != ErrUnauthenticated {
		t.Fatalf("Authenticate after ResetPassword = %v, want ErrUnauthenticated", err)
	}

	// 14. Can login with new password
	finalToken, _, err := svc.Login(ctx, "owner", "brand-new-password-long")
	if err != nil || finalToken == "" {
		t.Fatalf("Login with new password failed: %v", err)
	}
}
