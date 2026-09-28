// Package auth owns single-owner authentication (ADR-0008): one-time setup,
// Argon2id password login, and server-side sessions looked up by the hash of
// an opaque cookie value.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
)

const (
	// SessionTTL is the absolute lifetime of a login session.
	SessionTTL = 30 * 24 * time.Hour
	// MinPasswordLength and MaxPasswordLength bound accepted passwords; the
	// maximum caps Argon2 work on hostile input.
	MinPasswordLength = 12
	MaxPasswordLength = 1024

	// Argon2id parameters (RFC 9106 second recommended option).
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16

	// lastSeenGranularity limits session writes to one per interval.
	lastSeenGranularity = 5 * time.Minute
)

var (
	ErrAlreadySetUp       = errors.New("owner account already exists")
	ErrNotSetUp           = errors.New("owner account has not been set up")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUnauthenticated    = errors.New("no valid session")
	ErrInvalidInput       = errors.New("invalid input")

	validUsername = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	// dummyHash keeps login timing the same whether or not the username exists.
	dummyHash = mustHash("iroha-dummy-password")
)

// Principal is the authenticated owner behind a request.
type Principal struct {
	UserID      string
	Username    string
	DisplayName string
	SessionID   string
	CSRFToken   string
	// ReauthenticatedAt is when this session last re-confirmed the password.
	ReauthenticatedAt *time.Time
}

type Service struct {
	db  *gorm.DB
	now func() time.Time
	// passkeys is nil until ConfigurePasskeys succeeds.
	passkeys *passkeyConfig
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// SetupRequired reports whether the owner account still has to be created.
func (s *Service) SetupRequired(ctx context.Context) (bool, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&models.User{}).Count(&n).Error
	return n == 0, err
}

// Setup creates the sole owner and returns a new session token. The
// single-owner unique index makes a concurrent second setup fail.
func (s *Service) Setup(ctx context.Context, username, password string) (string, Principal, error) {
	username = strings.TrimSpace(username)
	if !validUsername.MatchString(username) {
		return "", Principal{}, fmt.Errorf("%w: username must match %s", ErrInvalidInput, validUsername)
	}
	if err := checkPassword(password); err != nil {
		return "", Principal{}, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return "", Principal{}, err
	}
	id, err := ids.New()
	if err != nil {
		return "", Principal{}, err
	}
	now := s.now()
	user := models.User{ID: id, Username: username, CreatedAt: now}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&models.User{}).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return ErrAlreadySetUp
		}
		if err := tx.Create(&user).Error; err != nil {
			if isUniqueViolation(err) {
				return ErrAlreadySetUp
			}
			return err
		}
		return tx.Create(&models.UserPassword{UserID: id, PasswordHash: hash, UpdatedAt: now}).Error
	})
	if err != nil {
		return "", Principal{}, err
	}
	return s.createSession(ctx, user)
}

// Login verifies the owner's password and returns a new session token.
func (s *Service) Login(ctx context.Context, username, password string) (string, Principal, error) {
	if len(password) > MaxPasswordLength {
		return "", Principal{}, ErrInvalidCredentials
	}
	var user models.User
	res := s.db.WithContext(ctx).Where("username = ?", strings.TrimSpace(username)).Limit(1).Find(&user)
	if res.Error != nil {
		return "", Principal{}, res.Error
	}
	stored := dummyHash
	if res.RowsAffected == 1 {
		var pw models.UserPassword
		if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).Take(&pw).Error; err != nil {
			return "", Principal{}, err
		}
		stored = pw.PasswordHash
	}
	ok, err := verifyPassword(stored, password)
	if err != nil {
		return "", Principal{}, err
	}
	if !ok || res.RowsAffected == 0 {
		return "", Principal{}, ErrInvalidCredentials
	}
	return s.createSession(ctx, user)
}

