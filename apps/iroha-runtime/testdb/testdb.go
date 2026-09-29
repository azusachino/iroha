// Package testdb resolves the database that DB-backed tests may use.
//
// Integration tests truncate tables and delete the owner account, so they must
// never run against a database that holds real data. DSN refuses anything that
// is not clearly disposable.
package testdb

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
)

const (
	// EnvDatabaseURL names the database the tests connect to.
	EnvDatabaseURL = "DATABASE_URL"
	// EnvDisposable ("1") declares that the database behind EnvDatabaseURL is
	// throwaway (for example a per-run container), whatever it is called.
	EnvDisposable = "IROHA_TEST_DB_DISPOSABLE"
	// DisposablePrefix marks a database name as safe to wipe.
	DisposablePrefix = "iroha_test"
)

// DSN returns the DATABASE_URL for an integration test. It skips when the
// variable is unset and fails when the database is not disposable.
func DSN(t testing.TB) string {
	t.Helper()
	dsn := os.Getenv(EnvDatabaseURL)
	if dsn == "" {
		t.Skipf("%s is not set; run `make test-integration`", EnvDatabaseURL)
	}
	if err := Check(dsn, os.Getenv(EnvDisposable) == "1"); err != nil {
		t.Fatal(err)
	}
	return dsn
}

// Check reports an error unless dsn names a database whose name starts with
// DisposablePrefix or the caller declared it disposable.
func Check(dsn string, disposable bool) error {
	if disposable {
		return nil
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("integration tests refuse an unparseable %s: %w", EnvDatabaseURL, err)
	}
	name := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasPrefix(name, DisposablePrefix) {
		return fmt.Errorf("integration tests wipe tables and refuse database %q: use a database named %s* (make test-integration creates one) or set %s=1 for a throwaway database", name, DisposablePrefix, EnvDisposable)
	}
	return nil
}
