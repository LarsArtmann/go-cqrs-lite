package systemscenario

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// Option configures a Scenario at construction.
type Option func(*scenarioConfig)

type scenarioConfig struct {
	awaitTimeout  time.Duration
	quietWindow   time.Duration
	clock         system.Clock
	captureFilter func(command.Command) bool
}

func defaultConfig() scenarioConfig {
	return scenarioConfig{awaitTimeout: 5 * time.Second}
}

// WithAwaitTimeout sets how long poll-awaiting assertions (ThenQuery, and
// every assertion after TimeAdvances) wait before failing. Default: 5s.
func WithAwaitTimeout(d time.Duration) Option {
	return func(c *scenarioConfig) {
		if d > 0 {
			c.awaitTimeout = d
		}
	}
}

// WithClock replaces the scenario's default [system.ManualClock] (frozen at
// a fixed epoch) with another time source. TimeAdvances requires the
// ManualClock, so tests that exercise timers should not override it.
func WithClock(clock system.Clock) Option {
	return func(c *scenarioConfig) {
		if clock != nil {
			c.clock = clock
		}
	}
}

// WithQuietWindow bounds how long [WhenPhase.ThenNoEvents] waits for
// silence under await mode (after TimeAdvances or Await). Default: the full
// await timeout — the negative assertion must actually watch for late
// arrivals, not pass the instant it is called. Pass a shorter window to
// make negative tests fast while still catching arrivals within it.
func WithQuietWindow(d time.Duration) Option {
	return func(c *scenarioConfig) {
		if d > 0 {
			c.quietWindow = d
		}
	}
}

// WithCommandCaptureFilter restricts command capture to commands the
// filter accepts: background noise (lifecycle sweeps, keepalives) stops
// diluting ThenCommands baselines. The filter runs on dispatch goroutines
// inside the capture lock — it must be pure and fast. Filtered commands are
// never captured, so no Then* assertion sees them.
func WithCommandCaptureFilter(filter func(command.Command) bool) Option {
	return func(c *scenarioConfig) {
		if filter != nil {
			c.captureFilter = filter
		}
	}
}

// manualClockEpoch is the frozen instant every scenario starts at:
// deterministic timestamps in events and timers, no wall-clock coupling.
var manualClockEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// capturedCommand records one command dispatched through the system's
// dispatcher: the command and its dispatch outcome.
type capturedCommand struct {
	cmd command.Command
	err error
}

// Scenario is a booted system under Given/When/Then test. Construct via
// [System]; never zero-value it.
type Scenario struct {
	t        testing.TB
	ctx      context.Context
	sys      *system.System
	cfg      scenarioConfig
	asserted bool

	cmdMu            sync.Mutex
	capturedCommands []capturedCommand

	// baselines snapshot the journal and the captured-command log when the
	// first When act begins; Then* assertions diff against them.
	baselineEventIDs   map[string]struct{}
	baselineCommandIDs map[string]struct{}
	actStarted         bool

	// lastErr is the outcome of the most recent command act; lastQueryResult
	// the outcome of the most recent query act.
	lastErr         error
	lastQueryResult any

	// versionHints tracks the next version per stream for [Scenario.Event],
	// counting minted-but-unappended events so multiple mints in one Given
	// stamp strictly increasing versions.
	versionHints map[string]event.Version

	// awaitMode makes Then* assertions poll (set by TimeAdvances).
	awaitMode bool

	// clock is the scenario's time source (default: ManualClock frozen at
	// manualClockEpoch, injected into system.New via system.WithClock).
	clock system.Clock
}

// System boots a full system via [system.New] with the SAME DomainConfig and
// DeploymentConfig a production binary uses, installs command capture,
// starts the system, and registers cleanup. The returned Scenario drives
// Given/When/Then chains; a scenario that never runs a Then* assertion fails
// the test (vacuous-pass guard).
func System(
	t testing.TB,
	ctx context.Context,
	domain system.DomainConfig,
	deploy system.DeploymentConfig,
	opts ...Option,
) *Scenario {
	t.Helper()

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.clock == nil {
		cfg.clock = system.NewManualClock(manualClockEpoch)
	}

	sys, err := system.New(ctx, domain, deploy, system.WithClock(cfg.clock))
	if err != nil {
		t.Fatalf("systemscenario: system.New: %v", err)
	}

	sc := newScenario(t, ctx, sys, cfg)

	if err := sys.Start(ctx); err != nil {
		_ = sys.Close()

		t.Fatalf("systemscenario: system.Start: %v", err)
	}

	t.Cleanup(func() { sc.shutdown() })
	sc.requireTerminalAssertion()

	return sc
}

// Adopt wraps an ALREADY-BOOTED system (started or not) — the
// wrapper-library entry point: when a facade owns the system.New call (e.g.
// a service object exposing System()), Adopt installs command capture and
// the vacuous-pass guard over it so Given/When/Then works against the
// wrapper's boot. The caller owns the lifecycle — Adopt registers no
// shutdown cleanup.
func Adopt(
	t testing.TB,
	ctx context.Context,
	sys *system.System,
	opts ...Option,
) *Scenario {
	t.Helper()

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	sc := newScenario(t, ctx, sys, cfg)
	sc.requireTerminalAssertion()

	return sc
}

