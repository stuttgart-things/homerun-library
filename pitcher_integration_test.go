//go:build integration

/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"context"
	"strconv"
	"strings"
	"testing"

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
