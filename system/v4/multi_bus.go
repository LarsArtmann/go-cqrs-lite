package system

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// busEntry pairs a fan-out publisher with its deployment-config target name
// (empty for positionally added publishers). Pairing them in one slice keeps
// names and publishers from drifting apart.
type busEntry struct {
	name      string
	publisher event.Publisher
}

// MultiBus fans out event publishing to multiple [event.Publisher] instances.
// This implements D9 (multi-bus support): events from a source-of-truth instance
// can be published to multiple buses simultaneously (e.g., GoChannel for local
// projections + NATS for cross-service distribution).
//
// Entries may carry the Publish target name from the deployment config;
// address them via [MultiBus.PublisherByName] or [System.PublisherFor].
// Positionally added entries stay reachable via [MultiBus.Publishers].
//
// Publish is synchronous: all publishers receive the events before Publish
// returns. If any publisher fails, the first error is returned (remaining
// publishers may or may not have received the events).
type MultiBus struct {
	mu      sync.RWMutex
	entries []busEntry
}

// NewMultiBus creates a MultiBus from the given publishers. The entries are
// unnamed; name them with [MultiBus.AddNamedPublisher] instead.
func NewMultiBus(publishers ...event.Publisher) *MultiBus {
	entries := make([]busEntry, len(publishers))
	for i, p := range publishers {
		entries[i] = busEntry{publisher: p}
	}

	return &MultiBus{entries: entries}
}

// AddPublisher appends an unnamed publisher to the fan-out list.
func (m *MultiBus) AddPublisher(p event.Publisher) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = append(m.entries, busEntry{publisher: p})
}

// AddNamedPublisher appends a publisher bound to a Publish target name from
// the deployment config, addressable via [MultiBus.PublisherByName].
func (m *MultiBus) AddNamedPublisher(name string, p event.Publisher) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = append(m.entries, busEntry{name: name, publisher: p})
}

// Publishers returns a snapshot of the child publishers in fan-out order.
// Index 0 is always the local bus (if included during construction).
func (m *MultiBus) Publishers() []event.Publisher {
	m.mu.RLock()
	defer m.mu.RUnlock()

	publishers := make([]event.Publisher, len(m.entries))
	for i, e := range m.entries {
		publishers[i] = e.publisher
	}

	return publishers
}

// PublisherByName returns the fan-out publisher bound to the given Publish
// target name. Positionally added entries have no name and are not reachable
// by name. The bool is false when no entry is bound to that name.
func (m *MultiBus) PublisherByName(name string) (event.Publisher, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, e := range m.entries {
		if e.name != "" && e.name == name {
			return e.publisher, true
		}
	}

	return nil, false
}

// Names returns the named entries' target names in fan-out order. Unnamed
// entries are skipped.
func (m *MultiBus) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.entries))
	for _, e := range m.entries {
		if e.name != "" {
			names = append(names, e.name)
		}
	}

	return names
}

// Publish sends events to all registered publishers sequentially.
// Returns the first error encountered.
func (m *MultiBus) Publish(ctx context.Context, events ...event.Event) error {
	m.mu.RLock()
	entries := slices.Clone(m.entries)
	m.mu.RUnlock()

	for i, e := range entries {
		if err := e.publisher.Publish(ctx, events...); err != nil {
			return fmt.Errorf("system: multi-bus publisher %s: %w", describeEntry(i, e), err)
		}
	}

	return nil
}

// describeEntry names a fan-out entry for error messages: the target name
// when bound, the positional index otherwise.
func describeEntry(index int, e busEntry) string {
	if e.name != "" {
		return fmt.Sprintf("%q", e.name)
	}

	return fmt.Sprintf("%d", index)
}

// Compile-time assertion.
var _ event.Publisher = (*MultiBus)(nil)
