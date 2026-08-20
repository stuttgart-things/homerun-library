//go:build integration

/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	rejson "github.com/nitishm/go-rejson/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnqueueMessageInRedisStreamsIntegration talks to a real Redis with the
// RedisJSON module loaded. It is excluded from `go test ./...` and runs only
// under `go test -tags=integration ./...`, which the Dagger pipeline invokes
// with a redis-stack service bound (see .dagger/main.go).
//
// Connection details come from the environment so the same test works against
// the Dagger service binding and against a local `redis-stack` container.
func TestEnqueueMessageInRedisStreamsIntegration(t *testing.T) {
	rc := RedisConfig{
		Addr:     GetEnv("REDIS_ADDR", "localhost"),
		Port:     GetEnv("REDIS_PORT", "6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		Stream:   GetEnv("REDIS_STREAM", "messages"),
	}

	msg := Message{
		Title:     "Test Message",
		Message:   "This is a test message",
		Severity:  "info",
		Author:    "test-user",
		Timestamp: "2025-11-11T06:45:00Z",
		System:    "test-system",
		Tags:      "integration-test",
	}

	t.Run("enqueues into the configured stream", func(t *testing.T) {
		objectID, streamID, err := EnqueueMessageInRedisStreams(msg, rc)

		require.NoError(t, err)
		assert.Equal(t, rc.Stream, streamID)
		assert.NotEmpty(t, objectID)
		assert.Contains(t, objectID, msg.System, "object ID should carry the originating system")
	})

	t.Run("stream override wins over RedisConfig.Stream", func(t *testing.T) {
		override := "integration-override-" + GenerateUUID()

		objectID, streamID, err := EnqueueMessageInRedisStreams(msg, rc, override)

		require.NoError(t, err)
		assert.Equal(t, override, streamID)
		assert.NotEmpty(t, objectID)
	})

	t.Run("stored JSON round-trips through GetMessageJSON", func(t *testing.T) {
		objectID, _, err := EnqueueMessageInRedisStreams(msg, rc)
		require.NoError(t, err)

		handler, cleanup := newTestJSONHandler(t, rc)
		defer cleanup()

		stored, err := GetMessageJSON(objectID, handler)
		require.NoError(t, err)
		assert.Equal(t, msg.Title, stored.Title)
		assert.Equal(t, msg.Message, stored.Message)
		assert.Equal(t, msg.Severity, stored.Severity)
		assert.Equal(t, msg.Timestamp, stored.Timestamp)
	})

	t.Run("unknown object ID is reported as an error", func(t *testing.T) {
		handler, cleanup := newTestJSONHandler(t, rc)
		defer cleanup()

		_, err := GetMessageJSON("does-not-exist-"+GenerateUUID(), handler)
		assert.Error(t, err)
	})
}

// newTestJSONHandler builds a ReJSON handler for the configured Redis and
// returns a cleanup that closes the underlying client. The library itself does
// not expose a constructor for this — see #97.
func newTestJSONHandler(t *testing.T, rc RedisConfig) (*rejson.Handler, func()) {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr:     rc.Addr + ":" + rc.Port,
		Password: rc.Password,
	})

	handler := rejson.NewReJSONHandler()
	handler.SetGoRedisClientWithContext(context.Background(), client)

	return handler, func() {
		if err := client.Close(); err != nil {
			t.Logf("closing test redis client: %v", err)
		}
	}
}

