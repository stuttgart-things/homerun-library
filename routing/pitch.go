/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	homerun "github.com/stuttgart-things/homerun-library/v4"
	"go.yaml.in/yaml/v3"
)

// The paths omni-pitcher accepts pitches on. A pitch to any other path is
// answered 404 and published nowhere.
const (
	PitchPath        = "/pitch"
	PitchPathGrafana = "/pitch/grafana"
	PitchPathGitHub  = "/pitch/github"
)

// PitchPaths lists every path omni-pitcher accepts pitches on.
var PitchPaths = []string{PitchPath, PitchPathGrafana, PitchPathGitHub}

// StreamRoutes is omni-pitcher's routing file, the one ROUTES_CONFIG points
// at: an allowlist of streams, a default stream and ordered rules that pick a
// stream per message. Mirrors homerun2-omni-pitcher internal/routing.
type StreamRoutes struct {
	Streams       []string      `yaml:"streams" json:"streams"`
	DefaultStream string        `yaml:"default_stream" json:"defaultStream"`
	Routes        []StreamRoute `yaml:"routes" json:"routes"`
}

// StreamRoute is one rule: a message every matcher of Match fits goes to
// Stream.
type StreamRoute struct {
	Match  RouteMatch `yaml:"match" json:"match"`
	Stream string     `yaml:"stream" json:"stream"`
}

