/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	homerun "github.com/stuttgart-things/homerun-library/v4"
	"go.yaml.in/yaml/v3"
)

// ledDefaultColors are led-catcher's built-in severity colors, which a
// profile's colors extend and override (homerun2-led-catcher
// profile/engine.py DEFAULT_COLORS). critical is deliberately absent: it is
// not in led-catcher's defaults either.
var ledDefaultColors = map[string][3]int{
	"error":   {255, 0, 0},
	"warning": {255, 165, 0},
	"success": {0, 255, 0},
	"info":    {0, 100, 255},
	"debug":   {128, 128, 128},
}

// ledWhite is the color led-catcher renders a severity without a color in -
// its "unknown severity" color.
var ledWhite = [3]int{255, 255, 255}

// LEDRule is one entry of a led-catcher profile's displayRules.
type LEDRule struct {
	Name    string   `json:"name"`
	Systems []string `json:"systems"`
	// Severity is lower-cased, as led-catcher stores it. Empty matches every
	// severity.
	Severity []string `json:"severity,omitempty"`
	Kind     string   `json:"kind"`
	Text     string   `json:"text,omitempty"`
	Image    string   `json:"image,omitempty"`
	Font     string   `json:"font"`
	Duration float64  `json:"duration"`
	Hold     bool     `json:"hold"`
}

// LEDDisplay is what led-catcher shows for a matched message.
type LEDDisplay struct {
	LEDRule
	// Text is the rule's text template rendered for the message, when
	// TextRendered is true. Otherwise it is the raw template: only plain
	// {{ variable }} substitutions are rendered here, not Jinja2 filters or
	// statements.
	TextRendered bool   `json:"textRendered"`
	Color        [3]int `json:"color"`
}

// LEDProfile is a parsed homerun2-led-catcher profile (the profile.yaml in
// ConfigMap homerun2-led-catcher-profile).
type LEDProfile struct {
	// Rules in profile document order, which is the order they match in.
	Rules []LEDRule
	// Colors are the severity colors: led-catcher's defaults, overridden and
	// extended by the profile's colors.
	Colors map[string][3]int
}

// ParseLEDProfile parses a led-catcher profile the way led-catcher's
// load_profile reads it: missing fields take led-catcher's defaults (kind
// text, font 6x10.bdf, duration 5), severities are lower-cased, a color that
// is not a list of three integers is ignored, and a rule name given twice
// keeps its first position with its last definition.
//
// led-catcher parses YAML 1.1 (PyYAML), this parses YAML 1.2. The two
// disagree on unquoted yes/no/on/off, which only hold takes as a value, and
// hold reads them as PyYAML does.
func ParseLEDProfile(data []byte) (*LEDProfile, error) {
	p := &LEDProfile{Colors: make(map[string][3]int, len(ledDefaultColors))}
	for k, v := range ledDefaultColors {
		p.Colors[k] = v
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse led-catcher profile: %w", err)
	}
	if len(doc.Content) == 0 || !pythonTruthy(doc.Content[0]) {
		return p, nil // led-catcher: `if not data: return Profile()`
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("led-catcher profile: top level is not a mapping")
	}

	if colors := mappingValue(root, "colors"); colors != nil {
		if colors.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("led-catcher profile: colors is not a mapping")
		}
		for i := 0; i+1 < len(colors.Content); i += 2 {
			var rgb []int
			if err := colors.Content[i+1].Decode(&rgb); err == nil && len(rgb) == 3 {
				p.Colors[colors.Content[i].Value] = [3]int{rgb[0], rgb[1], rgb[2]}
			}
		}
	}

	rules := mappingValue(root, "displayRules")
	if rules == nil {
		return p, nil
	}
	if rules.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("led-catcher profile: displayRules is not a mapping")
	}
	for i := 0; i+1 < len(rules.Content); i += 2 {
		name := rules.Content[i].Value
		rule, err := parseLEDRule(name, rules.Content[i+1])
		if err != nil {
			return nil, fmt.Errorf("led-catcher profile: rule %q: %w", name, err)
		}
		if at := slices.IndexFunc(p.Rules, func(r LEDRule) bool { return r.Name == name }); at >= 0 {
			p.Rules[at] = rule
		} else {
			p.Rules = append(p.Rules, rule)
		}
	}
	return p, nil
}

