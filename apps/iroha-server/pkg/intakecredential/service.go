// Package intakecredential owns the Health Auto Export intake credentials:
// one high-entropy token per uploading device, persisted only as a SHA-256
// verifier. A random 256-bit token needs no slow password hash.
package intakecredential

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"gorm.io/gorm"
)

const (
	// tokenBytes is the random entropy of an issued token (ADR-0008: at least 32).
	tokenBytes = 32
	// TokenPrefix marks Iroha intake tokens so secret scanners can recognize a leak.
	TokenPrefix = "iroha_hae_"
)

var (
	// ErrNotProvisioned means no active credential exists; intake fails closed.
	ErrNotProvisioned = errors.New("health intake credential is not provisioned")
	// ErrInvalid means the presented token matches no active credential.
	ErrInvalid = errors.New("invalid health intake credential")
	// ErrNotFound means the credential to revoke does not exist or is already revoked.
	ErrNotFound = errors.New("intake credential not found")

	validName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

type Service struct {
	db  *gorm.DB
	now func() time.Time
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// Issue creates a credential for the named device and returns the plaintext
// token once. Existing credentials stay active until revoked, so a rotation
// can overlap with the device being updated.
func (s *Service) Issue(ctx context.Context, name string) (models.IntakeCredential, string, error) {
	if !validName.MatchString(name) {
		return models.IntakeCredential{}, "", fmt.Errorf("credential name %q must match %s", name, validName)
	}
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return models.IntakeCredential{}, "", err
	}
	id, err := ids.New()
	if err != nil {
		return models.IntakeCredential{}, "", err
	}
	token := TokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	row := models.IntakeCredential{ID: id, Name: name, TokenSHA256: verifier(token), CreatedAt: s.now()}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return models.IntakeCredential{}, "", err
	}
	return row, token, nil
}

// List returns every credential, newest first, including revoked ones.
func (s *Service) List(ctx context.Context) ([]models.IntakeCredential, error) {
	var rows []models.IntakeCredential
	err := s.db.WithContext(ctx).Order("created_at desc").Find(&rows).Error
	return rows, err
}

// Revoke disables one active credential immediately.
func (s *Service) Revoke(ctx context.Context, id uuid.UUID) error {
	res := s.db.WithContext(ctx).Model(&models.IntakeCredential{}).
		Where("id = ? and revoked_at is null", id).
		Update("revoked_at", s.now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Verify returns the active credential matching token and records its use.
// The lookup is by verifier, so no constant-time comparison is needed.
func (s *Service) Verify(ctx context.Context, token string) (models.IntakeCredential, error) {
	var row models.IntakeCredential
	if strings.HasPrefix(token, TokenPrefix) {
		res := s.db.WithContext(ctx).Where("token_sha256 = ? and revoked_at is null", verifier(token)).Limit(1).Find(&row)
		if res.Error != nil {
			return models.IntakeCredential{}, res.Error
		}
		if res.RowsAffected == 1 {
			now := s.now()
			if err := s.db.WithContext(ctx).Model(&row).UpdateColumn("last_used_at", now).Error; err != nil {
				return models.IntakeCredential{}, err
			}
			row.LastUsedAt = &now
			return row, nil
		}
	}
	var active int64
	if err := s.db.WithContext(ctx).Model(&models.IntakeCredential{}).Where("revoked_at is null").Count(&active).Error; err != nil {
		return models.IntakeCredential{}, err
	}
	if active == 0 {
		return models.IntakeCredential{}, ErrNotProvisioned
	}
	return models.IntakeCredential{}, ErrInvalid
}

func verifier(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
