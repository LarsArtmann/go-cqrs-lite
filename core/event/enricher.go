package event

import "context"

// ContextEnricher extracts event options from a context.
// Use it to automatically inject correlation IDs, user IDs,
// and other request-scoped metadata into events.
type ContextEnricher func(ctx context.Context) []Option

// CompositeEnricher combines multiple enrichers into one.
func CompositeEnricher(enrichers ...ContextEnricher) ContextEnricher {
	return func(ctx context.Context) []Option {
		opts := make([]Option, 0, len(enrichers))

		for _, e := range enrichers {
			opts = append(opts, e(ctx)...)
		}

		return opts
	}
}

// EnrichEvent applies all options from the enricher to the event.
func EnrichEvent(ctx context.Context, evt *Core, enricher ContextEnricher) {
	for _, opt := range enricher(ctx) {
		opt(evt)
	}
}
