/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// These four started life in homerun2-omni-pitcher's internal/pitcher and were
// copied into homerun2-core-catcher; they run against the real 1s/2s backoff.

func TestWaitForReady_FirstTry(t *testing.T) {
	var calls atomic.Int32
	probe := func(ctx context.Context) error {
		calls.Add(1)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := WaitForReady(ctx, probe, 100*time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("probe calls = %d, want 1 (happy path must not retry)", got)
	}
}

func TestWaitForReady_SucceedsAfterRetries(t *testing.T) {
	var calls atomic.Int32
	probe := func(ctx context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			return errors.New("not ready yet")
		}
		return nil
	}

	// WaitForReady's first backoff is 1s. A budget of 5s is enough for two
	// retry sleeps (1s + 2s) plus a hair of slack.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	if err := WaitForReady(ctx, probe, 100*time.Millisecond); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("probe calls = %d, want 3", got)
	}
	// First sleep 1s + second sleep 2s = 3s of backoff before the third probe.
	if elapsed := time.Since(start); elapsed < 2900*time.Millisecond {
		t.Errorf("elapsed = %v, want at least 2.9s (skipped backoff?)", elapsed)
	}
}

func TestWaitForReady_BudgetExhausted(t *testing.T) {
	var calls atomic.Int32
	probeErr := errors.New("redis unreachable")
	probe := func(ctx context.Context) error {
		calls.Add(1)
		return probeErr
	}

	// 50ms budget, 10ms per attempt - guarantees we hit the deadline quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := WaitForReady(ctx, probe, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when budget exhausted, got nil")
	}
	if !errors.Is(err, probeErr) {
		t.Errorf("error %q does not wrap last probe error", err.Error())
	}
	if !strings.HasPrefix(err.Error(), "after 1 attempts:") {
		t.Errorf("error %q missing attempt-count prefix", err.Error())
	}
	if calls.Load() < 1 {
		t.Error("probe never called")
	}
}

func TestWaitForReady_StopsImmediatelyOnContextCancel(t *testing.T) {
	probe := func(ctx context.Context) error {
		return errors.New("boom")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	start := time.Now()
	err := WaitForReady(ctx, probe, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when ctx already canceled")
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Errorf("elapsed = %v, want <200ms (did the helper sleep through cancel?)", elapsed)
	}
}

// shortBackoff scales the backoff down so the schedule itself can be tested
// without waiting for it.
func shortBackoff(t *testing.T) {
	t.Helper()
	initial, maxBackoff := readyInitialBackoff, readyMaxBackoff
	readyInitialBackoff, readyMaxBackoff = time.Millisecond, 4*time.Millisecond
	t.Cleanup(func() { readyInitialBackoff, readyMaxBackoff = initial, maxBackoff })
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	SetLogger(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { SetLogger(nil) })
	return &buf
}

func TestWaitForReady_BackoffDoublesUpToCap(t *testing.T) {
	shortBackoff(t)
	buf := captureLog(t)

	var calls atomic.Int32
	probe := func(ctx context.Context) error {
		if calls.Add(1) < 6 {
			return errors.New("not ready yet")
		}
		return nil
	}

	if err := WaitForReady(context.Background(), probe, time.Second); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}

	var sleeps []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if _, after, ok := strings.Cut(line, "next_sleep="); ok {
			sleeps = append(sleeps, after)
		}
	}
	want := []string{"1ms", "2ms", "4ms", "4ms", "4ms"}
	if strings.Join(sleeps, ",") != strings.Join(want, ",") {
		t.Errorf("sleeps = %v, want %v", sleeps, want)
	}
	if !strings.Contains(buf.String(), "readiness probe succeeded after retries") ||
		!strings.Contains(buf.String(), "attempts=6") {
		t.Errorf("late success was not logged with the attempt count:\n%s", buf.String())
	}
}

func TestWaitForReady_PerAttemptTimeoutBoundsEachProbe(t *testing.T) {
	shortBackoff(t)

	var calls atomic.Int32
	probe := func(ctx context.Context) error {
		if calls.Add(1) == 1 {
			// A hanging first attempt must be cut off, not wait out the budget.
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	if err := WaitForReady(ctx, probe, 20*time.Millisecond); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("elapsed = %v, the first attempt was not bounded by its timeout", elapsed)
	}
}

func TestWaitForReady_NonPositivePerAttemptTimeout(t *testing.T) {
	var sawDeadline atomic.Bool
	probe := func(ctx context.Context) error {
		_, ok := ctx.Deadline()
		sawDeadline.Store(ok)
		return nil
	}

	// context.WithTimeout(ctx, 0) is already expired; a zero per-attempt
	// timeout must not turn every probe into a guaranteed failure.
	if err := WaitForReady(context.Background(), probe, 0); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if sawDeadline.Load() {
		t.Error("probe got a deadline although no timeout was set")
	}
}

func TestWaitForRedis_GivesUpAtTimeout(t *testing.T) {
	// Nothing listens on port 1; the dial is refused at once, so this needs
	// no Redis and only checks that the budget bounds the wait.
	rc := RedisConfig{Addr: "127.0.0.1", Port: "1"}
	start := time.Now()
	err := WaitForRedis(rc, 1500*time.Millisecond)
	if err == nil {
		t.Fatal("expected an error with no Redis listening")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Errorf("error %q does not name the address", err.Error())
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("WaitForRedis took %v, want it bounded by the 1.5s budget", elapsed)
	}
}

func TestWaitForRedis_MissingAddressFailsAtOnce(t *testing.T) {
	cases := []struct {
		name string
		rc   RedisConfig
	}{
		{"no address", RedisConfig{Port: "6379"}},
		{"no port", RedisConfig{Addr: "localhost"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			if err := WaitForRedis(tc.rc, time.Minute); err == nil {
				t.Fatal("expected an error for an incomplete RedisConfig")
			}
			if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
				t.Errorf("elapsed = %v, a configuration error must not be retried", elapsed)
			}
		})
	}
}

func TestWaitForRedisContext_StopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	start := time.Now()
	err := WaitForRedisContext(ctx, RedisConfig{Addr: "127.0.0.1", Port: "1"}, time.Minute)
	if err == nil {
		t.Fatal("expected an error with no Redis listening")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("elapsed = %v, want the cancel to end the one-minute wait", elapsed)
	}
}

func TestParseRedisStartupTimeout(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"unset uses default", "", DefaultRedisStartupTimeout, false},
		{"whitespace uses default", "  ", DefaultRedisStartupTimeout, false},
		{"seconds", "90s", 90 * time.Second, false},
		{"minutes", "5m", 5 * time.Minute, false},
		{"surrounding whitespace", " 2m ", 2 * time.Minute, false},
		{"bare number is not a duration", "120", 0, true},
		{"garbage", "soon", 0, true},
		{"zero", "0s", 0, true},
		{"negative", "-10s", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseRedisStartupTimeout(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseRedisStartupTimeout(%q) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ParseRedisStartupTimeout(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestLoadRedisStartupTimeout(t *testing.T) {
	t.Setenv("REDIS_STARTUP_TIMEOUT", "45s")
	got, err := LoadRedisStartupTimeout()
	if err != nil || got != 45*time.Second {
		t.Errorf("LoadRedisStartupTimeout() = %v, %v, want 45s, nil", got, err)
	}

	t.Setenv("REDIS_STARTUP_TIMEOUT", "120")
	if _, err := LoadRedisStartupTimeout(); err == nil {
		t.Error("LoadRedisStartupTimeout() accepted a bare number")
	}
}
