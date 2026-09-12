/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The field name services declare in their RediSearch index has to be the one
// Enqueue writes, or the NUMERIC attribute indexes nothing.
func TestRediSearchTimestampFieldIsTheDocumentTag(t *testing.T) {
	f, ok := reflect.TypeOf(storedMessage{}).FieldByName("TimestampUnix")
	if !ok {
		t.Fatal("storedMessage has no TimestampUnix field")
	}
	if tag := strings.Split(f.Tag.Get("json"), ",")[0]; tag != RediSearchTimestampField {
		t.Errorf("json tag %q, RediSearchTimestampField %q", tag, RediSearchTimestampField)
	}
}

func TestStoredMessageDocument(t *testing.T) {
	msg := Message{
		Title: "t", Message: "m", Severity: "error", Author: "a", Timestamp: "2026-09-12T07:07:14Z",
		System: "github", Tags: "x,y", AssigneeAddress: "aa", AssigneeName: "an", Artifacts: "ar", URL: "u",
	}
	raw, err := json.Marshal(newStoredMessage(msg))
	if err != nil {
		t.Fatal(err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 12, 7, 7, 14, 0, time.UTC).Unix()
	if got, ok := doc[RediSearchTimestampField].(float64); !ok || int64(got) != want {
		t.Errorf("%s = %v, want %d", RediSearchTimestampField, doc[RediSearchTimestampField], want)
	}

	// Every Message field keeps its JSON name at the top level of the
	// document, where the catchers' JSON.GET and the index's $.field paths
	// expect it.
	plain, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(plain, &fields); err != nil {
		t.Fatal(err)
	}
	for k, v := range fields {
		if doc[k] != v {
			t.Errorf("document %s = %v, want %v", k, doc[k], v)
		}
	}
	if len(doc) != len(fields)+1 {
		t.Errorf("document has %d fields, want the %d Message fields and %s", len(doc), len(fields), RediSearchTimestampField)
	}

	// Readers decode the document into a Message; the extra field must not
	// change what they get.
	var back Message
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back != msg {
		t.Errorf("decoded %+v, want %+v", back, msg)
	}
}

func TestStoredMessageFallsBackToNow(t *testing.T) {
	SetLogger(nil)
	for _, ts := range []string{"", "yesterday", "2020-01-02 03:04:05"} {
		before := time.Now().Unix()
		doc := newStoredMessage(Message{Timestamp: ts, System: "sys"})
		after := time.Now().Unix()
		if doc.TimestampUnix < before || doc.TimestampUnix > after {
			t.Errorf("timestamp %q: %s = %d, want the current time", ts, RediSearchTimestampField, doc.TimestampUnix)
		}
		if doc.Timestamp != ts {
			t.Errorf("timestamp %q was changed to %q; the document keeps what the producer sent", ts, doc.Timestamp)
		}
	}
}

func TestEventUnix(t *testing.T) {
	cases := []struct {
		ts           string
		want         int64
		wantFallback bool
	}{
		{"2020-01-02T03:04:05Z", time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC).Unix(), false},
		{"2020-01-02T05:04:05+02:00", time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC).Unix(), false},
		{"2020-01-02T03:04:05.123456Z", time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC).Unix(), false},
		{"", 0, true},
		{"not a time", 0, true},
	}
	for _, tc := range cases {
		got, fallback := eventUnix(Message{Timestamp: tc.ts})
		if (fallback != "") != tc.wantFallback {
			t.Errorf("eventUnix(%q) fallback = %q, want fallback %v", tc.ts, fallback, tc.wantFallback)
		}
		if !tc.wantFallback && got != tc.want {
			t.Errorf("eventUnix(%q) = %d, want %d", tc.ts, got, tc.want)
		}
	}
	if _, fallback := eventUnix(Message{}); fallback != fallbackNoTimestamp {
		t.Errorf("fallback for an empty timestamp = %q, want %q", fallback, fallbackNoTimestamp)
	}
}
