# Migrating from v3 to v4

v4 changes the module path, three exported symbols and the RediSearch schema.
Everything else is source-compatible with v3.2.0.

```bash
go get github.com/stuttgart-things/homerun-library/v4@v4.0.0
```

Update the import path — the package name is unchanged:

```go
homerun "github.com/stuttgart-things/homerun-library/v4"
```

## 1. `Message.Url` → `Message.URL`

Go initialisms are upper-case. The JSON tag stays `url`, so **nothing on the
wire, in Redis JSON or in the RediSearch index changes** — this is a
compile-time rename only.

```go
msg := homerun.Message{URL: "https://example.com/build/42"}   // was Url:
```

Mechanical fix across a service:

```bash
grep -rl '\.Url\b\|Url:' --include='*.go' . | xargs sed -i 's/\bUrl:/URL:/g; s/\.Url\b/.URL/g'
```

## 2. `SendToHomerun` returns a `Response`

```go
// v3
answer, resp, err := homerun.SendToHomerun(dest, token, body, false)
fmt.Println(resp.Status, string(answer))

// v4
resp, err := homerun.SendToHomerun(dest, token, body, false)
fmt.Println(resp.Status, string(resp.Body))
```

v3 returned the raw `*http.Response` after already reading and closing its body,
so a caller that tried to read `Body` got nothing and one that closed it was
closing a closed body. `Response` carries what a caller can actually use:

```go
type Response struct {
    StatusCode int
    Status     string
    Header     http.Header
    Body       []byte
}

func (r Response) OK() bool   // 2xx
```

A non-2xx answer is still not an error — `err` reports whether the request could
be made and read at all. Check `resp.OK()` for the endpoint's verdict.

The same applies to `SendToHomerunContext`.

## 3. RediSearch: `timestamp` is NUMERIC and carries the event time

Two fixes in the schema `StoreInRediSearch` creates:

- **`timestamp` is `NUMERIC`, not `TEXT`.** RediSearch cannot range-query TEXT,
  so `@timestamp:[1757836800 +inf]` — "everything since yesterday" — was not
  expressible. Sorting worked, filtering did not.
- **The indexed value is `Message.Timestamp`, not `time.Now()`.** v3 recorded the
  moment of *indexing*. In a queue-backed system that differs from the event time
  on every retry, backlog and replay, and the original was then unrecoverable.

A missing or unparseable `Message.Timestamp` falls back to the current time and
**logs a warning** (install a logger with `homerun.SetLogger` to see it).

### This needs the index recreated

A field's type cannot be changed in place, and the library only checks whether
the index *exists*. An index created by v3 keeps the old TEXT schema, so the
range query keeps failing until you recreate it:

```bash
redis-cli FT.DROPINDEX <index>        # keeps the documents
# restart the service; StoreInRediSearch recreates the index with the new schema
```

`FT.DROPINDEX <index> DD` also deletes the indexed documents — only use it if you
intend to lose the history.

## 4. `StoreInRediSearch` is deprecated, and now refuses an incompatible index

`StoreInRediSearch` maintains a second, **hash-based** copy of every message in
its own key. If your index is defined `ON JSON`, a hash written into it is stored
but never indexed — `FT.SEARCH` will not find it and `num_docs` does not move.

In v3 this failed **silently**: the call returned `nil`, and the orphaned key had
no TTL, so nothing that prunes by search result would ever delete it. That is not
hypothetical — `homerun2-omni-pitcher` creates its index `ON JSON` at startup, so
every call against it produced exactly that.

v4 detects the mismatch and returns an error instead:

```
redisearch index homerun-idx indexes JSON keys, but StoreInRediSearch writes
hashes: documents would be stored but never indexed. Recreate the index with
ON HASH, or index the Redis JSON documents written by Enqueue instead
```

**If you index the Redis JSON documents** (the `ON JSON` case), you do not need
`StoreInRediSearch` at all: `Enqueue` already writes the JSON document, and an
index over those keys picks it up automatically with the correct event
timestamp. Drop the call.

**If you own a dedicated `ON HASH` index**, `StoreInRediSearch` keeps working with
the fixes above.

It is deprecated either way and scheduled for removal in v5.

### Making the JSON index range-queryable

The JSON documents carry `timestamp` as an RFC3339 **string**, which has the same
range-query limitation. Indexing it as `NUMERIC` is not possible without a
numeric field in the document. Until then, consumers filtering by time over a
JSON index have to fetch and compare in Go — which is what
`homerun2-scout`'s retention does today.

## 5. `redisearch-go` v1 → v2

Internal: the library now uses `github.com/RediSearch/redisearch-go/v2`. This
only affects you if your own code imports `redisearch-go` directly.
