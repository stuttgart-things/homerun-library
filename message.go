/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"encoding/json"
	"log"

	rejson "github.com/nitishm/go-rejson/v4"
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
