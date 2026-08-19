/*
Copyright © 2024 Patrick Hermann
patrick.hermann@sva.de
*/

package homerun

import (
	"context"
	"fmt"

	rejson "github.com/nitishm/go-rejson/v4"
	"github.com/nitishm/go-rejson/v4/clients"
	"github.com/pterm/pterm"
	"github.com/redis/go-redis/v9"
)

var (
	logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelTrace)
)

// streamMaxLen caps the Redis stream length on publish. It matches the value
// the previous redisqueue-based implementation used, so stream trimming
// behaviour is unchanged.
const streamMaxLen = 10000

// Pitcher owns the Redis connection used to publish messages.
//
// Every *redis.Client carries its own connection pool, so a client that is
// created per published message and never closed leaks a pool per message.
// Callers that publish more than once should create one Pitcher, reuse it, and
// Close it when done.
type Pitcher struct {
	rc     RedisConfig
	client *redis.Client
}

// NewPitcher opens a Redis client for the given configuration. The caller owns
// it and must call Close.
func NewPitcher(rc RedisConfig) *Pitcher {
	return &Pitcher{
		rc: rc,
		client: redis.NewClient(&redis.Options{
			Addr:     rc.Addr + ":" + rc.Port,
			Password: rc.Password,
			DB:       0,
		}),
	}
}

// Close releases the underlying Redis connection pool.
func (p *Pitcher) Close() error {
	return p.client.Close()
}

// Enqueue stores msg as a Redis JSON object and enqueues its ID into a Redis
// Stream, returning the generated object ID and the stream it was written to.
//
// The optional variadic streamOverride, if set and non-empty, publishes to that
// stream instead of the configured one. Only the first value is used.
func (p *Pitcher) Enqueue(
	ctx context.Context,
	msg Message,
	streamOverride ...string,
) (objectID, streamID string, err error) {

	// The ReJSON handler binds a context at setup rather than per call, so it
	// is built here to carry the caller's context.
	redisJSONHandler := rejson.NewReJSONHandler()
	var conn clients.GoRedisClientConn = p.client
	redisJSONHandler.SetGoRedisClientWithContext(ctx, conn)

	objectID = GenerateUUID() + "-" + msg.System
	if err = setRedisJSON(redisJSONHandler, msg, objectID); err != nil {
		return objectID, "", err
	}

	streamID = resolveStream(p.rc, streamOverride...)
	streamValues := map[string]interface{}{
		"messageID": objectID,
	}

	if err = p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamID,
		MaxLen: streamMaxLen,
		Values: streamValues,
	}).Err(); err != nil {
		return objectID, streamID, fmt.Errorf("failed to enqueue message in redis stream %s: %w", streamID, err)
	}

	logger.Info(
		"MESSAGE WAS ENQUEUED IN REDIS STREAMS",
		logger.Args(streamID, streamValues),
	)

	return objectID, streamID, nil
}

// setRedisJSON writes obj as a Redis JSON document under key.
func setRedisJSON(handler *rejson.Handler, obj interface{}, key string) error {
	res, err := handler.JSONSet(key, ".", obj)
	if err != nil {
		return fmt.Errorf("failed to set redis JSON for object %s: %w", key, err)
	}

	if status, ok := res.(string); !ok || status != "OK" {
		return fmt.Errorf("failed to set redis JSON for object %s: unexpected response %v", key, res)
	}

	return nil
}

// EnqueueMessageInRedisStreams stores a Message object in Redis JSON and enqueues its ID into a Redis Stream.
//
// It performs two operations:
//  1. Generates a unique object ID (UUID + System) and writes the Message as a Redis JSON object.
//  2. Adds an entry into the configured Redis Stream, linking the stream entry to the JSON object.
//
// This is the one-shot form: it opens a Redis connection, publishes, and closes
// it again. Callers that publish repeatedly should use NewPitcher and reuse the
// Pitcher instead of paying for a new connection pool per message.
//
// Parameters:
//
//   - msg: The Message struct to store. Fields include:
//
//   - Title: A short title of the message
//
//   - Message: The actual message content
//
//   - Severity: Message severity level (e.g. info, warning, error, success)
//
//   - Author: The creator of the message
//
//   - Timestamp: ISO-8601 timestamp (e.g. 2025-09-14T10:00:00Z)
//
//   - System: Originating system (e.g. "demo-system")
//
//   - Tags: Comma-separated list of tags
//
//   - AssigneeAddress: Email or address of the assignee
//
//   - AssigneeName: Name of the assignee
//
//   - Artifacts: Related artifacts (e.g. container image, build artifact)
//
//   - Url: Related URL (e.g. link to deployment dashboard)
//
//   - rc: Redis connection details (Addr, Port, Password, Stream)
//
// Returns:
//   - objectID: The generated Redis JSON object ID
//   - streamID: The name of the Redis stream where the entry was enqueued
//
// Example:
//
//	objectID, streamID, err := homerun.EnqueueMessageInRedisStreams(
//		homerun.Message{
//			Title:    "Deployment Notification",
//			Message:  "Service xyz deployed successfully",
//			Severity: "success",
//			Author:   "ci-pipeline",
//			System:   "demo-system",
//		},
//		homerun.RedisConfig{
//			Addr:     "localhost",
//			Port:     "6379",
//			Password: "",
//			Stream:   "messages",
//		},
//	)
//
// The optional variadic streamOverride, if set and non-empty, publishes to that
// stream instead of rc.Stream. Only the first value is used. This lets a single
// process route messages to different streams without mutating shared
// RedisConfig.
func EnqueueMessageInRedisStreams(
	msg Message,
	rc RedisConfig,
	streamOverride ...string,
) (objectID, streamID string, err error) {

	pitcher := NewPitcher(rc)
	defer func() {
		if closeErr := pitcher.Close(); closeErr != nil {
			logger.Warn("failed to close redis client", logger.Args("error", closeErr))
		}
	}()

	return pitcher.Enqueue(context.Background(), msg, streamOverride...)
}

// resolveStream picks the effective stream name: the first non-empty streamOverride
// wins, otherwise rc.Stream. Extra override entries are ignored (a warning is logged).
func resolveStream(rc RedisConfig, streamOverride ...string) string {
	if len(streamOverride) > 1 {
		logger.Warn(
			"EnqueueMessageInRedisStreams received multiple streamOverride values; using the first",
			logger.Args("count", len(streamOverride)),
		)
	}
	if len(streamOverride) > 0 && streamOverride[0] != "" {
		return streamOverride[0]
	}
	return rc.Stream
}
