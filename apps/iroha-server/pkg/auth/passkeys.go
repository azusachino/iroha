package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

const (
	// ReauthWindow is how long a password re-confirmation authorizes adding
	// or removing a passkey (ADR-0008 item 6).
	ReauthWindow = 5 * time.Minute
	// ceremonyTTL bounds a WebAuthn ceremony; its challenge is single-use.
	ceremonyTTL = 5 * time.Minute
	// maxCeremonies caps pending ceremonies so anonymous callers cannot grow
	// server memory without bound.
	maxCeremonies = 256
)

var (
	ErrPasskeysDisabled = errors.New("passkeys are not configured")
	ErrReauthRequired   = errors.New("recent password confirmation required")
	ErrCeremony         = errors.New("passkey ceremony expired or unknown")
	ErrPasskeyNotFound  = errors.New("passkey not found")
)

// PasskeyInfo is the public view of one registered passkey.
type PasskeyInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

type passkeyConfig struct {
	wa         *webauthn.WebAuthn
	ceremonies *ceremonyStore
}

// ConfigurePasskeys enables WebAuthn for rpID (the site's host name) and the
// exact HTTPS origins browsers present. Trust never comes from forwarded
// headers: the library checks each response against these values.
func (s *Service) ConfigurePasskeys(rpID string, origins []string) error {
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: "iroha",
		RPOrigins:     origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		},
	})
	if err != nil {
		return err
	}
	s.passkeys = &passkeyConfig{wa: wa, ceremonies: newCeremonyStore(s.now)}
	return nil
}

// PasskeysEnabled reports whether ConfigurePasskeys succeeded.
func (s *Service) PasskeysEnabled() bool { return s.passkeys != nil }

// Reauthenticate re-confirms the owner's password for the given session.
func (s *Service) Reauthenticate(ctx context.Context, who Principal, password string) error {
	userID, sessionID, err := principalIDs(who)
	if err != nil {
		return err
	}
	var pw models.UserPassword
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Take(&pw).Error; err != nil {
		return err
	}
	if len(password) > MaxPasswordLength {
		return ErrInvalidCredentials
	}
	ok, err := verifyPassword(pw.PasswordHash, password)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}
	return s.db.WithContext(ctx).Model(&models.Session{}).Where("id = ?", sessionID).
		Update("reauthenticated_at", s.now()).Error
}

func (s *Service) recentlyReauthenticated(who Principal) bool {
	return who.ReauthenticatedAt != nil && s.now().Sub(*who.ReauthenticatedAt) < ReauthWindow
}

// ListPasskeys returns the owner's passkeys, oldest first.
func (s *Service) ListPasskeys(ctx context.Context, who Principal) ([]PasskeyInfo, error) {
	userID, _, err := principalIDs(who)
	if err != nil {
		return nil, err
	}
	var rows []models.Passkey
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PasskeyInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, passkeyInfo(row))
	}
	return out, nil
}

// BeginPasskeyRegistration starts adding a passkey. It needs a recent
// password confirmation and returns the browser options plus a ceremony id.
func (s *Service) BeginPasskeyRegistration(ctx context.Context, who Principal) (any, string, error) {
	if s.passkeys == nil {
		return nil, "", ErrPasskeysDisabled
	}
	if !s.recentlyReauthenticated(who) {
		return nil, "", ErrReauthRequired
	}
	user, err := s.webauthnUser(ctx, who, true)
	if err != nil {
		return nil, "", err
	}
	creation, session, err := s.passkeys.wa.BeginRegistration(user,
		webauthn.WithExclusions(webauthn.Credentials(user.credentials).CredentialDescriptors()))
	if err != nil {
		return nil, "", err
	}
	id, err := s.passkeys.ceremonies.put(ceremony{kind: "register", userID: user.id, session: *session})
	if err != nil {
		return nil, "", err
	}
	return creation, id, nil
}

