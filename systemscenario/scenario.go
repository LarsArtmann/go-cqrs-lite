package systemscenario

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	awaitTimeout time.Duration
	clock        system.Clock
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
	t        *testing.T
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
	t *testing.T,
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

	sc := &Scenario{ //nolint:exhaustruct_v5 // baselines set at the first When act
		t:            t,
		ctx:          ctx,
		sys:          sys,
		cfg:          cfg,
		clock:        cfg.clock,
		versionHints: make(map[string]event.Version),
	}
	sys.UseCommandMiddleware(sc.captureMiddleware())

	if err := sys.Start(ctx); err != nil {
		_ = sys.Close()

		t.Fatalf("systemscenario: system.Start: %v", err)
	}

	t.Cleanup(func() { sc.shutdown() })
	sc.requireTerminalAssertion()

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

// Clock returns the scenario's time source — a [system.ManualClock] frozen
// at 2026-01-01T00:00:00Z by default. Consumer Timers closures compute
// deterministic FireAt values from it and wire
// scheduling.WithClock(sys.Clock().Now) so When().TimeAdvances(d) makes
// timers fire without sleeping.
func (s *Scenario) Clock() system.Clock { return s.clock }

// requireTerminalAssertion registers a cleanup that fails the test if no
// Then* assertion ever ran — a scenario without a terminal assertion would
// otherwise pass vacuously. Port of scenario/dsl.go's guard.
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

			s.cmdMu.Lock()
			s.capturedCommands = append(s.capturedCommands, capturedCommand{cmd: cmd, err: err})
			s.cmdMu.Unlock()

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

// journal returns the full event journal, ordered by occurrence.
func (s *Scenario) journal() []event.Event {
	s.t.Helper()

	journal, ok := s.sys.EventStore().(event.Journal)
	if !ok {
		s.t.Fatal("systemscenario: event store does not implement event.Journal (ReadAll)")
	}

	events, err := journal.ReadAll(s.ctx)
	if err != nil {
		s.t.Fatalf("systemscenario: read journal: %v", err)
	}

	return events
}

// journalIDSet returns the set of event IDs currently in the journal.
func (s *Scenario) journalIDSet() map[string]struct{} {
	ids := make(map[string]struct{})

	for _, evt := range s.journal() {
		ids[evt.ID().String()] = struct{}{}
	}

	return ids
}

// commandIDSet returns the set of command IDs captured so far.
func (s *Scenario) commandIDSet() map[string]struct{} {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()

	ids := make(map[string]struct{}, len(s.capturedCommands))
	for _, entry := range s.capturedCommands {
		ids[entry.cmd.ID().String()] = struct{}{}
	}

	return ids
}

// actEvents returns the journal events emitted since the act baseline,
// preserving journal order.
func (s *Scenario) actEvents() []event.Event {
	events := s.journal()
	acted := make([]event.Event, 0, len(events))

	for _, evt := range events {
		if _, given := s.baselineEventIDs[evt.ID().String()]; !given {
			acted = append(acted, evt)
		}
	}

	return acted
}

// actCommands returns the commands dispatched since the act baseline,
// preserving dispatch order.
func (s *Scenario) actCommands() []command.Command {
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()

	acted := make([]command.Command, 0, len(s.capturedCommands))
	for _, entry := range s.capturedCommands {
		if _, given := s.baselineCommandIDs[entry.cmd.ID().String()]; !given {
			acted = append(acted, entry.cmd)
		}
	}

	return acted
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

// describeEvents renders one line per event for failure diagnostics:
// type, version, stream, and actor.
func describeEvents(events []event.Event) string {
	if len(events) == 0 {
		return "(no events)"
	}

	out := ""

	var outSb315 strings.Builder
	for _, evt := range events {
		fmt.Fprintf(&outSb315, "\n  - %s v%d on %s:%s (actor: %s)",
			evt.Type(), evt.Version(), evt.StreamType(), evt.StreamID(), evt.Metadata().ActorID)
	}

	out += outSb315.String()

	return out
}
