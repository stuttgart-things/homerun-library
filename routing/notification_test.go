/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"encoding/json"
	"strings"
	"testing"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

// Cases in this file are ported from homerun2-notification-catcher
// internal/notify/filter_test.go and internal/config/notify_test.go.

func matches(f NotificationFilters, msg homerun.Message) bool {
	c := &NotificationConfig{Outputs: []NotificationOutput{{Name: "o", Type: "webhook", Filters: f}}}
	return len(c.Evaluate(msg)) == 1
}

func TestNotificationMatches_EmptyFiltersMatchEverything(t *testing.T) {
	if !matches(NotificationFilters{}, homerun.Message{Severity: "info"}) {
		t.Error("empty filters should match every message")
	}
}

func TestNotificationMatches_SeverityMin(t *testing.T) {
	cases := []struct {
		min, sev string
		want     bool
	}{
		{"warning", "critical", true},
		{"warning", "error", true},
		{"warning", "warning", true},
		{"warning", "info", false},
		{"warning", "debug", false},
		{"info", "success", true},
		{"critical", "warning", false},
		{"critical", "error", true},
		{"error", "critical", true},
		// an unknown message severity counts as info
		{"info", "weird", true},
		{"warning", "weird", false},
		{"WARNING", "Error", true},
	}
	for _, tc := range cases {
		if got := matches(NotificationFilters{SeverityMin: tc.min}, homerun.Message{Severity: tc.sev}); got != tc.want {
			t.Errorf("min=%s sev=%s: got %v want %v", tc.min, tc.sev, got, tc.want)
		}
	}
}

func TestNotificationMatches_MatchMapIsExactAndCaseInsensitive(t *testing.T) {
	msg := homerun.Message{System: "Kubernetes", Author: "alertmanager", Severity: "warning"}

	if !matches(NotificationFilters{Match: map[string]string{"system": "kubernetes"}}, msg) {
		t.Error("system match should be case-insensitive")
	}
	if matches(NotificationFilters{Match: map[string]string{"system": "kubernet"}}, msg) {
		t.Error("match should be exact, not prefix")
	}
	if !matches(NotificationFilters{Match: map[string]string{"System": "Kubernetes", "author": "ALERTMANAGER"}}, msg) {
		t.Error("multiple match keys should AND, case-insensitive on key + value")
	}
}

func TestNotificationMatches_TagsContainIsOR(t *testing.T) {
	msg := homerun.Message{Tags: "infra,storage,disk"}

	if !matches(NotificationFilters{TagsContain: []string{"compute", "infra"}}, msg) {
		t.Error("any-substring should match infra")
	}
	if matches(NotificationFilters{TagsContain: []string{"compute", "network"}}, msg) {
		t.Error("no needle present → should not match")
	}
	if !matches(NotificationFilters{TagsContain: []string{"INFRA"}}, msg) {
		t.Error("substring match should be case-insensitive")
	}
}

func TestNotificationMatches_MessageContains(t *testing.T) {
	msg := homerun.Message{Message: "node01 OOM on container redis"}

	if !matches(NotificationFilters{MessageContains: []string{"disk", "OOM"}}, msg) {
		t.Error("OOM substring should match")
	}
	if matches(NotificationFilters{MessageContains: []string{"disk", "cpu"}}, msg) {
		t.Error("no needle present → should not match")
	}
}

func TestNotificationMatches_AllFiltersANDed(t *testing.T) {
	msg := homerun.Message{
		Severity: "warning",
		System:   "kubernetes",
		Tags:     "infra,storage",
		Message:  "disk almost full on node01",
	}
	f := NotificationFilters{
		SeverityMin:     "warning",
		Match:           map[string]string{"system": "kubernetes"},
		TagsContain:     []string{"infra"},
		MessageContains: []string{"disk"},
	}
	if !matches(f, msg) {
		t.Error("all rules satisfied → should match")
	}

	lower := msg
	lower.Severity = "info"
	if matches(f, lower) {
		t.Error("severity drops below floor → should not match")
	}

	other := msg
	other.System = "openshift"
	if matches(f, other) {
		t.Error("system mismatch → should not match")
	}
}

