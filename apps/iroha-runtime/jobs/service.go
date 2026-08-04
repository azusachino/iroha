package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCanceled  = "canceled"

	KindAppleImportParse      = "apple_import_parse"
	KindGPXImportParse        = "gpx_import_parse"
	KindFITImportParse        = "fit_import_parse"
	KindTCXImportParse        = "tcx_import_parse"
	KindStravaImportParse     = "strava_import_parse"
	KindMediaIntakeParse      = "media_intake_parse"
	KindMediaConnectorSync    = "media_connector_sync"
	KindMediaSyncAniList      = "media_sync_anilist"
	KindMediaSyncBangumi      = "media_sync_bangumi"
	KindHealthFullDumpRequest = "health_full_dump_request"
	KindProjectionRefresh     = "projection_refresh"
	KindPublicSummaryRefresh  = "public_summary_refresh"
	KindParserReprocess       = "parser_reprocess"
	KindGeocodeRefresh        = "geocode_refresh"

	ScheduleKindInterval = "interval"
	ScheduleKindManual   = "manual"

	DefaultMaxAttempts  = 3
	DefaultLimit        = 50
	DefaultLeaseTimeout = 5 * time.Minute
)

var (
	ErrNoJobAvailable = errors.New("no job available")
	ErrUnknownJobKind = errors.New("unknown job kind")
)

type Handler func(context.Context, models.Job) error

type Service struct {
	db       *gorm.DB
	logger   *slog.Logger
	handlers map[string]Handler
}

type EnqueueInput struct {
	Kind        string
	Payload     any
	Priority    int
	MaxAttempts int
	RunAfter    time.Time
}

type ScheduleInput struct {
	Kind         string
	ScheduleKind string
	ScheduleExpr string
	Payload      any
	NextRunAt    *time.Time
	Enabled      bool
}

type ListFilters struct {
	Kind   string
	Status string
	Limit  int
}

func NewService(db *gorm.DB, logger *slog.Logger, handlers map[string]Handler) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if handlers == nil {
		handlers = map[string]Handler{}
	}
	return &Service{db: db, logger: logger, handlers: handlers}
}

func (s *Service) Enqueue(input EnqueueInput) (models.Job, error) {
	return s.EnqueueTx(nil, input)
}

func (s *Service) EnqueueTx(tx *gorm.DB, input EnqueueInput) (models.Job, error) {
	if input.Kind == "" {
		return models.Job{}, fmt.Errorf("job kind is required")
	}

	payload, err := marshalPayload(input.Payload)
	if err != nil {
		return models.Job{}, err
	}

	id, err := ids.New()
	if err != nil {
		return models.Job{}, err
	}

	now := time.Now().UTC()
	runAfter := input.RunAfter
	if runAfter.IsZero() {
		runAfter = now
	}
	maxAttempts := input.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}

	job := models.Job{
		ID:          id,
		Kind:        input.Kind,
		Status:      StatusQueued,
		Priority:    input.Priority,
		PayloadJSON: payload,
		MaxAttempts: maxAttempts,
		RunAfter:    runAfter.UTC(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	db := s.db
	if tx != nil {
		db = tx
	}

	if err := db.Create(&job).Error; err != nil {
		return models.Job{}, err
	}
	s.logger.Info("enqueued job", "job_id", job.ID.String(), "kind", job.Kind, "run_after", job.RunAfter)
	return job, nil
}

func (s *Service) List(filters ListFilters) ([]models.Job, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = DefaultLimit
	}
	query := s.db.Model(&models.Job{})
	if filters.Kind != "" {
		query = query.Where("kind = ?", filters.Kind)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	var result []models.Job
	if err := query.Order("created_at desc").Limit(limit).Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) Get(id uuid.UUID) (models.Job, bool, error) {
	var job models.Job
	result := s.db.First(&job, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return models.Job{}, false, nil
		}
		return models.Job{}, false, result.Error
	}
	return job, true, nil
}