func parseLEDRule(name string, n *yaml.Node) (LEDRule, error) {
	rule := LEDRule{Name: name, Kind: "text", Font: "6x10.bdf", Duration: 5}
	if n.Kind != yaml.MappingNode {
		return rule, fmt.Errorf("not a mapping")
	}

	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i].Value, n.Content[i+1]
		var err error
		switch key {
		case "systems":
			err = decodeList(value, &rule.Systems)
		case "severity":
			err = decodeList(value, &rule.Severity)
			for j, s := range rule.Severity {
				rule.Severity[j] = strings.ToLower(s)
			}
		case "kind":
			err = value.Decode(&rule.Kind)
		case "text":
			err = value.Decode(&rule.Text)
		case "image":
			err = value.Decode(&rule.Image)
		case "font":
			err = value.Decode(&rule.Font)
		case "duration":
			// Python's float() also takes a numeric string.
			rule.Duration, err = strconv.ParseFloat(strings.TrimSpace(value.Value), 64)
			if err != nil || value.Kind != yaml.ScalarNode {
				err = fmt.Errorf("duration %q is not a number", value.Value)
			}
		case "hold":
			rule.Hold = pythonTruthy(value)
		}
		if err != nil {
			return rule, fmt.Errorf("%s: %w", key, err)
		}
	}
	return rule, nil
}

// decodeList decodes a list of strings. led-catcher iterates the value, so a
// null there fails on it instead of meaning "no entries".
func decodeList(n *yaml.Node, out *[]string) error {
	if isNull(n) {
		return fmt.Errorf("is null")
	}
	return n.Decode(out)
}

// yaml11False are the scalars PyYAML (YAML 1.1) reads as false that YAML 1.2
// reads as strings.
var yaml11False = map[string]bool{"no": true, "No": true, "NO": true, "off": true, "Off": true, "OFF": true}

// pythonTruthy mirrors Python's bool() of the value PyYAML reads for n, which
// is what led-catcher applies to hold: false, no, off, 0, null, "" and empty
// collections are false; everything else is true - the quoted string "false"
// included.
func pythonTruthy(n *yaml.Node) bool {
	if n.Kind != yaml.ScalarNode {
		return len(n.Content) > 0
	}
	switch n.Tag {
	case "!!null":
		return false
	case "!!bool":
		b, _ := strconv.ParseBool(n.Value)
		return b
	case "!!int", "!!float":
		f, err := strconv.ParseFloat(n.Value, 64)
		return err != nil || f != 0
	case "!!str":
		if n.Style == 0 && yaml11False[n.Value] {
			return false
		}
		return n.Value != ""
	default:
		return n.Value != ""
	}
}

// Evaluate implements Profile, mirroring led-catcher's match_rule: the first
// rule in document order whose systems contain the message's system
// (case-insensitive, or "*") and whose severity list contains its severity
// (case-insensitive; an empty list matches any) wins.
//
// led-catcher reads a message without a system as system "unknown" and one
// without a severity as "info", and so does this.
func (p *LEDProfile) Evaluate(msg homerun.Message) []Reaction {
	system := strings.ToLower(orDefault(msg.System, "unknown"))
	severity := strings.ToLower(orDefault(msg.Severity, "info"))

	for _, rule := range p.Rules {
		systemMatch := slices.ContainsFunc(rule.Systems, func(s string) bool {
			return s == "*" || strings.ToLower(s) == system
		})
		if !systemMatch {
			continue
		}
		if len(rule.Severity) > 0 && !slices.Contains(rule.Severity, severity) {
			continue
		}

		d := LEDDisplay{LEDRule: rule}
		d.Text, d.TextRendered = renderLEDText(rule.Text, msg)

		var problem string
		d.Color, problem = p.color(severity)
		return []Reaction{{Rule: rule.Name, Summary: ledSummary(d), Problem: problem, Details: d}}
	}
	return nil
}

