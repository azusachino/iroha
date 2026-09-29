package rawfiles

import (
	"context"
	"errors"
	"os"
	"time"
)

const (
	DefaultRetention       = 7 * 24 * time.Hour
	DefaultFailedRetention = 30 * 24 * time.Hour
)

const latestImportStatus = `(select j.status from tb_import_jobs j where j.raw_file_id = f.id order by j.created_at desc limit 1)`

// Purge deletes the bytes of raw files past their retention window and marks
// the rows purged. Completed imports use retention; failed ones use
// failedRetention; queued, parsing and job-less files are never purged. The
// file is removed before the row is marked, so a crash only repeats work.
func (s *Service) Purge(ctx context.Context, retention, failedRetention time.Duration) (int, error) {
	now := time.Now().UTC()
	var rows []struct {
		ID          string
		StoragePath string
	}
	err := s.db.WithContext(ctx).Raw(`select f.id, f.storage_path from tb_raw_files f
where f.purged_at is null and (
  (f.created_at < ? and `+latestImportStatus+` = 'completed') or
  (f.created_at < ? and `+latestImportStatus+` = 'failed'))`,
		now.Add(-retention), now.Add(-failedRetention)).Scan(&rows).Error
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, row := range rows {
		if err := os.Remove(row.StoragePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return purged, err
		}
		if err := s.db.WithContext(ctx).Exec(`update tb_raw_files set purged_at = ? where id = ?`, now, row.ID).Error; err != nil {
			return purged, err
		}
		purged++
	}
	return purged, nil
}
