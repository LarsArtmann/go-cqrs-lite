package multisig_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/signing/v2/internal/testutil"
)

func collectingPublisher() (event.PublisherFunc, *[]event.Event) {
	return testutil.CollectingPublisher()
}

func trackingHandler() (func(context.Context, event.Event) error, func() bool) {
	h, w := testutil.TrackingHandler()

	return h, w
}

var noopHandler = testutil.NoopHandler

func makeTestEvent(t *testing.T) event.Event {
	return testutil.MakeTestEvent(t)
}

func tamperEvent(tb testing.TB, evt event.Event) event.Event {
	return testutil.TamperEvent(tb, evt)
}
