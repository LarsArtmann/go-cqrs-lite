package dgraphengine

// Option configures optional engine capabilities at construction time.
type Option func(*dgraphEngine)

// WithContentionObserver returns an Option registering a callback invoked
// once for every contention retry inside the engine's backoff loop (attempt
// starts at 1 for the first retry). retryOnContention retries SILENTLY
// otherwise — correct for tests, but production Alpha contention storms
// (every parallel writer conflicts on the shared dgraph.type predicate)
// deserve operator visibility. The engine deliberately does NOT depend on a
// metrics library (production-dep budget is 3, enforced by check-arch);
// wire OTel — or any counter — from the outside:
//
//	eng, err := dgraphengine.New(addr,
//		dgraphengine.WithContentionObserver(func(attempt int) {
//			contentionRetries.Add(ctx, 1) // e.g. cqrs.dgraph.contention_retry
//		}))
func WithContentionObserver(fn func(attempt int)) Option {
	return func(e *dgraphEngine) {
		e.contentionObserver = fn
	}
}

// countContentionRetry reports one contention retry to the observer (if any).
func (e *dgraphEngine) countContentionRetry(attempt int) {
	if e.contentionObserver == nil {
		return
	}

	e.contentionObserver(attempt)
}
