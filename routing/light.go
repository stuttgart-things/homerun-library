/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	homerun "github.com/stuttgart-things/homerun-library/v4"
	"go.yaml.in/yaml/v3"
)

// lightColors are the local color and palette names light-catcher resolves
// itself (homerun2-light-catcher internal/profile GetColor). Any other color
// name is looked up as a palette of the WLED device at send time.
var lightColors = map[string]bool{
	"sunset": true, "beach": true, "forest": true, "ocean": true,
	"red": true, "yellow": true, "orange": true, "green": true, "blue": true,
	"white": true,
}

// lightFx are light-catcher's fallback effect names (homerun2-light-catcher
// internal/profile FxMap), matched case-insensitively. light-catcher asks the
// device first, so a name outside this list may still be a device effect.
var lightFx = map[string]bool{
	"solid": true, "blink": true, "breathe": true, "wipe": true, "scan": true,
	"twinkle": true, "strobe": true, "fireworks": true, "rainbow": true,
	"candle": true, "chase": true, "dynamic": true, "chase rainbow": true,
	"aurora": true, "blurz": true, "dj light": true,
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
	// Palette names a device palette explicitly.
	Palette  string `yaml:"palette" json:"palette,omitempty"`
	Segments []int  `yaml:"segments" json:"segments,omitempty"`
	Endpoint string `yaml:"endpoint" json:"endpoint"`
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
			Summary: lightSummary(e),
			Details: e,
		}
		r.Problem = lightProblem(e)
		return []Reaction{r}
	}
	return nil
}

// lightSummary describes e in one line. Names light-catcher resolves only on
// the device are marked: whether the device knows them shows at send time.
func lightSummary(e LightEffect) string {
	fx, look := e.Fx, e.Color
	if !lightFxLocal(e.Fx) {
		fx += " (device effect)"
	}
	switch {
	case e.Color != "" && !lightColors[e.Color]:
		look += " (device palette)"
	case e.Color == "":
		look = e.Palette + " (device palette)"
	case e.Palette != "":
		look += " on device palette " + e.Palette
	}
	return fmt.Sprintf("WLED %s in %s for %s on %s", fx, look, lightDuration(e.Duration), e.Endpoint)
}

// lightProblem says why light-catcher cannot send e, checked in the order
// light-catcher resolves it - the first failure aborts the send. Names it
// resolves on the device (palettes, effects outside its fallback table) are
// not problems: only the device can tell.
func lightProblem(e LightEffect) string {
	fx := strings.TrimSpace(e.Fx)
	switch {
	case e.Color == "" && e.Palette == "":
		return "neither color nor palette: nothing is sent"
	case e.Color != "" && e.Palette != "" && !lightColors[e.Color]:
		// light-catcher takes a non-local color beside an explicit palette
		// for a typo, not for a second device palette.
		return fmt.Sprintf("unknown color %q next to palette %q: nothing is sent", e.Color, e.Palette)
	case fx == "":
		return "no fx: nothing is sent"
	case strings.HasPrefix(fx, "-") && lightFxLocal(fx):
		return fmt.Sprintf("negative fx %q: nothing is sent", e.Fx)
	case e.Endpoint == "":
		return "no endpoint: nothing is sent"
	}
	return ""
}

// lightFxLocal reports whether light-catcher can resolve fx without the
// device: an effect ID, or a name in its fallback table.
func lightFxLocal(fx string) bool {
	fx = strings.TrimSpace(fx)
	if _, err := strconv.Atoi(fx); err == nil {
		return true
	}
	return lightFx[strings.ToLower(fx)]
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
