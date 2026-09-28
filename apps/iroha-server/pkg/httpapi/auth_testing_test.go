package httpapi

import (
	"context"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/auth"
	"github.com/google/uuid"
)

const testCSRFToken = "test-csrf"

// allowAllAuth authenticates every request as the owner so route tests can
// exercise handlers; auth_test.go covers the real gate.
type allowAllAuth struct{}

func (allowAllAuth) SetupRequired(context.Context) (bool, error) { return false, nil }
func (allowAllAuth) Setup(context.Context, string, string) (string, auth.Principal, error) {
	return "", auth.Principal{}, auth.ErrAlreadySetUp
}

func (allowAllAuth) Login(context.Context, string, string) (string, auth.Principal, error) {
	return "", auth.Principal{}, auth.ErrInvalidCredentials
}

func (allowAllAuth) Authenticate(context.Context, string) (auth.Principal, error) {
	return auth.Principal{Username: "owner", CSRFToken: testCSRFToken}, nil
}
func (allowAllAuth) Logout(context.Context, string) error { return nil }
func (allowAllAuth) SetDisplayName(_ context.Context, _ uuid.UUID, name string) (string, error) {
	return name, nil
}
