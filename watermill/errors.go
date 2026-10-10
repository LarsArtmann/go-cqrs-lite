package watermill

import (
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
