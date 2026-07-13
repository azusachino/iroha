package rawfiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/gorm"
)

var safeNamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type Service struct {
	db      *gorm.DB
	dataDir string
}

type CreateInput struct {
	File             multipart.File
	OriginalFilename string
	ContentType      string
	SourceKind       string
	UploadedVia      string
}

func NewService(db *gorm.DB, dataDir string) (*Service, error) {
	// Resolve to an absolute path so the persisted storage_path is
	// independent of the process working directory. The import worker
	// (iroha-job) runs from a different CWD than the server and opens the
	// raw file by its stored path, so a relative path would not resolve.
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absDataDir, 0o755); err != nil {
		return nil, err
	}
	return &Service{db: db, dataDir: absDataDir}, nil
}

func (s *Service) Create(input CreateInput) (models.RawFile, bool, error) {
	now := time.Now().UTC()
	id, err := ids.New()
	if err != nil {
		return models.RawFile{}, false, err
	}

	tempDir := filepath.Join(s.dataDir, "tmp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return models.RawFile{}, false, err
	}

	tempFile, err := os.CreateTemp(tempDir, "raw-*")
	if err != nil {
		return models.RawFile{}, false, err
	}
	tempPath := tempFile.Name()
	defer func() { _ = os.Remove(tempPath) }()

	hasher := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(tempFile, hasher), input.File)
	closeErr := tempFile.Close()
	if copyErr != nil {
		return models.RawFile{}, false, copyErr
	}
	if closeErr != nil {
		return models.RawFile{}, false, closeErr
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	existing, found, err := s.findByHash(hash)
	if err != nil {
		return models.RawFile{}, false, err
	}
	if found {
		return existing, true, nil
	}

	storagePath := s.storagePath(now, id.String(), input.OriginalFilename)
	if err := os.MkdirAll(filepath.Dir(storagePath), 0o755); err != nil {
		return models.RawFile{}, false, err
	}
	if err := os.Rename(tempPath, storagePath); err != nil {
		return models.RawFile{}, false, err
	}

	rawFile := models.RawFile{
		ID:               id,
		SHA256:           hash,
		OriginalFilename: input.OriginalFilename,
		ContentType:      input.ContentType,
		SizeBytes:        size,
		StoragePath:      storagePath,
		SourceKind:       input.SourceKind,
		UploadedVia:      input.UploadedVia,
		CreatedAt:        now,
	}

	if err := s.db.Create(&rawFile).Error; err != nil {
		return models.RawFile{}, false, err
	}

	return rawFile, false, nil
}

func (s *Service) StoreSnapshot(ctx context.Context, snapshot connector.Snapshot) (models.RawFile, error) {
	if err := ctx.Err(); err != nil {
		return models.RawFile{}, err
	}
	if snapshot.SourceKind == "" {
		return models.RawFile{}, errors.New("snapshot source kind is required")
	}
	if len(snapshot.Body) == 0 {
		return models.RawFile{}, errors.New("snapshot body is empty")
	}
	if snapshot.Filename == "" {
		snapshot.Filename = snapshot.SourceKind + ".json"
	}
	now := time.Now().UTC()
	hashBytes := sha256.Sum256(snapshot.Body)
	hash := hex.EncodeToString(hashBytes[:])
	existing, found, err := s.findByHash(hash)
	if err != nil {
		return models.RawFile{}, err
	}
	if found {
		return existing, nil
	}
	id, err := ids.New()
	if err != nil {
		return models.RawFile{}, err
	}
	storagePath := s.storagePath(now, id.String(), snapshot.Filename)
	if err := os.MkdirAll(filepath.Dir(storagePath), 0o755); err != nil {
		return models.RawFile{}, err
	}
	if err := os.WriteFile(storagePath, snapshot.Body, 0o644); err != nil {
		return models.RawFile{}, err
	}
	rawFile := models.RawFile{
		ID:               id,
		SHA256:           hash,
		OriginalFilename: snapshot.Filename,
		ContentType:      snapshot.ContentType,
		SizeBytes:        int64(len(snapshot.Body)),
		StoragePath:      storagePath,
		SourceKind:       snapshot.SourceKind,
		UploadedVia:      "connector",
		CreatedAt:        now,
	}
	if err := s.db.Create(&rawFile).Error; err != nil {
		_ = os.Remove(storagePath)
		return models.RawFile{}, err
	}
	return rawFile, nil
}

func (s *Service) List(limit int) ([]models.RawFile, error) {
	var rawFiles []models.RawFile
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	err := s.db.Order("created_at desc").Limit(limit).Find(&rawFiles).Error
	return rawFiles, err
}

func (s *Service) Get(id string) (models.RawFile, bool, error) {
	decoded, err := ids.Decode(ids.RawFilePrefix, id)
	if err != nil {
		return models.RawFile{}, false, err
	}

	var rawFile models.RawFile
	err = s.db.First(&rawFile, "id = ?", decoded).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.RawFile{}, false, nil
	}
	if err != nil {
		return models.RawFile{}, false, err
	}
	return rawFile, true, nil
}

func (s *Service) findByHash(hash string) (models.RawFile, bool, error) {
	var rawFile models.RawFile
	err := s.db.First(&rawFile, "sha256 = ?", hash).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.RawFile{}, false, nil
	}
	if err != nil {
		return models.RawFile{}, false, err
	}
	return rawFile, true, nil
}

func (s *Service) storagePath(createdAt time.Time, id string, filename string) string {
	return filepath.Join(
		s.dataDir,
		"raw-files",
		createdAt.Format("2006"),
		createdAt.Format("01"),
		id,
		safeFilename(filename),
	)
}

func safeFilename(filename string) string {
	base := filepath.Base(filename)
	base = strings.TrimSpace(base)
	if base == "" || base == "." {
		return "upload.bin"
	}
	return safeNamePattern.ReplaceAllString(base, "_")
}
