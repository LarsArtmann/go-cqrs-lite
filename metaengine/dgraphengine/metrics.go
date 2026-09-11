package dgraphengine

import (
	"context"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// contentionCounter is the testable seam for the retry counter: the real
// OTel counter (a sealed interface — it carries an unexported method, so
// fakes cannot implement it directly) satisfies this narrow view, and tests
// substitute their own recorder.
type contentionCounter interface {
	Add(ctx context.Context, incr int64, opts ...cqrsotel.AddOption)
}

// newContentionRetryCounter builds the contention-retry counter from the
// global meter provider: a no-op when no provider is configured (zero cost),
// a real instrument after otel.Setup. retryOnContention retries SILENTLY
// otherwise — correct for tests, but production Alpha contention storms
// (every parallel writer conflicts on the shared dgraph.type predicate)
// deserve operator visibility.
func newContentionRetryCounter() contentionCounter {
	meter := cqrsotel.NewMeter("metaengine.dgraphengine")

	counter, err := meter.Int64Counter(
		"cqrs.dgraph.contention_retry",
		cqrsotel.CounterMetricWithDescription(
			"Dgraph operations retried after transient contention (aborted txns / pending-txn Alter rejections)",
		),
	)
	if err != nil {
		// A misconfigured provider must never break the engine: fall back to
		// silent (nil counter is guarded at the single call site).
		return nil //nolint:nilnil // explicit disabled sentinel, guarded
	}

	return counter
}

// countContentionRetry records one contention retry on the counter.
func (e *dgraphEngine) countContentionRetry(ctx context.Context) {
	if e.contentionRetry == nil {
		return
	}

	e.contentionRetry.Add(ctx, 1)
}
