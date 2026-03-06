package main

import (
	homerun "github.com/stuttgart-things/homerun-library" // use module path from go.mod
)

func main() {

	redisAddr := homerun.GetEnv("REDIS_ADDR", "localhost")
	redisPort := homerun.GetEnv("REDIS_PORT", "6379")
	redisPassword := homerun.GetEnv("REDIS_PASSWORD", "")
	redisStream := homerun.GetEnv("REDIS_STREAM", "messages")

	objectID, streamID, err := homerun.EnqueueMessageInRedisStreams(
		homerun.Message{
			Title:           "Deployment Notification",
			Message:         "Service xyz deployed successfully",
			Severity:        "success",
			Author:          "ci-pipeline",
			Timestamp:       "2025-09-14T10:00:00Z", // normally you'd auto-generate this
			System:          "demo-system",
			Tags:            "deployment,production,success",
			AssigneeAddress: "ops-team@example.com",
			AssigneeName:    "Ops Team",
			Artifacts:       "docker://registry.example.com/xyz:1.0.0",
			Url:             "http://example.com/deployment/xyz",
		},
		homerun.RedisConfig{
			Addr:     redisAddr,
			Port:     redisPort,
			Password: redisPassword,
			Stream:   redisStream,
		},
	)

	if err != nil {
		println("Error:", err.Error())
		return
	}
	println("Object ID:", objectID)
	println("Stream ID:", streamID)
}
