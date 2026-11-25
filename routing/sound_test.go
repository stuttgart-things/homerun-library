/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"strings"
	"testing"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

// Cases in this file are ported from homerun2-sound-catcher
// tests/test_profile.py.

// soundShippedProfile is homerun2-sound-catcher profile.yaml.
const soundShippedProfile = `quietHours:
  from: "18:00"
  to: "08:00"
  weekends: true
  timezone: Europe/Berlin
  allow: [critical]

rules:
  critical:
    severity: [critical]
    sound: alarm.wav
    say: "Critical{% if system not in spoken_title %} on {{ system }}{% endif %}: {{ spoken_title }}"
    volume: 100

  errors:
    severity: [error]
    sound: chime.wav
    say: "Error{% if system not in spoken_title %} on {{ system }}{% endif %}: {{ spoken_title }}"
    volume: 85

  warnings:
    severity: [warning]
    sound: ding.wav
    say: "Warning{% if system not in spoken_title %} from {{ system }}{% endif %}: {{ spoken_title }}"
    volume: 70

  success:
    severity: [success]
    sound: success.wav
    volume: 60

  quiet:
    severity: [info, debug]
    silent: true
`

// soundTimerRule is the sound-catcher rule of homerun2-timer-pitcher (#147).
const soundTimerRule = `  timer:
    systems: [timer]
    severity: [warning]
    sound: alarm.wav
    say: "{{ title }}"
`

func mustSound(t *testing.T, doc string) *SoundProfile {
	t.Helper()
	p, err := ParseSoundProfile([]byte(doc))
	if err != nil {
		t.Fatalf("ParseSoundProfile: %v", err)
	}
	return p
}

func soundMatch(t *testing.T, p *SoundProfile, msg homerun.Message) (*SoundAction, Reaction) {
	t.Helper()
	r := p.Evaluate(msg)
	if len(r) == 0 {
		return nil, Reaction{}
	}
	a := r[0].Details.(SoundAction)
	return &a, r[0]
}

func TestSoundEvaluate_FirstMatchWinsAndFiltersCombine(t *testing.T) {
	p := mustSound(t, `rules:
  tagged: {systems: [ci], tags: [prod], say: prod}
  ci: {systems: [ci], severity: [error], say: ci}
  any: {systems: ["*"], say: any}
`)
	for _, tc := range []struct {
		msg  homerun.Message
		want string
	}{
		{homerun.Message{System: "ci", Tags: "prod, nightly"}, "tagged"},
		{homerun.Message{System: "ci", Severity: "ERROR"}, "ci"},
		{homerun.Message{System: "ci", Severity: "info"}, "any"},
	} {
		if a, _ := soundMatch(t, p, tc.msg); a == nil || a.Name != tc.want {
			t.Errorf("%+v: got %+v, want %s", tc.msg, a, tc.want)
		}
	}
}

func TestSoundEvaluate_SystemsAreExactAndEmptyMatchesAny(t *testing.T) {
	p := mustSound(t, "rules:\n  ci: {systems: [ci], say: ci}\n  rest: {say: rest}\n")
	if a, _ := soundMatch(t, p, homerun.Message{System: "CI"}); a == nil || a.Name != "rest" {
		t.Errorf("sound-catcher compares systems case-sensitively: got %+v", a)
	}
	if a, _ := soundMatch(t, p, homerun.Message{}); a == nil || a.Name != "rest" {
		t.Errorf("a rule without systems matches any: got %+v", a)
	}
	if got := p.Systems(); len(got) != 1 || got[0] != "ci" {
		t.Errorf("Systems() = %v", got)
	}
}

func TestSoundEvaluate_RenderFlattensAndCapsText(t *testing.T) {
	p := mustSound(t, "rules:\n  r:\n    say: \"{{ title }}\\n\\n{{ message }}\"\n")
	if a, _ := soundMatch(t, p, homerun.Message{Title: "a", Message: "b   c"}); a == nil || a.Said != "a b c" || !a.SayRendered {
		t.Errorf("got %+v", a)
	}
	if a, _ := soundMatch(t, p, homerun.Message{Title: strings.Repeat("x", 2000)}); a == nil || len(a.Said) != 500 {
		t.Errorf("not cut at 500")
	}
}

func TestSoundEvaluate_UndefinedNameIsAProblem(t *testing.T) {
	for _, say := range []string{"{{ nope }}", "{{ tag('until') }}", "{{ timestamp }}"} {
		p := mustSound(t, "rules:\n  r:\n    say: \""+say+"\"\n")
		a, r := soundMatch(t, p, homerun.Message{})
		if a == nil || a.SayRendered || !strings.Contains(r.Problem, "nothing is said: nothing plays") {
			t.Errorf("%s: got %+v, problem %q", say, a, r.Problem)
		}
		if got := p.problems(); len(got) != 1 || got[0].Rule != "r" {
			t.Errorf("%s: problems() = %+v", say, got)
		}
	}
	// With a clip the clip still plays.
	p := mustSound(t, "rules:\n  r:\n    sound: a.wav\n    say: \"{{ nope }}\"\n")
	if _, r := soundMatch(t, p, homerun.Message{}); strings.Contains(r.Problem, "nothing plays") || r.Problem == "" {
		t.Errorf("problem %q", r.Problem)
	}
}