// newScenario assembles the Scenario and installs the capture middleware.
func newScenario(
	t testing.TB,
	ctx context.Context,
	sys *system.System,
	cfg scenarioConfig,
) *Scenario {
	t.Helper()

	sc := &Scenario{ //nolint:exhaustruct_v5 // baselines set at the first When act
		t:            t,
		ctx:          ctx,
		sys:          sys,
		cfg:          cfg,
		clock:        cfg.clock,
		versionHints: make(map[string]event.Version),
	}
	sys.UseCommandMiddleware(sc.captureMiddleware())

	return sc
}

// shutdown drains and closes the booted system on a background context so
// cleanup works even when the scenario context is already cancelled.
func (s *Scenario) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.sys.GracefulClose(ctx)
}

// System returns the booted system for escape hatches (typed queries via
// MetaEngine, direct store access, health inspection).
func (s *Scenario) System() *system.System { return s.sys }

// Phase returns the current WhenPhase for the package-level generic
// assertions ([ThenPayload], [ThenQueryTyped]) that Go methods cannot
// express. Safe before any act; the assertions themselves require one.
func (s *Scenario) Phase() *WhenPhase { return &WhenPhase{sc: s} }

// Clock returns the scenario's time source — a [system.ManualClock] frozen
// at 2026-01-01T00:00:00Z by default. Consumer Timers closures compute
// deterministic FireAt values from it and wire
// scheduling.WithClock(sys.Clock().Now) so When().TimeAdvances(d) makes
// timers fire without sleeping.
func (s *Scenario) Clock() system.Clock { return s.clock }

// requireTerminalAssertion registers a cleanup that fails the test if no
// Then* assertion ever ran — a scenario without a terminal assertion would
// otherwise pass vacuously. Port of scenario/dsl.go's guard. Accepts TB so
// benchmarks can drive scenarios without a vacuous failure.
func (s *Scenario) requireTerminalAssertion() {
	s.t.Cleanup(func() {
		if !s.asserted {
			s.t.Errorf(
				"systemscenario: no Then* assertion ran — this test passes vacuously; " +
					"end the chain with Then, ThenEvents, ThenQuery, ThenError, or a sibling",
			)
		}
	})
}

// captureMiddleware records every command dispatched through the system's
// dispatcher, in dispatch order, with its outcome. Installed by [System]
// before Start; ThenCommands diffs the log against the act baseline.
func (s *Scenario) captureMiddleware() command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			err := next(ctx, cmd)

			if s.cfg.captureFilter == nil || s.cfg.captureFilter(cmd) {
				s.cmdMu.Lock()
				s.capturedCommands = append(s.capturedCommands, capturedCommand{cmd: cmd, err: err})
				s.cmdMu.Unlock()
			}

			return err
		}
	}
}

// markActStart snapshots the journal and captured-command baselines at the
// first When act. Given-phase seeding happens before it, so Then* assertions
// see only the acts' effects.
func (s *Scenario) markActStart() {
	if s.actStarted {
		return
	}

	s.actStarted = true
	s.baselineEventIDs = s.journalIDSet()
	s.baselineCommandIDs = s.commandIDSet()
}

// requireAct fails the test when a Then* assertion runs before any When act.
func (s *Scenario) requireAct(method string) {
	s.t.Helper()
	s.asserted = true

	if !s.actStarted {
		s.t.Fatal(
			"systemscenario: call a When act (When, WhenEvent, WhenQuery) before " + method + "()",
		)
	}
}

// Event mints a correctly-versioned domain event for the stream ref via
// [event.New], stamping the next per-stream version (harness-tracked from
// the store), so Given clauses compose without manual version bookkeeping:
//
//	sc.Given(sc.Event("task.created", ref, TaskCreated{Title: "ship"}))
func (s *Scenario) Event(
	eventType event.Type,
	ref id.StreamRef,
	payload any,
	opts ...event.Option,
) event.Event {
	s.t.Helper()

	evt, err := event.New(eventType, ref.ID, ref.Type, s.nextVersion(ref), payload, opts...)
	if err != nil {
		s.t.Fatalf("systemscenario.Event(%s): %v", eventType, err)
	}

	return evt
}

// nextVersion returns the version the next event on ref should carry,
// counting minted-but-unappended events so consecutive [Scenario.Event]
// calls in one Given stamp strictly increasing versions.
func (s *Scenario) nextVersion(ref id.StreamRef) event.Version {
	s.t.Helper()

	key := ref.String()
	if next, ok := s.versionHints[key]; ok {
		s.versionHints[key] = next + 1

		return next
	}

	next := event.Version(1)
	if events, err := s.sys.EventStore().Load(s.ctx, ref); err == nil && len(events) > 0 {
		next = events[len(events)-1].Version() + 1
	} else if err != nil && !errors.Is(err, event.ErrStreamNotFound) {
		s.t.Fatalf("systemscenario.Event: load %s: %v", ref, err)
	}

	s.versionHints[key] = next + 1

	return next
}
