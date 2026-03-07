# homerun-library

Shared Go library module for the **homerun** microservice family by [stuttgart-things](https://github.com/stuttgart-things).

## Overview

homerun-library provides common building blocks used across homerun microservices:

| Module | Description |
|---|---|
| **Message** | Core message struct with `NewMessage` constructor, JSON serialization and Redis JSON retrieval |
| **Pitcher** | Enqueue messages into Redis Streams with Redis JSON storage |
| **Send** | HTTP POST client for sending messages to homerun endpoints |
| **RediSearch** | Full-text search indexing of messages via RediSearch |
| **Print** | Table rendering utilities using go-pretty |
| **Helpers** | UUID generation, random selection, environment variable utilities |

## Quick Start

```bash
go get github.com/stuttgart-things/homerun-library
```

```go
import homerun "github.com/stuttgart-things/homerun-library"
```

See [Usage Examples](usage.md) for detailed code samples.

## Architecture

```
homerun-library
├── message.go      # Message struct, NewMessage, RedisConfig, Redis JSON retrieval
├── pitcher.go      # Redis Streams enqueueing
├── send.go         # HTTP sending, template rendering
├── redisearch.go   # RediSearch indexing
├── helpers.go      # Utility functions
└── print.go        # Table output
```

All functions operate on the central `Message` struct which represents a notification/event in the homerun system. Redis connection details are passed via the `RedisConfig` struct.
