/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"testing"

	rejson "github.com/nitishm/go-rejson/v4"
	"github.com/stretchr/testify/mock"
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

}

// Mock for the sthingsCli package to mock GetRedisJSON
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) GetRedisJSON(handler *rejson.Handler, key string) ([]byte, bool) {
	args := m.Called(handler, key)
	return args.Get(0).([]byte), args.Bool(1)
}
