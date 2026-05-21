package event

import "context"

type contextEnricher func(ctx context.Context) []Option

func compositeEnricher(enrichers ...contextEnricher) contextEnricher {
	return func(ctx context.Context) []Option {
		opts := make([]Option, 0, len(enrichers))

		for _, e := range enrichers {
			opts = append(opts, e(ctx)...)
		}

		return opts
	}
}

func enrichEvent(ctx context.Context, evt *Core, enricher contextEnricher) {
	for _, opt := range enricher(ctx) {
		opt(evt)
	}
}
