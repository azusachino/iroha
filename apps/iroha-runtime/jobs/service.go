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
	"gorm.io/gorm/clause"
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
	KindMediaBridgeRefresh    = "media_bridge_refresh"
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
	ErrClaimLost      = errors.New("job claim lost")
	ErrClaimRequired  = errors.New("job claim required")
)

type Handler func(context.Context, models.Job) error

// Claim identifies one ownership generation of a durable job. Attempts is
// monotonic and therefore fences a worker even if a lease is reassigned to
// another worker with the same job ID.
type Claim struct {
	JobID    uuid.UUID
	WorkerID string
	Attempt  int
}

func ClaimFor(job models.Job, workerID string) Claim {
	return Claim{JobID: job.ID, WorkerID: workerID, Attempt: job.Attempts}
}

func (c Claim) matches(job models.Job) bool {
	return job.ID == c.JobID && job.Status == StatusRunning &&
		job.Attempts == c.Attempt && job.LockedBy != nil && *job.LockedBy == c.WorkerID
}

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
	Kinds  []string
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
	if len(filters.Kinds) > 0 {
		query = query.Where("kind in ?", filters.Kinds)
	} else if filters.Kind != "" {
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
	noJobAvailable := false
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
			noJobAvailable = true
		}
		return nil
	})
	if err != nil {
		return models.Job{}, err
	}
	if noJobAvailable {
		return models.Job{}, ErrNoJobAvailable
	}
	return job, nil
}

func (s *Service) Complete(claim Claim) error {
	now := time.Now().UTC()
	result := s.db.Model(&models.Job{}).
		Where("id = ? and status = ? and attempts = ? and locked_by = ?", claim.JobID, StatusRunning, claim.Attempt, claim.WorkerID).
		Updates(map[string]any{
			"status":        StatusCompleted,
			"finished_at":   &now,
			"locked_by":     nil,
			"locked_at":     nil,
			"updated_at":    now,
			"error_message": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrClaimLost
	}
	return nil
}

func (s *Service) Fail(claim Claim, job models.Job, cause error) error {
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

	result := s.db.Model(&models.Job{}).
		Where("id = ? and status = ? and attempts = ? and locked_by = ?", claim.JobID, StatusRunning, claim.Attempt, claim.WorkerID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrClaimLost
	}
	return nil
}

func (s *Service) ProcessNext(ctx context.Context, workerID string) (models.Job, error) {
	job, err := s.ClaimNext(workerID)
	if err != nil {
		return models.Job{}, err
	}
	claim := ClaimFor(job, workerID)

	handler, ok := s.handlers[job.Kind]
	if !ok {
		err := fmt.Errorf("%w: %s", ErrUnknownJobKind, job.Kind)
		if failErr := s.Fail(claim, job, err); failErr != nil {
			return job, failErr
		}
		return job, err
	}

	workCtx, cancel := context.WithCancel(WithClaim(ctx, claim))
	defer cancel()
	claimLost := make(chan struct{})
	go s.refreshLeaseLoop(workCtx, claim, func() {
		close(claimLost)
		cancel()
	})

	if err := handler(workCtx, job); err != nil {
		select {
		case <-claimLost:
			return job, ErrClaimLost
		default:
		}
		if failErr := s.Fail(claim, job, err); failErr != nil {
			return job, failErr
		}
		return job, err
	}

	select {
	case <-claimLost:
		return job, ErrClaimLost
	default:
	}
	if err := s.Complete(claim); err != nil {
		return job, err
	}
	return job, nil
}

func (s *Service) refreshLeaseLoop(ctx context.Context, claim Claim, onLost func()) {
	ticker := time.NewTicker(DefaultLeaseTimeout / 3)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if err := s.Heartbeat(claim, now.UTC()); err != nil {
				if errors.Is(err, ErrClaimLost) {
					onLost()
				} else {
					s.logger.Error("refresh job lease", "job_id", claim.JobID.String(), "worker", claim.WorkerID, "error", err)
				}
				return
			}
		}
	}
}

// Heartbeat refreshes a lease only for the exact claim that acquired it.
func (s *Service) Heartbeat(claim Claim, now time.Time) error {
	result := s.db.Model(&models.Job{}).
		Where("id = ? and status = ? and attempts = ? and locked_by = ?", claim.JobID, StatusRunning, claim.Attempt, claim.WorkerID).
		Updates(map[string]any{"locked_at": now.UTC(), "updated_at": now.UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrClaimLost
	}
	return nil
}

// WithClaim makes a worker claim available to a handler's protected writes.
func WithClaim(ctx context.Context, claim Claim) context.Context {
	return context.WithValue(ctx, claimContextKey{}, claim)
}

// ClaimFromContext returns the claim attached by ProcessNext, if any.
func ClaimFromContext(ctx context.Context) (Claim, bool) {
	claim, ok := ctx.Value(claimContextKey{}).(Claim)
	return claim, ok
}

type claimContextKey struct{}

// ProtectedTransaction locks and checks the current job claim before allowing
// a handler to publish protected data. The caller must lock any source scope
// rows after this job row, keeping the lock order deterministic.
func ProtectedTransaction(ctx context.Context, db *gorm.DB, claim Claim, publish func(*gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job models.Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&job, "id = ?", claim.JobID).Error; err != nil {
			return err
		}
		if !claim.matches(job) {
			return ErrClaimLost
		}
		return publish(tx)
	})
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
