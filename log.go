/*
Copyright © 2026 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"log/slog"
	"sync/atomic"
)

// A library should be silent unless the program embedding it asks for output.
// Until v3.1.7 this package installed a pterm logger at trace level at import
// time, so every consumer got ANSI-coloured decoration on stdout with no way to
// silence, redirect or reformat it - which corrupts a TUI and makes a JSON log
// stream unparseable.
//
// The messages themselves are useful, so they are kept and routed through an
// injected *slog.Logger that discards everything by default.
var logger atomic.Pointer[slog.Logger]

func init() {
	logger.Store(slog.New(slog.DiscardHandler))
}

// SetLogger installs the logger this package writes to. Passing nil restores
// the default, which discards every record.
//
// Typical use from a service that already has a logger:
//
//	homerun.SetLogger(slog.Default())
//
// It is safe to call at any time and from any goroutine.
func SetLogger(l *slog.Logger) {
	if l == nil {
		l = slog.New(slog.DiscardHandler)
	}
	logger.Store(l)
}

// log returns the currently installed logger. It never returns nil.
func log() *slog.Logger {
	return logger.Load()
}
