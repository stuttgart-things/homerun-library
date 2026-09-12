/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"fmt"
	"slices"
	"strings"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

// Matrix is the static view of one stream: for every severity and system,
// what each catcher reading the stream does with a message carrying just that
// severity and system.
//
// Rules that also look at tags or message text (light-catcher tags,
// notification-catcher tags_contain and message_contains) do not match such a
// message, so a cell shows what happens to a message without them. For the
// same reason a led-catcher text is shown as its template, not rendered. Use
// DryRun for a concrete message.
type Matrix struct {
	Stream string `json:"stream"`
	// Catchers are the catchers reading the stream, in component order.
	Catchers []string `json:"catchers"`
	// Systems are the systems any of those catchers' rules name, in rule
	// order, followed by OtherSystem.
	Systems    []string `json:"systems"`
	Severities []string `json:"severities"`
	// Cells holds one cell per system and severity, systems outermost.
	Cells []Cell `json:"cells"`
}

// Cell is one system × severity entry of a Matrix.
type Cell struct {
	System   string `json:"system"`
	Severity string `json:"severity"`
	// Deliveries holds one entry per catcher in Matrix.Catchers.
	Deliveries []Delivery `json:"deliveries"`
}

// Cell returns the cell for system and severity, or false if the matrix has
// none.
func (m Matrix) Cell(system, severity string) (Cell, bool) {
	for _, c := range m.Cells {
		if c.System == system && c.Severity == severity {
			return c, true
		}
	}
	return Cell{}, false
}

// BuildMatrix builds the static view of stream. A nil severities uses
// Severities.
func BuildMatrix(components []Component, stream string, severities []string) Matrix {
	if severities == nil {
		severities = Severities
	}
	m := Matrix{Stream: stream, Severities: severities}

	var readers []Component
	for _, c := range components {
		if c.reads(stream) {
			readers = append(readers, c)
			m.Catchers = append(m.Catchers, c.Name)
			if c.Profile != nil {
				m.Systems = appendSystems(m.Systems, c.Profile.Systems())
			}
		}
	}
	m.Systems = append(m.Systems, OtherSystem)

	for _, system := range m.Systems {
		for _, severity := range severities {
			msg := homerun.Message{System: system, Severity: severity}
			deliveries := DryRun(readers, stream, msg)
			showLEDTemplates(readers, deliveries)
			m.Cells = append(m.Cells, Cell{
				System:     system,
				Severity:   severity,
				Deliveries: deliveries,
			})
		}
	}
	return m
}

// showLEDTemplates puts the text template of each led-catcher rule into a
// cell instead of its rendering. A cell's message carries only a system and a
// severity, so the rendering would be "(other): " or "github: " - text
// led-catcher never displays for a real message.
func showLEDTemplates(readers []Component, deliveries []Delivery) {
	for i := range deliveries {
		k := slices.IndexFunc(readers, func(c Component) bool { return c.Name == deliveries[i].Component })
		if k < 0 {
			continue
		}
		profile, ok := readers[k].Profile.(*LEDProfile)
		if !ok || len(deliveries[i].Reactions) == 0 {
			continue
		}
		reactions := slices.Clone(deliveries[i].Reactions)
		for j, r := range reactions {
			d, isLED := r.Details.(LEDDisplay)
			rule := slices.IndexFunc(profile.Rules, func(rule LEDRule) bool { return rule.Name == r.Rule })
			if !isLED || rule < 0 {
				continue
			}
			d.Text, d.TextRendered = profile.Rules[rule].Text, false
			reactions[j].Details, reactions[j].Summary = d, ledSummary(d)
		}
		deliveries[i].Reactions = reactions
	}
}

// FindingKind classifies a Finding.
type FindingKind string

