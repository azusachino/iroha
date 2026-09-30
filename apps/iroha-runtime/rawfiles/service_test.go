package rawfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewServiceRestrictsExistingRawFiles(t *testing.T) {
	root := t.TempDir()
	rawFilesDir := filepath.Join(root, "raw-files")
	if err := os.MkdirAll(rawFilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Evidence written before the permission hardening kept its old modes.
	legacyDir := filepath.Join(rawFilesDir, "2026", "08", "id")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyFile := filepath.Join(legacyDir, "export.zip")
	if err := os.WriteFile(legacyFile, []byte("evidence"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewService(nil, root); err != nil {
		t.Fatal(err)
	}
	assertMode(t, rawFilesDir, privateDirMode)
	assertMode(t, filepath.Join(rawFilesDir, "2026"), privateDirMode)
	assertMode(t, legacyDir, privateDirMode)
	assertMode(t, legacyFile, privateFileMode)
}

func TestStorePrivateUploadPermissions(t *testing.T) {
	root := t.TempDir()
	service := Service{dataDir: root}
	rawFilesDir := filepath.Join(root, "raw-files")
	if err := os.MkdirAll(rawFilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	storageDir := filepath.Join(rawFilesDir, "2026", "09", "id")
	if err := service.ensureStorageDir(storageDir); err != nil {
		t.Fatal(err)
	}
	tempDir := filepath.Join(root, "tmp")
	if err := ensurePrivateDir(tempDir); err != nil {
		t.Fatal(err)
	}
	tempFile, err := os.CreateTemp(tempDir, "raw-*")
	if err != nil {
		t.Fatal(err)
	}
	tempPath := tempFile.Name()
	if _, err := tempFile.Write([]byte("upload")); err != nil {
		t.Fatal(err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tempPath, 0o644); err != nil {
		t.Fatal(err)
	}

	storedPath := filepath.Join(storageDir, "upload.bin")
	if err := storePrivateUpload(tempPath, storedPath); err != nil {
		t.Fatal(err)
	}
	assertMode(t, rawFilesDir, privateDirMode)
	assertMode(t, storageDir, privateDirMode)
	assertMode(t, tempDir, privateDirMode)
	assertMode(t, storedPath, privateFileMode)
}

func TestStorePrivateSnapshotPermissions(t *testing.T) {
	root := t.TempDir()
	service := Service{dataDir: root}
	rawFilesDir := filepath.Join(root, "raw-files")
	if err := os.MkdirAll(rawFilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	storageDir := filepath.Join(rawFilesDir, "2026", "09", "id")
	if err := service.ensureStorageDir(storageDir); err != nil {
		t.Fatal(err)
	}

	storedPath := filepath.Join(storageDir, "snapshot.json")
	if err := writePrivateSnapshot(storedPath, []byte("snapshot")); err != nil {
		t.Fatal(err)
	}
	assertMode(t, rawFilesDir, privateDirMode)
	assertMode(t, storageDir, privateDirMode)
	assertMode(t, storedPath, privateFileMode)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode = %04o, want %04o", path, got, want)
	}
}

func TestNewServiceLeavesSymlinkTargetsOutsideRawFiles(t *testing.T) {
	root := t.TempDir()
	rawFilesDir := filepath.Join(root, "raw-files")
	if err := os.MkdirAll(rawFilesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(outside, []byte("not evidence"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(rawFilesDir, "link")); err != nil {
		t.Fatal(err)
	}

	if _, err := NewService(nil, root); err != nil {
		t.Fatal(err)
	}
	assertMode(t, outside, 0o644)
}
