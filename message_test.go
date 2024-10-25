/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"testing"
	"time"
)

// NewMessage erstellt ein neues Message-Objekt
func NewMessage(author, content, severity string) *Message {

	timestamp := time.Now().Format(time.RFC3339)

	return &Message{
		Author:    author,
		Message:   content,
		Timestamp: timestamp,
		Severity:  severity,
	}
}

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

}