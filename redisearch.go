/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"time"

	"github.com/RediSearch/redisearch-go/redisearch"

	sthingsCli "github.com/stuttgart-things/sthingsCli"
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

func StoreInRediSearch(message Message, redisConnection map[string]string) {

	// CREATE REDISEARCH CLIENT
	connectionPool := sthingsCli.CreateRedisConnectionPool(redisConnection["addr"]+":"+redisConnection["port"], redisConnection["password"])
	rediSearchClient := redisearch.NewClientFromPool(connectionPool, redisConnection["index"])

	// CHECK/CREATE INDEX
	indexExists, err := sthingsCli.CheckIfRedisSearchIndexExists(rediSearchClient)
	if !indexExists && err == nil {
		sthingsCli.CreateRedisSearchIndex(rediSearchClient, redisSearchSchema)
		logger.Info("INDEX DID NOT EXIST, BUT WAS NOW CREATED", logger.Args("", redisConnection["index"]))
	}

	// INDEX THE DOCUMENTS ON INDEX
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

	sthingsCli.IndexDocument(rediSearchClient, doc)

	logger.Info("DOCUMENT WAS CREATED ON REDISEARCH", logger.Args("", doc, documentID, doc))
}
