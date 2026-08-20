/*
Copyright © 2024 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
	resp, err := SendToHomerun(destination, token, renderedBody, true)
	if err != nil {
		t.Fatalf("SendToHomerun returned unexpected error: %v", err)
	}
	if !resp.OK() {
		t.Errorf("Expected a 2xx answer, got %s", resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") == "" {
		t.Error("Expected the response headers to be carried through")
	}

	// Verify the response body
	expectedResponse := `{"status":"success"}`
	if string(resp.Body) != expectedResponse {
		t.Errorf("Expected response %s, got %s", expectedResponse, string(resp.Body))
	}
}

func TestSendToHomerunInvalidURL(t *testing.T) {
	_, err := SendToHomerun("://invalid-url", "token", []byte(`{}`), true)
	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}
}

func TestSendToHomerunConnectionRefused(t *testing.T) {
	_, err := SendToHomerun("http://127.0.0.1:1", "token", []byte(`{}`), true)
	if err == nil {
		t.Fatal("Expected error for connection refused, got nil")
	}
}

func TestRenderBodyInvalidTemplate(t *testing.T) {
	_, err := RenderBody("{{.Unclosed", nil)
	if err == nil {
		t.Fatal("Expected error for invalid template, got nil")
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
		result, err := RenderBody(test.templateData, test.object)
		if err != nil {
			t.Fatalf("RenderBody returned unexpected error: %v", err)
		}
		if result != test.expected {
			t.Errorf("For template '%s' and object %v, expected '%s' but got '%s'", test.templateData, test.object, test.expected, result)
		}
	}
}

func TestRenderBodyEscapesJSON(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "double quotes",
			msg:  Message{Title: `Build "42" failed`, Message: "unexpected end of JSON"},
		},
		{
			name: "newlines and tabs",
			msg:  Message{Title: "stack trace", Message: "line1\nline2\tindented"},
		},
		{
			name: "backslashes",
			msg:  Message{Title: `C:\builds\42`, Artifacts: `\\share\artifacts`},
		},
		{
			name: "control characters and unicode",
			msg:  Message{Title: "bell\x07", Message: "grüße 🎉"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := RenderBody(HomeRunBodyData, tt.msg)
			if err != nil {
				t.Fatalf("RenderBody returned unexpected error: %v", err)
			}

			var got Message
			if err := json.Unmarshal([]byte(rendered), &got); err != nil {
				t.Fatalf("rendered body is not valid JSON: %v\n%s", err, rendered)
			}

			if got.Title != tt.msg.Title {
				t.Errorf("Title round-trip mismatch:\n got %q\nwant %q", got.Title, tt.msg.Title)
			}
			if got.Message != tt.msg.Message {
				t.Errorf("Message round-trip mismatch:\n got %q\nwant %q", got.Message, tt.msg.Message)
			}
			if got.Artifacts != tt.msg.Artifacts {
				t.Errorf("Artifacts round-trip mismatch:\n got %q\nwant %q", got.Artifacts, tt.msg.Artifacts)
			}
		})
	}
}

func TestRenderBodyResistsFieldInjection(t *testing.T) {
	// A caller that controls a single field must not be able to set another one.
	msg := Message{
		Title:  `benign", "Severity": "critical`,
		Author: `x", "System": "spoofed`,
	}

	rendered, err := RenderBody(HomeRunBodyData, msg)
	if err != nil {
		t.Fatalf("RenderBody returned unexpected error: %v", err)
	}

	var got Message
	if err := json.Unmarshal([]byte(rendered), &got); err != nil {
		t.Fatalf("rendered body is not valid JSON: %v\n%s", err, rendered)
	}

	if got.Severity != "" {
		t.Errorf("Severity was injected through Title: %q", got.Severity)
	}
	if got.System != "" {
		t.Errorf("System was injected through Author: %q", got.System)
	}
	if got.Title != msg.Title {
		t.Errorf("Title round-trip mismatch:\n got %q\nwant %q", got.Title, msg.Title)
	}
}

func TestJSONEscape(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{`plain`, `plain`},
		{`with "quotes"`, `with \"quotes\"`},
		{"tab\there", `tab\there`},
		{`back\slash`, `back\\slash`},
		{nil, ``},
		{42, `42`},
	}

	for _, tt := range tests {
		got, err := jsonEscape(tt.in)
		if err != nil {
			t.Fatalf("jsonEscape(%v) returned unexpected error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("jsonEscape(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
