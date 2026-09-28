package jobs

import (
	"errors"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

var (
	// ErrJobNotFound means no job or schedule matched.
	ErrJobNotFound = errors.New("job not found")
	// ErrJobState means the job is not in a state that allows the action.
	ErrJobState = errors.New("job cannot change from its current state")
)

// Retry enqueues a fresh copy of a failed or canceled job. The original row
// stays as history.
func (s *Service) Retry(id uuid.UUID) (models.Job, error) {
	job, found, err := s.Get(id)
	if err != nil {
		return models.Job{}, err
	}
	if !found {
		return models.Job{}, ErrJobNotFound
	}
	if job.Status != StatusFailed && job.Status != StatusCanceled {
		return models.Job{}, ErrJobState
	}
	return s.Enqueue(EnqueueInput{
		Kind:        job.Kind,
		Payload:     job.PayloadJSON,
		Priority:    job.Priority,
		MaxAttempts: job.MaxAttempts,
	})
}

// Cancel stops a job that has not started yet.
func (s *Service) Cancel(id uuid.UUID) error {
	now := time.Now().UTC()
	res := s.db.Model(&models.Job{}).
		Where("id = ? and status = ?", id, StatusQueued).
		Updates(map[string]any{"status": StatusCanceled, "finished_at": now, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		if _, found, err := s.Get(id); err != nil {
			return err
		} else if !found {
			return ErrJobNotFound
		}
		return ErrJobState
	}
	return nil
}

// ListSchedules returns every recurring schedule, by kind.
func (s *Service) ListSchedules() ([]models.JobSchedule, error) {
	var rows []models.JobSchedule
	err := s.db.Order("kind").Find(&rows).Error
	return rows, err
}

// SetScheduleEnabled pauses or resumes a schedule.
func (s *Service) SetScheduleEnabled(kind string, enabled bool) error {
	return s.updateSchedule(kind, map[string]any{"enabled": enabled})
}

// RunScheduleNow makes a schedule due immediately; the worker's scheduler
// enqueues it on its next poll.
func (s *Service) RunScheduleNow(kind string) error {
	return s.updateSchedule(kind, map[string]any{"next_run_at": time.Now().UTC()})
}

func (s *Service) updateSchedule(kind string, updates map[string]any) error {
	updates["updated_at"] = time.Now().UTC()
	res := s.db.Model(&models.JobSchedule{}).Where("kind = ?", kind).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrJobNotFound
	}
	return nil
}
