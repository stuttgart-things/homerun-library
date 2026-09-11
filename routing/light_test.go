/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"strings"
	"testing"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

// Cases in this file are ported from homerun2-light-catcher
// internal/profile/profile_test.go, so the evaluator is held to the catcher's
// own expectations.

const lightTestProfile = `---
effects:
  error-git:
    systems:
      - gitlab
      - github
    severity:
      - ERROR
    fx: Blurz
    duration: 3
    color: sunset
    segments:
      - 0
    endpoint: http://wled:8080
  info:
    systems:
      - "*"
    severity:
      - INFO
    fx: DJ Light
    duration: 3
    color: ocean
    segments:
      - 0
    endpoint: http://localhost:8080
  success:
    systems:
      - "*"
    severity:
      - SUCCESS
    fx: Aurora
    duration: 3
    color: forest
    segments:
      - 0
    endpoint: http://localhost:8080
`

const lightTagProfile = `effects:
  tabletennis-match:
    systems: [tabletennis]
    severity: [success]
    tags: [transition=match_won]
    fx: Fireworks
    color: sunset
    endpoint: http://wled
  tabletennis-point-a:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point, side=a]
    fx: Solid
    color: blue
    endpoint: http://wled
  tabletennis-point-b:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point, side=b]
    fx: Solid
    color: red
    endpoint: http://wled
  success:
    systems: ["*"]
    severity: [success]
    fx: Aurora
    color: forest
    endpoint: http://wled
  info:
    systems: ["*"]
    severity: [info]
    fx: DJ Light
    color: ocean
    endpoint: http://wled
`

func mustLight(t *testing.T, doc string) *LightProfile {
	t.Helper()
	p, err := ParseLightProfile([]byte(doc))
	if err != nil {
		t.Fatalf("ParseLightProfile: %v", err)
	}
	return p
}

// lightMatch returns the matched effect, or nil.
func lightMatch(p *LightProfile, system, severity, tags string) *LightEffect {
	r := p.Evaluate(homerun.Message{System: system, Severity: severity, Tags: tags})
	if len(r) == 0 {
		return nil
	}
	e := r[0].Details.(LightEffect)
	return &e
}

func TestParseLightProfile_DocumentOrder(t *testing.T) {
	p := mustLight(t, lightTestProfile)
	var names []string
	for _, e := range p.Effects {
		names = append(names, e.Name)
	}
	if got := strings.Join(names, ","); got != "error-git,info,success" {
		t.Errorf("effects = %s, want document order", got)
	}
	if e := p.Effects[0]; e.Fx != "Blurz" || e.Duration != 3 || e.Endpoint != "http://wled:8080" || len(e.Segments) != 1 {
		t.Errorf("error-git decoded as %+v", e)
	}
}

func TestParseLightProfile_Rejects(t *testing.T) {
	cases := map[string]string{
		"invalid YAML":        "effects: [unclosed",
		"duration not an int": "effects:\n  a:\n    duration: 3s\n",
		"duplicate effect":    "effects:\n  a:\n    fx: Solid\n  a:\n    fx: Blink\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseLightProfile([]byte(doc)); err == nil {
				t.Error("expected an error, light-catcher cannot load this profile")
			}
		})
	}
}

func TestParseLightProfile_NoEffects(t *testing.T) {
	for _, doc := range []string{"", "effects:\n", "other: 1\n"} {
		p := mustLight(t, doc)
		if len(p.Effects) != 0 || len(p.Evaluate(homerun.Message{System: "x", Severity: "info"})) != 0 {
			t.Errorf("%q: expected an empty profile", doc)
		}
	}
}

func TestLightEvaluate(t *testing.T) {
	p := mustLight(t, lightTestProfile)

	cases := []struct {
		name, system, severity, tags string
		wantFx                       string // "" = no match
	}{
		{"exact system", "github", "ERROR", "", "Blurz"},
		{"wildcard", "anything", "INFO", "", "DJ Light"},
		{"no match", "github", "WARNING", "", ""},
		{"severity case-insensitive", "gitlab", "error", "", "Blurz"},
		{"system is case-sensitive", "GitHub", "ERROR", "", ""},
		{"profile without tags ignores message tags", "github", "ERROR", "transition=point,side=a", "Blurz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := lightMatch(p, tc.system, tc.severity, tc.tags)
			switch {
			case tc.wantFx == "" && e != nil:
				t.Errorf("expected no match, got %s", e.Name)
			case tc.wantFx != "" && (e == nil || e.Fx != tc.wantFx):
				t.Errorf("expected %s, got %+v", tc.wantFx, e)
			}
		})
	}
}

