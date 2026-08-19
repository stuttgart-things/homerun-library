/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"testing"
)

func TestNewMessage(t *testing.T) {
	author := "Patrick Hermann"
	content := "This is a test message."
	severity := "INFO"

	msg := NewMessage(author, content, severity)

	if msg.Author != author {
		t.Errorf("expected author %s, got %s", author, msg.Author)
	}

	if msg.Message != content {
		t.Errorf("expected content %s, got %s", content, msg.Message)
	}

	if msg.Severity != severity {
		t.Errorf("expected severity %s, got %s", severity, msg.Severity)
	}

	if msg.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}