// color returns the color led-catcher renders severity in, and a problem when
// that is the white of an unknown severity.
func (p *LEDProfile) color(severity string) ([3]int, string) {
	if c, ok := p.Colors[severity]; ok {
		return c, ""
	}
	return ledWhite, fmt.Sprintf("severity %q has no color in the profile: it renders white, the color for an unknown severity", severity)
}

func (p *LEDProfile) problems() []Reaction {
	var out []Reaction
	for _, rule := range p.Rules {
		severities := rule.Severity
		if len(severities) == 0 {
			severities = Severities
		}
		for _, sev := range severities {
			if _, problem := p.color(sev); problem != "" {
				out = append(out, Reaction{Rule: rule.Name, Problem: problem})
			}
		}
	}
	return out
}

func ledSummary(d LEDDisplay) string {
	var what string
	switch d.Kind {
	case "image", "gif":
		what = fmt.Sprintf("%s %s", d.Kind, d.Image)
	default:
		what = fmt.Sprintf("%s %q", d.Kind, d.Text)
	}
	// Only the still modes honour hold, and a scroll lasts as long as it takes
	// whatever the duration (led-catcher docs/profile-reference.md).
	var how string
	switch {
	case d.Hold && (d.Kind == "static" || d.Kind == "image" || d.Kind == "score"):
		how = "until replaced"
	case d.Kind == "text" || d.Kind == "ticker":
		how = "scrolling"
	default:
		how = fmt.Sprintf("for %gs", d.Duration)
	}
	return fmt.Sprintf("LED %s in rgb(%d,%d,%d) %s", what, d.Color[0], d.Color[1], d.Color[2], how)
}

// ledTemplateTag matches one Jinja2 tag of any kind.
var ledTemplateTag = regexp.MustCompile(`\{\{.*?\}\}|\{%.*?%\}|\{#.*?#\}`)

// ledVariable matches a tag that is a bare variable, e.g. {{ title }}.
var ledVariable = regexp.MustCompile(`^\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}$`)

// renderLEDText renders the plain {{ variable }} substitutions of a
// led-catcher text template with the variables led-catcher passes. A template
// using anything else - filters, expressions, statements - is returned
// unrendered with false: a half-rendered text would look like the real output
// and not be.
func renderLEDText(text string, msg homerun.Message) (string, bool) {
	vars := map[string]string{
		"title":    msg.Title,
		"message":  msg.Message,
		"severity": orDefault(msg.Severity, "info"),
		"system":   orDefault(msg.System, "unknown"),
		"author":   orDefault(msg.Author, "unknown"),
		"tags":     msg.Tags,
		"url":      msg.URL,
	}

	// Jinja2 drops a single trailing newline of the template by default.
	source := strings.TrimSuffix(text, "\n")

	rendered := true
	out := ledTemplateTag.ReplaceAllStringFunc(source, func(tag string) string {
		m := ledVariable.FindStringSubmatch(tag)
		if m == nil {
			rendered = false
			return tag
		}
		return vars[m[1]] // Jinja2 renders an undefined variable as ""
	})
	if !rendered {
		return text, false
	}
	return out, true
}

// Systems implements Profile.
func (p *LEDProfile) Systems() []string {
	var out []string
	for _, r := range p.Rules {
		out = appendSystems(out, r.Systems)
	}
	return out
}

func mappingValue(m *yaml.Node, key string) *yaml.Node {
	var found *yaml.Node
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			found = m.Content[i+1] // the last one wins, as in PyYAML
		}
	}
	return found
}

func isNull(n *yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && n.Tag == "!!null"
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
