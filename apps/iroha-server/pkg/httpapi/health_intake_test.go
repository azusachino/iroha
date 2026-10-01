package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/intakecredential"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeIntakeVerifier struct {
	token string
	err   error
}

func (f fakeIntakeVerifier) Verify(_ context.Context, token string) (models.IntakeCredential, error) {
	if f.err != nil {
		return models.IntakeCredential{}, f.err
	}
	if token != f.token {
		return models.IntakeCredential{}, intakecredential.ErrInvalid
	}
	return models.IntakeCredential{Name: "primary"}, nil
}

func TestBearerToken(t *testing.T) {
	for header, want := range map[string]string{
		"Bearer secret":       "secret",
		"bearer secret":       "secret",
		"":                    "",
		"secret":              "",
		"Bearer":              "",
		"Basic secret":        "",
		"Bearer secret extra": "",
	} {
		if got := bearerToken(header); got != want {
			t.Errorf("bearerToken(%q) = %q, want %q", header, got, want)
		}
	}
}

func postHealthIntake(t *testing.T, verifier HealthIntakeVerifier, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	server := NewServer(Dependencies{
		Config:                  config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}},
		HealthIntakeCredentials: verifier,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/intake/health", strings.NewReader("not-json"))
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	server.ServeHTTP(recorder, request)
	return recorder
}

func TestHealthIntakeFailsClosedWithoutVerifier(t *testing.T) {
	if code := postHealthIntake(t, nil, "Bearer anything").Code; code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", code)
	}
}

func TestHealthIntakeFailsClosedWhenNotProvisioned(t *testing.T) {
	verifier := fakeIntakeVerifier{err: intakecredential.ErrNotProvisioned}
	if code := postHealthIntake(t, verifier, "Bearer anything").Code; code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", code)
	}
}

func TestHealthIntakeRejectsMissingOrWrongCredentialBeforeBodyProcessing(t *testing.T) {
	verifier := fakeIntakeVerifier{token: "secret"}
	for _, authorization := range []string{"", "Bearer wrong", "secret"} {
		recorder := postHealthIntake(t, verifier, authorization)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("authorization %q: status = %d, want 401", authorization, recorder.Code)
		}
		if recorder.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("authorization %q: missing WWW-Authenticate", authorization)
		}
	}
}

func TestHealthIntakeAcceptsValidCredential(t *testing.T) {
	// A valid credential reaches body validation, so an invalid body yields 400.
	if code := postHealthIntake(t, fakeIntakeVerifier{token: "secret"}, "Bearer secret").Code; code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
}

func TestHealthIntakeVerifierFailureIsServerError(t *testing.T) {
	server := NewServer(Dependencies{
		Config:                  config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}},
		HealthIntakeCredentials: fakeIntakeVerifier{err: errors.New("db down")},
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/intake/health", strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer secret")
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

type fakeTestEnqueuer struct{}

func (f *fakeTestEnqueuer) EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error) {
	return models.Job{ID: uuid.New()}, nil
}

func TestHealthIntakeAcceptsValidHealthAutoExportPayload(t *testing.T) {
	dbName := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.RawFile{}, &models.SourceReceipt{}, &models.ImportJob{}, &models.SourceInstance{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_source_instances_provider_key ON tb_source_instances(provider, instance_key)").Error; err != nil {
		t.Fatalf("create index: %v", err)
	}

	tempDir := t.TempDir()
	rawFileService, err := rawfiles.NewService(db, tempDir)
	if err != nil {
		t.Fatalf("new rawfiles service: %v", err)
	}

	importService := imports.NewService(db, slog.Default(), "test", &fakeTestEnqueuer{}, nil)

	server := NewServer(Dependencies{
		Logger:                  slog.Default(),
		Config:                  config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}},
		HealthIntakeCredentials: fakeIntakeVerifier{token: "valid-hae-token"},
		RawFileService:          rawFileService,
		ImportService:           importService,
	})

	body := `{"data":{"metrics":[{"name":"step_count","units":"count","data":[{"date":"2026-09-20 00:00:00 +0900","qty":1000}]}]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/intake/health", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-hae-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body: %s", rec.Code, rec.Body.String())
	}

	var resp healthIntakeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !strings.HasPrefix(resp.RawFileID, "raw_") {
		t.Errorf("raw_file_id = %q, want raw_ prefix", resp.RawFileID)
	}
	if !strings.HasPrefix(resp.ImportID, "imp_") {
		t.Errorf("import_id = %q, want imp_ prefix", resp.ImportID)
	}
	if resp.Status != "queued" {
		t.Errorf("status = %q, want queued", resp.Status)
	}

	var rawCount int64
	if err := db.Model(&models.RawFile{}).Count(&rawCount).Error; err != nil || rawCount != 1 {
		t.Errorf("tb_raw_files count = %d (err: %v), want 1", rawCount, err)
	}

	var receiptCount int64
	if err := db.Model(&models.SourceReceipt{}).Count(&receiptCount).Error; err != nil || receiptCount != 1 {
		t.Errorf("tb_source_receipts count = %d (err: %v), want 1", receiptCount, err)
	}

	var jobCount int64
	if err := db.Model(&models.ImportJob{}).Count(&jobCount).Error; err != nil || jobCount != 1 {
		t.Errorf("tb_import_jobs count = %d (err: %v), want 1", jobCount, err)
	}
}
