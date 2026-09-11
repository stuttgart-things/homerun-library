/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"strings"
	"testing"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

// Cases in this file are ported from homerun2-led-catcher tests/test_profile.py
// and tests/test_default_profile.py.

// ledTestProfile is homerun2-led-catcher tests/profile.yaml.
const ledTestProfile = `displayRules:
  github-error:
    systems: [github, gitlab]
    severity: [ERROR]
    kind: gif
    image: sunset.gif
    duration: 5

  scale-weight:
    systems: [scale]
    severity: [INFO]
    kind: static
    text: "{{ message | replace('WEIGHT: ','') }}g"
    font: 6x10.bdf
    duration: 3

  warning-all:
    systems: ["*"]
    severity: [WARNING]
    kind: text
    text: "{{ system }}: {{ title }}"
    font: 6x10.bdf
    duration: 5

  default-info:
    systems: ["*"]
    severity: [INFO, SUCCESS]
    kind: text
    text: "{{ system }}: {{ title }}"
    font: 6x10.bdf
    duration: 5

colors:
  error: [255, 0, 0]
  warning: [255, 165, 0]
  success: [0, 255, 0]
  info: [0, 100, 255]
  debug: [128, 128, 128]
`

// ledShippedProfile is the profileData default of homerun2-led-catcher
// kcl/schema.k, the profile the kustomize base ships.
const ledShippedProfile = `displayRules:
  error-all:
    systems: ["*"]
    severity: [ERROR, CRITICAL]
    kind: text
    text: "{{ system }}: {{ title }}"
    font: 6x10.bdf
    duration: 5
  warning-all:
    systems: ["*"]
    severity: [WARNING]
    kind: text
    text: "{{ system }}: {{ title }}"
    font: 6x10.bdf
    duration: 5
  tabletennis-score:
    systems: [tabletennis]
    severity: [INFO, SUCCESS]
    kind: static
    text: "{{ title }}"
    font: 6x10.bdf
    hold: true
    duration: 3
  default-info:
    systems: ["*"]
    severity: [INFO, SUCCESS]
    kind: text
    text: "{{ system }}: {{ title }}"
    font: 6x10.bdf
    duration: 5
colors:
  error: [255, 0, 0]
  critical: [255, 0, 0]
  warning: [255, 165, 0]
  success: [0, 255, 0]
  info: [0, 100, 255]
  debug: [128, 128, 128]
`

func mustLED(t *testing.T, doc string) *LEDProfile {
	t.Helper()
	p, err := ParseLEDProfile([]byte(doc))
	if err != nil {
		t.Fatalf("ParseLEDProfile: %v", err)
	}
	return p
}

func ledMatch(t *testing.T, p *LEDProfile, msg homerun.Message) (*LEDDisplay, Reaction) {
	t.Helper()
	r := p.Evaluate(msg)
	if len(r) == 0 {
		return nil, Reaction{}
	}
	d := r[0].Details.(LEDDisplay)
	return &d, r[0]
}

func TestParseLEDProfile(t *testing.T) {
	p := mustLED(t, ledTestProfile)
	var names []string
	for _, r := range p.Rules {
		names = append(names, r.Name)
	}
	if got := strings.Join(names, ","); got != "github-error,scale-weight,warning-all,default-info" {
		t.Errorf("rules = %s, want document order", got)
	}
	if p.Colors["error"] != [3]int{255, 0, 0} {
		t.Errorf("error color = %v", p.Colors["error"])
	}
	if got := p.Rules[0].Severity; len(got) != 1 || got[0] != "error" {
		t.Errorf("severity not lower-cased: %v", got)
	}
}

func TestParseLEDProfile_Empty(t *testing.T) {
	for _, doc := range []string{"", "---\n", "{}\n", "[]\n"} {
		p := mustLED(t, doc)
		if len(p.Rules) != 0 {
			t.Errorf("%q: expected no rules", doc)
		}
		if _, ok := p.Colors["error"]; !ok {
			t.Errorf("%q: default colors should still be present", doc)
		}
		if r := p.Evaluate(homerun.Message{Title: "test", Severity: "info", System: "github"}); len(r) != 0 {
			t.Errorf("%q: empty profile matched: %v", doc, r)
		}
	}
}

func TestParseLEDProfile_Defaults(t *testing.T) {
	p := mustLED(t, "displayRules:\n  bare:\n    systems: [\"*\"]\n")
	r := p.Rules[0]
	if r.Kind != "text" || r.Font != "6x10.bdf" || r.Duration != 5 || r.Hold || len(r.Severity) != 0 {
		t.Errorf("defaults not applied: %+v", r)
	}
}

func TestParseLEDProfile_Rejects(t *testing.T) {
	cases := map[string]string{
		"invalid YAML":             "displayRules: [unclosed",
		"top level is a list":      "- a\n- b\n",
		"displayRules null":        "displayRules:\n",
		"rule is not a mapping":    "displayRules:\n  a: text\n",
		"severity null":            "displayRules:\n  a:\n    systems: [\"*\"]\n    severity:\n",
		"duration is not a number": "displayRules:\n  a:\n    duration: 5s\n",
		"colors is a list":         "colors: [1, 2, 3]\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseLEDProfile([]byte(doc)); err == nil {
				t.Error("expected an error, led-catcher cannot load this profile")
			}
		})
	}
}

