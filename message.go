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
	Url             string `json:"url,omitempty"`             // empty
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
