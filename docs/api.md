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
    Url             string `json:"url,omitempty"`
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

## Functions

### Constructors

#### `NewMessage`

Creates a new Message with the given author, content, severity, and an auto-generated timestamp.

```go
func NewMessage(author, content, severity string) *Message
```

---

### Messaging

#### `EnqueueMessageInRedisStreams`

Stores a Message as Redis JSON and enqueues its ID into a Redis Stream.

```go
func EnqueueMessageInRedisStreams(
    msg Message,
    rc RedisConfig,
) (objectID, streamID string, err error)
```

**Parameters:**

- `msg` - The Message to store
- `rc` - Redis connection config (uses `Addr`, `Port`, `Password`, `Stream`)

**Returns:** The generated object ID, the stream name, and an error if enqueueing failed.

---

#### `StoreInRediSearch`

Indexes a Message in RediSearch for full-text search capabilities.

```go
func StoreInRediSearch(message Message, rc RedisConfig) error
```

**Parameters:**

- `message` - The Message to index
- `rc` - Redis connection config (uses `Addr`, `Port`, `Password`, `Index`)

**Returns:** An error if the index check fails.

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
) ([]byte, *http.Response, error)
```

**Parameters:**

- `destination` - Target URL
- `token` - Authentication token (set as `X-Auth-Token` header)
- `renderedBody` - JSON body to send
- `insecure` - Skip TLS certificate verification

**Returns:** The response body, the HTTP response, and an error if the request failed.

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

## Variables

#### `HomeRunBodyData`

Default JSON template string for rendering a Message body.

```go
var HomeRunBodyData string
```
