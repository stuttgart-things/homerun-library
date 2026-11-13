/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// The payload is stored as a nested JSON value, so a consumer reading the
// document gets the object, not an escaped string.
func TestStoredMessagePayloadIsNested(t *testing.T) {
	msg := Message{Title: "t", Timestamp: "2026-09-12T07:07:14Z"}
	obj := map[string]any{"kind": "Result", "spec": map[string]any{"error": []any{map[string]any{"text": "back-off"}}}}
	if err := msg.SetPayload(obj); err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(newStoredMessage(msg))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(doc["payload"], obj) {
		t.Errorf("document payload = %#v, want %#v", doc["payload"], obj)
	}

	var back Message
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	var gotObj map[string]any
	if err := json.Unmarshal([]byte(back.Payload), &gotObj); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotObj, obj) {
		t.Errorf("decoded payload = %#v, want %#v", gotObj, obj)
	}
}

// Without a payload the document and the HTTP body look as before.
func TestEmptyPayloadIsOmitted(t *testing.T) {
	raw, err := json.Marshal(Message{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "payload") {
		t.Errorf("empty payload serialized: %s", raw)
	}
}

// Message stays comparable with ==; services compare messages in tests and
// code, and a slice-typed field would break them at compile time.
func TestMessageIsComparable(t *testing.T) {
	a := Message{Title: "t", Payload: `{"a":1}`}
	b := a
	if a != b {
		t.Error("equal messages compare unequal")
	}
	b.Payload = `{"a":2}`
	if a == b {
		t.Error("different payloads compare equal")
	}
}

// A payload of JSON null decodes to an empty payload.
func TestNullPayloadIsEmpty(t *testing.T) {
	var m Message
	if err := json.Unmarshal([]byte(`{"title":"t","payload":null}`), &m); err != nil {
		t.Fatal(err)
	}
	if m.Payload != "" {
		t.Errorf("payload = %q, want empty", m.Payload)
	}
}

// Producers that post a Message as JSON (k8s-pitcher to omni-pitcher /pitch)
// and receivers that decode it keep the payload.
func TestPayloadSurvivesJSONRoundTrip(t *testing.T) {
	in := Message{Title: "t", Payload: `{"a":[1,2]}`}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Message
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if string(out.Payload) != `{"a":[1,2]}` {
		t.Errorf("payload = %s", out.Payload)
	}
}

func TestSetPayload(t *testing.T) {
	var m Message
	if err := m.SetPayload([]string{"x"}); err != nil {
		t.Fatal(err)
	}
	if string(m.Payload) != `["x"]` {
		t.Errorf("payload = %s", m.Payload)
	}

	if err := m.SetPayload(nil); err != nil || m.Payload != "" {
		t.Errorf("nil: err %v, payload %s", err, m.Payload)
	}

	if err := m.SetPayload(make(chan int)); err == nil {
		t.Error("unmarshalable value accepted")
	}

	big := strings.Repeat("x", MaxPayloadSize)
	m.Payload = `{"keep":true}`
	if err := m.SetPayload(big); err == nil {
		t.Error("payload above MaxPayloadSize accepted")
	}
	if string(m.Payload) != `{"keep":true}` {
		t.Errorf("failed SetPayload changed the payload to %.40s", m.Payload)
	}
}

func TestCheckPayload(t *testing.T) {
	for _, tc := range []struct {
		raw string
		ok  bool
	}{
		{"", true},
		{`{"a":1}`, true},
		{`"text"`, true},
		{`{"a":`, false},
		{`not json`, false},
	} {
		if err := checkPayload(tc.raw); (err == nil) != tc.ok {
			t.Errorf("checkPayload(%q) = %v, want ok %v", tc.raw, err, tc.ok)
		}
	}
}

// Enqueue rejects a bad payload before it talks to Redis, so no half-written
// document or stream entry can result.
func TestEnqueueRejectsInvalidPayload(t *testing.T) {
	p := NewPitcher(RedisConfig{Addr: "127.0.0.1", Port: "1", Stream: "s"})
	defer func() { _ = p.Close() }()

	_, _, err := p.Enqueue(context.Background(), Message{Title: "t", Payload: `{broken`})
	if err == nil || !strings.Contains(err.Error(), "payload") {
		t.Errorf("err = %v, want a payload error", err)
	}
}