// Authenticate resolves a session cookie value to its owner.
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	if token == "" {
		return Principal{}, ErrUnauthenticated
	}
	var session models.Session
	now := s.now()
	res := s.db.WithContext(ctx).
		Where("token_sha256 = ? and revoked_at is null and expires_at > ?", digest(token), now).
		Limit(1).Find(&session)
	if res.Error != nil {
		return Principal{}, res.Error
	}
	if res.RowsAffected == 0 {
		return Principal{}, ErrUnauthenticated
	}
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", session.UserID).Take(&user).Error; err != nil {
		return Principal{}, err
	}
	if now.Sub(session.LastSeenAt) > lastSeenGranularity {
		if err := s.db.WithContext(ctx).Model(&session).UpdateColumn("last_seen_at", now).Error; err != nil {
			return Principal{}, err
		}
	}
	return principal(user, session), nil
}

// Logout revokes the session behind token. Unknown tokens are ignored.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.db.WithContext(ctx).Model(&models.Session{}).
		Where("token_sha256 = ? and revoked_at is null", digest(token)).
		Update("revoked_at", s.now()).Error
}

// ResetPassword is the operator-only break-glass reset: it replaces the
// owner's password hash and revokes every active session.
func (s *Service) ResetPassword(ctx context.Context, password string) (string, error) {
	if err := checkPassword(password); err != nil {
		return "", err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return "", err
	}
	var user models.User
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Limit(1).Find(&user)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotSetUp
		}
		now := s.now()
		if err := tx.Model(&models.UserPassword{}).Where("user_id = ?", user.ID).
			Updates(map[string]any{"password_hash": hash, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Session{}).Where("user_id = ? and revoked_at is null", user.ID).
			Update("revoked_at", now).Error
	})
	return user.Username, err
}

func (s *Service) createSession(ctx context.Context, user models.User) (string, Principal, error) {
	token, err := randomToken()
	if err != nil {
		return "", Principal{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return "", Principal{}, err
	}
	id, err := ids.New()
	if err != nil {
		return "", Principal{}, err
	}
	now := s.now()
	session := models.Session{
		ID: id, UserID: user.ID, TokenSHA256: digest(token), CSRFToken: csrf,
		CreatedAt: now, ExpiresAt: now.Add(SessionTTL), LastSeenAt: now,
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return "", Principal{}, err
	}
	return token, principal(user, session), nil
}

func principal(user models.User, session models.Session) Principal {
	return Principal{
		UserID:      ids.Encode(ids.UserPrefix, user.ID),
		Username:    user.Username,
		DisplayName: displayName(user),
		SessionID:   ids.Encode(ids.SessionPrefix, session.ID),
		CSRFToken:   session.CSRFToken,

		ReauthenticatedAt: session.ReauthenticatedAt,
	}
}

// SetDisplayName sets or clears (empty) the owner's display name.
func (s *Service) SetDisplayName(ctx context.Context, userID uuid.UUID, name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := len([]rune(name)); n > 64 {
		return "", fmt.Errorf("%w: display name must be at most 64 characters", ErrInvalidInput)
	}
	var value *string
	if name != "" {
		value = &name
	}
	err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("display_name", value).Error
	return name, err
}

func displayName(user models.User) string {
	if user.DisplayName != nil {
		return *user.DisplayName
	}
	return ""
}

func checkPassword(password string) error {
	if n := len([]rune(password)); n < MinPasswordLength || len(password) > MaxPasswordLength {
		return fmt.Errorf("%w: password must be %d to %d characters", ErrInvalidInput, MinPasswordLength, MaxPasswordLength)
	}
	return nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

func verifyPassword(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("unsupported password hash")
	}
	var version int
	var memory, iterations uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errors.New("unsupported argon2 version")
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads); err != nil {
		return false, errors.New("malformed argon2 parameters")
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := enc.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func mustHash(password string) string {
	hash, err := hashPassword(password)
	if err != nil {
		panic(err)
	}
	return hash
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func digest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
