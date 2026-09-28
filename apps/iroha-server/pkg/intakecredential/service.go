// Package intakecredential owns the Health Auto Export intake credential: a
// high-entropy token whose SHA-256 verifier is the only persisted form.
package intakecredential

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// tokenBytes is the random entropy of an issued token (ADR-0008: at least 32).
const tokenBytes = 32

var (
	// ErrNotProvisioned means no credential has been issued; intake fails closed.
	ErrNotProvisioned = errors.New("health intake credential is not provisioned")
	// ErrInvalid means the presented token does not match the current verifier.
	ErrInvalid = errors.New("invalid health intake credential")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Rotate issues a new token, replaces the stored verifier atomically, and
// returns the plaintext once. Any previously issued token stops working.
func (s *Service) Rotate(ctx context.Context) (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	row := models.HealthIntakeCredential{Singleton: true, TokenSHA256: verifier(token), RotatedAt: time.Now().UTC()}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "singleton"}},
		DoUpdates: clause.AssignmentColumns([]string{"token_sha256", "rotated_at"}),
	}).Create(&row).Error
	if err != nil {
		return "", err
	}
	return token, nil
}

// Verify checks a presented token against the stored verifier.
func (s *Service) Verify(ctx context.Context, token string) error {
	var row models.HealthIntakeCredential
	res := s.db.WithContext(ctx).Limit(1).Find(&row)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotProvisioned
	}
	if token == "" || subtle.ConstantTimeCompare([]byte(verifier(token)), []byte(row.TokenSHA256)) != 1 {
		return ErrInvalid
	}
	return nil
}

func verifier(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
