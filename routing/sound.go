/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	homerun "github.com/stuttgart-things/homerun-library/v4"
	"go.yaml.in/yaml/v3"
)

// soundMaxSpokenChars is homerun2-sound-catcher's MAX_SPOKEN_CHARS: a
// rendered announcement is cut there.
const soundMaxSpokenChars = 500

// soundTitleTags matches leading tags like "[WARN] " that sound-catcher's
// spoken_title drops.
var soundTitleTags = regexp.MustCompile(`^\s*(?:\[[^\]\n]{1,24}\]\s*)+`)

// SoundRule is one entry of a sound-catcher profile's rules.
type SoundRule struct {
	Name string `json:"name"`
	// Systems are matched exactly (case-sensitive). Empty or "*" matches any.
	Systems []string `json:"systems,omitempty"`
	// Severity is lower-cased, as sound-catcher stores it. Empty matches any.
	Severity []string `json:"severity,omitempty"`
	// Tags must each equal one whole, trimmed element of the message's tags.
	Tags []string `json:"tags,omitempty"`
	// Sound is a clip in sound-catcher's SOUNDS_DIR, played first.
	Sound string `json:"sound,omitempty"`
	// Say is the announcement, a Jinja template over the message.
	Say    string `json:"say,omitempty"`
	Volume int    `json:"volume"`
	// Silent matches and plays nothing.
	Silent bool `json:"silent,omitempty"`
}

// SoundQuietHours is a sound-catcher profile's quietHours: inside the window
// only the Allow severities make a sound.
type SoundQuietHours struct {
	// From and To are HH:MM; the window may cross midnight.
	From     string   `json:"from"`
	To       string   `json:"to"`
	Weekends bool     `json:"weekends,omitempty"`
	Timezone string   `json:"timezone,omitempty"`
	Allow    []string `json:"allow"`
}

// SoundAction is what sound-catcher does with a matched message.
type SoundAction struct {
	SoundRule
	// Said is the rule's say template rendered for the message, when
	// SayRendered is true. Otherwise it is the raw template: only plain
	// {{ variable }} substitutions are rendered here.
	Said        string `json:"said,omitempty"`
	SayRendered bool   `json:"sayRendered"`
	// QuietHoursSilence is true when the profile's quiet hours silence the
	// message's severity: it makes no sound inside that window.
	QuietHoursSilence bool `json:"quietHoursSilence,omitempty"`
}

// SoundProfile is a parsed homerun2-sound-catcher profile (profile.yaml).
type SoundProfile struct {
	// Rules in profile document order, which is the order they match in.
	Rules      []SoundRule
	QuietHours *SoundQuietHours
}

// ParseSoundProfile parses a sound-catcher profile the way its parse_profile
// reads it and rejects what it rejects: a sound that is a path, a volume
// outside 0-100, and a rule with neither sound, say nor silent. Severities and
// quietHours' allow are lower-cased; allow defaults to [critical].
//
// sound-catcher also rejects a say template Jinja cannot parse; that is not
// checked here.
func ParseSoundProfile(data []byte) (*SoundProfile, error) {
	p := &SoundProfile{}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse sound-catcher profile: %w", err)
	}
	if len(doc.Content) == 0 || !pythonTruthy(doc.Content[0]) {
		return p, nil // sound-catcher: `yaml.safe_load(...) or {}`
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("sound-catcher profile: top level is not a mapping")
	}

	if q := mappingValue(root, "quietHours"); q != nil && pythonTruthy(q) {
		quiet, err := parseSoundQuietHours(q)
		if err != nil {
			return nil, fmt.Errorf("sound-catcher profile: quietHours: %w", err)
		}
		p.QuietHours = quiet
	}

	rules := mappingValue(root, "rules")
	if rules == nil || !pythonTruthy(rules) {
		return p, nil // `data.get("rules") or {}`
	}
	if rules.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("sound-catcher profile: rules is not a mapping")
	}
	for i := 0; i+1 < len(rules.Content); i += 2 {
		name := rules.Content[i].Value
		rule, err := parseSoundRule(name, rules.Content[i+1])
		if err != nil {
			return nil, fmt.Errorf("sound-catcher profile: rule %s: %w", name, err)
		}
		// A Python dict keeps a repeated key at its first position.
		if at := slices.IndexFunc(p.Rules, func(r SoundRule) bool { return r.Name == name }); at >= 0 {
			p.Rules[at] = rule
		} else {
			p.Rules = append(p.Rules, rule)
		}
	}
	return p, nil
}