const (
	// FindingUnreadStream: a pitcher publishes to a stream no catcher reads.
	// The pitcher still reports success and nothing happens.
	FindingUnreadStream FindingKind = "unread-stream"
	// FindingSharedConsumerGroup: different catchers read a stream in the same
	// consumer group, so each message reaches only one of them.
	FindingSharedConsumerGroup FindingKind = "shared-consumer-group"
	// FindingUncoveredSeverity: a catcher with rules does not react to a
	// severity from a system its rules do not name.
	FindingUncoveredSeverity FindingKind = "uncovered-severity"
	// FindingBrokenRule: a rule matches but cannot act as configured.
	FindingBrokenRule FindingKind = "broken-rule"
)

// Finding is one configuration problem found by Check.
type Finding struct {
	Kind       FindingKind `json:"kind"`
	Component  string      `json:"component,omitempty"`
	Stream     string      `json:"stream,omitempty"`
	Rule       string      `json:"rule,omitempty"`
	Severities []string    `json:"severities,omitempty"`
	Message    string      `json:"message"`
}

// Check looks for configuration problems across components.
//
// Broken rules are found for the Profiles of this package, including rules no
// message currently reaches.
//
// mustReact are the severities every catcher with rules is expected to react
// to, e.g. error and critical: a catcher that has no rule for one of them from
// an arbitrary system is reported. Catchers without rules react to everything
// and are not checked. A catcher whose rules deliberately cover only some
// systems will be reported too, so choose mustReact per catcher kind if that
// is intended.
func Check(components []Component, mustReact []string) []Finding {
	var findings []Finding

	for _, p := range components {
		if p.Role != RolePitcher {
			continue
		}
		for _, s := range p.Streams {
			if !slices.ContainsFunc(components, func(c Component) bool { return c.reads(s) }) {
				findings = append(findings, Finding{
					Kind:      FindingUnreadStream,
					Component: p.Name,
					Stream:    s,
					Message:   fmt.Sprintf("%s publishes to stream %q, which no catcher reads", p.Name, s),
				})
			}
		}
	}

	for _, stream := range catcherStreams(components) {
		groups := map[string][]string{}
		var order []string
		for _, c := range components {
			if c.ConsumerGroup == "" || !c.reads(stream) {
				continue
			}
			if _, ok := groups[c.ConsumerGroup]; !ok {
				order = append(order, c.ConsumerGroup)
			}
			groups[c.ConsumerGroup] = append(groups[c.ConsumerGroup], c.Name)
		}
		for _, g := range order {
			if names := groups[g]; len(names) > 1 {
				findings = append(findings, Finding{
					Kind:    FindingSharedConsumerGroup,
					Stream:  stream,
					Message: fmt.Sprintf("%s read stream %q in the same consumer group %q: each message reaches only one of them", strings.Join(names, ", "), stream, g),
				})
			}
		}
	}

	seenBroken := map[string]bool{}
	for _, c := range components {
		if c.Role != RoleCatcher || c.Profile == nil {
			continue
		}

		var uncovered []string
		for _, sev := range mustReact {
			if len(c.Profile.Evaluate(homerun.Message{System: OtherSystem, Severity: sev})) == 0 {
				uncovered = append(uncovered, sev)
			}
		}
		if len(uncovered) > 0 {
			findings = append(findings, Finding{
				Kind:       FindingUncoveredSeverity,
				Component:  c.Name,
				Severities: uncovered,
				Message:    fmt.Sprintf("%s does not react to severity %s from a system its rules do not name", c.Name, strings.Join(uncovered, ", ")),
			})
		}

		if lister, ok := c.Profile.(problemLister); ok {
			for _, r := range lister.problems() {
				key := c.Name + "\x00" + r.Rule + "\x00" + r.Problem
				if seenBroken[key] {
					continue
				}
				seenBroken[key] = true
				findings = append(findings, Finding{
					Kind:      FindingBrokenRule,
					Component: c.Name,
					Rule:      r.Rule,
					Message:   r.Problem,
				})
			}
		}
	}

	return findings
}

// catcherStreams returns every stream a catcher reads, in first-seen order.
func catcherStreams(components []Component) []string {
	var out []string
	for _, c := range components {
		if c.Role != RoleCatcher {
			continue
		}
		for _, s := range c.Streams {
			if !slices.Contains(out, s) {
				out = append(out, s)
			}
		}
	}
	return out
}
