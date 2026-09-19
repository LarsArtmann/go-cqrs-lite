package engine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sync"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// DefaultLease is how long a Due claim fences other pollers when no lease is
// configured. Alias of [metaengine.DefaultClaimLease].
const DefaultLease = metaengine.DefaultClaimLease

// TimerStore adapts a metaengine.DueClaimer engine into a
// [scheduling.TimerStore]. Multiple TimerStore instances over DIFFERENT
// engines coexist freely; over the SAME engine collection they share one
// claim table and fence each other (multi-instance safe, at-least-once).
type TimerStore[P any] struct {
	claims     metaengine.DueClaimer
	collection string
	owner      string
	lease      time.Duration

	mu sync.Mutex
	// epochs maps timer ID → the DueAt of the generation THIS store
	// claimed last (MarkFired's epoch guard). Entries are removed on
	// MarkFired/Cancel; an unknown epoch degrades to an unconditional
	// delete (the pre-ADR-0142 behavior, still correct for the common
	// Schedule→Due→MarkFired cycle).
	epochs map[string]time.Time
}

// Option configures a TimerStore.
type Option func(*config)

type config struct {
	collection string
	owner      string
	lease      time.Duration
}

// WithCollection names the engine claims collection (default "timers").
// One collection per payload type P is the norm.
func WithCollection(name string) Option {
	return func(c *config) {
		if name != "" {
			c.collection = name
		}
	}
}

// WithOwner names the claiming owner stamped into leases (default
// "scheduler"); owner-scoped when several dispatchers share one store.
func WithOwner(owner string) Option {
	return func(c *config) {
		if owner != "" {
			c.owner = owner
		}
	}
}

// WithLease sets how long a Due claim fences other pollers (default
// [DefaultLease]).
func WithLease(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.lease = d
		}
	}
}

// NewTimerStore adapts any engine implementing [metaengine.DueClaimer] into
// a [scheduling.TimerStore]. Engines without the capability are rejected —
// check metaengine.SupportsDueClaims(eng) first when the engine is
// operator-supplied.
func NewTimerStore[P any](eng metaengine.Engine, opts ...Option) (*TimerStore[P], error) {
	claims, ok := eng.(metaengine.DueClaimer)
	if !ok {
		return nil, fmt.Errorf(
			"scheduling/engine: engine %s does not implement metaengine.DueClaimer (ADR-0142 capability)",
			eng.Profile().Name,
		)
	}

	cfg := config{collection: "timers", owner: "scheduler", lease: DefaultLease}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &TimerStore[P]{
		claims:     claims,
		collection: cfg.collection,
		owner:      cfg.owner,
		lease:      cfg.lease,
		epochs:     map[string]time.Time{},
	}, nil
}

// Schedule records a timer; idempotent by ID (an existing unfired timer is
// left untouched — TimerStore contract).
func (s *TimerStore[P]) Schedule(ctx context.Context, t scheduling.Timer[P]) error {
	payload, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("scheduling/engine: encode timer: %w", err)
	}

	if err := s.claims.ClaimInsert(ctx, s.collection, t.ID.Get(), t.FireAt, payload); err != nil {
		return fmt.Errorf("scheduling/engine: schedule %s: %w", t.ID.Get(), err)
	}

	return nil
}

// Due claims and returns timers whose FireAt is at or before now, ordered by
// FireAt ascending (ID tie-break). Claiming is the delivery fence: a second
// Due before MarkFired/Cancel sees nothing (single dispatcher), and a
// crashed dispatcher's timers become claimable again after the lease
// (multi-instance, at-least-once).
func (s *TimerStore[P]) Due(ctx context.Context, now time.Time) ([]scheduling.Timer[P], error) {
	var due []scheduling.Timer[P]

	for {
		batch, err := s.claims.ClaimDue(ctx, metaengine.ClaimDueRequest{
			Collection: s.collection,
			Owner:      s.owner,
			Lease:      s.lease,
			Now:        now,
		})
		if err != nil {
			return due, fmt.Errorf("scheduling/engine: due claim: %w", err)
		}

		for _, claim := range batch {
			var t scheduling.Timer[P]

			if err := json.Unmarshal(claim.Payload, &t); err != nil {
				return due, fmt.Errorf("scheduling/engine: decode timer %s: %w", claim.Key, err)
			}

			s.rememberEpoch(t.ID.Get(), claim.DueAt)
			due = append(due, t)
		}

		if len(batch) < metaengine.DefaultClaimLimit {
			return due, nil
		}
	}
}

// MarkFired removes a timer after dispatch, epoch-guarded: a stale
// MarkFired for generation N cannot delete a re-scheduled generation N+1
// scheduled under the same ID (the scheduler.go race, fixed structurally).
func (s *TimerStore[P]) MarkFired(ctx context.Context, id scheduling.TimerID) error {
	if epoch, ok := s.takeEpoch(id.Get()); ok {
		if err := s.claims.ClaimDeleteIfDue(ctx, s.collection, id.Get(), epoch); err != nil {
			return fmt.Errorf("scheduling/engine: mark fired %s: %w", id.Get(), err)
		}

		return nil
	}

	if err := s.claims.ClaimDelete(ctx, s.collection, id.Get()); err != nil {
		return fmt.Errorf("scheduling/engine: mark fired %s: %w", id.Get(), err)
	}

	return nil
}

// Cancel removes a timer before it fires.
func (s *TimerStore[P]) Cancel(ctx context.Context, id scheduling.TimerID) error {
	s.takeEpoch(id.Get())

	if err := s.claims.ClaimDelete(ctx, s.collection, id.Get()); err != nil {
		return fmt.Errorf("scheduling/engine: cancel %s: %w", id.Get(), err)
	}

	return nil
}

func (s *TimerStore[P]) rememberEpoch(key string, dueAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.epochs[key] = dueAt
}

func (s *TimerStore[P]) takeEpoch(key string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	epoch, ok := s.epochs[key]
	if ok {
		delete(s.epochs, key)
	}

	return epoch, ok
}
