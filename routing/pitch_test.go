/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

// Cases for StreamRoutes are ported from homerun2-omni-pitcher
// internal/routing/router_test.go and loader_test.go, and those for
// PreparePitch from internal/handlers/pitch.go, so both are held to
// omni-pitcher's own expectations.

func TestParseStreamRoutes(t *testing.T) {
	r, err := ParseStreamRoutes([]byte(`
streams:
  - messages
  - github-events
default_stream: messages
routes:
  - match: { endpoint: /pitch/github }
    stream: github-events
`))
	if err != nil {
		t.Fatalf("ParseStreamRoutes() unexpected error: %v", err)
	}
	if r.DefaultStream != "messages" {
		t.Errorf("default_stream = %q, want %q", r.DefaultStream, "messages")
	}
	if len(r.Routes) != 1 || r.Routes[0].Stream != "github-events" || r.Routes[0].Match.Endpoint != "/pitch/github" {
		t.Errorf("unexpected routes: %#v", r.Routes)
	}
}

func TestParseStreamRoutesRejects(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{"invalid yaml", "streams: [oops\n", "parse routes"},
		{"empty file", "", "streams must be non-empty"},
		{"empty streams", "default_stream: x\n", "streams must be non-empty"},
		{"empty stream entry", "streams: [a, '']\ndefault_stream: a\n", "streams contains an empty entry"},
		{"duplicate stream", "streams: [a, a]\ndefault_stream: a\n", `streams contains duplicate "a"`},
		{"no default", "streams: [a]\n", "default_stream is required"},
		{"default not in allowlist", "streams: [a]\ndefault_stream: b\n", `default_stream "b" is not in streams allowlist`},
		{
			"route stream not in allowlist",
			"streams: [a]\ndefault_stream: a\nroutes:\n  - match: {endpoint: /x}\n    stream: b\n",
			`routes[0]: stream "b" is not in streams allowlist`,
		},
		{
			"route without matchers",
			"streams: [a]\ndefault_stream: a\nroutes:\n  - stream: a\n",
			"routes[0]: at least one matcher is required",
		},
		{
			"route without stream",
			"streams: [a]\ndefault_stream: a\nroutes:\n  - match: {system: s}\n",
			"routes[0]: stream is required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseStreamRoutes([]byte(tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestStreamRoutesValidateMinimal(t *testing.T) {
	r := StreamRoutes{Streams: []string{"a"}, DefaultStream: "a"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveFirstMatchWins(t *testing.T) {
	r := &StreamRoutes{
		Streams:       []string{"messages", "github-events", "grafana-alerts"},
		DefaultStream: "messages",
		Routes: []StreamRoute{
			{Match: RouteMatch{Endpoint: PitchPathGitHub}, Stream: "github-events"},
			{Match: RouteMatch{System: "grafana"}, Stream: "grafana-alerts"},
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cases := []struct {
		endpoint   string
		msg        homerun.Message
		wantStream string
		wantRule   int
	}{
		{PitchPathGitHub, homerun.Message{System: "grafana"}, "github-events", 0},
		{PitchPathGrafana, homerun.Message{System: "grafana"}, "grafana-alerts", 1},
		{PitchPath, homerun.Message{}, "messages", -1},
	}
	for _, tc := range cases {
		stream, rule := r.Resolve(tc.endpoint, tc.msg)
		if stream != tc.wantStream || rule != tc.wantRule {
			t.Errorf("Resolve(%q, %+v) = %q, %d; want %q, %d", tc.endpoint, tc.msg, stream, rule, tc.wantStream, tc.wantRule)
		}
	}
}

func TestResolveANDWithinRule(t *testing.T) {
	r := &StreamRoutes{
		Streams:       []string{"a", "b"},
		DefaultStream: "a",
		Routes:        []StreamRoute{{Match: RouteMatch{Endpoint: PitchPath, System: "grafana"}, Stream: "b"}},
	}
	if got, _ := r.Resolve(PitchPath, homerun.Message{System: "grafana"}); got != "b" {
		t.Errorf("both matchers should match: got %q", got)
	}
	if got, _ := r.Resolve(PitchPath, homerun.Message{System: "github"}); got != "a" {
		t.Errorf("one matcher mismatched should fall through: got %q", got)
	}
}

func TestResolveNil(t *testing.T) {
	var r *StreamRoutes
	if stream, rule := r.Resolve(PitchPath, homerun.Message{}); stream != "" || rule != -1 {
		t.Errorf("nil StreamRoutes.Resolve = %q, %d; want \"\", -1", stream, rule)
	}
	if got := r.Targets(); got != nil {
		t.Errorf("nil StreamRoutes.Targets = %v, want nil", got)
	}
}

func TestRouteMatchMatches(t *testing.T) {
	cases := []struct {
		name  string
		match RouteMatch
		ep    string
		msg   homerun.Message
		hit   bool
	}{
		{"endpoint substring", RouteMatch{Endpoint: "/pitch/github"}, "/pitch/github", homerun.Message{}, true},
		{"endpoint substring partial", RouteMatch{Endpoint: "github"}, "/pitch/github", homerun.Message{}, true},
		{"endpoint no match", RouteMatch{Endpoint: "github"}, "/pitch/grafana", homerun.Message{}, false},
		{"system substring", RouteMatch{System: "graf"}, "/x", homerun.Message{System: "grafana"}, true},
		{"system is case-sensitive", RouteMatch{System: "Grafana"}, "/x", homerun.Message{System: "grafana"}, false},
		{"author substring", RouteMatch{Author: "depend"}, "/x", homerun.Message{Author: "dependabot[bot]"}, true},
		{"tag_contains", RouteMatch{TagContains: "release"}, "/x", homerun.Message{Tags: "ci,release,prod"}, true},
		{"tag_contains miss", RouteMatch{TagContains: "release"}, "/x", homerun.Message{Tags: "ci,prod"}, false},
		{"tag_contains spans tags", RouteMatch{TagContains: "ci,rel"}, "/x", homerun.Message{Tags: "ci,release"}, true},
		{"title_contains any", RouteMatch{TitleContains: []string{"error", "failure"}}, "/x", homerun.Message{Title: "build failure on main"}, true},
		{"title_contains none", RouteMatch{TitleContains: []string{"error", "failure"}}, "/x", homerun.Message{Title: "release 1.0"}, false},
		{"title_contains only empty entries", RouteMatch{TitleContains: []string{""}}, "/x", homerun.Message{Title: "anything"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.match.Matches(tc.ep, tc.msg); got != tc.hit {
				t.Errorf("Matches() = %v, want %v", got, tc.hit)
			}
		})
	}
}

func TestStreamRoutesTargets(t *testing.T) {
	r := &StreamRoutes{
		Streams:       []string{"messages", "tabletennis", "github-events", "unused"},
		DefaultStream: "messages",
		Routes: []StreamRoute{
			{Match: RouteMatch{System: "tabletennis"}, Stream: "tabletennis"},
			{Match: RouteMatch{Author: "bot"}, Stream: "messages"},
			{Match: RouteMatch{Endpoint: PitchPathGitHub}, Stream: "github-events"},
			{Match: RouteMatch{System: "table"}, Stream: "tabletennis"},
		},
	}
	want := []string{"messages", "tabletennis", "github-events"}
	if got := r.Targets(); !slices.Equal(got, want) {
		t.Errorf("Targets() = %v, want %v", got, want)
	}
}

// The routing file deployed on homerun2-test1, where the e2e run of
// homerun2-config-viewer#9 saw a tabletennis message leave messages.
func TestResolveHomerun2Test1Routes(t *testing.T) {
	r, err := ParseStreamRoutes([]byte(`streams:
  - messages
  - tabletennis
default_stream: messages
routes:
  - match:
      system: tabletennis
    stream: tabletennis
`))
	if err != nil {
		t.Fatal(err)
	}
	if stream, rule := r.Resolve(PitchPath, homerun.Message{System: "tabletennis", Severity: "info"}); stream != "tabletennis" || rule != 0 {
		t.Errorf("tabletennis: got %q, %d", stream, rule)
	}
	if stream, rule := r.Resolve(PitchPath, homerun.Message{System: "github", Severity: "error"}); stream != "messages" || rule != -1 {
		t.Errorf("github: got %q, %d", stream, rule)
	}
}

func TestRouteMatchSummary(t *testing.T) {
	cases := []struct {
		match RouteMatch
		want  string
	}{
		{RouteMatch{System: "tabletennis"}, `system contains "tabletennis"`},
		{RouteMatch{Endpoint: "/pitch/github", Author: "bot"}, `endpoint contains "/pitch/github" and author contains "bot"`},
		{RouteMatch{TagContains: "release", TitleContains: []string{"fail"}}, `tags contains "release" and title contains "fail"`},
		{RouteMatch{TitleContains: []string{"error", "failure"}}, `title contains any of "error", "failure"`},
		{RouteMatch{TitleContains: []string{""}}, `title contains any of ""`},
		{RouteMatch{}, "no matchers"},
	}
	for _, tc := range cases {
		if got := tc.match.Summary(); got != tc.want {
			t.Errorf("Summary(%+v) = %q, want %q", tc.match, got, tc.want)
		}
	}
}

func TestPreparePitch(t *testing.T) {
	now := time.Date(2026, 9, 12, 7, 7, 14, 0, time.UTC)

	cases := []struct {
		name          string
		msg           homerun.Message
		wantErr       error
		wantMsg       homerun.Message
		wantDefaulted []string
	}{
		{
			name:    "no title",
			msg:     homerun.Message{Message: "m"},
			wantErr: ErrPitchTitleRequired,
		},
		{
			name:    "title is checked before message",
			msg:     homerun.Message{},
			wantErr: ErrPitchTitleRequired,
		},
		{
			name:    "no message",
			msg:     homerun.Message{Title: "t"},
			wantErr: ErrPitchMessageRequired,
		},
		{
			name: "every default",
			msg:  homerun.Message{Title: "t", Message: "m", Tags: "a,b"},
			wantMsg: homerun.Message{
				Title: "t", Message: "m", Tags: "a,b",
				Severity: "info", Author: "unknown", Timestamp: "2026-09-12T07:07:14Z", System: OmniPitcherSystem,
			},
			wantDefaulted: []string{"severity", "author", "timestamp", "system"},
		},
		{
			name: "critical without system",
			msg:  homerun.Message{Title: "t", Message: "m", Severity: "critical", Author: "e2e"},
			wantMsg: homerun.Message{
				Title: "t", Message: "m", Severity: "critical", Author: "e2e",
				Timestamp: "2026-09-12T07:07:14Z", System: OmniPitcherSystem,
			},
			wantDefaulted: []string{"timestamp", "system"},
		},
		{
			name: "nothing to default",
			msg: homerun.Message{
				Title: "t", Message: "m", Severity: "error", Author: "a", Timestamp: "2020-01-01T00:00:00Z", System: "github",
			},
			wantMsg: homerun.Message{
				Title: "t", Message: "m", Severity: "error", Author: "a", Timestamp: "2020-01-01T00:00:00Z", System: "github",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := PreparePitch(tc.msg, now)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Message != tc.wantMsg {
				t.Errorf("message = %+v, want %+v", p.Message, tc.wantMsg)
			}
			if !slices.Equal(p.Defaulted, tc.wantDefaulted) {
				t.Errorf("defaulted = %v, want %v", p.Defaulted, tc.wantDefaulted)
			}
		})
	}
}
