package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
)

type ctxKeyRequestScope struct{}

// RequestScope carries the request-correlation fields that
// [RequestScopeEnricher] propagates into event metadata: who correlated the
// request (CorrelationID), which request it was (RequestID), and which client
// made it (IPAddress, UserAgent, ClientID). Zero-valued fields are skipped by
// the enricher, so a partially filled scope enriches only what is known.
//
// This is the correlation counterpart to [WithActorContext] (who initiated)
// and [WithCommandCausality] (which command caused): the scope answers
// "which request". Bridges from HTTP frameworks store their request fields
// here once, at the edge.
type RequestScope struct {
	CorrelationID id.CorrelationID
	RequestID     id.RequestID
	IPAddress     IPAddress
	UserAgent     UserAgent
	ClientID      id.ClientID
}

// WithRequestScope stores the [RequestScope] in the context so events created
// during this execution automatically record their request correlation.
// Use with decider's WithEnricher option and [RequestScopeEnricher].
func WithRequestScope(ctx context.Context, scope RequestScope) context.Context {
	return context.WithValue(ctx, ctxKeyRequestScope{}, scope)
}

// RequestScopeFromContext returns the [RequestScope] stored in the context.
// The second return is false when no scope was set or every field is zero.
func RequestScopeFromContext(ctx context.Context) (RequestScope, bool) {
	scope, ok := ctx.Value(ctxKeyRequestScope{}).(RequestScope)
	if !ok || scope.isZero() {
		return RequestScope{}, false
	}

	return scope, true
}

// RequestScopeEnricher is a [ContextEnricher] that propagates the
// [RequestScope] from the context into event metadata — only the non-zero
// fields, so absent values never overwrite present metadata. Compose with
// [ActorEnricher] and [CommandCausalityEnricher] via [CompositeEnricher] for
// the full audit trail.
//
//	repo, _ := decider.NewRepository[State](store, bus, d,
//	    decider.WithEnricher(event.CompositeEnricher(
//	        event.ActorEnricher,
//	        event.RequestScopeEnricher,
//	    )))
func RequestScopeEnricher(ctx context.Context) []Option {
	scope, ok := RequestScopeFromContext(ctx)
	if !ok {
		return nil
	}

	opts := make([]Option, 0, 5)

	if !scope.CorrelationID.IsZero() {
		opts = append(opts, WithCorrelationID(scope.CorrelationID))
	}

	if !scope.RequestID.IsZero() {
		opts = append(opts, WithRequestID(scope.RequestID))
	}

	if scope.IPAddress != "" {
		opts = append(opts, WithIPAddress(scope.IPAddress))
	}

	if scope.UserAgent != "" {
		opts = append(opts, WithUserAgent(scope.UserAgent))
	}

	if !scope.ClientID.IsZero() {
		opts = append(opts, WithClientID(scope.ClientID))
	}

	return opts
}

// isZero reports whether every correlation field is unset.
func (s RequestScope) isZero() bool {
	return s.CorrelationID.IsZero() &&
		s.RequestID.IsZero() &&
		s.IPAddress == "" &&
		s.UserAgent == "" &&
		s.ClientID.IsZero()
}
