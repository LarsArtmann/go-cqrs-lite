package decider

import "github.com/larsartmann/go-cqrs-lite/core/event"

// RepositoryOption configures a Repository.
type RepositoryOption[State any] func(*Repository[State])

// WithOutbox enables outbox support for reliable event publishing.
// When configured, Execute appends events to the outbox instead of
// publishing directly to the bus.
func WithOutbox[State any](outbox event.Outbox) RepositoryOption[State] {
	return func(r *Repository[State]) {
		r.outbox = outbox
	}
}
