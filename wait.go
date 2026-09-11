/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultRedisStartupTimeout is how long startup waits for Redis to answer
// when REDIS_STARTUP_TIMEOUT is unset. A freshly installed redis-stack took
// ~70s to answer; a fixed 30s restarted services before Redis was up
// (homerun2-omni-pitcher#180, homerun2-core-catcher#117).
const DefaultRedisStartupTimeout = 120 * time.Second

// redisPingTimeout bounds a single PING while waiting for Redis.
const redisPingTimeout = 5 * time.Second

// Backoff between readiness attempts: 1s, 2s, 4s, 8s, then 16s. Variables
// rather than constants only so tests can shorten them.
var (
	readyInitialBackoff = time.Second
	readyMaxBackoff     = 16 * time.Second
)

// WaitForReady retries probe with exponential backoff (1s, 2s, 4s, 8s, capped
// at 16s) until probe returns nil or ctx is done. Each attempt runs under a
// child context bounded by perAttemptTimeout; a non-positive
// perAttemptTimeout leaves the attempt bounded by ctx alone.
//
// Every failed attempt is logged at warn level, a success after retries at
// info level (see SetLogger). When ctx is done it returns the last probe error
// wrapped with the attempt count. It never exits the process - what a failed
// wait means is the caller's decision.
//
// ctx must eventually be done: with no deadline and a probe that never
// succeeds, WaitForReady waits forever.
//
// Intended use: smooth over short readiness races at startup (a redis-stack
// installed alongside, Cilium identity propagation in a fresh namespace, a
// sidecar still booting) without turning genuine misconfiguration into a
// silent hang.
func WaitForReady(ctx context.Context, probe func(context.Context) error, perAttemptTimeout time.Duration) error {
	backoff := readyInitialBackoff

	for attempt := 1; ; attempt++ {
		err := runProbe(ctx, probe, perAttemptTimeout)
		if err == nil {
			if attempt > 1 {
				log().Info("readiness probe succeeded after retries", "attempts", attempt)
			}
			return nil
		}

		if ctx.Err() != nil {
			return fmt.Errorf("after %d attempts: %w", attempt, err)
		}

		log().Warn("readiness probe failed, retrying",
			"attempt", attempt,
			"error", err,
			"next_sleep", backoff.String(),
		)

		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("after %d attempts: %w", attempt, err)
		}

		backoff = min(backoff*2, readyMaxBackoff)
	}
}

func runProbe(ctx context.Context, probe func(context.Context) error, perAttemptTimeout time.Duration) error {
	if perAttemptTimeout <= 0 {
		return probe(ctx)
	}
	attemptCtx, cancel := context.WithTimeout(ctx, perAttemptTimeout)
	defer cancel()
	return probe(attemptCtx)
}

// WaitForRedis blocks until Redis answers PING or timeout expires. A consumer
// or publisher that dials Redis exactly once at startup crashloops when Redis
// is seconds away from ready; this is the wait to put in front of that.
//
// It is WaitForRedisContext with context.Background().
func WaitForRedis(rc RedisConfig, timeout time.Duration) error {
	return WaitForRedisContext(context.Background(), rc, timeout)
}

// WaitForRedisContext is WaitForRedis that also stops when ctx is done, so a
// service receiving SIGTERM while still waiting for Redis can shut down
// instead of sitting out the rest of the timeout.
//
// A RedisConfig without an address or port fails at once: retrying cannot fix
// it.
func WaitForRedisContext(ctx context.Context, rc RedisConfig, timeout time.Duration) error {
	if err := rc.validateConnection(); err != nil {
		return err
	}

	client := redis.NewClient(&redis.Options{
		Addr:     rc.Addr + ":" + rc.Port,
		Password: rc.Password,
	})
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log().Warn("failed to close redis client", "error", closeErr)
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := WaitForReady(ctx, func(ctx context.Context) error {
		return client.Ping(ctx).Err()
	}, redisPingTimeout); err != nil {
		return fmt.Errorf("redis at %s:%s not ready: %w", rc.Addr, rc.Port, err)
	}
	return nil
}

// LoadRedisStartupTimeout reads REDIS_STARTUP_TIMEOUT, a Go duration such as
// "90s" or "2m". Unset or blank means DefaultRedisStartupTimeout. See
// ParseRedisStartupTimeout for what counts as invalid.
func LoadRedisStartupTimeout() (time.Duration, error) {
	return ParseRedisStartupTimeout(os.Getenv("REDIS_STARTUP_TIMEOUT"))
}

// ParseRedisStartupTimeout is the parser behind LoadRedisStartupTimeout.
//
// An unparsable or non-positive value is an error rather than a fallback to
// the default: a typo here should fail startup loudly, not quietly restore a
// budget nobody chose. A bare number ("120") is not a Go duration and is
// rejected.
func ParseRedisStartupTimeout(v string) (time.Duration, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return DefaultRedisStartupTimeout, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("REDIS_STARTUP_TIMEOUT %q: %w", v, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("REDIS_STARTUP_TIMEOUT %q: must be positive", v)
	}
	return d, nil
}
