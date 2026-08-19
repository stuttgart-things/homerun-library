package homerun

import "testing"

func TestResolveStream(t *testing.T) {
	rc := RedisConfig{Stream: "default-stream"}

	tests := []struct {
		name     string
		override []string
		want     string
	}{
		{"no override uses rc.Stream", nil, "default-stream"},
		{"non-empty override wins", []string{"releases"}, "releases"},
		{"empty-string override falls back", []string{""}, "default-stream"},
		{"multiple overrides use the first", []string{"first", "second"}, "first"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveStream(rc, tt.override...)
			if got != tt.want {
				t.Errorf("resolveStream(%v) = %q, want %q", tt.override, got, tt.want)
			}
		})
	}
}

func TestPitcherCloseReleasesTheClient(t *testing.T) {
	// NewPitcher does not dial, so this needs no server: it asserts that the
	// pool is owned and can be released.
	p := NewPitcher(RedisConfig{Addr: "127.0.0.1", Port: "1", Stream: "s"})

	if err := p.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}
}

func TestEnqueueOnUnreachableRedisReturnsError(t *testing.T) {
	// Port 1 is never a Redis. The point is that the failure is reported rather
	// than swallowed, and that the caller still gets the generated object ID.
	objectID, streamID, err := EnqueueMessageInRedisStreams(
		Message{Title: "unreachable", System: "test"},
		RedisConfig{Addr: "127.0.0.1", Port: "1", Stream: "messages"},
	)

	if err == nil {
		t.Fatal("expected an error for an unreachable Redis, got nil")
	}
	if objectID == "" {
		t.Error("expected the generated object ID to be returned alongside the error")
	}
	if streamID != "" {
		t.Errorf("expected no stream ID when the JSON write failed, got %q", streamID)
	}
}
