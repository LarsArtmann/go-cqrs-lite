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

	// Engines and closers for lifecycle management.
	engines []metaengine.Engine
	closers []func() error

	// handlerCount tracks registered command handlers for introspection.
	cmdHandlerCount int

	started bool
	stopped bool
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
func (s *System) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}

	s.stopped = true

	var firstErr error

	if s.projHost != nil {
		if err := s.projHost.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("system: stop projection host: %w", err)
		}
	}

	for _, closer := range s.closers {
		if err := closer(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	for _, eng := range s.engines {
		_ = eng.Close()
	}

	return firstErr
}
