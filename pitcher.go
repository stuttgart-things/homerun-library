/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"context"
	"os"

	"github.com/nitishm/go-rejson/v4"
	"github.com/nitishm/go-rejson/v4/clients"
	"github.com/pterm/pterm"
	sthingsCli "github.com/stuttgart-things/sthingsCli"
)

var (
	logger           = pterm.DefaultLogger.WithLevel(pterm.LogLevelTrace)
	redisAddress     = os.Getenv("REDIS_SERVER")
	redisPort        = os.Getenv("REDIS_PORT")
	redisPassword    = os.Getenv("REDIS_PASSWORD")
	redisJSONHandler = rejson.NewReJSONHandler()
	redisStream      = os.Getenv("REDIS_STREAM")
)

func EnqueueMessageInRedisStreams(msg Message, system string) (objectID, streamID string) {
	var redisClient = sthingsCli.CreateRedisClient(redisAddress+":"+redisPort, redisPassword)
	var conn clients.GoRedisClientConn = redisClient

	redisJSONHandler.SetGoRedisClientWithContext(context.Background(), conn)

	// SET TO REDIS JSON
	objectID = GenerateUUID() + "-" + system
	sthingsCli.SetRedisJSON(redisJSONHandler, msg, objectID)

	// SET TO REDIS STREAMS
	streamID = redisStream
	streamValues := map[string]interface{}{
		"messageID": objectID,
	}

	enqueue := sthingsCli.EnqueueDataInRedisStreams(redisAddress+":"+redisPort, redisPassword, streamID, streamValues)

	if enqueue {
		logger.Info("MESSAGE WAS ENQUEUED IN REDIS STREAMS", logger.Args("", streamID, streamValues))
	} else {
		logger.Error("MESSAGE WAS NOT ENQUEUED IN REDIS STREAMS", logger.Args("", streamID, streamValues))
	}

	return objectID, streamID
}
