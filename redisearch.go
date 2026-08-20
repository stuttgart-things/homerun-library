/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"fmt"
	"strings"
	"time"

	"github.com/RediSearch/redisearch-go/redisearch"
	redigo "github.com/gomodule/redigo/redis"
)

var (
	redisSearchSchema = redisearch.NewSchema(redisearch.DefaultOptions).
		AddField(redisearch.NewTextFieldOptions("title", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("message", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("severity", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("author", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("timestamp", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("system", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("tags", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("assigneeaddress", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("assigneename", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("artifacts", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("url", redisearch.TextFieldOptions{Sortable: true}))
)

// newRediSearchPool builds a redigo pool for the given configuration.
//
// AUTH is only sent when a password is configured: a Redis without
// requirepass answers "ERR Client sent AUTH, but no password is set" and the
// dial fails, which is what happened with an unconditional AUTH.
func newRediSearchPool(rc RedisConfig) *redigo.Pool {
	return &redigo.Pool{
		MaxIdle:   10,
		MaxActive: 10,
		Dial: func() (redigo.Conn, error) {
			conn, err := redigo.Dial("tcp", rc.Addr+":"+rc.Port)
			if err != nil {
				return nil, err
			}
			if rc.Password != "" {
				if _, err := conn.Do("AUTH", rc.Password); err != nil {
					_ = conn.Close()
					return nil, err
				}
			}
			return conn, nil
		},
	}
}

// StoreInRediSearch indexes a Message in the configured RediSearch index,
// creating the index first if it does not exist yet.
//
// Every failure is returned. The previous implementation delegated index
// creation and document indexing to helpers that called log.Fatalf, so a Redis
// hiccup terminated the calling process instead of returning an error - while
// this function's signature promised otherwise.
func StoreInRediSearch(message Message, rc RedisConfig) error {
	if err := rc.validateConnection(); err != nil {
		return err
	}
	if rc.Index == "" {
		return fmt.Errorf("no redisearch index configured")
	}

	connectionPool := newRediSearchPool(rc)
	defer func() {
		if err := connectionPool.Close(); err != nil {
			log().Warn("failed to close redisearch connection pool", "error", err)
		}
	}()

	rediSearchClient := redisearch.NewClientFromPool(connectionPool, rc.Index)

	// CHECK/CREATE INDEX
	indexExists, err := rediSearchIndexExists(rediSearchClient)
	if err != nil {
		return fmt.Errorf("failed to check redisearch index %s: %w", rc.Index, err)
	}
	if !indexExists {
		if err := rediSearchClient.CreateIndex(redisSearchSchema); err != nil {
			return fmt.Errorf("failed to create redisearch index %s: %w", rc.Index, err)
		}
		log().Info("redisearch index created", "index", rc.Index)
	}

	// INDEX THE DOCUMENT
	documentID := time.Now().Format(time.RFC3339Nano) + "-" + message.System
	doc := redisearch.NewDocument(documentID, 1.0)
	doc.Set("title", message.Title).
		Set("message", message.Message).
		Set("severity", message.Severity).
		Set("author", message.Author).
		Set("timestamp", time.Now().Unix()).
		Set("system", message.System).
		Set("tags", message.Tags).
		Set("assigneeaddress", message.AssigneeAddress).
		Set("assigneename", message.AssigneeName).
		Set("artifacts", message.Artifacts).
		Set("url", message.Url)

	if err := rediSearchClient.Index(doc); err != nil {
		return fmt.Errorf("failed to index document %s: %w", documentID, err)
	}

	log().Info("document indexed in redisearch", "index", rc.Index, "documentID", documentID)
	return nil
}

// rediSearchIndexExists reports whether the client's index exists. RediSearch
// answers a missing index with an "Unknown Index name" error rather than an
// empty result, so that case is translated into (false, nil).
func rediSearchIndexExists(client *redisearch.Client) (bool, error) {
	if _, err := client.Info(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unknown index name") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
