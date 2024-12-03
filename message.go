/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"encoding/json"
	"log"

	"github.com/nitishm/go-rejson/v4"
	sthingsCli "github.com/stuttgart-things/sthingsCli"
)

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

func GetMessageJSON(redisJSONid string, redisJSONHandler *rejson.Handler) (jsonMessage Message, err error) {

	// GET JSON AS MESSAGE OBJECT
	obj, exist := sthingsCli.GetRedisJSON(redisJSONHandler, redisJSONid)

	if !exist {
		log.Printf("No JSON object found for ID: %s", redisJSONid)
		return
	}

	log.Printf("Raw JSON from Redis: %s\n", string(obj))

	jsonMessage = Message{}
	err = json.Unmarshal(obj, &jsonMessage)
	if err != nil {
		log.Fatalf("FAILED TO JSON UNMARSHAL: %v", err)
	}

	if jsonMessage.Message == "" {
		log.Println("Warning: Message field is empty.")
	}

	return jsonMessage, nil
}
