package jobs

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	minClaimBackoff = 100 * time.Millisecond
	maxClaimBackoff = 15 * time.Second
)

// RunConfig configures the worker pool driven by Run.
type RunConfig struct {
	WorkerID     string
	Concurrency  int
	PollInterval time.Duration
}

// Run drains the queue with Concurrency workers until ctx is canceled. Each
// worker loops ProcessNext back-to-back and sleeps PollInterval ONLY when the
// queue is empty (drain-then-idle) — a burst of N jobs no longer costs N*poll.
// SKIP LOCKED in ClaimNext keeps parallel workers from colliding. A separate
// goroutine promotes due schedules into the queue each PollInterval.
func (s *Service) Run(ctx context.Context, cfg RunConfig) {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.scheduleLoop(ctx, cfg.PollInterval)
	}()
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.workerLoop(ctx, fmt.Sprintf("%s#%d", cfg.WorkerID, n), cfg.PollInterval)
		}(i)
	}
	wg.Wait()
}

func (s *Service) workerLoop(ctx context.Context, workerID string, pollInterval time.Duration) {
	backoff := minClaimBackoff
	for {
		if ctx.Err() != nil {
			return
		}
		job, err := s.ProcessNext(ctx, workerID)
		switch {
		case errors.Is(err, ErrNoJobAvailable):
			backoff = minClaimBackoff
			if !sleepCtx(ctx, pollInterval) {
				return
			}
		case err != nil:
			if job.ID == uuid.Nil {
				jitter := time.Duration(rand.Int63n(int64(backoff/2) + 1))
				sleepDur := backoff + jitter
				s.logger.Error("claim job failed, backing off", "worker", workerID, "backoff", sleepDur, "error", err)
				if !sleepCtx(ctx, sleepDur) {
					return
				}
				if backoff < maxClaimBackoff {
					backoff = min(backoff*2, maxClaimBackoff)
				}
			} else {
				backoff = minClaimBackoff
				// Fail already recorded the outcome; move straight to the next job.
				s.logger.Error("process job", "worker", workerID, "job_id", job.ID.String(), "kind", job.Kind, "error", err)
			}
		default:
			backoff = minClaimBackoff
			s.logger.Info("processed job", "worker", workerID, "job_id", job.ID.String(), "kind", job.Kind)
		}
	}
}

func (s *Service) scheduleLoop(ctx context.Context, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := s.EnqueueDueSchedules(DefaultLimit)
			if err != nil {
				s.logger.Error("enqueue due schedules", "error", err)
			} else if n > 0 {
				s.logger.Info("enqueued due schedules", "count", n)
			}
		}
	}
}

// sleepCtx sleeps for d or until ctx is canceled; returns false if canceled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
