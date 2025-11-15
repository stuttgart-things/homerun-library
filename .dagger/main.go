package main

import (
	"context"
	"crypto/rand"
	"dagger/dagger/internal/dagger"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type Dagger struct{}

// TestResult represents the result of a single test
type TestResult struct {
	TestPath string  `json:"test_path"`
	Status   string  `json:"status"` // "passed" or "failed"
	Duration float64 `json:"duration_seconds"`
	Error    string  `json:"error,omitempty"`
}

// TestReport represents the full test report
type TestReport struct {
	TotalTests    int          `json:"total_tests"`
	PassedTests   int          `json:"passed_tests"`
	FailedTests   int          `json:"failed_tests"`
	TotalDuration float64      `json:"total_duration_seconds"`
	Timestamp     string       `json:"timestamp"`
	Results       []TestResult `json:"results"`
}

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

	report := TestReport{
		TotalTests: len(tests),
		Timestamp:  time.Now().Format(time.RFC3339),
		Results:    make([]TestResult, 0, len(tests)),
	}

	allOK := true
	startTime := time.Now()

	for _, t := range tests {
		testStart := time.Now()
		_, err := m.RunTestWithRedis(ctx, source, goVersion, t)
		duration := time.Since(testStart).Seconds()

		result := TestResult{
			TestPath: t,
			Duration: duration,
		}

		if err != nil {
			fmt.Printf("❌ Test failed: %s (%v) [%.2fs]\n", t, err, duration)
			result.Status = "failed"
			result.Error = err.Error()
			report.FailedTests++
			allOK = false
		} else {
			fmt.Printf("✅ Test passed: %s [%.2fs]\n", t, duration)
			result.Status = "passed"
			report.PassedTests++
		}

		report.Results = append(report.Results, result)
	}

	report.TotalDuration = time.Since(startTime).Seconds()

	// Generate and print the report
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Printf("⚠️  Failed to generate report: %v\n", err)
	} else {
		fmt.Printf("\n=== Test Report ===\n%s\n", string(reportJSON))
	}

	return allOK
}

// RunAllTestsWithReport runs all tests and exports the test report as a file
func (m *Dagger) RunAllTestsWithReport(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	// +default="1.25.4"
	goVersion string,
) *dagger.File {
	tests := []string{
		"tests/helpers/pick_random.go",
		"tests/pitcher/pitch_message.go",
		"tests/table/print_demo.go",
	}

	report := TestReport{
		TotalTests: len(tests),
		Timestamp:  time.Now().Format(time.RFC3339),
		Results:    make([]TestResult, 0, len(tests)),
	}

	startTime := time.Now()

	for _, t := range tests {
		testStart := time.Now()
		_, err := m.RunTestWithRedis(ctx, source, goVersion, t)
		duration := time.Since(testStart).Seconds()

		result := TestResult{
			TestPath: t,
			Duration: duration,
		}

		if err != nil {
			fmt.Printf("❌ Test failed: %s (%v) [%.2fs]\n", t, err, duration)
			result.Status = "failed"
			result.Error = err.Error()
			report.FailedTests++
		} else {
			fmt.Printf("✅ Test passed: %s [%.2fs]\n", t, duration)
			result.Status = "passed"
			report.PassedTests++
		}

		report.Results = append(report.Results, result)
	}

	report.TotalDuration = time.Since(startTime).Seconds()

	// Generate JSON report
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Printf("⚠️  Failed to generate report: %v\n", err)
		reportJSON = []byte(fmt.Sprintf(`{"error": "Failed to generate report: %v"}`, err))
	}

	fmt.Printf("\n=== Test Report ===\n%s\n", string(reportJSON))

	// Return the report as a Dagger file
	return dag.Directory().
		WithNewFile("test-report.json", string(reportJSON)).
		File("test-report.json")
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
