package graph

import (
	"context"
	"fmt"
	"slices"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
)

// Handler processes one event, merging nodes and edges through sink. The
// handler is driver-agnostic: it never references Neo4j, Cypher, or any
// concrete driver. The backend is fixed when the [GraphProjection] is
// constructed.
type Handler func(ctx context.Context, evt cqrsevent.Event, sink GraphSink) error

// GraphProjection is an [cqrsevent.Projection] that materialises events into a
// graph read model — nodes and edges — via a [GraphDriver].
//
// It is the graph counterpart to [storage.RelationalProjection] (multi-table
// relational) and [stack.Materialize] (single-document-per-key KV). Use it
// when the dominant read patterns are variable-depth traversal, adjacency,
// path-finding, or connected-component queries — patterns the relational tier
// serves poorly (recursive CTEs) and the document tier cannot express at all.
//
// All writes within one Handle call are atomic (driver transaction): a partial
// failure leaves the graph untouched and the event can be retried.
type GraphProjection struct {
	name    string
	driver  GraphDriver
	handler Handler
	types   []cqrsevent.Type
}

// NewGraphProjection creates a projection that materialises events into driver.
// handler is driver-agnostic; types filters which event types the projection
// receives.
func NewGraphProjection(
	name string,
	driver GraphDriver,
	handler Handler,
	types []cqrsevent.Type,
) (*GraphProjection, error) {
	if name == "" {
		return nil, errNoName
	}

	if driver == nil {
		return nil, errNilDriver
	}

	if handler == nil {
		return nil, errNilHandler
	}

	return &GraphProjection{
		name:    name,
		driver:  driver,
		handler: handler,
		types:   slices.Clone(types),
	}, nil
}

// Name implements [cqrsevent.Projection].
func (p *GraphProjection) Name() string { return p.name }

// EventTypes implements [cqrsevent.Projection].
func (p *GraphProjection) EventTypes() []cqrsevent.Type { return slices.Clone(p.types) }

// Handle runs the handler inside a driver transaction, committing on success
// and rolling back on error.
func (p *GraphProjection) Handle(ctx context.Context, evt cqrsevent.Event) error {
	err := p.driver.RunInTx(func(sink GraphSink) error {
		return p.handler(ctx, evt, sink)
	})
	if err != nil {
		return fmt.Errorf("graph projection %q: %w", p.name, err)
	}

	return nil
}

// Close closes the underlying driver. After Close, Handle returns an error.
// Callers that share a driver across projections should not Close it here.
func (p *GraphProjection) Close() error {
	if p.driver == nil {
		return nil
	}

	if err := p.driver.Close(); err != nil {
		return fmt.Errorf("graph projection %q: close driver: %w", p.name, err)
	}

	return nil
}
