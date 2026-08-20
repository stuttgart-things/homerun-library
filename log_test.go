/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// A library must not write to the process output unless the program asks it to.
// Before #101 the package installed a pterm trace logger at import time, so
// this could not hold.
func TestLoggerIsSilentByDefault(t *testing.T) {
	SetLogger(nil)

	// Capture the real process stdout, not an injected writer - that is exactly
	// what the pterm logger wrote to and what an injected writer would miss.
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	log().Info("must not be visible", "key", "value")
	log().Warn("must not be visible either")

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe: %v", err)
	}
	os.Stdout = orig

	written, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read pipe: %v", err)
	}
	if len(written) != 0 {
		t.Errorf("library wrote to stdout without a logger installed: %q", written)
	}
}

func TestSetLoggerRoutesRecords(t *testing.T) {
	var buf bytes.Buffer
	SetLogger(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { SetLogger(nil) })

	log().Info("hello", "stream", "messages")

	got := buf.String()
	if !strings.Contains(got, "hello") || !strings.Contains(got, "stream=messages") {
		t.Errorf("record was not routed to the installed logger, got %q", got)
	}
}

func TestSetLoggerNilRestoresSilence(t *testing.T) {
	var buf bytes.Buffer
	SetLogger(slog.New(slog.NewTextHandler(&buf, nil)))
	SetLogger(nil)

	log().Info("must not reach the previous logger")

	if buf.Len() != 0 {
		t.Errorf("expected no output after SetLogger(nil), got %q", buf.String())
	}
}

// EnqueueMessageInRedisStreams must not print anything on the failure path
// either - it logs a warning when closing the client fails and returns the real
// error to the caller.
func TestEnqueueFailureIsSilentByDefault(t *testing.T) {
	SetLogger(nil)
	_, _, err := EnqueueMessageInRedisStreams(Message{Title: "t"}, RedisConfig{})
	if err == nil {
		t.Fatal("expected an error for an unconfigured RedisConfig")
	}
	if !strings.Contains(err.Error(), "no redis address configured") {
		t.Errorf("expected a configuration error, got %v", err)
	}
}