// TestPitcherDoesNotLeakConnections publishes repeatedly through the one-shot
// API and asserts the server does not accumulate a client per message.
//
// Before the lifecycle fix every call created a go-redis client (and the
// stream producer created a second one) and closed neither, so a long-running
// pitcher leaked two pools per published message.
func TestPitcherDoesNotLeakConnections(t *testing.T) {
	rc := RedisConfig{
		Addr:     GetEnv("REDIS_ADDR", "localhost"),
		Port:     GetEnv("REDIS_PORT", "6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		Stream:   "leak-check-" + GenerateUUID(),
	}

	control := redis.NewClient(&redis.Options{
		Addr:     rc.Addr + ":" + rc.Port,
		Password: rc.Password,
	})
	defer func() { _ = control.Close() }()

	const messages = 15

	before := connectedClients(t, control)
	for i := 0; i < messages; i++ {
		if _, _, err := EnqueueMessageInRedisStreams(
			Message{Title: "leak check", System: "test"}, rc,
		); err != nil {
			t.Fatalf("publish %d failed: %v", i, err)
		}
	}
	after := connectedClients(t, control)

	// Redis reaps closed connections asynchronously, so allow some slack -
	// what must not happen is growth proportional to the number of messages.
	if after > before+5 {
		t.Errorf("connected clients grew from %d to %d over %d messages; connections are leaking",
			before, after, messages)
	}
}

// TestPitcherReusesOneConnection publishes through a single Pitcher, which is
// the path callers should use for repeated publishing.
func TestPitcherReusesOneConnection(t *testing.T) {
	rc := RedisConfig{
		Addr:     GetEnv("REDIS_ADDR", "localhost"),
		Port:     GetEnv("REDIS_PORT", "6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		Stream:   "reuse-check-" + GenerateUUID(),
	}

	pitcher := NewPitcher(rc)
	defer func() { _ = pitcher.Close() }()

	for i := 0; i < 5; i++ {
		objectID, streamID, err := pitcher.Enqueue(context.Background(),
			Message{Title: "reuse check", System: "test"})
		require.NoError(t, err)
		assert.Equal(t, rc.Stream, streamID)
		assert.NotEmpty(t, objectID)
	}
}

// TestStoreInRediSearchIntegration covers the path that previously could not
// run at all against a Redis without requirepass: the connection pool sent
// AUTH unconditionally and every dial failed.
func TestStoreInRediSearchIntegration(t *testing.T) {
	rc := RedisConfig{
		Addr:     GetEnv("REDIS_ADDR", "localhost"),
		Port:     GetEnv("REDIS_PORT", "6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		Index:    "homerun-integration-" + strings.ReplaceAll(GenerateUUID(), "-", ""),
	}

	msg := Message{
		Title:    "indexed by the integration suite",
		Message:  "body",
		Severity: "info",
		Author:   "test",
		System:   "test-system",
		Tags:     "integration-test",
	}

	t.Run("creates the index on first use and indexes the document", func(t *testing.T) {
		require.NoError(t, StoreInRediSearch(msg, rc))
	})

	t.Run("reuses the existing index on the second call", func(t *testing.T) {
		require.NoError(t, StoreInRediSearch(msg, rc))
	})
}

// connectedClients reads connected_clients out of INFO clients.
func connectedClients(t *testing.T, client *redis.Client) int {
	t.Helper()

	info, err := client.Info(context.Background(), "clients").Result()
	require.NoError(t, err)

	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "connected_clients:"); ok {
			n, err := strconv.Atoi(strings.TrimSpace(rest))
			require.NoError(t, err)
			return n
		}
	}

	t.Fatalf("connected_clients not found in INFO output:\n%s", info)
	return 0
}

// TestContextIsHonouredOnTheRedisPath checks that the context variants added in
// #99 actually reach Redis. Without a real server a cancelled context and a
// broken connection are indistinguishable, so these belong here rather than in
// the hermetic suite.
func TestContextIsHonouredOnTheRedisPath(t *testing.T) {
	rc := RedisConfig{
		Addr:     GetEnv("REDIS_ADDR", "localhost"),
		Port:     GetEnv("REDIS_PORT", "6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		Stream:   GetEnv("REDIS_STREAM", "messages"),
		Index:    GetEnv("REDIS_INDEX", "homerun-integration"),
	}
	msg := Message{Title: "ctx", Message: "ctx", Severity: "info", System: "test-system"}

	t.Run("EnqueueMessageInRedisStreamsContext succeeds with a live context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		objectID, streamID, err := EnqueueMessageInRedisStreamsContext(ctx, msg, rc)

		require.NoError(t, err)
		assert.Equal(t, rc.Stream, streamID)
		assert.NotEmpty(t, objectID)
	})

	t.Run("EnqueueMessageInRedisStreamsContext fails on a cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already cancelled before the call

		_, _, err := EnqueueMessageInRedisStreamsContext(ctx, msg, rc)

		require.Error(t, err, "a cancelled context must not reach Redis")
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("StoreInRediSearchContext succeeds with a live context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, StoreInRediSearchContext(ctx, msg, rc))
	})

	t.Run("StoreInRediSearchContext fails on a cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := StoreInRediSearchContext(ctx, msg, rc)

		require.Error(t, err, "a cancelled context must not reach Redis")
	})
}

// The redigo pool used by the RediSearch path had no timeouts at all, so a
// Redis that accepts the connection and then stops answering blocked the caller
// forever. This drives that exact case with a listener that accepts and is
// silent, and asserts the call returns rather than hangs.
func TestRediSearchDoesNotHangOnASilentServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	// Accept and then never write a byte.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	rc := RedisConfig{Addr: host, Port: port, Index: "homerun-hang-test"}

	done := make(chan error, 1)
	go func() { done <- StoreInRediSearch(Message{Title: "hang"}, rc) }()

	select {
	case err := <-done:
		require.Error(t, err, "expected a timeout error from the silent server")
	case <-time.After(redisReadTimeout + 10*time.Second):
		t.Fatal("StoreInRediSearch hung on a server that accepts and never answers")
	}
}

// redisSearchTestConfig points at the integration Redis with a unique index so
// each test owns its own schema.
func redisSearchTestConfig(index string) RedisConfig {
	return RedisConfig{
		Addr:     GetEnv("REDIS_ADDR", "localhost"),
		Port:     GetEnv("REDIS_PORT", "6379"),
		Password: GetEnv("REDIS_PASSWORD", ""),
		Stream:   GetEnv("REDIS_STREAM", "messages"),
		Index:    index,
	}
}

