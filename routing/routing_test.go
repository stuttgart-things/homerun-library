/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"errors"
	"slices"
	"strings"
	"testing"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

func TestParseStreams(t *testing.T) {
	cases := []struct {
		name, streams, stream, def string
		want                       []string
	}{
		{"REDIS_STREAMS wins", "a, b,,c ", "legacy", "messages", []string{"a", "b", "c"}},
		{"blank REDIS_STREAMS falls back", " , ", "legacy", "messages", []string{"legacy"}},
		{"REDIS_STREAM", "", "legacy", "messages", []string{"legacy"}},
		{"component default", "", "", "alerts", []string{"alerts"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseStreams(tc.streams, tc.stream, tc.def); !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// test1 is the homerun2-test1 stack from #122: everything on "messages",
// except demo-pitcher, which kept its base default "homerun".
func test1(t *testing.T) []Component {
	t.Helper()
	return []Component{
		{Name: "homerun2-omni-pitcher", Role: RolePitcher, Streams: []string{"messages"}},
		{Name: "homerun2-demo-pitcher", Role: RolePitcher, Streams: []string{"homerun"}},
		{Name: "homerun2-core-catcher", Role: RoleCatcher, Streams: []string{"messages"}, ConsumerGroup: "homerun2-core-catcher"},
		{Name: "homerun2-light-catcher", Role: RoleCatcher, Streams: []string{"messages"}, ConsumerGroup: "homerun2-light-catcher",
			Profile: mustLight(t, lightTestProfile)},
		{Name: "homerun2-led-catcher", Role: RoleCatcher, Streams: []string{"messages"}, ConsumerGroup: "homerun2-led-catcher",
			Profile: mustLED(t, ledTestProfile)},
	}
}

func delivery(t *testing.T, deliveries []Delivery, component string) Delivery {
	t.Helper()
	for _, d := range deliveries {
		if d.Component == component {
			return d
		}
	}
	t.Fatalf("no delivery for %s in %+v", component, deliveries)
	return Delivery{}
}

func TestDryRun(t *testing.T) {
	components := test1(t)
	msg := homerun.Message{Title: "Build failed", Severity: "ERROR", System: "github"}

	got := DryRun(components, "messages", msg)
	if len(got) != 3 {
		t.Fatalf("expected one delivery per catcher, got %+v", got)
	}

	if d := delivery(t, got, "homerun2-core-catcher"); !d.Receives || len(d.Reactions) != 1 || d.Reactions[0].Summary != "receives every message" {
		t.Errorf("core-catcher: %+v", d)
	}
	if d := delivery(t, got, "homerun2-light-catcher"); len(d.Reactions) != 1 || d.Reactions[0].Rule != "error-git" {
		t.Errorf("light-catcher: %+v", d)
	}
	if d := delivery(t, got, "homerun2-led-catcher"); len(d.Reactions) != 1 || d.Reactions[0].Rule != "github-error" {
		t.Errorf("led-catcher: %+v", d)
	}

	// What demo-pitcher published on test1: nobody reads it.
	for _, d := range DryRun(components, "homerun", msg) {
		if d.Receives || len(d.Reactions) != 0 {
			t.Errorf("%s must not receive a message on stream homerun: %+v", d.Component, d)
		}
	}
}

func TestDryRun_SharedConsumerGroup(t *testing.T) {
	components := []Component{
		{Name: "a", Role: RoleCatcher, Streams: []string{"messages"}, ConsumerGroup: "g"},
		{Name: "b", Role: RoleCatcher, Streams: []string{"messages", "alerts"}, ConsumerGroup: "g"},
		{Name: "c", Role: RoleCatcher, Streams: []string{"alerts"}, ConsumerGroup: "g"},
		{Name: "d", Role: RoleCatcher, Streams: []string{"messages"}},
	}
	got := DryRun(components, "messages", homerun.Message{})
	if d := delivery(t, got, "a"); !slices.Equal(d.SharedWith, []string{"b"}) {
		t.Errorf("a shares the group on messages with b only: %v", d.SharedWith)
	}
	if d := delivery(t, got, "d"); d.SharedWith != nil {
		t.Errorf("an unknown group is not shared: %v", d.SharedWith)
	}
	if d := delivery(t, got, "c"); d.Receives || d.SharedWith != nil {
		t.Errorf("c does not read messages: %+v", d)
	}
}

func TestDryRun_InvalidProfile(t *testing.T) {
	components := []Component{{
		Name: "light", Role: RoleCatcher, Streams: []string{"messages"},
		Profile: InvalidProfile{Err: errors.New("yaml: line 3: bad")},
	}}
	d := DryRun(components, "messages", homerun.Message{Severity: "error"})[0]
	if len(d.Reactions) != 1 || !strings.Contains(d.Reactions[0].Problem, "line 3") {
		t.Errorf("got %+v", d)
	}
}

func TestBuildMatrix(t *testing.T) {
	m := BuildMatrix(test1(t), "messages", nil)

	if !slices.Equal(m.Catchers, []string{"homerun2-core-catcher", "homerun2-light-catcher", "homerun2-led-catcher"}) {
		t.Errorf("catchers = %v", m.Catchers)
	}
	if want := []string{"gitlab", "github", "scale", OtherSystem}; !slices.Equal(m.Systems, want) {
		t.Errorf("systems = %v, want %v", m.Systems, want)
	}
	if len(m.Cells) != len(m.Systems)*len(Severities) {
		t.Errorf("got %d cells, want %d", len(m.Cells), len(m.Systems)*len(Severities))
	}

	// The gap from #122: an error from an unnamed system lights the WLED
	// nowhere and shows nothing on the LED matrix.
	cell, ok := m.Cell(OtherSystem, "error")
	if !ok {
		t.Fatal("no cell for (other)/error")
	}
	for _, name := range []string{"homerun2-light-catcher", "homerun2-led-catcher"} {
		if d := delivery(t, cell.Deliveries, name); len(d.Reactions) != 0 {
			t.Errorf("%s reacts to (other)/error: %+v", name, d.Reactions)
		}
	}

	cell, _ = m.Cell("github", "error")
	if d := delivery(t, cell.Deliveries, "homerun2-led-catcher"); len(d.Reactions) != 1 || d.Reactions[0].Rule != "github-error" {
		t.Errorf("led-catcher github/error: %+v", d)
	}

	if _, ok := m.Cell("nope", "error"); ok {
		t.Error("Cell must report a missing cell")
	}
}

func TestCheck(t *testing.T) {
	components := test1(t)
	components = append(components,
		Component{Name: "second-core", Role: RoleCatcher, Streams: []string{"messages"}, ConsumerGroup: "homerun2-core-catcher"},
		Component{Name: "broken-light", Role: RoleCatcher, Streams: []string{"messages"},
			Profile: mustLight(t, "effects:\n  e:\n    systems: [\"*\"]\n    severity: [error, critical]\n    tags: [x]\n    fx: Sparkle\n    color: red\n    endpoint: http://w\n")},
	)

	findings := Check(components, []string{"error", "critical"})

	byKind := map[FindingKind][]Finding{}
	for _, f := range findings {
		byKind[f.Kind] = append(byKind[f.Kind], f)
	}

	if f := byKind[FindingUnreadStream]; len(f) != 1 || f[0].Component != "homerun2-demo-pitcher" || f[0].Stream != "homerun" {
		t.Errorf("unread-stream: %+v", f)
	}
	if f := byKind[FindingSharedConsumerGroup]; len(f) != 1 || !strings.Contains(f[0].Message, "homerun2-core-catcher, second-core") {
		t.Errorf("shared-consumer-group: %+v", f)
	}

	var uncovered []string
	for _, f := range byKind[FindingUncoveredSeverity] {
		uncovered = append(uncovered, f.Component+"="+strings.Join(f.Severities, "+"))
	}
	// broken-light's rule needs a tag, so it does not react to a plain error either.
	want := []string{"homerun2-light-catcher=error+critical", "homerun2-led-catcher=error+critical", "broken-light=error+critical"}
	if !slices.Equal(uncovered, want) {
		t.Errorf("uncovered-severity = %v, want %v", uncovered, want)
	}

	// Found although no plain message reaches the rule.
	if f := byKind[FindingBrokenRule]; len(f) != 1 || f[0].Component != "broken-light" || f[0].Rule != "e" || !strings.Contains(f[0].Message, "Sparkle") {
		t.Errorf("broken-rule: %+v", f)
	}
}

func TestCheck_Clean(t *testing.T) {
	components := []Component{
		{Name: "pitcher", Role: RolePitcher, Streams: []string{"messages"}},
		{Name: "core", Role: RoleCatcher, Streams: []string{"messages"}, ConsumerGroup: "core"},
		{Name: "led", Role: RoleCatcher, Streams: []string{"messages"}, ConsumerGroup: "led", Profile: mustLED(t, ledShippedProfile)},
	}
	if findings := Check(components, []string{"error", "critical", "warning"}); len(findings) != 0 {
		t.Errorf("expected no findings, got %+v", findings)
	}
}
