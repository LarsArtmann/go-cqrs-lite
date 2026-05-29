package event

import "context"

// ContextEnricher extracts event options from a context.
// Use WithEnricher on repositories to automatically enrich events
// with context-derived metadata (correlation IDs, user IDs, etc.).
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

func enrichEvent(ctx context.Context, evt *ImmutableEvent, enricher ContextEnricher) {
	for _, opt := range enricher(ctx) {
		opt(evt)
	}
}
