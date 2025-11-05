/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendToHomerun(t *testing.T) {
	// Mock server to simulate the homerun service
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate the request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Validate the Content-Type header
		if r.Header.Get("Content-Type") != contentType {
			t.Errorf("Expected Content-Type %s, got %s", contentType, r.Header.Get("Content-Type"))
		}

		// Validate the X-Auth-Token header
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("Expected X-Auth-Token test-token, got %s", r.Header.Get("X-Auth-Token"))
		}

		// Write a response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer mockServer.Close()

	// Test data
	destination := mockServer.URL
	token := "test-token"
	renderedBody := []byte(`{"message":"hello"}`)

	// Call the function
	response, resp := SendToHomerun(destination, token, renderedBody, true)
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Error closing response body: %v", err)
		}
	}()

	fmt.Println(resp.Status)

	// Verify the response
	expectedResponse := `{"status":"success"}`
	if string(response) != expectedResponse {
		t.Errorf("Expected response %s, got %s", expectedResponse, string(response))
	}
}

func TestRenderBody(t *testing.T) {
	tests := []struct {
		templateData string
		object       interface{}
		expected     string
	}{
		{
			templateData: "Hello, {{.Name}}!",
			object:       map[string]string{"Name": "Alice"},
			expected:     "Hello, Alice!",
		},
		{
			templateData: "Age: {{.Age}}",
			object:       map[string]int{"Age": 30},
			expected:     "Age: 30",
		},
		{
			templateData: "Empty: {{.Missing}}",
			object:       map[string]string{},
			expected:     "Empty: <no value>", // Default Go template behavior
		},
	}

	for _, test := range tests {
		result := RenderBody(test.templateData, test.object)
		if result != test.expected {
			t.Errorf("For template '%s' and object %v, expected '%s' but got '%s'", test.templateData, test.object, test.expected, result)
		}
	}
}

func TestEnqueueMessageInRedisStreams(t *testing.T) {
	msg := Message{
		Title:     "Test Message",
		Message:   "This is a test message",
		Severity:  "info",
		Author:    "test-user",
		Timestamp: "2025-11-11T06:45:00Z",
		System:    "test-system",
		Tags:      "unit-test",
	}

	conn := map[string]string{
		"addr":     "localhost",
		"port":     "6379",
		"password": "",
		"stream":   "test-stream",
	}

	objectID, streamID := EnqueueMessageInRedisStreams(msg, conn)

	assert.NotEmpty(t, objectID)
	assert.Equal(t, "test-stream", streamID)
}
