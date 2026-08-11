package sqliteengine

import (
	"context"
	"errors"
	"time"
)

// ErrNoProber is returned by Probe when no probe function has been configured
// via SetProber. ProbeEngine's IsRemote check prevents this from being hit for
// local SQLite databases; it only surfaces if a caller explicitly probes a
// non-remote engine that has not called SetProber.
var ErrNoProber = errors.New("sqliteengine: no prober configured")

// ProberSetter is implemented by *sqliteEngine. Wrapper packages (e.g.
// tursoengine) use it to inject a live probe function so ProbeEngine can
// measure RTT to a remote server without exporting the concrete engine type.
type ProberSetter interface {
	SetProber(fn func(context.Context) (time.Duration, error))
}

// SetProber installs a probe function used by [metaengine.ProbeEngine] to
// measure live RTT. Wrapper engines (e.g. tursoengine wrapping a remote libSQL
// connection) call this at construction time to inject a probe that times a
// real network round-trip (e.g. db.PingContext). Local SQLite databases should
// not call SetProber — their NetworkRTT is structurally zero.
func (e *sqliteEngine) SetProber(fn func(context.Context) (time.Duration, error)) {
	e.probeFn = fn
}

// Probe measures the current round-trip by calling the configured probe
// function. It implements [metaengine.Prober] so ProbeEngine can build a live
// RTT tracker. Returns ErrNoProber when no probe function is set.
func (e *sqliteEngine) Probe(ctx context.Context) (time.Duration, error) {
	if e.probeFn == nil {
		return 0, ErrNoProber
	}

	return e.probeFn(ctx)
}
