package decider

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

func (r *Repository[State]) applyEnricher(ctx context.Context, events []event.Event) {
	if r.enricher == nil {
		return
	}

	opts := r.enricher(ctx)
	if len(opts) == 0 {
		return
	}

	for _, evt := range events {
		for _, opt := range opts {
			opt(evt)
		}
	}
}