// FinishPasskeyRegistration verifies the browser's response and stores the
// new credential under name.
func (s *Service) FinishPasskeyRegistration(ctx context.Context, who Principal, ceremonyID, name string, r *http.Request) (PasskeyInfo, error) {
	if s.passkeys == nil {
		return PasskeyInfo{}, ErrPasskeysDisabled
	}
	if !s.recentlyReauthenticated(who) {
		return PasskeyInfo{}, ErrReauthRequired
	}
	name = strings.TrimSpace(name)
	if n := len([]rune(name)); n < 1 || n > 64 {
		return PasskeyInfo{}, fmt.Errorf("%w: passkey name must be 1 to 64 characters", ErrInvalidInput)
	}
	c, ok := s.passkeys.ceremonies.take(ceremonyID, "register")
	if !ok {
		return PasskeyInfo{}, ErrCeremony
	}
	user, err := s.webauthnUser(ctx, who, false)
	if err != nil {
		return PasskeyInfo{}, err
	}
	if user.id != c.userID {
		return PasskeyInfo{}, ErrCeremony
	}
	credential, err := s.passkeys.wa.FinishRegistration(user, c.session, r)
	if err != nil {
		return PasskeyInfo{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	raw, err := json.Marshal(credential)
	if err != nil {
		return PasskeyInfo{}, err
	}
	id, err := ids.New()
	if err != nil {
		return PasskeyInfo{}, err
	}
	row := models.Passkey{ID: id, UserID: user.id, CredentialID: credential.ID, Credential: raw, Name: name, CreatedAt: s.now()}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return PasskeyInfo{}, err
	}
	return passkeyInfo(row), nil
}

// RenamePasskey changes a passkey's label.
func (s *Service) RenamePasskey(ctx context.Context, who Principal, id uuid.UUID, name string) error {
	userID, _, err := principalIDs(who)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if n := len([]rune(name)); n < 1 || n > 64 {
		return fmt.Errorf("%w: passkey name must be 1 to 64 characters", ErrInvalidInput)
	}
	res := s.db.WithContext(ctx).Model(&models.Passkey{}).Where("id = ? and user_id = ?", id, userID).Update("name", name)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}

// DeletePasskey removes a passkey; it needs a recent password confirmation.
func (s *Service) DeletePasskey(ctx context.Context, who Principal, id uuid.UUID) error {
	if !s.recentlyReauthenticated(who) {
		return ErrReauthRequired
	}
	userID, _, err := principalIDs(who)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Where("id = ? and user_id = ?", id, userID).Delete(&models.Passkey{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPasskeyNotFound
	}
	return nil
}

// BeginPasskeyLogin starts a discoverable (username-less) passkey sign-in.
func (s *Service) BeginPasskeyLogin() (any, string, error) {
	if s.passkeys == nil {
		return nil, "", ErrPasskeysDisabled
	}
	assertion, session, err := s.passkeys.wa.BeginDiscoverableLogin()
	if err != nil {
		return nil, "", err
	}
	id, err := s.passkeys.ceremonies.put(ceremony{kind: "login", session: *session})
	if err != nil {
		return nil, "", err
	}
	return assertion, id, nil
}

// FinishPasskeyLogin verifies a passkey assertion and starts a session.
func (s *Service) FinishPasskeyLogin(ctx context.Context, ceremonyID string, r *http.Request) (string, Principal, error) {
	if s.passkeys == nil {
		return "", Principal{}, ErrPasskeysDisabled
	}
	c, ok := s.passkeys.ceremonies.take(ceremonyID, "login")
	if !ok {
		return "", Principal{}, ErrCeremony
	}
	var found *webauthnUser
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		var user models.User
		res := s.db.WithContext(ctx).Where("webauthn_handle = ?", userHandle).Limit(1).Find(&user)
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, ErrInvalidCredentials
		}
		u, err := s.loadWebauthnUser(ctx, user)
		if err != nil {
			return nil, err
		}
		found = u
		return u, nil
	}
	_, credential, err := s.passkeys.wa.FinishPasskeyLogin(handler, c.session, r)
	if err != nil || found == nil {
		return "", Principal{}, ErrInvalidCredentials
	}
	raw, err := json.Marshal(credential)
	if err != nil {
		return "", Principal{}, err
	}
	now := s.now()
	if err := s.db.WithContext(ctx).Model(&models.Passkey{}).Where("credential_id = ?", credential.ID).
		Updates(map[string]any{"credential": raw, "last_used_at": now}).Error; err != nil {
		return "", Principal{}, err
	}
	return s.createSession(ctx, found.user)
}