func TestLightEvaluate_FirstMatchWins(t *testing.T) {
	specificFirst := mustLight(t, `effects:
  tabletennis-win:
    systems: [tabletennis]
    severity: [success]
    fx: Fireworks
  success:
    systems: ["*"]
    severity: [success]
    fx: Aurora
`)
	if e := lightMatch(specificFirst, "tabletennis", "success", ""); e == nil || e.Fx != "Fireworks" {
		t.Errorf("specific rule declared first: got %+v", e)
	}
	if e := lightMatch(specificFirst, "gitlab", "success", ""); e == nil || e.Fx != "Aurora" {
		t.Errorf("wildcard for another system: got %+v", e)
	}

	wildcardFirst := mustLight(t, `effects:
  success:
    systems: ["*"]
    severity: [success]
    fx: Aurora
  tabletennis-win:
    systems: [tabletennis]
    severity: [success]
    fx: Fireworks
`)
	if e := lightMatch(wildcardFirst, "tabletennis", "success", ""); e == nil || e.Fx != "Aurora" {
		t.Errorf("wildcard declared first must win regardless of specificity: got %+v", e)
	}
}

func TestLightEvaluate_Tags(t *testing.T) {
	p := mustLight(t, lightTagProfile)

	cases := []struct {
		name, system, sev, tags string
		wantFx, wantColor       string
	}{
		{"tag rule precedes wildcard declared after it", "tabletennis", "SUCCESS", "match=36c17b30,set=3,transition=match_won,side=a", "Fireworks", "sunset"},
		{"side a point", "tabletennis", "INFO", "match=36c17b30,set=2,transition=point,side=a", "Solid", "blue"},
		{"side b point", "tabletennis", "INFO", "match=36c17b30,set=2,transition=point,side=b", "Solid", "red"},
		{"partial tags fall through to wildcard", "tabletennis", "INFO", "transition=point", "DJ Light", "ocean"},
		{"set won falls through to wildcard", "tabletennis", "SUCCESS", "transition=set_won,side=a", "Aurora", "forest"},
		{"no tags falls through to wildcard", "tabletennis", "INFO", "", "DJ Light", "ocean"},
		{"tags on another system do not match tag rules", "other", "INFO", "transition=point,side=a", "DJ Light", "ocean"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := lightMatch(p, tc.system, tc.sev, tc.tags)
			if e == nil || e.Fx != tc.wantFx || e.Color != tc.wantColor {
				t.Errorf("got %+v, want %s/%s", e, tc.wantFx, tc.wantColor)
			}
		})
	}
}

func TestLightTagsMatch(t *testing.T) {
	cases := []struct {
		name     string
		required []string
		tags     string
		want     bool
	}{
		{"no required tags, no message tags", nil, "", true},
		{"no required tags, message tags", nil, "side=a", true},
		{"single tag present", []string{"side=a"}, "match=1,set=2,side=a", true},
		{"substring is not a match", []string{"side=a"}, "side=ab", false},
		{"superstring is not a match", []string{"side=ab"}, "side=a", false},
		{"all tags required (AND)", []string{"transition=point", "side=a"}, "transition=point,side=b", false},
		{"all tags present in any order", []string{"side=a", "transition=point"}, "match=36c17b30,set=2,transition=point,side=a", true},
		{"whitespace around elements ignored", []string{" side=a "}, "set=2, side=a ,x", true},
		{"case-sensitive", []string{"side=A"}, "side=a", false},
		{"required tag, message without tags", []string{"side=a"}, "", false},
		{"empty required entry never matches", []string{""}, "a,,b", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lightTagsMatch(tc.required, tc.tags); got != tc.want {
				t.Errorf("lightTagsMatch(%q, %q) = %v, want %v", tc.required, tc.tags, got, tc.want)
			}
		})
	}
}

func TestLightEvaluate_Problems(t *testing.T) {
	cases := []struct {
		name, effect, wantProblem string
	}{
		{"valid", "fx: Solid\n    color: red\n    endpoint: http://wled", ""},
		{"unknown color", "fx: Solid\n    color: purple\n    endpoint: http://wled", `unknown color or palette "purple"`},
		{"missing color", "fx: Solid\n    endpoint: http://wled", `unknown color or palette ""`},
		{"unknown fx", "fx: Sparkle\n    color: red\n    endpoint: http://wled", `unknown fx "Sparkle"`},
		{"color is checked before fx", "fx: Sparkle\n    color: purple\n    endpoint: http://wled", "unknown color"},
		{"no endpoint", "fx: Solid\n    color: red", "no endpoint"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := mustLight(t, "effects:\n  e:\n    systems: [\"*\"]\n    severity: [info]\n    "+tc.effect+"\n")
			r := p.Evaluate(homerun.Message{System: "s", Severity: "info"})
			if len(r) != 1 {
				t.Fatalf("expected one reaction, got %v", r)
			}
			if tc.wantProblem == "" && r[0].Problem != "" || !strings.Contains(r[0].Problem, tc.wantProblem) {
				t.Errorf("problem = %q, want %q", r[0].Problem, tc.wantProblem)
			}
			if got := len(p.problems()); (got == 0) != (tc.wantProblem == "") {
				t.Errorf("problems() listed %d, want a problem: %v", got, tc.wantProblem != "")
			}
		})
	}
}

func TestLightSystems(t *testing.T) {
	p := mustLight(t, lightTestProfile+"  more:\n    systems: [github, ci]\n")
	if got := strings.Join(p.Systems(), ","); got != "gitlab,github,ci" {
		t.Errorf("Systems() = %s", got)
	}
}
