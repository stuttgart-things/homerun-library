package main

import (
	"context"
	"crypto/rand"
	"dagger/dagger/internal/dagger"
	"encoding/base64"
	"fmt"
)

type Dagger struct{}

func (m *Dagger) RunAllTests(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	// +default="1.25.4"
	goVersion string,
) bool {
	tests := []string{
		"tests/helpers/pick_random.go",
		"tests/pitcher/pitch_message.go",
		"tests/table/print_demo.go",
	}

	allOK := true

	for _, t := range tests {
		_, err := m.RunTestWithRedis(ctx, source, goVersion, t)
		if err != nil {
			fmt.Printf("❌ Test failed: %s (%v)\n", t, err)
			allOK = false
		} else {
			fmt.Printf("✅ Test passed: %s\n", t)
		}
	}

	return allOK
}

// helper: generate a random password
func randomPassword(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (m *Dagger) RunTestWithRedis(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	// +default="1.25.4"
	goVersion string,
	testPath string,
) (string, error) {
	// generate random redis password
	generatedRedisPassword, err := randomPassword(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate redis password: %w", err)
	}

	// START REDIS SERVICE IN BACKGROUND
	redis := dag.Container().
		From("redis/redis-stack-server:7.2.0-v18").
		WithEnvVariable("REDIS_ARGS", "--requirepass "+generatedRedisPassword).
		WithExposedPort(6379).
		AsService()

	// RUN TEST CONTAINER
	return dag.Container().
		From("golang:"+goVersion+"-alpine").
		WithMountedDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("gomod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("gobuild")).
		WithServiceBinding("redis", redis).
		WithEnvVariable("REDIS_ADDR", "redis").
		WithEnvVariable("REDIS_PORT", "6379").
		WithEnvVariable("REDIS_STREAM", "messages").
		WithEnvVariable("REDIS_PASSWORD", generatedRedisPassword).
		WithExec([]string{"go", "mod", "download"}).
		WithExec([]string{"go", "run", testPath}).
		Stdout(ctx)
}
