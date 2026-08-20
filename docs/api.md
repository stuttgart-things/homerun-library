# API Reference

## Types

### Message

The central data type representing a homerun notification/event.

```go
type Message struct {
    Title           string `json:"title,omitempty"`
    Message         string `json:"message,omitempty"`
    Severity        string `json:"severity,omitempty"`
    Author          string `json:"author,omitempty"`
    Timestamp       string `json:"timestamp,omitempty"`
    System          string `json:"system,omitempty"`
    Tags            string `json:"tags,omitempty"`
    AssigneeAddress string `json:"assigneeaddress,omitempty"`
    AssigneeName    string `json:"assigneename,omitempty"`
    Artifacts       string `json:"artifacts,omitempty"`
    URL             string `json:"url,omitempty"`
}
```

---

### RedisConfig

Holds Redis connection details used by pitcher and redisearch functions.

```go
type RedisConfig struct {
    Addr     string // Redis host address
    Port     string // Redis port
    Password string // Redis password
    Stream   string // Redis stream name (used by EnqueueMessageInRedisStreams)
    Index    string // RediSearch index name (used by StoreInRediSearch)
}
```

### Pitcher

Owns the Redis connection used for publishing. Every `*redis.Client` carries its
own connection pool, so callers that publish more than once should create one
`Pitcher`, reuse it, and `Close` it when done.

```go
type Pitcher struct { /* ... */ }
```

## Functions

### Constructors

#### `NewMessage`

Creates a new Message with the given author, content, severity, and an auto-generated timestamp.

```go
func NewMessage(author, content, severity string) *Message
```

---

### Messaging

#### `NewPitcher` / `Pitcher.Enqueue` / `Pitcher.Close`

The reusable form. Opens one Redis connection, publishes through it, and hands
the lifetime to the caller.

```go
func NewPitcher(rc RedisConfig) *Pitcher
func (p *Pitcher) Enqueue(
    ctx context.Context,
    msg Message,
    streamOverride ...string,
) (objectID, streamID string, err error)
func (p *Pitcher) Close() error
```

```go
pitcher := homerun.NewPitcher(rc)
defer pitcher.Close()

for msg := range messages {
    objectID, streamID, err := pitcher.Enqueue(ctx, msg)
    // ...
}
```

`Enqueue` takes a `context.Context`, so a publish can be cancelled or bounded by
the caller. The optional variadic `streamOverride` publishes to a different
stream than `rc.Stream`; only the first non-empty value is used.

---

#### `EnqueueMessageInRedisStreams`

Stores a Message as Redis JSON and enqueues its ID into a Redis Stream.

```go
func EnqueueMessageInRedisStreams(
    msg Message,
    rc RedisConfig,
    streamOverride ...string,
) (objectID, streamID string, err error)
```

**Parameters:**

- `msg` - The Message to store
- `rc` - Redis connection config (uses `Addr`, `Port`, `Password`, `Stream`)
- `streamOverride` - Optional stream name overriding `rc.Stream`

**Returns:** The generated object ID, the stream name, and an error if enqueueing failed.

This is the one-shot form: it opens a connection, publishes, and closes it
again. For repeated publishing use `NewPitcher` instead of paying for a new
connection pool per message.

---

#### `EnqueueMessageInRedisStreamsContext`

`EnqueueMessageInRedisStreams` bounded by a context, so the publish can be
cancelled, given a deadline, and carry a request-scoped trace.

```go
func EnqueueMessageInRedisStreamsContext(
    ctx context.Context,
    msg Message,
    rc RedisConfig,
    streamOverride ...string,
) (objectID, streamID string, err error)
```

The context-free form is exactly this with `context.Background()`.

---

#### `StoreInRediSearch`

Indexes a Message in RediSearch for full-text search capabilities.

```go
func StoreInRediSearch(message Message, rc RedisConfig) error
func StoreInRediSearchContext(ctx context.Context, message Message, rc RedisConfig) error
```

!!! warning "Deprecated in v4, removal in v5"
    `StoreInRediSearch` maintains a second, hash-based copy of every message.
    If you index the Redis JSON documents that `Enqueue` already writes — the
    `ON JSON` case — you do not need it: those documents are indexed
    automatically, with the correct event timestamp. See
    [Migration v3 → v4](migration-v4.md).

`ctx` bounds establishing the connection. `redisearch-go` exposes no
context-aware command API, so the individual commands are bounded by read and
write deadlines on the connection pool instead (10s each, dial 5s). Before
v3.2.0 the pool had **no timeouts at all**, so a Redis that accepted the
connection and then stopped answering blocked the caller forever.

**Schema.** `timestamp` is indexed as `NUMERIC` (so `@timestamp:[x +inf]` range
queries work) and carries `Message.Timestamp`, the time the event happened. Up
to v3 it was `TEXT` and held the moment of *indexing*. A missing or unparseable
`Message.Timestamp` falls back to the current time and logs a warning.

