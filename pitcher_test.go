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
