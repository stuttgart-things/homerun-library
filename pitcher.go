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

func EnqueueMessageInRedisStreams(msg Message, redisConnection map[string]string) (objectID, streamID string) {
	var redisJSONHandler = rejson.NewReJSONHandler()
	var redisClient = sthingsCli.CreateRedisClient(redisConnection["addr"]+":"+redisConnection["port"], redisConnection["password"])
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

	enqueue := sthingsCli.EnqueueDataInRedisStreams(redisConnection["addr"]+":"+redisConnection["port"], redisConnection["password"], streamID, streamValues)

	if enqueue {
		logger.Info("MESSAGE WAS ENQUEUED IN REDIS STREAMS", logger.Args("", streamID, streamValues))
	} else {
		logger.Error("MESSAGE WAS NOT ENQUEUED IN REDIS STREAMS", logger.Args("", streamID, streamValues))
	}

	return objectID, streamID
}