func parseSoundRule(name string, n *yaml.Node) (SoundRule, error) {
	rule := SoundRule{Name: name, Volume: 100}
	if isNull(n) {
		n = &yaml.Node{Kind: yaml.MappingNode} // `raw or {}`
	}
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
		case "tags":
			err = decodeList(value, &rule.Tags)
		case "sound":
			rule.Sound = pythonStr(value)
		case "say":
			rule.Say = pythonStr(value)
		case "volume":
			rule.Volume, err = pythonInt(value)
		case "silent":
			rule.Silent = pythonTruthy(value)
		}
		if err != nil {
			return rule, fmt.Errorf("%s: %w", key, err)
		}
	}

	switch {
	case rule.Sound != "" && !isBareName(rule.Sound):
		return rule, fmt.Errorf("sound must be a file name in SOUNDS_DIR, not a path")
	case rule.Volume < 0 || rule.Volume > 100:
		return rule, fmt.Errorf("volume must be 0-100")
	case rule.Sound == "" && rule.Say == "" && !rule.Silent:
		return rule, fmt.Errorf("needs sound, say, or silent: true")
	}
	return rule, nil
}

func parseSoundQuietHours(n *yaml.Node) (*SoundQuietHours, error) {
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("not a mapping")
	}
	q := &SoundQuietHours{Allow: []string{"critical"}}
	for _, key := range []string{"from", "to"} {
		v := mappingValue(n, key)
		if v == nil {
			return nil, fmt.Errorf("%s is missing", key)
		}
		clock, err := soundClockTime(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		if key == "from" {
			q.From = clock
		} else {
			q.To = clock
		}
	}
	if v := mappingValue(n, "weekends"); v != nil {
		q.Weekends = pythonTruthy(v)
	}
	if v := mappingValue(n, "timezone"); v != nil {
		q.Timezone = pythonStr(v)
	}
	if v := mappingValue(n, "allow"); v != nil {
		var allow []string
		if err := decodeList(v, &allow); err != nil {
			return nil, fmt.Errorf("allow: %w", err)
		}
		for i, s := range allow {
			allow[i] = strings.ToLower(s)
		}
		q.Allow = allow
	}
	return q, nil
}

// soundClockTime mirrors sound-catcher's _clock_time: HH:MM, or an integer
// read as minutes since midnight (PyYAML reads an unquoted 19:00 as 1140).
func soundClockTime(n *yaml.Node) (string, error) {
	if n.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("%q is not a time", n.Value)
	}
	if n.Tag == "!!int" {
		m, err := strconv.Atoi(n.Value)
		if err != nil || m < 0 || m >= 24*60 {
			return "", fmt.Errorf("%q is not a time", n.Value)
		}
		return fmt.Sprintf("%02d:%02d", m/60, m%60), nil
	}
	for _, layout := range []string{"15:04", "15:04:05"} {
		if t, err := time.Parse(layout, n.Value); err == nil {
			return t.Format("15:04"), nil
		}
	}
	return "", fmt.Errorf("%q is not a time", n.Value)
}

