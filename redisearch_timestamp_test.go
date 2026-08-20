/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/RediSearch/redisearch-go/v2/redisearch"
)

// Up to v3 the indexed timestamp was time.Now(), i.e. the moment of indexing
// rather than of the event. For a queue-backed system those differ on every
// retry, backlog or replay, and the original was then unrecoverable.
func TestEventTimestampUsesTheMessageTimestamp(t *testing.T) {
	SetLogger(nil)

	const ts = "2020-01-02T03:04:05Z"
	want := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC).Unix()

	got := eventTimestamp(Message{Timestamp: ts})
	if got != want {
		t.Errorf("eventTimestamp = %d, want %d (the event time, not the indexing time)", got, want)
	}
	if got >= time.Now().Unix()-60 {
		t.Errorf("eventTimestamp returned something close to now (%d); it is indexing the wrong value", got)
	}
}

func TestEventTimestampFallsBackLoudly(t *testing.T) {
	cases := []struct {
		name    string
		message Message
		logWant string
	}{
		{"empty", Message{System: "sys"}, "no timestamp"},
		{"unparseable", Message{Timestamp: "yesterday", System: "sys"}, "not RFC3339"},
		{"wrong layout", Message{Timestamp: "2020-01-02 03:04:05", System: "sys"}, "not RFC3339"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			SetLogger(slog.New(slog.NewTextHandler(&buf, nil)))
			t.Cleanup(func() { SetLogger(nil) })

			before := time.Now().Unix()
			got := eventTimestamp(tc.message)
			after := time.Now().Unix()

			if got < before || got > after {
				t.Errorf("expected a fallback to the current time, got %d", got)
			}
			// Falling back is fine; falling back silently is what reintroduces
			// the defect this fixes.
			if !strings.Contains(buf.String(), tc.logWant) {
				t.Errorf("fallback was not logged, got %q", buf.String())
			}
			if !strings.Contains(buf.String(), "level=WARN") {
				t.Errorf("expected the fallback at WARN level, got %q", buf.String())
			}
		})
	}
}

func TestSchemaDeclaresTimestampNumeric(t *testing.T) {
	// RediSearch cannot range-query a TEXT field, so @timestamp:[x +inf] is not
	// expressible unless this is NUMERIC.
	for _, field := range redisSearchSchema.Fields {
		if field.Name != "timestamp" {
			continue
		}
		if field.Type != redisearch.NumericField {
			t.Errorf("timestamp is declared as field type %v, want NumericField", field.Type)
		}
		return
	}
	t.Fatal("schema has no timestamp field")
}
