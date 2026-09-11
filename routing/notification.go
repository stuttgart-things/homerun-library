/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"fmt"
	"strings"

	homerun "github.com/stuttgart-things/homerun-library/v4"
	"go.yaml.in/yaml/v3"
)

// NotificationFilters narrows which messages reach a notification-catcher
// output (homerun2-notification-catcher internal/config OutputFilters).
type NotificationFilters struct {
	SeverityMin     string            `yaml:"severity_min,omitempty" json:"severityMin,omitempty"`
	Match           map[string]string `yaml:"match,omitempty" json:"match,omitempty"`
	TagsContain     []string          `yaml:"tags_contain,omitempty" json:"tagsContain,omitempty"`
	MessageContains []string          `yaml:"message_contains,omitempty" json:"messageContains,omitempty"`
}

// NotificationOutput is one notification-catcher output: a Teams channel or a
// webhook.
//
// Its URL and headers are deliberately not kept. They are secrets or
// references to secrets, and knowing where a message would go does not need
// them.
type NotificationOutput struct {
	Name    string              `yaml:"name" json:"name"`
	Type    string              `yaml:"type" json:"type"`
	Filters NotificationFilters `yaml:"filters,omitempty" json:"filters"`
}

// NotificationConfig is a parsed homerun2-notification-catcher config.yaml.
type NotificationConfig struct {
	Outputs []NotificationOutput
}

// notificationMatchFields are the Message fields a filters.match key may name.
var notificationMatchFields = map[string]bool{
	"severity": true, "system": true, "author": true, "title": true,
	"assigneename": true, "assigneeaddress": true,
}

// ParseNotificationConfig parses a notification-catcher config.yaml and
// rejects what notification-catcher refuses to start with: an output without a
// name, a duplicate name, an unknown type, a type without its URL, an unknown
// severity_min or match key.
//
// ${VAR} references are not interpolated. notification-catcher resolves them
// from its own environment and fails to start when one is unset; that cannot
// be checked from the config alone.
func ParseNotificationConfig(data []byte) (*NotificationConfig, error) {
	var doc struct {
		Outputs []struct {
			NotificationOutput `yaml:",inline"`
			WebhookURL         string `yaml:"webhook_url"`
			URL                string `yaml:"url"`
		} `yaml:"outputs"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse notification-catcher config: %w", err)
	}

	cfg := &NotificationConfig{}
	seen := map[string]bool{}
	for i, o := range doc.Outputs {
		prefix := fmt.Sprintf("output[%d]", i)
		if o.Name == "" {
			return nil, fmt.Errorf("notification-catcher config: %s: name is required", prefix)
		}
		prefix = fmt.Sprintf("output[%d] %q", i, o.Name)
		if seen[o.Name] {
			return nil, fmt.Errorf("notification-catcher config: %s: duplicate name", prefix)
		}
		seen[o.Name] = true

		switch o.Type {
		case "msteams":
			if strings.TrimSpace(o.WebhookURL) == "" {
				return nil, fmt.Errorf("notification-catcher config: %s: webhook_url is required for type %q", prefix, o.Type)
			}
		case "webhook":
			if strings.TrimSpace(o.URL) == "" {
				return nil, fmt.Errorf("notification-catcher config: %s: url is required for type %q", prefix, o.Type)
			}
		case "":
			return nil, fmt.Errorf("notification-catcher config: %s: type is required", prefix)
		default:
			return nil, fmt.Errorf("notification-catcher config: %s: unknown type %q", prefix, o.Type)
		}

		if s := strings.TrimSpace(o.Filters.SeverityMin); s != "" {
			if _, ok := severityRank(s); !ok {
				return nil, fmt.Errorf("notification-catcher config: %s: filters.severity_min %q is not a known severity", prefix, s)
			}
		}
		for k := range o.Filters.Match {
			if !notificationMatchFields[strings.ToLower(k)] {
				return nil, fmt.Errorf("notification-catcher config: %s: filters.match key %q is not a recognised field", prefix, k)
			}
		}

		cfg.Outputs = append(cfg.Outputs, o.NotificationOutput)
	}
	return cfg, nil
}

// Evaluate implements Profile, mirroring notification-catcher's Dispatch and
// Matches: every output whose filters all match receives the message.
//
//   - severity_min: the message's severity ranks at least as high; an unknown
//     message severity counts as info
//   - match: every key names a field whose value equals, case-insensitively
//   - tags_contain / message_contains: at least one entry is a
//     case-insensitive substring
func (c *NotificationConfig) Evaluate(msg homerun.Message) []Reaction {
	var out []Reaction
	for _, o := range c.Outputs {
		if !notificationMatches(o.Filters, msg) {
			continue
		}
		out = append(out, Reaction{
			Rule:    o.Name,
			Summary: fmt.Sprintf("%s output %s", o.Type, o.Name),
			Details: o,
		})
	}
	return out
}

func notificationMatches(f NotificationFilters, msg homerun.Message) bool {
	if !severityAtLeast(msg.Severity, f.SeverityMin) {
		return false
	}
	for k, want := range f.Match {
		got := notificationField(msg, k)
		if !strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(want)) {
			return false
		}
	}
	if len(f.TagsContain) > 0 && !anySubstring(msg.Tags, f.TagsContain) {
		return false
	}
	if len(f.MessageContains) > 0 && !anySubstring(msg.Message, f.MessageContains) {
		return false
	}
	return true
}

func severityAtLeast(msgSev, minSev string) bool {
	if strings.TrimSpace(minSev) == "" {
		return true
	}
	minRank, ok := severityRank(minSev)
	if !ok {
		return true
	}
	msgRank, ok := severityRank(msgSev)
	if !ok {
		msgRank, _ = severityRank("info")
	}
	return msgRank >= minRank
}

// severityRank ranks a severity as notification-catcher does; error and
// critical share the top rank.
func severityRank(s string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return 0, true
	case "info":
		return 1, true
	case "success":
		return 2, true
	case "warning":
		return 3, true
	case "critical", "error":
		return 4, true
	default:
		return 0, false
	}
}

func notificationField(msg homerun.Message, key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "severity":
		return msg.Severity
	case "system":
		return msg.System
	case "author":
		return msg.Author
	case "title":
		return msg.Title
	case "assigneename":
		return msg.AssigneeName
	case "assigneeaddress":
		return msg.AssigneeAddress
	default:
		return ""
	}
}

func anySubstring(haystack string, needles []string) bool {
	h := strings.ToLower(haystack)
	for _, n := range needles {
		if n != "" && strings.Contains(h, strings.ToLower(n)) {
			return true
		}
	}
	return false
}

// Systems implements Profile.
func (c *NotificationConfig) Systems() []string {
	var out []string
	for _, o := range c.Outputs {
		for k, v := range o.Filters.Match {
			if strings.EqualFold(strings.TrimSpace(k), "system") {
				out = appendSystems(out, []string{strings.TrimSpace(v)})
			}
		}
	}
	return out
}