// isBareName mirrors sound-catcher's is_bare_name: a file name without a
// directory part.
func isBareName(name string) bool {
	return name != "" && name != "." && name != ".." && path.Base(name) == name &&
		!strings.ContainsAny(name, `/\`)
}

// pythonStr mirrors Python's str() of the scalar PyYAML reads for n.
func pythonStr(n *yaml.Node) string {
	switch {
	case isNull(n):
		return "None"
	case n.Kind == yaml.ScalarNode && n.Tag == "!!bool":
		if b, _ := strconv.ParseBool(n.Value); b {
			return "True"
		}
		return "False"
	}
	return n.Value
}

// pythonInt mirrors Python's int() of the scalar PyYAML reads for n: an
// integer, a float truncated, or a string holding an integer.
func pythonInt(n *yaml.Node) (int, error) {
	if n.Kind == yaml.ScalarNode {
		switch n.Tag {
		case "!!float":
			if f, err := strconv.ParseFloat(n.Value, 64); err == nil && !math.IsInf(f, 0) && !math.IsNaN(f) {
				return int(f), nil
			}
		case "!!bool":
			if b, _ := strconv.ParseBool(n.Value); b {
				return 1, nil
			}
			return 0, nil
		default:
			if i, err := strconv.Atoi(strings.TrimSpace(n.Value)); err == nil {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("%q is not an integer", n.Value)
}

// Evaluate implements Profile, mirroring sound-catcher's Profile.match: the
// first rule in document order whose systems contain the message's system
// (exact; empty or "*" matches any), whose severities contain its severity
// (case-insensitive; empty matches any) and whose tags are all among the
// message's tags wins.
//
// sound-catcher reads a message without a system as "unknown" and one
// without a severity as "info", and so does this. Quiet hours are not
// applied, since Evaluate has no clock: QuietHoursSilence says whether they
// would silence the message.
func (p *SoundProfile) Evaluate(msg homerun.Message) []Reaction {
	system := orDefault(msg.System, "unknown")
	severity := strings.ToLower(orDefault(msg.Severity, "info"))

	for _, rule := range p.Rules {
		if len(rule.Systems) > 0 && !slices.Contains(rule.Systems, system) && !slices.Contains(rule.Systems, "*") {
			continue
		}
		if len(rule.Severity) > 0 && !slices.Contains(rule.Severity, severity) {
			continue
		}
		if !lightTagsMatch(rule.Tags, msg.Tags) {
			continue
		}

		a := SoundAction{SoundRule: rule}
		var problem string
		if !rule.Silent {
			a.Said, a.SayRendered, problem = renderSoundSay(rule.Say, msg)
			if problem != "" && rule.Sound == "" {
				problem += ": nothing plays"
			}
			a.QuietHoursSilence = p.QuietHours != nil && !slices.Contains(p.QuietHours.Allow, severity)
		}
		return []Reaction{{Rule: rule.Name, Summary: p.soundSummary(a), Problem: problem, Details: a}}
	}
	return nil
}

// renderSoundSay renders a say template's plain {{ variable }} substitutions
// with the variables sound-catcher passes, then collapses whitespace and
// cuts it at soundMaxSpokenChars as sound-catcher does. Anything else is
// returned unrendered with false. sound-catcher renders with StrictUndefined,
// so an unknown variable fails the template and nothing is said: that is the
// returned problem.
func renderSoundSay(say string, msg homerun.Message) (string, bool, string) {
	if say == "" {
		return "", true, ""
	}
	vars := map[string]string{
		"title":        msg.Title,
		"spoken_title": spokenTitle(msg.Title),
		"message":      msg.Message,
		"severity":     orDefault(msg.Severity, "info"),
		"system":       orDefault(msg.System, "unknown"),
		"author":       orDefault(msg.Author, "unknown"),
		"tags":         msg.Tags,
		"url":          msg.URL,
	}

	rendered, problem := true, ""
	out := ledTemplateTag.ReplaceAllStringFunc(say, func(tag string) string {
		m := soundVariable.FindStringSubmatch(tag)
		if m == nil {
			// A call of a name sound-catcher does not pass, e.g. tag('x'),
			// fails as an undefined variable too.
			if c := soundCall.FindStringSubmatch(tag); c != nil && problem == "" {
				if _, ok := vars[c[1]]; ok {
					rendered = false
					return tag
				}
				problem = fmt.Sprintf("say calls %s(), which sound-catcher does not define: the template fails and nothing is said", c[1])
			}
			rendered = false
			return tag
		}
		v, ok := vars[m[1]]
		if !ok && problem == "" {
			problem = fmt.Sprintf("say uses {{ %s }}, which sound-catcher does not define: the template fails and nothing is said", m[1])
		}
		return v
	})
	if problem != "" {
		return say, false, problem
	}
	if !rendered {
		return say, false, ""
	}
	out = strings.Join(strings.Fields(out), " ")
	if r := []rune(out); len(r) > soundMaxSpokenChars {
		out = string(r[:soundMaxSpokenChars])
	}
	return out, true, ""
}

// soundVariable matches a tag that is a bare variable, e.g. {{ title }}.
var soundVariable = regexp.MustCompile(`^\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}$`)

// soundCall matches a tag that calls a bare name, e.g. {{ tag('x') }}.
var soundCall = regexp.MustCompile(`^\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(`)

// spokenTitle mirrors sound-catcher's spoken_title: the title without leading
// tags like [WARN].
func spokenTitle(title string) string {
	if s := strings.TrimSpace(soundTitleTags.ReplaceAllString(title, "")); s != "" {
		return s
	}
	return strings.TrimSpace(title)
}

func (p *SoundProfile) soundSummary(a SoundAction) string {
	if a.Silent {
		return "sound: silent, matches and plays nothing"
	}
	var parts []string
	if a.Sound != "" {
		parts = append(parts, fmt.Sprintf("plays %s at %d%%", a.Sound, a.Volume))
	}
	if a.Say != "" {
		parts = append(parts, fmt.Sprintf("says %q", a.Said))
	}
	s := "sound: " + strings.Join(parts, ", then ")
	if a.QuietHoursSilence {
		q := p.QuietHours
		window := q.From + "-" + q.To
		if q.Weekends {
			window += " and weekends"
		}
		if q.Timezone != "" {
			window += " (" + q.Timezone + ")"
		}
		s += fmt.Sprintf("; silent in quiet hours %s, which allow only %s", window, strings.Join(q.Allow, ", "))
	}
	return s
}

// Systems implements Profile.
func (p *SoundProfile) Systems() []string {
	var out []string
	for _, r := range p.Rules {
		out = appendSystems(out, r.Systems)
	}
	return out
}

// problems lists the rules whose say template fails in sound-catcher for
// every message: it uses a name sound-catcher does not define.
func (p *SoundProfile) problems() []Reaction {
	var out []Reaction
	for _, rule := range p.Rules {
		if rule.Silent {
			continue
		}
		if _, _, problem := renderSoundSay(rule.Say, homerun.Message{}); problem != "" {
			if rule.Sound == "" {
				problem += ": nothing plays"
			}
			out = append(out, Reaction{Rule: rule.Name, Problem: problem})
		}
	}
	return out
}
