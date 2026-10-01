package jobs

import (
	"context"
	"testing"
	"time"
)

func TestSleepCtxReturnsFalseOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if sleepCtx(ctx, 5*time.Second) {
		t.Fatal("expected sleepCtx to return false when context is canceled")
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("sleepCtx took too long to return after cancel: %v", elapsed)
	}
}

func TestSleepCtxReturnsTrueWhenTimerExpires(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	if !sleepCtx(ctx, 10*time.Millisecond) {
		t.Fatal("expected sleepCtx to return true when timer expires")
	}
	if elapsed := time.Since(start); elapsed < 10*time.Millisecond {
		t.Fatalf("sleepCtx returned too early: %v", elapsed)
	}
}