func (s *Service) ClaimNext(workerID string) (models.Job, error) {
	if workerID == "" {
		return models.Job{}, fmt.Errorf("worker id is required")
	}

	now := time.Now().UTC()
	var job models.Job
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := recoverExpiredJobs(tx, now); err != nil {
			return err
		}
		result := tx.Raw(`
			update tb_jobs
			set status = ?,
			    attempts = attempts + 1,
			    locked_by = ?,
			    locked_at = ?,
			    started_at = coalesce(started_at, ?),
			    updated_at = ?
			where id = (
				select id
				from tb_jobs
				where status = ?
				  and run_after <= ?
				  and attempts < max_attempts
				order by priority desc, run_after asc, created_at asc
				for update skip locked
				limit 1
			)
			returning *
		`, StatusRunning, workerID, now, now, now, StatusQueued, now).Scan(&job)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNoJobAvailable
		}
		return nil
	})
	if err != nil {
		return models.Job{}, err
	}
	return job, nil
}

func (s *Service) Complete(jobID uuid.UUID) error {
	now := time.Now().UTC()
	return s.db.Model(&models.Job{}).
		Where("id = ? and status = ?", jobID, StatusRunning).
		Updates(map[string]any{
			"status":        StatusCompleted,
			"finished_at":   &now,
			"locked_by":     nil,
			"locked_at":     nil,
			"updated_at":    now,
			"error_message": nil,
		}).Error
}

func (s *Service) Fail(job models.Job, cause error) error {
	if cause == nil {
		cause = fmt.Errorf("job failed")
	}

	now := time.Now().UTC()
	message := cause.Error()
	nextStatus := StatusQueued
	updates := map[string]any{
		"locked_by":     nil,
		"locked_at":     nil,
		"updated_at":    now,
		"error_message": &message,
	}
	if job.Attempts >= job.MaxAttempts {
		nextStatus = StatusFailed
		updates["finished_at"] = &now
	} else {
		updates["run_after"] = now.Add(retryDelayFor(cause, job.Attempts))
	}
	updates["status"] = nextStatus

	return s.db.Model(&models.Job{}).
		Where("id = ? and status = ?", job.ID, StatusRunning).
		Updates(updates).Error
}

func (s *Service) ProcessNext(ctx context.Context, workerID string) (models.Job, error) {
	job, err := s.ClaimNext(workerID)
	if err != nil {
		return models.Job{}, err
	}

	handler, ok := s.handlers[job.Kind]
	if !ok {
		err := fmt.Errorf("%w: %s", ErrUnknownJobKind, job.Kind)
		if failErr := s.Fail(job, err); failErr != nil {
			return job, failErr
		}
		return job, err
	}

	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go s.refreshLeaseLoop(workCtx, job.ID, workerID)

	if err := handler(workCtx, job); err != nil {
		if failErr := s.Fail(job, err); failErr != nil {
			return job, failErr
		}
		return job, err
	}

	if err := s.Complete(job.ID); err != nil {
		return job, err
	}
	return job, nil
}

func (s *Service) refreshLeaseLoop(ctx context.Context, jobID uuid.UUID, workerID string) {
	ticker := time.NewTicker(DefaultLeaseTimeout / 3)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			result := s.db.Model(&models.Job{}).
				Where("id = ? and status = ? and locked_by = ?", jobID, StatusRunning, workerID).
				Updates(map[string]any{"locked_at": now.UTC(), "updated_at": now.UTC()})
			if result.Error != nil {
				s.logger.Error("refresh job lease", "job_id", jobID.String(), "worker", workerID, "error", result.Error)
				return
			}
			if result.RowsAffected == 0 {
				return
			}
		}
	}
}