**Index compatibility.** The function writes hashes. If the index exists and is
defined `ON JSON`, it returns an error rather than writing a document that would
be stored but never indexed — which is what v3 did silently.

**Changing the schema requires the index to be recreated** (`FT.DROPINDEX`); the
library only checks whether the index exists.

**Parameters:**

- `message` - The Message to index
- `rc` - Redis connection config (uses `Addr`, `Port`, `Password`, `Index`)

**Returns:** An error if the index is unset, unreachable, cannot be created, or
the document cannot be indexed. Every failure is returned - none of them
terminates the calling process.

---

#### `GetMessageJSON`

Retrieves a Message from Redis JSON by its ID.

```go
func GetMessageJSON(
    redisJSONid string,
    redisJSONHandler *rejson.Handler,
) (jsonMessage Message, err error)
```

**Returns:** The deserialized Message and an error if the object was not found or unmarshalling failed.

---

### HTTP Sending

#### `SendToHomerun`

Sends a rendered message body to a homerun endpoint via HTTP POST.

```go
func SendToHomerun(
    destination, token string,
    renderedBody []byte,
    insecure bool,
) (Response, error)
```

**Parameters:**

- `destination` - Target URL
- `token` - Authentication token (set as `X-Auth-Token` header)
- `renderedBody` - JSON body to send
- `insecure` - Skip TLS certificate verification

**Returns:** A `Response` and an error if the request could not be made or read.

```go
type Response struct {
    StatusCode int         // e.g. 200
    Status     string      // e.g. "200 OK"
    Header     http.Header
    Body       []byte      // fully read
}

func (r Response) OK() bool  // 2xx
```

A non-2xx answer is **not** an error — `err` reports whether the request could be
made and read at all. Check `resp.OK()` for the endpoint's verdict.

Up to v3 this returned the raw `*http.Response` after already reading and
closing its body. See [Migration v3 → v4](migration-v4.md).

Bounded by `DefaultHTTPTimeout` (30s) even without a context.

---

#### `SendToHomerunContext`

`SendToHomerun` bounded by a context.

```go
func SendToHomerunContext(
    ctx context.Context,
    destination, token string,
    renderedBody []byte,
    insecure bool,
) (Response, error)
```

Whichever expires first — the context deadline or `DefaultHTTPTimeout` — ends
the call.

---

#### `SetHTTPClient`

Installs the HTTP client both send functions use, for callers that need their
own transport, timeout, proxy, instrumentation or retry wrapper.

```go
func SetHTTPClient(c *http.Client)
```

Passing `nil` restores the built-in clients. A custom client is used for both
secure and insecure calls, so the `insecure` argument becomes the caller's
responsibility. Safe to call from any goroutine.

**Connection reuse.** By default the library keeps one client — and therefore
one connection pool — per TLS configuration. Up to and including v3.1.9 it built
a fresh `http.Transport` per call, so every send paid a new TCP and TLS
handshake and no connection was ever reused. Ten sends now share one connection;
they used to open ten.

---

#### `RenderBody`

Renders a Go template string with the given data object.

```go
func RenderBody(templateData string, object interface{}) (string, error)
```

**Returns:** The rendered string and an error if template parsing or execution failed.

---

### Helpers

#### `GenerateUUID`

Returns a new random UUID v4 string.

```go
func GenerateUUID() string
```

---

#### `GetRandomObject`

Returns a random element from a string slice.

```go
func GetRandomObject(input []string) string
```

---

#### `EnvVarExists`

Returns true if the environment variable exists and is non-empty.

```go
func EnvVarExists(varName string) bool
```

---

#### `GetEnv`

Returns the environment variable value or a fallback default.

```go
func GetEnv(key, fallback string) string
```

---

### Output

#### `PrintTable`

Renders a formatted table to the given writer.

```go
func PrintTable(output io.Writer, header, row table.Row, style table.Style)
```

### Logging

#### `SetLogger`

Installs the logger this package writes to. The library is **silent by default**:
until a logger is installed, every record is discarded.

```go
func SetLogger(l *slog.Logger)
```

Passing `nil` restores the default (discard). Safe to call at any time and from
any goroutine.

```go
homerun.SetLogger(slog.Default())
```

Up to and including v3.1.8 the package installed a `pterm` logger at trace level at import
time, so every consumer got ANSI-coloured output on stdout with no way to
silence, redirect or reformat it. Services that emit structured logs or draw a
TUI should now install their own logger, or leave it unset to get nothing.

The records the library emits:

| Level | Message | Attributes |
|---|---|---|
| `Info` | `message enqueued in redis stream` | `stream`, `messageID` |
| `Info` | `redisearch index created` | `index` |
| `Info` | `document indexed in redisearch` | `index`, `documentID` |
| `Warn` | `failed to close redis client` | `error` |
| `Warn` | `failed to close redisearch connection pool` | `error` |
| `Warn` | `received multiple streamOverride values, using the first` | `count` |

## Variables

#### `HomeRunBodyData`

Default JSON template string for rendering a Message body.

```go
var HomeRunBodyData string
```
