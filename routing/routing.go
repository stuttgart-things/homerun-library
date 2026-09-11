/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package routing

import (
	"slices"
	"strings"

	homerun "github.com/stuttgart-things/homerun-library/v4"
)

// Severities is the severity vocabulary the homerun2 catchers name in their
// profiles, from least to most severe. error and critical share a rank.
var Severities = []string{"debug", "info", "success", "warning", "error", "critical"}

// OtherSystem stands for "a system no rule names" in a Matrix. It is used as
// the System of the evaluated message, so it must not be a real system name.
const OtherSystem = "(other)"

// Role is what a component does with the streams it is configured with.
type Role string

const (
	// RolePitcher publishes to its streams.
	RolePitcher Role = "pitcher"
	// RoleCatcher reads its streams as a member of its consumer group.
	RoleCatcher Role = "catcher"
)

// Component is one deployed homerun2 component.
type Component struct {
	// Name identifies the component, e.g. its Deployment name. Replicas of one
	// Deployment are one Component.
	Name string `json:"name"`
	Role Role   `json:"role"`
	// Streams are the streams the component publishes to or reads, as
	// resolved by ParseStreams.
	Streams []string `json:"streams"`
	// ConsumerGroup is the catcher's CONSUMER_GROUP. Empty means unknown.
	ConsumerGroup string `json:"consumerGroup,omitempty"`
	// Profile decides what a catcher does with a message it reads. A nil
	// Profile acts on every message (core-catcher, scout). Ignored for
	// pitchers.
	Profile Profile `json:"-"`
}

// reads reports whether c is a catcher reading stream.
func (c Component) reads(stream string) bool {
	return c.Role == RoleCatcher && slices.Contains(c.Streams, stream)
}

// Profile is a catcher's parsed rule set.
type Profile interface {
	// Evaluate returns what the catcher does with msg, in the order it does
	// it. No reactions means the catcher ignores the message.
	Evaluate(msg homerun.Message) []Reaction
	// Systems returns the systems the rules name explicitly, without
	// wildcards, in rule order.
	Systems() []string
}

// Reaction is one thing a catcher does with a message.
type Reaction struct {
	// Rule names the effect, display rule or output that matched.
	Rule string `json:"rule,omitempty"`
	// Summary describes the reaction in one line.
	Summary string `json:"summary"`
	// Problem, when set, says why the reaction will not happen as the rule
	// intends: a rule that matches an unknown effect, say, so nothing lights
	// up at all.
	Problem string `json:"problem,omitempty"`
	// Details carries the schema-specific result: LightEffect, LEDDisplay or
	// NotificationOutput.
	Details any `json:"details,omitempty"`
}

// problemLister is implemented by the Profiles of this package: it lists the
// rules that cannot act as configured, whether or not a message reaches them.
type problemLister interface {
	problems() []Reaction
}

// acceptAll is the Profile of a catcher without rules.
var acceptAll = []Reaction{{Summary: "receives every message"}}

// InvalidProfile is the Profile of a catcher whose profile could not be
// parsed. It reacts to every message with the parse error as its problem.
type InvalidProfile struct {
	Err error
}

// Evaluate implements Profile.
func (p InvalidProfile) Evaluate(homerun.Message) []Reaction {
	return []Reaction{{Summary: "does nothing", Problem: "profile is invalid: " + p.Err.Error()}}
}

// Systems implements Profile.
func (InvalidProfile) Systems() []string { return nil }

func (p InvalidProfile) problems() []Reaction { return p.Evaluate(homerun.Message{}) }

// ParseStreams resolves the streams a component uses from its environment,
// the same way the catchers do:
//
//  1. streamsEnv (REDIS_STREAMS), comma-separated, blanks dropped
//  2. streamEnv (REDIS_STREAM), the legacy single stream
//  3. defaultStream, the component's built-in default
//
// The built-in default differs per component - notification-catcher reads
// "alerts" where the others read "messages" - which is exactly why it has to
// be passed in.
func ParseStreams(streamsEnv, streamEnv, defaultStream string) []string {
	var out []string
	for _, s := range strings.Split(streamsEnv, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	if len(out) > 0 {
		return out
	}
	if streamEnv != "" {
		return []string{streamEnv}
	}
	return []string{defaultStream}
}

// Delivery is what one catcher does with a message published to a stream.
type Delivery struct {
	Component string `json:"component"`
	// Receives reports whether the catcher reads the stream at all.
	Receives bool `json:"receives"`
	// SharedWith lists the other catchers reading the stream in the same
	// consumer group. Redis hands each message to only one member of a group,
	// so the catcher receives the message or one of these does - not both.
	SharedWith []string `json:"sharedWith,omitempty"`
	// Reactions is what the catcher does if it receives the message.
	Reactions []Reaction `json:"reactions,omitempty"`
}

// DryRun returns, for every catcher in components, what it would do with msg
// published to stream. Catchers that do not read the stream are included with
// Receives false, so a message that reaches nobody is visible as such.
func DryRun(components []Component, stream string, msg homerun.Message) []Delivery {
	var out []Delivery
	for _, c := range components {
		if c.Role != RoleCatcher {
			continue
		}
		d := Delivery{Component: c.Name, Receives: c.reads(stream)}
		if d.Receives {
			d.SharedWith = sharedGroup(components, c, stream)
			d.Reactions = evaluate(c, msg)
		}
		out = append(out, d)
	}
	return out
}

func evaluate(c Component, msg homerun.Message) []Reaction {
	if c.Profile == nil {
		return acceptAll
	}
	return c.Profile.Evaluate(msg)
}

// sharedGroup returns the other catchers reading stream in c's consumer group.
func sharedGroup(components []Component, c Component, stream string) []string {
	if c.ConsumerGroup == "" {
		return nil
	}
	var names []string
	for _, other := range components {
		if other.Name != c.Name && other.ConsumerGroup == c.ConsumerGroup && other.reads(stream) {
			names = append(names, other.Name)
		}
	}
	return names
}
