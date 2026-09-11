/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"fmt"
	"slices"
	"strings"

	homerun "github.com/stuttgart-things/homerun-library/v4"
	"go.yaml.in/yaml/v3"
)

// lightColors are the color and palette names light-catcher resolves
// (homerun2-light-catcher internal/profile GetColor).
var lightColors = map[string]bool{
	"sunset": true, "beach": true, "forest": true, "ocean": true,
	"red": true, "yellow": true, "green": true, "blue": true, "white": true,
}

// lightFx are the WLED effect names light-catcher knows
// (homerun2-light-catcher internal/profile FxMap).
var lightFx = map[string]bool{
	"Solid": true, "Blink": true, "Breathe": true, "Wipe": true, "Scan": true,
	"Twinkle": true, "Fireworks": true, "Rainbow": true, "Candle": true,
	"Chase": true, "Dynamic": true, "Chase Rainbow": true, "Aurora": true,
	"Blurz": true, "DJ Light": true,
}

// LightEffect is one entry of a light-catcher profile's effects.
type LightEffect struct {
	Name     string   `yaml:"-" json:"name"`
	Systems  []string `yaml:"systems" json:"systems"`
	Severity []string `yaml:"severity" json:"severity"`
	Tags     []string `yaml:"tags" json:"tags,omitempty"`
	Fx       string   `yaml:"fx" json:"fx"`
	Duration int      `yaml:"duration" json:"duration"`
	Color    string   `yaml:"color" json:"color"`
	Segments []int    `yaml:"segments" json:"segments,omitempty"`
	Endpoint string   `yaml:"endpoint" json:"endpoint"`
}

// LightProfile is a parsed homerun2-light-catcher profile (the profile.yaml
// in ConfigMap homerun2-light-catcher-profile).
type LightProfile struct {
	// Effects in profile document order, which is the order they match in.
	Effects []LightEffect
}

// ParseLightProfile parses a light-catcher profile. It rejects what
// light-catcher rejects: light-catcher loads the profile for every message and
// lights nothing when it does not parse.
func ParseLightProfile(data []byte) (*LightProfile, error) {
	var doc struct {
		Effects yaml.Node `yaml:"effects"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse light-catcher profile: %w", err)
	}

	// Decoding into a Go map, as light-catcher does, catches type errors and
	// duplicate names; the node walk recovers the document order the map
	// loses.
	var byName map[string]LightEffect
	if err := doc.Effects.Decode(&byName); err != nil {
		return nil, fmt.Errorf("failed to parse light-catcher profile: %w", err)
	}

	p := &LightProfile{}
	if doc.Effects.Kind != yaml.MappingNode {
		return p, nil
	}
	for i := 0; i+1 < len(doc.Effects.Content); i += 2 {
		name := doc.Effects.Content[i].Value
		effect := byName[name]
		effect.Name = name
		p.Effects = append(p.Effects, effect)
	}
	return p, nil
}

// Evaluate implements Profile, mirroring light-catcher's MatchEffect and
// SendToWLED: the first effect in document order whose systems (exact, or "*"),
// severity (case-insensitive) and tags all match wins.
func (p *LightProfile) Evaluate(msg homerun.Message) []Reaction {
	for _, e := range p.Effects {
		if !lightMatches(e, msg) {
			continue
		}
		r := Reaction{
			Rule:    e.Name,
			Summary: fmt.Sprintf("WLED %s in %s for %s on %s", e.Fx, e.Color, lightDuration(e.Duration), e.Endpoint),
			Details: e,
		}
		r.Problem = lightProblem(e)
		return []Reaction{r}
	}
	return nil
}

// lightProblem says why light-catcher cannot send e, checked in the order
// light-catcher resolves it - the first failure aborts the send.
func lightProblem(e LightEffect) string {
	switch {
	case !lightColors[e.Color]:
		return fmt.Sprintf("unknown color or palette %q: nothing is sent", e.Color)
	case !lightFx[e.Fx]:
		return fmt.Sprintf("unknown fx %q: nothing is sent", e.Fx)
	case e.Endpoint == "":
		return "no endpoint: nothing is sent"
	}
	return ""
}

func (p *LightProfile) problems() []Reaction {
	var out []Reaction
	for _, e := range p.Effects {
		if problem := lightProblem(e); problem != "" {
			out = append(out, Reaction{Rule: e.Name, Problem: problem})
		}
	}
	return out
}

func lightMatches(e LightEffect, msg homerun.Message) bool {
	systemMatch := false
	for _, s := range e.Systems {
		if s == "*" || s == msg.System {
			systemMatch = true
			break
		}
	}
	severityMatch := false
	for _, sev := range e.Severity {
		if strings.EqualFold(sev, msg.Severity) {
			severityMatch = true
			break
		}
	}
	return systemMatch && severityMatch && lightTagsMatch(e.Tags, msg.Tags)
}

// lightTagsMatch mirrors light-catcher's TagsMatch: every required tag must
// equal one whole, trimmed element of the comma-separated message tags.
func lightTagsMatch(required []string, messageTags string) bool {
	if len(required) == 0 {
		return true
	}
	present := make(map[string]bool)
	for _, tag := range strings.Split(messageTags, ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			present[tag] = true
		}
	}
	for _, tag := range required {
		if !present[strings.TrimSpace(tag)] {
			return false
		}
	}
	return true
}

func lightDuration(seconds int) string {
	if seconds <= 0 {
		return "until replaced"
	}
	return fmt.Sprintf("%ds", seconds)
}

// Systems implements Profile.
func (p *LightProfile) Systems() []string {
	var out []string
	for _, e := range p.Effects {
		out = appendSystems(out, e.Systems)
	}
	return out
}

// appendSystems adds the non-wildcard systems not yet in out.
func appendSystems(out, systems []string) []string {
	for _, s := range systems {
		if s != "*" && s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}