const notificationTestConfig = `outputs:
  - name: teams-ops
    type: msteams
    webhook_url: ${TEAMS_OPS_WEBHOOK}
    filters:
      severity_min: warning
      match:
        system: kubernetes
  - name: audit-webhook
    type: webhook
    url: https://audit.example.com/hook
    method: POST
    headers:
      Authorization: Bearer ${AUDIT_TOKEN}
  - name: storage
    type: webhook
    url: https://storage.example.com/hook
    filters:
      tags_contain: [storage]
`

func TestNotificationEvaluate_FansOutToEveryMatchingOutput(t *testing.T) {
	c, err := ParseNotificationConfig([]byte(notificationTestConfig))
	if err != nil {
		t.Fatal(err)
	}

	names := func(msg homerun.Message) string {
		var out []string
		for _, r := range c.Evaluate(msg) {
			out = append(out, r.Rule)
		}
		return strings.Join(out, ",")
	}

	if got := names(homerun.Message{System: "kubernetes", Severity: "error", Tags: "storage"}); got != "teams-ops,audit-webhook,storage" {
		t.Errorf("got %s", got)
	}
	if got := names(homerun.Message{System: "kubernetes", Severity: "info"}); got != "audit-webhook" {
		t.Errorf("got %s", got)
	}
	if got := c.Systems(); len(got) != 1 || got[0] != "kubernetes" {
		t.Errorf("Systems() = %v", got)
	}
}

func TestParseNotificationConfig_KeepsNoSecrets(t *testing.T) {
	c, err := ParseNotificationConfig([]byte(notificationTestConfig))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(c.Evaluate(homerun.Message{System: "kubernetes", Severity: "error", Tags: "storage"}))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"TEAMS_OPS_WEBHOOK", "audit.example.com", "AUDIT_TOKEN", "Authorization"} {
		if strings.Contains(string(encoded), secret) {
			t.Errorf("evaluation result contains %q: %s", secret, encoded)
		}
	}
}

func TestParseNotificationConfig_Rejects(t *testing.T) {
	cases := map[string]string{
		"invalid YAML":         "outputs: [unclosed",
		"missing name":         "outputs:\n  - type: webhook\n    url: http://x\n",
		"duplicate name":       "outputs:\n  - name: a\n    type: webhook\n    url: http://x\n  - name: a\n    type: webhook\n    url: http://y\n",
		"missing type":         "outputs:\n  - name: a\n    url: http://x\n",
		"unknown type":         "outputs:\n  - name: a\n    type: slack\n    url: http://x\n",
		"teams without url":    "outputs:\n  - name: a\n    type: msteams\n",
		"webhook without url":  "outputs:\n  - name: a\n    type: webhook\n    webhook_url: http://x\n",
		"unknown severity_min": "outputs:\n  - name: a\n    type: webhook\n    url: http://x\n    filters:\n      severity_min: fatal\n",
		"unknown match key":    "outputs:\n  - name: a\n    type: webhook\n    url: http://x\n    filters:\n      match:\n        tags: x\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseNotificationConfig([]byte(doc)); err == nil {
				t.Error("expected an error, notification-catcher refuses to start with this config")
			}
		})
	}
}

func TestParseNotificationConfig_NoOutputs(t *testing.T) {
	for _, doc := range []string{"", "outputs: []\n"} {
		c, err := ParseNotificationConfig([]byte(doc))
		if err != nil {
			t.Fatalf("%q: %v", doc, err)
		}
		if r := c.Evaluate(homerun.Message{Severity: "critical"}); len(r) != 0 {
			t.Errorf("%q: no outputs must send nothing, got %v", doc, r)
		}
	}
}
