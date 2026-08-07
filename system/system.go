// Package system is the deployer-driven composition root for go-cqrs-lite.
//
// System replaces [stack.Bundle] with a model where the consumer provides
// only domain types (DomainConfig) and the operator provides only
// infrastructure decisions (DeploymentConfig). The two are separate types,
// compiler-enforced per decision D11.
//
// The System owns ALL infrastructure wiring: storage instances, bus(es),
// projectionhost, and dispatchers (D6). Internally it uses N
// [metaengine.Store] instances across two layers (source-of-truth +
// projections), connected via the [metaengine.StreamLogBackend] interface.
package system

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// ─── Op[State] and Execute (D10: declarative command routing) ───

// Op[State] carries the information the System needs to execute a command
// against a decider. It is returned by command handlers registered via
// [RegisterCommand].
type Op[State any] struct {
	streamID   id.StreamID
	streamType id.StreamType
	decide     decider.DecideFunc[State]
}

// Execute creates an [Op] that the System will execute against the decider
// registered for streamType.
func Execute[State any](_ context.Context, streamID id.StreamID, streamType id.StreamType,
	decide decider.DecideFunc[State],
) Op[State] {
	return Op[State]{
		streamID:   streamID,
		streamType: streamType,
		decide:     decide,
	}
}

// StreamID returns the target stream for this Op.
func (o Op[State]) StreamID() id.StreamID { return o.streamID }

// StreamType returns the stream type (entity type) for this Op.
func (o Op[State]) StreamType() id.StreamType { return o.streamType }

// ─── System type ───

// System is the composition root that owns ALL infrastructure wiring (D6).
// It is constructed via [New] with a [DomainConfig] and a [DeploymentConfig].
//
// The consumer interacts with the System via:
//   - [System.Command] — register typed command handlers
//   - [System.RegisterDecider] — register a decider for a stream type
//   - [System.Query] — register typed query handlers
//   - [System.UseCommandMiddleware] — inject domain middleware
//   - [System.MetaEngine] — access the projection-layer store
//   - [System.Start] / [System.Stop] — lifecycle
type System struct {
	mu sync.RWMutex

	deployment DeploymentConfig

	// Adapters wrap StreamLogBackend as standard CQRS interfaces.
	eventStore event.Store
	cmdStore   command.Store
	queryStore query.QueryStore

	// snapStore holds the snapshot store (nil if engine lacks SnapshotBackend).
	snapStore snapshot.SnapshotStore

	// Infrastructure owned by the System (D6).
	repos    map[string]any // streamType -> *decider.Repository[State]
	deciders map[string]any // streamType -> decider.Decider[State]

	cmdDisp  *command.Dispatcher
	qryDisp  *query.Dispatcher
	projHost *projectionhost.Host
	bus      event.Bus
	pubBus   event.Publisher // publisher for decider repository (may be MultiBus)

	// Projection-layer metaengine store.
	projStore *metaengine.Store

	// Engines and engine names for lifecycle management.
	engines     []metaengine.Engine
	engineNames []string // parallel to engines; empty if unnamed

	// shutdownDeps declares ordering constraints for Close(). Each edge says
	// "before must close before after". Resources not in any edge keep their
	// creation order. Ported from [stack.Bundle].
	shutdownDeps []shutdownEdge

	// drainers are called by GracefulClose before Close to drain in-flight work.
	drainers []Drainer

	// handlerCount tracks registered command handlers for introspection.
	cmdHandlerCount int

	started bool
	stopped bool
}

// Drainer stops accepting new work and finishes in-flight work, bounded by ctx.
// Implemented by resources that need to drain before connections close (e.g.,
// event subscribers, projection runners). Drain is called by [GracefulClose]
// BEFORE [Close], so in-flight handlers complete before their underlying
// connections are dropped.
type Drainer interface {
	Drain(ctx context.Context) error
}

// shutdownEdge declares that Before must close before After during Close().
// Resource names are engine names from DeploymentConfig.Engines or the
// constant ProjectionHostResource.
type shutdownEdge struct {
	before, after string
}

// ProjectionHostResource is the resource name for the projection host in
// shutdown dependency declarations.
const ProjectionHostResource = "projection-host"

