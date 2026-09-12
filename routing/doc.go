/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

// Package routing answers what a homerun Message would trigger in which
// component, without publishing it.
//
// Three layers decide that, and all of them are normally invisible:
//
//  1. Which stream a message lands on. A pitcher writing to Redis itself
//     names its stream. omni-pitcher first fills in the fields a pitch left
//     empty (PreparePitch) and then picks a stream per message from its
//     routing file, the one ROUTES_CONFIG points at (ParseStreamRoutes).
//  2. Which catcher sees a message at all. That is decided by the streams and
//     the consumer group each component runs with (REDIS_STREAMS /
//     REDIS_STREAM and CONSUMER_GROUP), not by any profile. A pitcher on a
//     stream no catcher reads still reports success.
//  3. What a catcher does with it. That is decided by its profile, and every
//     catcher has its own schema: light-catcher's effects, led-catcher's
//     displayRules and notification-catcher's outputs.
//
// The package parses each profile schema into a Profile and evaluates it with
// the same rules the catcher applies (see ParseLightProfile,
// ParseLEDProfile, ParseNotificationConfig). On top of that:
//
//   - DryRun: what every catcher does with one message published to a stream.
//   - BuildMatrix: severity × system → which catcher reacts how.
//   - Check: pitchers writing to streams nobody reads, catchers splitting one
//     consumer group, severities no rule reacts to, rules that match but
//     cannot act.
//
// It reads nothing itself - no Kubernetes API, no Redis. A caller such as a
// config viewer collects the Deployments and ConfigMaps and hands the
// Components in, so the evaluation needs no RBAC and no access to message
// content.
//
// # Keeping in step with the catchers
//
// Each evaluator mirrors a catcher's matching code, and StreamRoutes and
// PreparePitch mirror omni-pitcher's, named in their doc comments. A change to
// how a service matches has to be made here as well, or the dry run will
// confidently describe behaviour the service no longer has. The Go services
// can import this code instead of keeping their own copy; led-catcher is
// Python, so its tests are the reference.
package routing