func (s *Service) CreateSchedule(input ScheduleInput) (models.JobSchedule, error) {
	if input.Kind == "" {
		return models.JobSchedule{}, fmt.Errorf("job kind is required")
	}
	if input.ScheduleKind == "" {
		return models.JobSchedule{}, fmt.Errorf("schedule kind is required")
	}
	if input.ScheduleExpr == "" {
		return models.JobSchedule{}, fmt.Errorf("schedule expression is required")
	}

	payload, err := marshalPayload(input.Payload)
	if err != nil {
		return models.JobSchedule{}, err
	}

	id, err := ids.New()
	if err != nil {
		return models.JobSchedule{}, err
	}

	now := time.Now().UTC()
	schedule := models.JobSchedule{
		ID:           id,
		Kind:         input.Kind,
		Enabled:      input.Enabled,
		ScheduleKind: input.ScheduleKind,
		ScheduleExpr: input.ScheduleExpr,
		PayloadJSON:  payload,
		NextRunAt:    input.NextRunAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.db.Create(&schedule).Error; err != nil {
		return models.JobSchedule{}, err
	}
	return schedule, nil
}

func (s *Service) EnqueueDueSchedules(limit int) (int, error) {
	if limit <= 0 || limit > 100 {
		limit = DefaultLimit
	}

	now := time.Now().UTC()
	var schedules []models.JobSchedule
	if err := s.db.
		Where("enabled = ? and next_run_at is not null and next_run_at <= ?", true, now).
		Order("next_run_at asc").
		Limit(limit).
		Find(&schedules).Error; err != nil {
		return 0, err
	}

	enqueued := 0
	for _, schedule := range schedules {
		claimed, err := s.enqueueSchedule(schedule, now)
		if err != nil {
			return enqueued, err
		}
		if claimed {
			enqueued++
		}
	}
	return enqueued, nil
}

func (s *Service) enqueueSchedule(schedule models.JobSchedule, now time.Time) (bool, error) {
	claimed := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		nextRunAt, enabled, err := nextScheduleRun(schedule, now)
		if err != nil {
			return err
		}
		updates := map[string]any{
			"last_run_at": &now,
			"next_run_at": nextRunAt,
			"enabled":     enabled,
			"updated_at":  now,
		}
		result := tx.Model(&models.JobSchedule{}).
			Where("id = ? and enabled = ? and next_run_at is not null and next_run_at <= ?", schedule.ID, true, now).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		claimed = true

		id, err := ids.New()
		if err != nil {
			return err
		}

		job := models.Job{
			ID:          id,
			Kind:        schedule.Kind,
			Status:      StatusQueued,
			PayloadJSON: schedule.PayloadJSON,
			MaxAttempts: DefaultMaxAttempts,
			RunAfter:    now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		return nil
	})
	return claimed, err
}

func nextScheduleRun(schedule models.JobSchedule, now time.Time) (*time.Time, bool, error) {
	switch schedule.ScheduleKind {
	case ScheduleKindManual:
		return nil, false, nil
	case ScheduleKindInterval:
		interval, err := time.ParseDuration(schedule.ScheduleExpr)
		if err != nil {
			return nil, schedule.Enabled, err
		}
		next := now.Add(interval)
		return &next, schedule.Enabled, nil
	default:
		return nil, schedule.Enabled, fmt.Errorf("unsupported schedule kind: %s", schedule.ScheduleKind)
	}
}

func retryDelay(attempts int) time.Duration {
	if attempts <= 1 {
		return 30 * time.Second
	}
	delay := time.Duration(attempts) * time.Minute
	if delay > 10*time.Minute {
		return 10 * time.Minute
	}
	return delay
}

type retryAfterError interface {
	RetryAfterDuration() (time.Duration, bool)
}

func retryDelayFor(cause error, attempts int) time.Duration {
	var retryAfter retryAfterError
	if errors.As(cause, &retryAfter) {
		if delay, ok := retryAfter.RetryAfterDuration(); ok {
			return delay
		}
	}
	return retryDelay(attempts)
}

func recoverExpiredJobs(tx *gorm.DB, now time.Time) error {
	leaseExpiredAt := now.Add(-DefaultLeaseTimeout)
	return tx.Exec(`
		update tb_jobs
		set status = case when attempts >= max_attempts then ? else ? end,
		    run_after = case when attempts >= max_attempts then run_after else ? end,
		    finished_at = case when attempts >= max_attempts then ? else finished_at end,
		    locked_by = null,
		    locked_at = null,
		    error_message = coalesce(error_message, 'worker lease expired'),
		    updated_at = ?
		where status = ? and locked_at is not null and locked_at <= ?
	`, StatusFailed, StatusQueued, now, now, now, StatusRunning, leaseExpiredAt).Error
}

func marshalPayload(payload any) (json.RawMessage, error) {
	if payload == nil {
		return json.RawMessage(`{}`), nil
	}
	if raw, ok := payload.(json.RawMessage); ok {
		if !json.Valid(raw) {
			return nil, fmt.Errorf("payload_json must be valid JSON")
		}
		return raw, nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if string(encoded) == "null" {
		return json.RawMessage(`{}`), nil
	}
	return encoded, nil
}