func TestSoundEvaluate_TemplateWithStatementsIsNotRendered(t *testing.T) {
	p := mustSound(t, soundShippedProfile)
	a, r := soundMatch(t, p, homerun.Message{Title: "[WARN] disk full", System: "k8s", Severity: "warning"})
	if a == nil || a.Name != "warnings" || a.SayRendered || r.Problem != "" {
		t.Fatalf("got %+v, problem %q", a, r.Problem)
	}
	if !strings.HasPrefix(r.Summary, `sound: plays ding.wav at 70%, then says "Warning{% if`) {
		t.Errorf("summary = %q", r.Summary)
	}
}

func TestParseSoundProfile_InvalidRulesFailLoading(t *testing.T) {
	for raw, want := range map[string]string{
		"{sound: ../etc/passwd}":      "file name",
		"{sound: a.wav, volume: 101}": "volume",
		"{systems: [x]}":              "needs sound",
		"":                            "needs sound", // `raw or {}`
		"{say: x, tags: null}":        "tags",
		"{say: x, volume: loud}":      "volume",
	} {
		_, err := ParseSoundProfile([]byte("rules:\n  bad: " + raw + "\n"))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want %q", raw, err, want)
		}
	}
}

func TestParseSoundProfile_Values(t *testing.T) {
	p := mustSound(t, `quietHours:
  from: 19:00
  to: 1140
rules:
  r: {say: x, volume: "70", severity: [ERROR], silent: "no"}
`)
	q := p.QuietHours
	if q == nil || q.From != "19:00" || q.To != "19:00" || len(q.Allow) != 1 || q.Allow[0] != "critical" {
		t.Errorf("quietHours = %+v", q)
	}
	r := p.Rules[0]
	if r.Volume != 70 || r.Severity[0] != "error" || !r.Silent {
		// PyYAML reads the quoted "no" as a non-empty string: truthy.
		t.Errorf("rule = %+v", r)
	}
	if _, err := ParseSoundProfile([]byte("quietHours:\n  from: \"18:00\"\nrules: {}\n")); err == nil {
		t.Error("quietHours without to fails loading in sound-catcher")
	}
}

func TestParseSoundProfile_Empty(t *testing.T) {
	for _, doc := range []string{"", "rules:\n", "rules: {}\n"} {
		p := mustSound(t, doc)
		if len(p.Rules) != 0 || p.Evaluate(homerun.Message{Severity: "critical"}) != nil {
			t.Errorf("%q: %+v", doc, p)
		}
	}
}

func TestSoundEvaluate_ShippedProfile(t *testing.T) {
	p := mustSound(t, soundShippedProfile)

	a, r := soundMatch(t, p, homerun.Message{Title: "down", System: "k8s", Severity: "critical"})
	if a == nil || a.Name != "critical" || a.QuietHoursSilence || strings.Contains(r.Summary, "quiet hours") {
		t.Errorf("critical is allowed in quiet hours: %+v %q", a, r.Summary)
	}
	a, r = soundMatch(t, p, homerun.Message{Title: "ok", System: "ci", Severity: "success"})
	want := "sound: plays success.wav at 60%; silent in quiet hours 18:00-08:00 and weekends (Europe/Berlin), which allow only critical"
	if a == nil || !a.QuietHoursSilence || r.Summary != want {
		t.Errorf("summary = %q, want %q", r.Summary, want)
	}
	a, r = soundMatch(t, p, homerun.Message{System: "radio"})
	if a == nil || a.Name != "quiet" || !a.Silent || r.Summary != "sound: silent, matches and plays nothing" {
		t.Errorf("info: %+v %q", a, r.Summary)
	}
	if got := p.problems(); len(got) != 0 {
		t.Errorf("problems() = %+v", got)
	}
}

// The timer rule of #147 rings for a timer, but sound-catcher's quiet hours
// allow severities profile-wide: a warning timer stays silent in them.
func TestSoundEvaluate_TimerRule(t *testing.T) {
	doc := strings.Replace(soundShippedProfile, "rules:\n", "rules:\n"+soundTimerRule, 1)
	p := mustSound(t, doc)
	a, r := soundMatch(t, p, homerun.Message{Title: "Coffee is ready", System: "timer", Severity: "warning", Tags: "timer=abc"})
	if a == nil || a.Name != "timer" || a.Said != "Coffee is ready" || !a.QuietHoursSilence {
		t.Fatalf("got %+v", a)
	}
	if !strings.HasPrefix(r.Summary, `sound: plays alarm.wav at 100%, then says "Coffee is ready"; silent in quiet hours`) {
		t.Errorf("summary = %q", r.Summary)
	}
}

func TestSpokenTitle(t *testing.T) {
	for title, want := range map[string]string{
		"[WARN] k8s-cluster disk usage above 80%": "k8s-cluster disk usage above 80%",
		"[OK] [CI] redis pipeline passed":         "redis pipeline passed",
		"disk [80%] full":                         "disk [80%] full",
		"[WARN]":                                  "[WARN]",
	} {
		if got := spokenTitle(title); got != want {
			t.Errorf("spokenTitle(%q) = %q, want %q", title, got, want)
		}
	}
}