// webauthnUser adapts the owner to the library's User interface.
type webauthnUser struct {
	id          uuid.UUID
	user        models.User
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte          { return u.user.WebauthnHandle }
func (u *webauthnUser) WebAuthnName() string        { return u.user.Username }
func (u *webauthnUser) WebAuthnDisplayName() string { return displayName(u.user) }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (s *Service) webauthnUser(ctx context.Context, who Principal, ensureHandle bool) (*webauthnUser, error) {
	userID, _, err := principalIDs(who)
	if err != nil {
		return nil, err
	}
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).Take(&user).Error; err != nil {
		return nil, err
	}
	if len(user.WebauthnHandle) == 0 && ensureHandle {
		handle := make([]byte, 64)
		if _, err := rand.Read(handle); err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Model(&user).Update("webauthn_handle", handle).Error; err != nil {
			return nil, err
		}
		user.WebauthnHandle = handle
	}
	return s.loadWebauthnUser(ctx, user)
}

func (s *Service) loadWebauthnUser(ctx context.Context, user models.User) (*webauthnUser, error) {
	var rows []models.Passkey
	if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).Find(&rows).Error; err != nil {
		return nil, err
	}
	creds := make([]webauthn.Credential, 0, len(rows))
	for _, row := range rows {
		var c webauthn.Credential
		if err := json.Unmarshal(row.Credential, &c); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return &webauthnUser{id: user.ID, user: user, credentials: creds}, nil
}

func principalIDs(who Principal) (uuid.UUID, uuid.UUID, error) {
	userID, err := ids.Decode(ids.UserPrefix, who.UserID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrUnauthenticated
	}
	sessionID, err := ids.Decode(ids.SessionPrefix, who.SessionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrUnauthenticated
	}
	return userID, sessionID, nil
}

func passkeyInfo(row models.Passkey) PasskeyInfo {
	return PasskeyInfo{
		ID:         ids.Encode(ids.PasskeyPrefix, row.ID),
		Name:       row.Name,
		CreatedAt:  row.CreatedAt,
		LastUsedAt: row.LastUsedAt,
	}
}

// ceremonyStore holds short-lived, single-use WebAuthn ceremony state in
// memory (exact for the single iroha-server replica).
type ceremony struct {
	kind    string
	userID  uuid.UUID
	session webauthn.SessionData
	expires time.Time
}

type ceremonyStore struct {
	mu    sync.Mutex
	now   func() time.Time
	items map[string]ceremony
}

func newCeremonyStore(now func() time.Time) *ceremonyStore {
	return &ceremonyStore{now: now, items: map[string]ceremony{}}
}

func (c *ceremonyStore) put(item ceremony) (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(raw)
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for key, existing := range c.items {
		if now.After(existing.expires) {
			delete(c.items, key)
		}
	}
	if len(c.items) >= maxCeremonies {
		return "", ErrCeremony
	}
	item.expires = now.Add(ceremonyTTL)
	c.items[id] = item
	return id, nil
}

func (c *ceremonyStore) take(id, kind string) (ceremony, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[id]
	delete(c.items, id)
	if !ok || item.kind != kind || c.now().After(item.expires) {
		return ceremony{}, false
	}
	return item, true
}