func redisTestClient(t *testing.T) *redis.Client {
	t.Helper()
	rc := redisSearchTestConfig("")
	client := redis.NewClient(&redis.Options{
		Addr:     rc.Addr + ":" + rc.Port,
		Password: rc.Password,
	})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// The whole point of making timestamp NUMERIC: "@timestamp:[x +inf]" is not
// expressible against a TEXT field, so this query is the regression test for
// the schema change.
func TestRediSearchTimestampIsRangeQueryable(t *testing.T) {
	ctx := context.Background()
	index := "rangequery-" + GenerateUUID()
	rc := redisSearchTestConfig(index)
	client := redisTestClient(t)
	t.Cleanup(func() { _ = client.Do(ctx, "FT.DROPINDEX", index, "DD").Err() })

	old := Message{Title: "old", Severity: "info", System: "sys", Timestamp: "2020-01-02T03:04:05Z"}
	recent := Message{Title: "recent", Severity: "info", System: "sys",
		Timestamp: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}

	require.NoError(t, StoreInRediSearchContext(ctx, old, rc))
	require.NoError(t, StoreInRediSearchContext(ctx, recent, rc))

	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	res, err := client.Do(ctx, "FT.SEARCH", index,
		"@timestamp:["+strconv.FormatInt(cutoff, 10)+" +inf]",
		"RETURN", "1", "title").Result()
	require.NoError(t, err, "range query over a NUMERIC timestamp must be expressible")

	rendered := fmt.Sprint(res)
	assert.Contains(t, rendered, "recent", "the recent message must fall inside the range")
	assert.NotContains(t, rendered, "old", "the 2020 message must fall outside the range")
}

// The indexed value must be the event time from the Message, not the moment of
// indexing - the defect that made every retry and replay wrong.
func TestRediSearchIndexesTheEventTime(t *testing.T) {
	ctx := context.Background()
	index := "eventtime-" + GenerateUUID()
	rc := redisSearchTestConfig(index)
	client := redisTestClient(t)
	t.Cleanup(func() { _ = client.Do(ctx, "FT.DROPINDEX", index, "DD").Err() })

	const stamp = "2020-01-02T03:04:05Z"
	want := strconv.FormatInt(time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC).Unix(), 10)

	require.NoError(t, StoreInRediSearchContext(ctx,
		Message{Title: "event", Severity: "info", System: "sys", Timestamp: stamp}, rc))

	res, err := client.Do(ctx, "FT.SEARCH", index, "*", "RETURN", "1", "timestamp").Result()
	require.NoError(t, err)
	assert.Contains(t, fmt.Sprint(res), want,
		"the index must carry the event time %s, not the time of indexing", want)
}

// An ON JSON index does not index hashes, so up to v3 StoreInRediSearch wrote
// an orphaned, never-expiring key and returned nil. This is the exact
// configuration homerun2-omni-pitcher creates at startup.
func TestStoreInRediSearchRefusesAnIncompatibleIndex(t *testing.T) {
	ctx := context.Background()
	index := "jsonindex-" + GenerateUUID()
	rc := redisSearchTestConfig(index)
	client := redisTestClient(t)
	t.Cleanup(func() { _ = client.Do(ctx, "FT.DROPINDEX", index, "DD").Err() })

	// Exactly what omni-pitcher's EnsureIndex issues.
	require.NoError(t, client.Do(ctx, "FT.CREATE", index, "ON", "JSON", "SCHEMA",
		"$.severity", "AS", "severity", "TEXT",
		"$.timestamp", "AS", "timestamp", "TEXT").Err())

	// A system name unique to this test: the document key is
	// "<RFC3339Nano>-<system>", and the integration Redis is shared with every
	// other test in the run.
	system := "refused-" + GenerateUUID()

	err := StoreInRediSearchContext(ctx,
		Message{Title: "invisible", Severity: "info", System: system,
			Timestamp: "2020-01-02T03:04:05Z"}, rc)

	require.Error(t, err, "writing hashes into an ON JSON index must not report success")
	assert.Contains(t, err.Error(), "JSON")
	assert.Contains(t, err.Error(), "hashes")

	// And no orphan was left behind: up to v3 this call wrote a hash keyed by
	// "<RFC3339Nano>-<system>" with no TTL, which nothing indexes and nothing
	// prunes, because retention finds documents through FT.SEARCH.
	keys, err := client.Keys(ctx, "*-"+system).Result()
	require.NoError(t, err)
	assert.Empty(t, keys, "the refused write must not leave an unindexed key behind")

	// num_docs is deliberately not asserted on: this index carries no prefix,
	// so it also backfills every Redis JSON document the rest of the suite
	// wrote, asynchronously after FT.CREATE. The key check above is the precise
	// statement of the claim.
}
