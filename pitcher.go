/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"context"

	"github.com/nitishm/go-rejson/v4"
	"github.com/nitishm/go-rejson/v4/clients"
	"github.com/pterm/pterm"
	sthingsCli "github.com/stuttgart-things/sthingsCli"
)

var (
	logger = pterm.DefaultLogger.WithLevel(pterm.LogLevelTrace)
)

// EnqueueMessageInRedisStreams stores a Message object in Redis JSON and enqueues its ID into a Redis Stream.
//
// It performs two operations:
//  1. Generates a unique object ID (UUID + System) and writes the Message as a Redis JSON object.
//  2. Adds an entry into the configured Redis Stream, linking the stream entry to the JSON object.
//
// Parameters:
//   - msg: The Message struct to store.
//   - redisConnection: A map containing Redis connection details. Expected keys:
//   - "addr"   → Redis host/address
//   - "port"   → Redis port
//   - "password" → Redis password
//   - "stream"   → Redis stream name
//
// Returns:
//   - objectID: The generated Redis JSON object ID
//   - streamID: The name of the Redis stream where the entry was enqueued
//
// Example:
//
//	objectID, streamID := homerun.EnqueueMessageInRedisStreams(
//		homerun.Message{System: "demo", Content: "Hello"},
//		map[string]string{
//			"addr":     "localhost",
//			"port":     "6379",
//			"password": "",
//			"stream":   "messages",
//		},
//	)
func EnqueueMessageInRedisStreams(
	msg Message,
	redisConnection map[string]string) (objectID, streamID string) {

	var redisJSONHandler = rejson.NewReJSONHandler()
	var redisClient = sthingsCli.CreateRedisClient(
		redisConnection["addr"]+":"+redisConnection["port"],
		redisConnection["password"])

	var conn clients.GoRedisClientConn = redisClient

	redisJSONHandler.SetGoRedisClientWithContext(context.Background(), conn)

	// SET TO REDIS JSON
	objectID = GenerateUUID() + "-" + msg.System
	sthingsCli.SetRedisJSON(redisJSONHandler, msg, objectID)

	// SET TO REDIS STREAMS
	streamID = redisConnection["stream"]
	streamValues := map[string]interface{}{
		"messageID": objectID,
	}

	enqueue := sthingsCli.EnqueueDataInRedisStreams(
		redisConnection["addr"]+":"+redisConnection["port"],
		redisConnection["password"],
		streamID,
		streamValues,
	)

	if enqueue {
		logger.Info(
			"MESSAGE WAS ENQUEUED IN REDIS STREAMS",
			logger.Args(streamID, streamValues))
	} else {
		logger.Error(
			"MESSAGE WAS NOT ENQUEUED IN REDIS STREAMS",
			logger.Args(streamID, streamValues))
	}

	return objectID, streamID
}