// orderedEngines returns engines sorted by shutdown dependencies. Engines not
// in any dependency edge keep their creation order. Cycles fall back to
// creation order for the affected engines.
func (s *System) orderedEngines() []metaengine.Engine {
	if len(s.shutdownDeps) == 0 || len(s.engineNames) == 0 {
		return s.engines
	}

	// Build a set of unique engine indices by name.
	nameToIdx := make(map[string]int, len(s.engineNames))
	for i, name := range s.engineNames {
		if _, exists := nameToIdx[name]; !exists {
			nameToIdx[name] = i
		}
	}

	// Build adjacency list from dependency edges.
	after := make([][]int, len(s.engines))
	inDegree := make([]int, len(s.engines))

	for _, edge := range s.shutdownDeps {
		beforeIdx, beforeOK := nameToIdx[edge.before]

		afterIdx, afterOK := nameToIdx[edge.after]
		if !beforeOK || !afterOK || beforeIdx == afterIdx {
			continue
		}

		after[beforeIdx] = append(after[beforeIdx], afterIdx)
		inDegree[afterIdx]++
	}

	// Kahn's algorithm for topological sort.
	queue := make([]int, 0, len(s.engines))
	for i := range s.engines {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	result := make([]metaengine.Engine, 0, len(s.engines))
	processed := 0

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]

		result = append(result, s.engines[idx])
		processed++

		for _, next := range after[idx] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// If there's a cycle, append remaining engines in creation order.
	if processed < len(s.engines) {
		appended := make(map[int]bool, len(result))

		for i := range s.engines {
			if inDegree[i] > 0 {
				result = append(result, s.engines[i])
				appended[i] = true
			}
		}
	}

	return result
}

// MetaEngine returns the projection-layer metaengine store, or nil if no
// projection instance is configured. Consumers use this for typed queries
// via [metaengine.ExecuteTyped] or [metaengine.NewReader].
func (s *System) MetaEngine() *metaengine.Store {
	return s.projStore
}

// CommandDispatcher returns the command dispatcher. Consumers use this to
// dispatch commands programmatically (in addition to HTTP handlers).
func (s *System) CommandDispatcher() *command.Dispatcher {
	return s.cmdDisp
}

// QueryDispatcher returns the query dispatcher.
func (s *System) QueryDispatcher() *query.Dispatcher {
	return s.qryDisp
}

// Publisher returns the event publisher used by decider repositories.
// This may be a MultiBus if the deployment configures multiple Publish targets.
func (s *System) Publisher() event.Publisher {
	return s.pubBus
}

// ProjectionHost returns the projection host, or nil if not configured.
func (s *System) ProjectionHost() *projectionhost.Host {
	return s.projHost
}

// Bus returns the event bus for pub/sub. Consumers use this to subscribe to
// events for custom side-effects (notifications, webhooks, etc.).
func (s *System) Bus() event.Bus {
	return s.bus
}

// UseCommandMiddleware adds domain middleware to the command dispatcher.
// This is the consumer's extension point for validation, authorization, etc.
func (s *System) UseCommandMiddleware(mw ...command.Middleware) {
	if s.cmdDisp == nil {
		return
	}

	for _, m := range mw {
		s.cmdDisp.Use(m)
	}
}

// RegisterDrainer registers a [Drainer] that will be called by [GracefulClose]
// before [Close]. Use this to ensure in-flight work (e.g., event subscribers,
// HTTP handlers) completes before infrastructure connections are dropped.
func (s *System) RegisterDrainer(d Drainer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.drainers = append(s.drainers, d)
}

// EventStore returns the event store backed by the source-of-truth instance.
func (s *System) EventStore() event.Store {
	return s.eventStore
}

// SnapshotStore returns the snapshot store if the engine implements
// SnapshotBackend, or nil otherwise. Consumers can use this to manage snapshots
// directly (e.g., manual SaveSnapshot after batch imports).
func (s *System) SnapshotStore() snapshot.SnapshotStore {
	return s.snapStore
}

// Close shuts down all owned infrastructure: projection host, engines, stores.
// All close errors are joined and returned (not just the first), matching
// [stack.Bundle.Close] behavior.
func (s *System) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}

	s.stopped = true

	var errs []error

	if s.projHost != nil {
		if err := s.projHost.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("system: stop projection host: %w", err))
		}
	}

	for _, eng := range s.orderedEngines() {
		if err := eng.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// GracefulClose drains in-flight work via any registered [Drainer] resources,
// then calls [Close] with a context-bounded timeout. It returns when Close
// finishes or the context is cancelled, whichever comes first. If the context
// expires during draining, Close is still attempted; if it expires during
// Close, resources may still be closing in the background.
//
// Use this instead of [Close] when you need a shutdown deadline (e.g., a
// Kubernetes SIGTERM grace period).
func (s *System) GracefulClose(ctx context.Context) error {
	// Phase 1: drain in-flight work.
	for _, d := range s.drainers {
		if err := d.Drain(ctx); err != nil {
			return fmt.Errorf("system: graceful drain: %w", err)
		}
	}

	// Phase 2: close with context race.
	done := make(chan error, 1)

	go func() { done <- s.Close() }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("system: graceful close: %w", ctx.Err())
	}
}

// ResetProjection resets a projection's checkpoint and read-model state so it
// replays from the beginning on the next Start. The projection host must be
// stopped before calling this method (projectionhost.Host.Reset returns an
// error if the host is running).
//
// The projection must implement [projectionhost.Resettable] for its read-model
// state to be cleared. The checkpoint is always reset regardless.
func (s *System) ResetProjection(ctx context.Context, name string) error {
	if s.projHost == nil {
		return ErrNoProjectionHost
	}

	return s.projHost.Reset(ctx, name)
}
