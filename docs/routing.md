# Routing: what would this message trigger?

```go
import "github.com/stuttgart-things/homerun-library/v4/routing"
```

Which catcher reacts to a message, and how, is decided in two places that are
both normally invisible:

1. **Which catcher sees it at all.** The streams (`REDIS_STREAMS` /
   `REDIS_STREAM`) and the `CONSUMER_GROUP` each component runs with. A pitcher
   on a stream no catcher reads still reports success.
2. **What the catcher does with it.** Its profile, in a schema of its own per
   catcher.

The `routing` package evaluates both without publishing anything. It reads
nothing itself — no Kubernetes API, no Redis. A caller, such as a config viewer,
collects the Deployments and ConfigMaps and passes `Component`s in.

## Profiles

| Catcher | Source | Parser | Matching |
|---|---|---|---|
| light-catcher | `profile.yaml` in ConfigMap `homerun2-light-catcher-profile` | `ParseLightProfile` | first effect in document order; `systems` exact or `*`; `severity` case-insensitive; every `tags` entry a whole element of the message tags |
| led-catcher | `profile.yaml` in ConfigMap `homerun2-led-catcher-profile` | `ParseLEDProfile` | first rule in document order; `systems` case-insensitive or `*`; `severity` case-insensitive, empty matches any; every `tags` entry a whole element of the message tags; a rule whose `image` template renders empty is skipped; missing system reads as `unknown`, missing severity as `info`; an `overlay` rule carries the next match as `Then`; `quietHours` are not applied |
| sound-catcher | `profile.yaml` | `ParseSoundProfile` | first rule in document order; `systems` **exact (case-sensitive)**, `*` or empty matches any; `severity` case-insensitive, empty matches any; every `tags` entry a whole element of the message tags; `silent: true` matches and plays nothing; `quietHours` not applied, but `SoundAction.QuietHoursSilence` says whether they silence the message's severity |
| notification-catcher | `config.yaml` | `ParseNotificationConfig` | every output whose filters all match: `severity_min` by rank, `match` exact case-insensitive, `tags_contain` / `message_contains` any substring |
| core-catcher, scout | — | `Profile: nil` | every message |

A parser rejects what the catcher cannot load. For a catcher whose profile does
not parse, use `routing.InvalidProfile{Err: err}`: it reports the error on every
message instead of pretending the catcher does nothing.

Each profile is a `Profile`:

```go
type Profile interface {
    Evaluate(msg homerun.Message) []Reaction
    Systems() []string
}

type Reaction struct {
    Rule    string // effect, display rule or output
    Summary string // one line
    Problem string // why it will not happen as configured
    Details any    // LightEffect, LEDDisplay, SoundAction or NotificationOutput
}
```

`Problem` catches rules that match but cannot act:

- light-catcher: unknown `color` or `fx`, or no `endpoint` — nothing is sent
- led-catcher: a severity without a color renders white, the "unknown severity"
  color (led-catcher's defaults have no `critical`)
- sound-catcher: a `say` template that uses a name sound-catcher does not
  define (e.g. `{{ timestamp }}` or `{{ tag('x') }}`) fails under
  `StrictUndefined`, so nothing is said — and nothing plays without a `sound`

A notification-catcher output's URL and headers are **not kept**. Knowing where
a message goes does not need the secret, and `${VAR}` references are not
interpolated.

## Components

```go
type Component struct {
    Name          string   // e.g. the Deployment name
    Role          Role     // RolePitcher or RoleCatcher
    Streams       []string
    ConsumerGroup string
    Profile       Profile  // catchers only; nil acts on every message
}
```

Resolve streams from a Deployment's environment with the component's own
default, which is not the same everywhere:

```go
streams := routing.ParseStreams(env["REDIS_STREAMS"], env["REDIS_STREAM"], "alerts") // notification-catcher
```

## Dry run

```go
msg := homerun.Message{Title: "Build failed", Severity: "error", System: "github"}

for _, d := range routing.DryRun(components, "messages", msg) {
    if !d.Receives {
        fmt.Printf("%s: does not read the stream\n", d.Component)
        continue
    }
    for _, r := range d.Reactions {
        fmt.Printf("%s: %s %s\n", d.Component, r.Summary, r.Problem)
    }
}
```

Every catcher appears, including those that do not read the stream, so a
message that reaches nobody is visible as such. `Delivery.SharedWith` lists
other catchers in the same consumer group on that stream: Redis hands each
message to only one of them.

For led-catcher, plain `{{ variable }}` and `{{ tag('name') }}` templates are
rendered (text and image). Anything else — filters, expressions, statements —
is returned as the raw template with `TextRendered: false` rather than
half-rendered; an image template that cannot be rendered here is taken to
match.

led-catcher rule options in the result (`LEDDisplay`):

- `overlay: true` — the display plays first, then the same message as the next
  matching rule draws it (`Then`, one step, no chain). Summary:
  `LED overlay static "…" … for 5s, then LED text "…" … scrolling`.
- `ambient: true` (homerun2-led-catcher#161, the radio-pitcher's rule) — shown
  only on a free panel, dropped when it is busy, cut short by any other
  display; never held. The summary ends in
  `ambient: only on a free panel, cut short by any other display`.

## Static matrix

```go
m := routing.BuildMatrix(components, "messages", nil) // nil = routing.Severities
cell, _ := m.Cell(routing.OtherSystem, "critical")
```

One cell per system × severity. Systems are those any rule names, plus
`routing.OtherSystem` for everything else. A cell evaluates a message carrying
only that system and severity, so rules that also require tags or message text
do not match in it.

## Check

```go
for _, f := range routing.Check(components, []string{"error", "critical"}) {
    fmt.Println(f.Kind, f.Message)
}
```

| Kind | Meaning |
|---|---|
| `unread-stream` | a pitcher publishes to a stream no catcher reads |
| `shared-consumer-group` | different catchers read a stream in one consumer group |
| `uncovered-severity` | a catcher with rules does not react to one of the given severities from a system its rules do not name |
| `broken-rule` | a rule cannot act as configured — found whether or not a message reaches it |

## Keeping in step with the catchers

Each evaluator mirrors a catcher's matching code. A change to how a catcher
matches must be made here as well, or the dry run describes behaviour the
catcher no longer has. The package's tests are ported from the catchers' own
tests; the Go catchers can import the evaluators instead of keeping a copy.