func TestParseLEDProfile_PythonReadings(t *testing.T) {
	p := mustLED(t, `displayRules:
  a:
    systems: [x]
    duration: "2.5"
    hold: "false"
  b:
    systems: [x]
    hold: no
  a:
    systems: [y]
    hold: 1
colors:
  info: [1, 2]
  warning: [1, 2, 3]
`)
	if len(p.Rules) != 2 || p.Rules[0].Name != "a" || p.Rules[1].Name != "b" {
		t.Fatalf("a duplicate rule must keep its first position: %+v", p.Rules)
	}
	if a := p.Rules[0]; a.Systems[0] != "y" || !a.Hold || a.Duration != 5 {
		t.Errorf("a duplicate rule must take its last definition: %+v", a)
	}
	if p.Rules[1].Hold {
		t.Error(`hold: no is false in PyYAML`)
	}
	if p.Colors["info"] != ledDefaultColors["info"] {
		t.Error("a color that is not three values must be ignored")
	}
	if p.Colors["warning"] != [3]int{1, 2, 3} {
		t.Error("a profile color must override the default")
	}

	quoted := mustLED(t, "displayRules:\n  a:\n    systems: [x]\n    duration: \"2.5\"\n    hold: \"false\"\n")
	if r := quoted.Rules[0]; r.Duration != 2.5 || !r.Hold {
		t.Errorf(`float("2.5") is 2.5 and bool("false") is True: %+v`, r)
	}
}

func TestLEDEvaluate_TestProfile(t *testing.T) {
	p := mustLED(t, ledTestProfile)

	t.Run("github error", func(t *testing.T) {
		d, _ := ledMatch(t, p, homerun.Message{Title: "Build failed", Severity: "error", System: "github"})
		if d == nil || d.Kind != "gif" || d.Image != "sunset.gif" || d.Color != [3]int{255, 0, 0} {
			t.Errorf("got %+v", d)
		}
	})
	t.Run("gitlab error, severity upper-case", func(t *testing.T) {
		d, _ := ledMatch(t, p, homerun.Message{Title: "Pipeline failed", Severity: "ERROR", System: "gitlab"})
		if d == nil || d.Kind != "gif" {
			t.Errorf("got %+v", d)
		}
	})
	t.Run("system is case-insensitive", func(t *testing.T) {
		d, _ := ledMatch(t, p, homerun.Message{Severity: "error", System: "GitHub"})
		if d == nil || d.Name != "github-error" {
			t.Errorf("got %+v", d)
		}
	})
	t.Run("scale info uses a Jinja2 filter", func(t *testing.T) {
		d, _ := ledMatch(t, p, homerun.Message{Title: "Weight", Message: "WEIGHT: 42", Severity: "info", System: "scale"})
		if d == nil || d.Kind != "static" {
			t.Fatalf("got %+v", d)
		}
		// led-catcher renders "42g"; a filter is not rendered here, and says so.
		if d.TextRendered || d.Text != "{{ message | replace('WEIGHT: ','') }}g" {
			t.Errorf("text = %q rendered=%v, want the raw template, unrendered", d.Text, d.TextRendered)
		}
	})
	t.Run("wildcard warning", func(t *testing.T) {
		d, _ := ledMatch(t, p, homerun.Message{Title: "Disk full", Severity: "warning", System: "ansible"})
		if d == nil || d.Kind != "text" || d.Text != "ansible: Disk full" || !d.TextRendered || d.Color != [3]int{255, 165, 0} {
			t.Errorf("got %+v", d)
		}
	})
	t.Run("wildcard info", func(t *testing.T) {
		d, _ := ledMatch(t, p, homerun.Message{Title: "Deploy OK", Severity: "info", System: "flux"})
		if d == nil || d.Text != "flux: Deploy OK" {
			t.Errorf("got %+v", d)
		}
	})
	t.Run("no match", func(t *testing.T) {
		if d, _ := ledMatch(t, p, homerun.Message{Title: "Debug trace", Severity: "debug", System: "internal"}); d != nil {
			t.Errorf("got %+v", d)
		}
	})
	t.Run("error from another system is not displayed", func(t *testing.T) {
		// The gap the shipped default had until 2026-09-05.
		if d, _ := ledMatch(t, p, homerun.Message{Severity: "critical", System: "k8s"}); d != nil {
			t.Errorf("got %+v", d)
		}
	})
}

func TestLEDEvaluate_MessageDefaults(t *testing.T) {
	// led-catcher reads a missing severity as info and a missing system as
	// unknown - and homerun.Message omits empty fields on the wire.
	p := mustLED(t, "displayRules:\n  unknown-info:\n    systems: [unknown]\n    severity: [info]\n    text: \"{{ system }}/{{ severity }}/{{ author }}\"\n")
	d, _ := ledMatch(t, p, homerun.Message{})
	if d == nil || d.Text != "unknown/info/unknown" {
		t.Errorf("got %+v", d)
	}
}

