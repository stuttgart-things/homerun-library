/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"encoding/json"
	"log"
	"time"

	rejson "github.com/nitishm/go-rejson/v4"
	sthingsCli "github.com/stuttgart-things/sthingsCli"
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

func GetMessageJSON(
	redisJSONid string,
	redisJSONHandler *rejson.Handler) (jsonMessage Message, err error) {

	// GET JSON AS MESSAGE OBJECT
	obj, exist := sthingsCli.GetRedisJSON(redisJSONHandler, redisJSONid)

	if exist {
		jsonMessage = Message{}
		err := json.Unmarshal(obj, &jsonMessage)
		if err != nil {
			log.Fatalf("FAILED TO JSON UNMARSHAL")
		}
	}

	return
}
