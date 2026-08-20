# stuttgart-things/homerun-library

Shared Go library module for the **homerun** microservice family.

[![Dagger Static Checks](https://github.com/stuttgart-things/homerun-library/actions/workflows/dagger-static-checks.yaml/badge.svg)](https://github.com/stuttgart-things/homerun-library/actions/workflows/dagger-static-checks.yaml)
[![Dagger Tests](https://github.com/stuttgart-things/homerun-library/actions/workflows/dagger-tests.yaml/badge.svg)](https://github.com/stuttgart-things/homerun-library/actions/workflows/dagger-tests.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/stuttgart-things/homerun-library/v4.svg)](https://pkg.go.dev/github.com/stuttgart-things/homerun-library/v4)

## Features

| Module | Description |
|---|---|
| **Message** | Core `Message` struct with `NewMessage` constructor, JSON serialization and Redis JSON retrieval |
| **Pitcher** | Enqueue messages into Redis Streams with Redis JSON storage |
| **Send** | HTTP POST client for sending messages to homerun endpoints + template rendering |
| **RediSearch** | Full-text search indexing of messages via RediSearch (deprecated, see migration guide) |
| **Print** | Table rendering utilities (go-pretty) |
| **Helpers** | UUID generation, random selection, environment variable utilities |

## Installation

```bash
go get github.com/stuttgart-things/homerun-library/v4
```

> **Upgrading from v3?** The module path, `Message.Url`, `SendToHomerun`'s return
> type and the RediSearch schema changed. See
> [docs/migration-v4.md](docs/migration-v4.md) — the RediSearch change needs the
> index recreated.

## Usage

### Send a message to homerun

```go
package main

import (
    "fmt"
    "time"
    homerun "github.com/stuttgart-things/homerun-library/v4"
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

    resp, err := homerun.SendToHomerun(
        "https://homerun.example.com/generic", "my-token", []byte(rendered), false,
    )
    if err != nil {
        panic(err)
    }
    if !resp.OK() {
        panic("homerun rejected the message: " + resp.Status)
    }

    fmt.Printf("Status: %s\nBody: %s\n", resp.Status, string(resp.Body))
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

### Publish repeatedly

`EnqueueMessageInRedisStreams` opens a Redis connection and closes it again per
call. A service that publishes continuously should own one connection instead:

```go
pitcher := homerun.NewPitcher(rc)
defer pitcher.Close()

objectID, streamID, err := pitcher.Enqueue(ctx, msg)
```

### Cancellation and timeouts

Every function that talks to the network has a context-taking variant. The
context-free forms are wrappers passing `context.Background()`, so nothing
breaks if you do not need one:

```go
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()

_, err := homerun.SendToHomerunContext(ctx, dest, token, body, false)
_, _, err = homerun.EnqueueMessageInRedisStreamsContext(ctx, msg, rc)
```

Even without a context, HTTP sends are bounded by `homerun.DefaultHTTPTimeout`
(30s) and Redis calls by connection deadlines. To use your own transport,
proxy, instrumentation or retry wrapper:

```go
homerun.SetHTTPClient(&http.Client{Timeout: 5 * time.Second})
```

### Logging

The library is silent by default. Install a logger to see what it is doing:

```go
import "log/slog"

homerun.SetLogger(slog.Default())
```

### Utility functions

```go
id := homerun.GenerateUUID()
item := homerun.GetRandomObject([]string{"a", "b", "c"})
addr := homerun.GetEnv("REDIS_ADDR", "localhost")
```

## Development

### Prerequisites

- Go 1.26.6+
- [Task](https://taskfile.dev/)
- [Dagger](https://dagger.io/)
- Docker (for Redis integration tests)

### Run tests

```bash
# Unit tests, no Redis required
go test ./...

# Integration tests, needs a running Redis (see docs/development.md)
go test -tags=integration ./...

# Full Dagger suite with a JSON report; starts Redis automatically
task test-all
```

### Available tasks

Run `task -l` for the authoritative list.

```bash
task test-all              # Full Dagger test suite, fails on a failing test
task ci-run-static-checks  # Lint + tests via Dagger, fails on a failing test
task govulncheck           # Reachable Go vulnerabilities, fails if any are found
task lint                  # golangci-lint
task check                 # pre-commit hooks
task release               # Semantic release
task tag                   # Commit, push & tag the module
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