func TestLEDEvaluate_EmptySeverityMatchesAny(t *testing.T) {
	p := mustLED(t, "displayRules:\n  any:\n    systems: [\"*\"]\n")
	if d, _ := ledMatch(t, p, homerun.Message{Severity: "whatever", System: "x"}); d == nil {
		t.Error("a rule without severity must match every severity")
	}
}

func TestLEDEvaluate_WhiteIsAProblem(t *testing.T) {
	p := mustLED(t, "displayRules:\n  all:\n    systems: [\"*\"]\n    severity: [critical, error]\n")

	d, r := ledMatch(t, p, homerun.Message{Severity: "CRITICAL", System: "x"})
	if d == nil || d.Color != ledWhite || !strings.Contains(r.Problem, "renders white") {
		t.Errorf("critical has no default color: got %+v, problem %q", d, r.Problem)
	}
	if _, r := ledMatch(t, p, homerun.Message{Severity: "error", System: "x"}); r.Problem != "" {
		t.Errorf("error has a default color, got problem %q", r.Problem)
	}
	if got := p.problems(); len(got) != 1 || got[0].Rule != "all" {
		t.Errorf("problems() = %+v", got)
	}
}

func TestLEDEvaluate_ShippedProfile(t *testing.T) {
	p := mustLED(t, ledShippedProfile)

	for _, sev := range []string{"error", "critical", "warning", "info", "success"} {
		d, r := ledMatch(t, p, homerun.Message{Title: "t", Severity: sev, System: "any-system"})
		if d == nil {
			t.Errorf("severity %s: no rule", sev)
			continue
		}
		if d.Color == ledWhite || r.Problem != "" {
			t.Errorf("severity %s renders white: %q", sev, r.Problem)
		}
	}

	info, _ := ledMatch(t, p, homerun.Message{Title: "t", Severity: "info", System: "s"})
	critical, _ := ledMatch(t, p, homerun.Message{Title: "t", Severity: "critical", System: "s"})
	if info.Color == critical.Color {
		t.Error("critical shares info's color")
	}

	for _, sev := range []string{"info", "success"} {
		d, _ := ledMatch(t, p, homerun.Message{Title: "SET 1:0", Severity: sev, System: "tabletennis"})
		if d == nil || d.Name != "tabletennis-score" || d.Kind != "static" || !d.Hold || d.Text != "SET 1:0" {
			t.Errorf("tabletennis %s: got %+v", sev, d)
		}
	}

	for _, system := range []string{"github", "k8s", "scale", "demo"} {
		d, _ := ledMatch(t, p, homerun.Message{Title: "t", Severity: "info", System: system})
		if d == nil || d.Kind != "text" || d.Text != system+": t" {
			t.Errorf("%s: got %+v", system, d)
		}
	}

	d, _ := ledMatch(t, p, homerun.Message{Title: "boom", Severity: "error", System: "tabletennis"})
	if d == nil || d.Kind != "text" || d.Hold {
		t.Errorf("tabletennis error must fall through to error-all: got %+v", d)
	}
}

func TestRenderLEDText(t *testing.T) {
	msg := homerun.Message{Title: "T", System: "S", URL: "http://u"}
	cases := []struct {
		text, want   string
		wantRendered bool
	}{
		{"{{ system }}: {{ title }}", "S: T", true},
		{"{{title}}", "T", true},
		{"{{ url }}", "http://u", true},
		{"{{ undefined }}x", "x", true},
		{"plain", "plain", true},
		{"{{ title }}\n", "T", true},
		{"{{ title | upper }}", "{{ title | upper }}", false},
		{"{% if title %}x{% endif %}", "{% if title %}x{% endif %}", false},
	}
	for _, tc := range cases {
		got, rendered := renderLEDText(tc.text, msg)
		if got != tc.want || rendered != tc.wantRendered {
			t.Errorf("renderLEDText(%q) = %q, %v; want %q, %v", tc.text, got, rendered, tc.want, tc.wantRendered)
		}
	}
}

func TestLEDSummary(t *testing.T) {
	cases := []struct {
		rule LEDRule
		want string
	}{
		{LEDRule{Kind: "static", Text: "7:5", Hold: true, Duration: 3}, `LED static "7:5" in rgb(0,100,255) until replaced`},
		{LEDRule{Kind: "text", Text: "x", Hold: true, Duration: 5}, `LED text "x" in rgb(0,100,255) scrolling`},
		{LEDRule{Kind: "gif", Image: "sunset.gif", Hold: true, Duration: 5}, `LED gif sunset.gif in rgb(0,100,255) for 5s`},
	}
	for _, tc := range cases {
		if got := ledSummary(LEDDisplay{LEDRule: tc.rule, Color: [3]int{0, 100, 255}}); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}
