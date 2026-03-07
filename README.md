# stuttgart-things/homerun-library

Shared Go library module for the **homerun** microservice family.

[![Dagger Static Checks](https://github.com/stuttgart-things/homerun-library/actions/workflows/dagger-static-checks.yaml/badge.svg)](https://github.com/stuttgart-things/homerun-library/actions/workflows/dagger-static-checks.yaml)
[![Dagger Tests](https://github.com/stuttgart-things/homerun-library/actions/workflows/dagger-tests.yaml/badge.svg)](https://github.com/stuttgart-things/homerun-library/actions/workflows/dagger-tests.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/stuttgart-things/homerun-library.svg)](https://pkg.go.dev/github.com/stuttgart-things/homerun-library)
[![Documentation](https://img.shields.io/badge/docs-MkDocs-blue)](https://stuttgart-things.github.io/homerun-library/)

## Features

| Module | Description |
|---|---|
| **Message** | Core `Message` struct with `NewMessage` constructor, JSON serialization and Redis JSON retrieval |
| **Pitcher** | Enqueue messages into Redis Streams with Redis JSON storage |
| **Send** | HTTP POST client for sending messages to homerun endpoints + template rendering |
| **RediSearch** | Full-text search indexing of messages via RediSearch |
| **Print** | Table rendering utilities (go-pretty) |
| **Helpers** | UUID generation, random selection, environment variable utilities |

## Installation

```bash
go get github.com/stuttgart-things/homerun-library
```

## Usage

### Send a message to homerun

```go
package main

import (
    "fmt"
    "time"
    homerun "github.com/stuttgart-things/homerun-library"
)

func main() {
    msg := homerun.Message{
        Title:     "Deployment Complete",
        Message:   "Service xyz deployed to production",
        Severity:  "success",
        Author:    "ci-pipeline",
        Timestamp: time.Now().Format(time.RFC3339),
        System:    "production",
        Tags:      "deployment,production",
    }

    rendered, err := homerun.RenderBody(homerun.HomeRunBodyData, msg)
    if err != nil {
        panic(err)
    }

    answer, resp, err := homerun.SendToHomerun(
        "https://homerun.example.com/generic", "my-token", []byte(rendered), false,
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("Status: %s\nBody: %s\n", resp.Status, string(answer))
}
```

### Enqueue into Redis Streams

```go
objectID, streamID, err := homerun.EnqueueMessageInRedisStreams(
    homerun.Message{
        Title:   "Build Finished",
        Message: "Build #42 completed",
        System:  "ci",
    },
    homerun.RedisConfig{
        Addr:     "localhost",
        Port:     "6379",
        Password: "",
        Stream:   "notifications",
    },
)
```

### Utility functions

```go
id := homerun.GenerateUUID()
item := homerun.GetRandomObject([]string{"a", "b", "c"})
addr := homerun.GetEnv("REDIS_ADDR", "localhost")
```

## Development

### Prerequisites

- Go 1.26.0+
- [Task](https://taskfile.dev/)
- [Dagger](https://dagger.io/)
- Docker (for Redis integration tests)

### Run tests

```bash
# Unit tests
go test ./...

# Integration tests via Dagger (starts Redis automatically)
task run-go-tests

# All tests with JSON report
task test-all
```

### Available tasks

```bash
task run-lint-stage    # Lint via Dagger
task run-go-tests      # Integration tests with Redis
task test-all          # All tests with report
task ci                # Full CI (validation + lint)
task release           # Semantic release
```

## Documentation

Full documentation is available via [MkDocs](https://stuttgart-things.github.io/homerun-library/):

```bash
pip install mkdocs-material
mkdocs serve
```

## Authors

```
Patrick Hermann, stuttgart-things 10/2024
Sina Schlatter, stuttgart-things 12/2024
```

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
