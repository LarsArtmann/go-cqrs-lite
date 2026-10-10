package watermill

import (
	"context"
	"fmt"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrMissingMetadata is returned when a required metadata field is missing from a Watermill message.
var ErrMissingMetadata error = errorfamily.NewRejection(
	"watermill.missing_metadata",
	"missing required metadata",
)

// ErrReentrantPublish is returned when Publish is called on a context that
// is marked as inside a synchronous bus delivery (event.MarkInDelivery): a
// nested synchronous publish from the delivering goroutine deadlocks the
// GoChannel backend under BlockPublishUntilSubscriberAck (ADR-0154; live
// stack evidence in docs/evidence/2026-10-09_deriver-bus-deadlock.md of the
// go-cqrs-lite repo). Dispatch asynchronously instead — deriver.WithAsyncDispatch,
// or any goroutine started with event.WithoutDeliveryMark.
var ErrReentrantPublish error = errorfamily.NewOrchestration(
	"watermill.reentrant_publish",
	"publish from inside a synchronous bus delivery would deadlock",
)

// rejectReentrant returns the [ErrReentrantPublish] guard error for a
// nested synchronous publish of kind ("event"/"command"), or nil when ctx
// is not inside a bus delivery — the shared guard both buses' Publish
// methods open with.
func rejectReentrant(ctx context.Context, kind string) error {
	if !cqrsevent.ContextInDelivery(ctx) {
		return nil
	}

	return fmt.Errorf(
		"%w: nested %s publish from a synchronous delivery handler",
		ErrReentrantPublish,
		kind,
	)
}
