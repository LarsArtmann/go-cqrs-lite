package graph

import (
	"context"
	"fmt"
	"slices"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsprojection "github.com/larsartmann/go-cqrs-lite/projection/v3"
)

// Handler processes one event, merging nodes and edges through sink. The
// handler is driver-agnostic: it never references Neo4j, Cypher, or any
// concrete driver. The backend is fixed when the [GraphProjection] is
// constructed.
type Handler func(ctx context.Context, evt cqrsevent.Event, sink GraphSink) error

// GraphProjection is an [cqrsprojection.Projection] that materialises events into a
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
	schema  *Schema
}

// ProjectionOption configures a [GraphProjection].
type ProjectionOption func(*GraphProjection)

// WithSchema attaches a [Schema] to the projection. Every write issued by the
// handler through the [GraphSink] is validated against the schema before it
// reaches the driver. This catches structural typos (unknown labels, unknown
// properties, edge endpoint mismatches) regardless of which [GraphDriver] is
// used — the validation happens at the projection boundary, not inside the
// driver.
func WithSchema(schema *Schema) ProjectionOption {
	return func(p *GraphProjection) {
		p.schema = schema
	}
}

// NewGraphProjection creates a projection that materialises events into driver.
// handler is driver-agnostic; types filters which event types the projection
// receives. Options allow attaching a [Schema] for write validation.
func NewGraphProjection(
	name string,
	driver GraphDriver,
	handler Handler,
	types []cqrsevent.Type,
	opts ...ProjectionOption,
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

	p := &GraphProjection{ //nolint:exhaustruct // schema applied via options below
		name:    name,
		driver:  driver,
		handler: handler,
		types:   slices.Clone(types),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

// Name implements [cqrsprojection.Projection].
func (p *GraphProjection) Name() string { return p.name }

// EventTypes implements [cqrsprojection.Projection].
func (p *GraphProjection) EventTypes() []cqrsevent.Type { return slices.Clone(p.types) }

// Handle runs the handler inside a driver transaction, committing on success
// and rolling back on error. If a schema is attached, the sink passed to the
// handler validates every write against the schema before it reaches the driver.
func (p *GraphProjection) Handle(ctx context.Context, evt cqrsevent.Event) error {
	err := p.driver.RunInTx(func(sink GraphSink) error {
		validated := wrapWithSchema(sink, p.schema)

		return p.handler(ctx, evt, validated)
	})
	if err != nil {
		return cqrsevent.Wrap(err, cqrsevent.Classify(err),
			"graph.projection.handle",
			fmt.Sprintf("projection %q", p.name))
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
		return cqrsevent.WrapInfrastructure(err, "graph.projection.close",
			fmt.Sprintf("projection %q: close driver", p.name))
	}

	return nil
}

var _ cqrsprojection.Projection = (*GraphProjection)(nil)