// RouteMatch holds the matchers of a rule. Each one is a case-sensitive
// substring test, unset ones are ignored, and all set ones have to match.
type RouteMatch struct {
	// Endpoint is matched against the path the pitch arrived on, such as
	// PitchPathGitHub.
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`
	System   string `yaml:"system,omitempty" json:"system,omitempty"`
	Author   string `yaml:"author,omitempty" json:"author,omitempty"`
	// TagContains is matched against the whole comma-separated Tags string,
	// not against single tags.
	TagContains string `yaml:"tag_contains,omitempty" json:"tagContains,omitempty"`
	// TitleContains matches when the title contains any of its non-empty
	// entries. A list of only empty entries never matches.
	TitleContains []string `yaml:"title_contains,omitempty" json:"titleContains,omitempty"`
}

// ParseStreamRoutes parses and validates an omni-pitcher routing file. An
// error means omni-pitcher does not start: it exits when its routing file
// does not load.
func ParseStreamRoutes(data []byte) (*StreamRoutes, error) {
	var r StreamRoutes
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse routes: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("invalid routes: %w", err)
	}
	return &r, nil
}

// Validate checks what omni-pitcher checks when it loads the file: a
// non-empty allowlist without empty or duplicate entries, a default stream
// from it, and rules that each have a matcher and an allowlisted stream.
func (r *StreamRoutes) Validate() error {
	if len(r.Streams) == 0 {
		return errors.New("streams must be non-empty")
	}

	seen := make(map[string]struct{}, len(r.Streams))
	for _, s := range r.Streams {
		if s == "" {
			return errors.New("streams contains an empty entry")
		}
		if _, dup := seen[s]; dup {
			return fmt.Errorf("streams contains duplicate %q", s)
		}
		seen[s] = struct{}{}
	}

	if r.DefaultStream == "" {
		return errors.New("default_stream is required")
	}
	if _, ok := seen[r.DefaultStream]; !ok {
		return fmt.Errorf("default_stream %q is not in streams allowlist", r.DefaultStream)
	}

	for i, route := range r.Routes {
		if !route.Match.HasAny() {
			return fmt.Errorf("routes[%d]: at least one matcher is required", i)
		}
		if route.Stream == "" {
			return fmt.Errorf("routes[%d]: stream is required", i)
		}
		if _, ok := seen[route.Stream]; !ok {
			return fmt.Errorf("routes[%d]: stream %q is not in streams allowlist", i, route.Stream)
		}
	}
	return nil
}

// Resolve returns the stream omni-pitcher publishes msg to when it arrives on
// endpoint, and the index of the rule that chose it: -1 when no rule matched
// and DefaultStream applies. Rules are tried in order; the first match wins.
//
// A nil StreamRoutes routes nothing and returns "" and -1: omni-pitcher then
// publishes to its single REDIS_STREAM.
func (r *StreamRoutes) Resolve(endpoint string, msg homerun.Message) (stream string, rule int) {
	if r == nil {
		return "", -1
	}
	for i, route := range r.Routes {
		if route.Match.Matches(endpoint, msg) {
			return route.Stream, i
		}
	}
	return r.DefaultStream, -1
}

// Targets returns the streams a message can be routed to: DefaultStream, then
// each rule's stream in rule order, without duplicates. An allowlisted stream
// no rule names is not among them.
func (r *StreamRoutes) Targets() []string {
	if r == nil {
		return nil
	}
	out := []string{r.DefaultStream}
	for _, route := range r.Routes {
		if !slices.Contains(out, route.Stream) {
			out = append(out, route.Stream)
		}
	}
	return out
}

// HasAny reports whether at least one matcher is set.
func (m RouteMatch) HasAny() bool {
	return m.Endpoint != "" || m.System != "" || m.Author != "" || m.TagContains != "" || len(m.TitleContains) > 0
}

// Matches reports whether a message arriving on endpoint fits every set
// matcher.
func (m RouteMatch) Matches(endpoint string, msg homerun.Message) bool {
	if m.Endpoint != "" && !strings.Contains(endpoint, m.Endpoint) {
		return false
	}
	if m.System != "" && !strings.Contains(msg.System, m.System) {
		return false
	}
	if m.Author != "" && !strings.Contains(msg.Author, m.Author) {
		return false
	}
	if m.TagContains != "" && !strings.Contains(msg.Tags, m.TagContains) {
		return false
	}
	if len(m.TitleContains) > 0 {
		hit := false
		for _, needle := range m.TitleContains {
			if needle != "" && strings.Contains(msg.Title, needle) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// Summary describes the matchers in one line, such as
// `system contains "tabletennis" and title contains any of "a", "b"`.
func (m RouteMatch) Summary() string {
	var parts []string
	add := func(field, value string) {
		if value != "" {
			parts = append(parts, fmt.Sprintf("%s contains %q", field, value))
		}
	}
	add("endpoint", m.Endpoint)
	add("system", m.System)
	add("author", m.Author)
	add("tags", m.TagContains)
	switch {
	case len(m.TitleContains) == 0:
	case len(m.TitleContains) == 1 && m.TitleContains[0] != "":
		add("title", m.TitleContains[0])
	default:
		quoted := make([]string, len(m.TitleContains))
		for i, s := range m.TitleContains {
			quoted[i] = strconv.Quote(s)
		}
		parts = append(parts, "title contains any of "+strings.Join(quoted, ", "))
	}
	if len(parts) == 0 {
		return "no matchers"
	}
	return strings.Join(parts, " and ")
}

// OmniPitcherSystem is the system PitchPath gives a message that names none.
const OmniPitcherSystem = "homerun2-omni-pitcher"

// The reasons PitchPath rejects a message. omni-pitcher answers them with 400
// "Title is required" and "Message is required".
var (
	ErrPitchTitleRequired   = errors.New("title is required")
	ErrPitchMessageRequired = errors.New("message is required")
)

// Pitch is a message as omni-pitcher publishes it.
type Pitch struct {
	Message homerun.Message `json:"message"`
	// Defaulted names the fields omni-pitcher filled in, by their JSON names.
	Defaulted []string `json:"defaulted,omitempty"`
}

// PreparePitch does to msg what omni-pitcher's PitchPath does before routing
// and publishing it: it rejects a message without title or message text, then
// fills in an empty severity (info), author (unknown), timestamp (now, RFC
// 3339) and system (OmniPitcherSystem). Mirrors homerun2-omni-pitcher
// internal/handlers/pitch.go. PitchPathGrafana and PitchPathGitHub build their
// messages from the webhook payload instead.
func PreparePitch(msg homerun.Message, now time.Time) (Pitch, error) {
	if msg.Title == "" {
		return Pitch{}, ErrPitchTitleRequired
	}
	if msg.Message == "" {
		return Pitch{}, ErrPitchMessageRequired
	}

	p := Pitch{Message: msg}
	setDefault := func(field *string, name, value string) {
		if *field == "" {
			*field = value
			p.Defaulted = append(p.Defaulted, name)
		}
	}
	setDefault(&p.Message.Severity, "severity", "info")
	setDefault(&p.Message.Author, "author", "unknown")
	setDefault(&p.Message.Timestamp, "timestamp", now.Format(time.RFC3339))
	setDefault(&p.Message.System, "system", OmniPitcherSystem)
	return p, nil
}
