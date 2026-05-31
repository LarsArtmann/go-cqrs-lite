package signing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/signing/internal/testutil"
)

func collectingPublisher() (event.PublisherFunc, *[]event.Event) {
	return testutil.CollectingPublisher()
}

func trackingHandler() (func(context.Context, event.Event) error, func() bool) {
	return testutil.TrackingHandler()
}

var noopHandler = testutil.NoopHandler

func makeTestEvent(t *testing.T) event.Event {
	return testutil.MakeTestEvent(t)
}

func tamperEvent(tb testing.TB, evt event.Event) event.Event {
	return testutil.TamperEvent(tb, evt)
}
