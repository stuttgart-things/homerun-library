/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"encoding/json"
	"fmt"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	rejson "github.com/nitishm/go-rejson/v4"
)

// RedisConfig holds Redis connection details used by pitcher and redisearch functions.
type RedisConfig struct {
	Addr     string // Redis host address
	Port     string // Redis port
	Password string // Redis password
	Stream   string // Redis stream name (used by EnqueueMessageInRedisStreams)
	Index    string // RediSearch index name (used by StoreInRediSearch)
}

// validateConnection reports whether the fields needed to reach Redis are set.
// Without this an empty Addr/Port dials ":" and surfaces as a generic
// "dial tcp: missing address" far from the actual configuration mistake.
func (rc RedisConfig) validateConnection() error {
	if rc.Addr == "" {
		return fmt.Errorf("no redis address configured")
	}
	if rc.Port == "" {
		return fmt.Errorf("no redis port configured")
	}
	return nil
}

// Message is a homerun notification or event.
//
// The string fields are the display side: what humans and the outputs (Teams,
// LED, sound) show. Payload is the machine side: the structured source object
// a pitcher carries along, for consumers that archive it or react to its
// fields instead of parsing display text.
type Message struct {
	Title           string `json:"title,omitempty"`           // if empty: info
	Message         string `json:"message,omitempty"`         // if empty: title
	Severity        string `json:"severity,omitempty"`        // default: info
	Author          string `json:"author,omitempty"`          // default: unknown
	Timestamp       string `json:"timestamp,omitempty"`       // generate timestamp func
	System          string `json:"system,omitempty"`          // default: unknown
	Tags            string `json:"tags,omitempty"`            // empty
	AssigneeAddress string `json:"assigneeaddress,omitempty"` // empty
	AssigneeName    string `json:"assigneename,omitempty"`    // empty
	Artifacts       string `json:"artifacts,omitempty"`       // empty
	URL             string `json:"url,omitempty"`             // empty

	// Payload is the structured source object as JSON, e.g. a Kubernetes
	// object or a webhook body. Enqueue stores it as a nested JSON value in
	// the Redis JSON document, not as a string. Optional; at most
	// MaxPayloadSize bytes. Set it with SetPayload.
	Payload RawJSON `json:"payload,omitempty"`
}

// RawJSON is an encoded JSON value that is marshalled as is, like
// json.RawMessage. It is a string rather than a byte slice so Message stays
// comparable with ==, which the services that use the library rely on.
type RawJSON string

// MarshalJSON returns r unchanged; an empty r is null.
func (r RawJSON) MarshalJSON() ([]byte, error) {
	if r == "" {
		return []byte("null"), nil
	}
	return []byte(r), nil
}

// UnmarshalJSON stores the encoded value as is; null is empty.
func (r *RawJSON) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*r = ""
		return nil
	}
	*r = RawJSON(b)
	return nil
}

// MaxPayloadSize is the largest Payload Enqueue and SetPayload accept, in
// bytes. Every message is kept as a Redis JSON document until retention prunes
// it, so a payload is meant to be one object, not a dump. Strip noise such as
// metadata.managedFields of Kubernetes objects before setting it.
const MaxPayloadSize = 256 * 1024

// SetPayload marshals v into m.Payload. A nil v clears it. It fails if v
// cannot be marshalled or the result exceeds MaxPayloadSize.
func (m *Message) SetPayload(v any) error {
	if v == nil {
		m.Payload = ""
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}
	if err := checkPayload(raw); err != nil {
		return err
	}
	m.Payload = RawJSON(raw)
	return nil
}

// checkPayload reports whether raw is usable as a Payload: empty, or valid
// JSON of at most MaxPayloadSize bytes.
func checkPayload[T ~string | ~[]byte](raw T) error {
	if len(raw) == 0 {
		return nil
	}
	if len(raw) > MaxPayloadSize {
		return fmt.Errorf("payload is %d bytes, more than MaxPayloadSize (%d)", len(raw), MaxPayloadSize)
	}
	if !json.Valid([]byte(raw)) {
		return fmt.Errorf("payload is not valid JSON")
	}
	return nil
}

// NewMessage creates a new Message with the given author, content, severity, and an auto-generated timestamp.
func NewMessage(author, content, severity string) *Message {
	return &Message{
		Author:    author,
		Message:   content,
		Timestamp: time.Now().Format(time.RFC3339),
		Severity:  severity,
	}
}

// GetMessageJSON reads the Redis JSON document stored under redisJSONid and
// decodes it into a Message.
func GetMessageJSON(
	redisJSONid string,
	redisJSONHandler *rejson.Handler) (jsonMessage Message, err error) {

	// GET JSON AS MESSAGE OBJECT
	raw, err := redisJSONHandler.JSONGet(redisJSONid, ".")
	if err != nil {
		return jsonMessage, fmt.Errorf("redis JSON object not found: %s: %w", redisJSONid, err)
	}

	obj, err := redigo.Bytes(raw, nil)
	if err != nil {
		return jsonMessage, fmt.Errorf("unexpected redis JSON payload for %s: %w", redisJSONid, err)
	}

	if err = json.Unmarshal(obj, &jsonMessage); err != nil {
		return jsonMessage, fmt.Errorf("failed to unmarshal message JSON: %w", err)
	}

	return
}
