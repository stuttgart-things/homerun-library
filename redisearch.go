/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RediSearch/redisearch-go/v2/redisearch"
	redigo "github.com/gomodule/redigo/redis"
)

var (
	redisSearchSchema = redisearch.NewSchema(redisearch.DefaultOptions).
		AddField(redisearch.NewTextFieldOptions("title", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("message", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("severity", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("author", redisearch.TextFieldOptions{Sortable: true})).
		// NUMERIC, not TEXT: RediSearch cannot range-query TEXT, so
		// @timestamp:[1757836800 +inf] - "everything since yesterday" - is not
		// expressible against a text field. Sorting worked, filtering did not.
		AddField(redisearch.NewNumericFieldOptions("timestamp", redisearch.NumericFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("system", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("tags", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("assigneeaddress", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("assigneename", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("artifacts", redisearch.TextFieldOptions{Sortable: true})).
		AddField(redisearch.NewTextFieldOptions("url", redisearch.TextFieldOptions{Sortable: true}))
)

// Redis dial and I/O deadlines for the RediSearch path.
//
// redigo.Dial applies no timeouts of its own, so before this a Redis that
// accepted the connection and then stopped answering blocked the caller
// forever. go-redis, used on the pitcher path, sets comparable defaults itself.
const (
	redisDialTimeout  = 5 * time.Second
	redisReadTimeout  = 10 * time.Second
	redisWriteTimeout = 10 * time.Second
)

// newRediSearchPool builds a redigo pool for the given configuration, dialling
// with ctx so a caller can cancel while a connection is being established.
//
// AUTH is only sent when a password is configured: a Redis without
// requirepass answers "ERR Client sent AUTH, but no password is set" and the
// dial fails, which is what happened with an unconditional AUTH.
//
// Index creation and document indexing themselves go through redisearch-go,
// which has no context-aware API, so ctx bounds the dial and the deadlines
// above bound each read and write.
func newRediSearchPool(ctx context.Context, rc RedisConfig) *redigo.Pool {
	return &redigo.Pool{
		MaxIdle:   10,
		MaxActive: 10,
		Dial: func() (redigo.Conn, error) {
			conn, err := redigo.DialContext(ctx, "tcp", rc.Addr+":"+rc.Port,
				redigo.DialConnectTimeout(redisDialTimeout),
				redigo.DialReadTimeout(redisReadTimeout),
				redigo.DialWriteTimeout(redisWriteTimeout),
			)
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
// Deprecated: prefer indexing the Redis JSON documents that Enqueue already
// writes. This function maintains a second, hash-based copy of every message in
// a separate key, which is redundant whenever an index over those JSON
// documents exists - and invisible if that index is defined ON JSON, since a
// hash is not indexed by it (see assertIndexIndexesHashes). It is kept and
// fixed in v4 for callers that own a dedicated ON HASH index, and is scheduled
// for removal in v5.
//
// Every failure is returned. The previous implementation delegated index
// creation and document indexing to helpers that called log.Fatalf, so a Redis
// hiccup terminated the calling process instead of returning an error - while
// this function's signature promised otherwise.
func StoreInRediSearch(message Message, rc RedisConfig) error {
	return StoreInRediSearchContext(context.Background(), message, rc)
}

// StoreInRediSearchContext is StoreInRediSearch bounded by ctx.
//
// ctx bounds establishing the connection; redisearch-go itself exposes no
// context-aware API, so the individual commands are bounded by the read and
// write deadlines set on the pool instead.
func StoreInRediSearchContext(ctx context.Context, message Message, rc RedisConfig) error {
	if err := rc.validateConnection(); err != nil {
		return err
	}
	if rc.Index == "" {
		return fmt.Errorf("no redisearch index configured")
	}

	connectionPool := newRediSearchPool(ctx, rc)
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
	} else if err := assertIndexIndexesHashes(connectionPool, rc.Index); err != nil {
		return err
	}

	// INDEX THE DOCUMENT
	documentID := time.Now().Format(time.RFC3339Nano) + "-" + message.System
	doc := redisearch.NewDocument(documentID, 1.0)
	doc.Set("title", message.Title).
		Set("message", message.Message).
		Set("severity", message.Severity).
		Set("author", message.Author).
		Set("timestamp", eventTimestamp(message)).
		Set("system", message.System).
		Set("tags", message.Tags).
		Set("assigneeaddress", message.AssigneeAddress).
		Set("assigneename", message.AssigneeName).
		Set("artifacts", message.Artifacts).
		Set("url", message.URL)

	if err := rediSearchClient.Index(doc); err != nil {
		return fmt.Errorf("failed to index document %s: %w", documentID, err)
	}

	log().Info("document indexed in redisearch", "index", rc.Index, "documentID", documentID)
	return nil
}

// assertIndexIndexesHashes fails when the existing index cannot hold the
// documents this function writes.
//
// This function writes hashes. An index created with ON JSON only indexes JSON
// keys, so a hash written into it is stored but never indexed: FT.SEARCH will
// not find it, num_docs does not move, and nothing that prunes by search result
// will ever delete it. Up to v3 this happened silently - the write returned nil
// and the caller had no way to tell the data was invisible.
//
// That is not a hypothetical: homerun2-omni-pitcher creates its index with
// ON JSON at startup, so every StoreInRediSearch call against it produced an
// orphaned, never-expiring key. Failing here turns that into something a caller
// can see and act on.
func assertIndexIndexesHashes(pool *redigo.Pool, index string) error {
	conn := pool.Get()
	defer func() { _ = conn.Close() }()

	raw, err := redigo.Values(conn.Do("FT.INFO", index))
	if err != nil {
		return fmt.Errorf("failed to read redisearch index info for %s: %w", index, err)
	}

	keyType, err := indexKeyType(raw)
	if err != nil {
		// An older RediSearch may not report index_definition at all. Do not
		// block the write on a field we cannot read.
		log().Warn("could not determine the key type of the redisearch index",
			"index", index, "error", err)
		return nil
	}

	if !strings.EqualFold(keyType, "hash") {
		return fmt.Errorf(
			"redisearch index %s indexes %s keys, but StoreInRediSearch writes hashes: "+
				"documents would be stored but never indexed. Recreate the index with ON HASH, "+
				"or index the Redis JSON documents written by Enqueue instead",
			index, strings.ToUpper(keyType))
	}

	return nil
}

// indexKeyType digs key_type out of the index_definition section of FT.INFO,
// which is a flat key/value array nested inside the outer one.
func indexKeyType(info []interface{}) (string, error) {
	for i := 0; i+1 < len(info); i += 2 {
		key, err := redigo.String(info[i], nil)
		if err != nil || key != "index_definition" {
			continue
		}
		definition, err := redigo.Values(info[i+1], nil)
		if err != nil {
			return "", fmt.Errorf("index_definition is not a list: %w", err)
		}
		for j := 0; j+1 < len(definition); j += 2 {
			field, err := redigo.String(definition[j], nil)
			if err != nil || field != "key_type" {
				continue
			}
			return redigo.String(definition[j+1], nil)
		}
		return "", fmt.Errorf("index_definition carries no key_type")
	}
	return "", fmt.Errorf("FT.INFO carries no index_definition")
}

// eventTimestamp returns the Unix time the message says the event happened.
//
// Up to v3 this field was populated with time.Now(), i.e. the moment of
// indexing rather than of the event. For a queue-backed system those are
// different values: any retry, backlog or replay made it wrong, and the
// original was then unrecoverable from the index.
//
// A missing or unparseable Message.Timestamp falls back to now, which is the
// best guess available - but it is logged, so a producer that does not set the
// field is visible rather than silently recorded with the wrong time.
func eventTimestamp(message Message) int64 {
	if message.Timestamp == "" {
		log().Warn("message has no timestamp, indexing with the current time",
			"system", message.System, "title", message.Title)
		return time.Now().Unix()
	}

	ts, err := time.Parse(time.RFC3339, message.Timestamp)
	if err != nil {
		log().Warn("message timestamp is not RFC3339, indexing with the current time",
			"system", message.System, "title", message.Title,
			"timestamp", message.Timestamp, "error", err)
		return time.Now().Unix()
	}

	return ts.Unix()
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
