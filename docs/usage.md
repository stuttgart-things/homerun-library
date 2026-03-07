# Usage Examples

## Send a Message via HTTP

```go
package main

import (
    "fmt"
    "time"
    homerun "github.com/stuttgart-things/homerun-library/v2/v2"
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
        "https://homerun.example.com/generic",
        "my-auth-token",
        []byte(rendered),
        false, // verify TLS
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("Status: %s\nBody: %s\n", resp.Status, string(answer))
}
```

## Create a Message with NewMessage

```go
msg := homerun.NewMessage("ci-pipeline", "Build #42 completed", "info")
fmt.Printf("Created at: %s\n", msg.Timestamp)
```

## Enqueue a Message into Redis Streams

```go
package main

import (
    "fmt"
    homerun "github.com/stuttgart-things/homerun-library/v2/v2"
)

func main() {
    objectID, streamID, err := homerun.EnqueueMessageInRedisStreams(
        homerun.Message{
            Title:    "Build Finished",
            Message:  "Build #42 completed successfully",
            Severity: "info",
            Author:   "build-system",
            System:   "ci",
            Tags:     "build,ci",
        },
        homerun.RedisConfig{
            Addr:     "localhost",
            Port:     "6379",
            Password: "",
            Stream:   "notifications",
        },
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("Stored as %s in stream %s\n", objectID, streamID)
}
```

## Index a Message in RediSearch

```go
package main

import homerun "github.com/stuttgart-things/homerun-library/v2/v2"

func main() {
    err := homerun.StoreInRediSearch(
        homerun.Message{
            Title:    "Alert",
            Message:  "Disk usage above 90%",
            Severity: "warning",
            Author:   "monitoring",
            System:   "infra",
            Tags:     "disk,alert",
        },
        homerun.RedisConfig{
            Addr:     "localhost",
            Port:     "6379",
            Password: "",
            Index:    "messages-idx",
        },
    )
    if err != nil {
        panic(err)
    }
}
```

## Print a Table

```go
package main

import (
    "os"
    "github.com/jedib0t/go-pretty/v6/table"
    homerun "github.com/stuttgart-things/homerun-library/v2/v2"
)

func main() {
    homerun.PrintTable(
        os.Stdout,
        table.Row{"Service", "Status", "Version"},
        table.Row{"homerun-api", "running", "v1.2.3"},
        table.StyleLight,
    )
}
```

## Utility Functions

```go
// Generate a UUID
id := homerun.GenerateUUID()

// Pick a random item
item := homerun.GetRandomObject([]string{"alpha", "beta", "gamma"})

// Get env var with fallback
addr := homerun.GetEnv("REDIS_ADDR", "localhost")

// Check if env var is set
if homerun.EnvVarExists("API_TOKEN") {
    // ...
}
```
