package testhelpers

import (
	"testing"

	th "github.com/larsartmann/go-cqrs-lite/testhelpers"
)

// TestMetrics is a type alias re-exported from the shared testhelpers module.
type TestMetrics = th.TestMetrics

var (
	AppendEventsHandler    = th.AppendEventsHandler
	NoopCommandHandler     = th.NoopCommandHandler
	NoopEventHandler       = th.NoopEventHandler
	NoopQueryHandler       = th.NoopQueryHandler
	FailingCommandHandler  = th.FailingCommandHandler
	FailingEventHandler    = th.FailingEventHandler
	PanicCommandHandler    = th.PanicCommandHandler
	PanicEventHandler      = th.PanicEventHandler
	CallbackCommandHandler = th.CallbackCommandHandler
	CommandMiddleware      = th.CommandMiddleware
	EventMiddleware        = th.EventMiddleware
)

// AssertCallOrder asserts the call order matches expected.
func AssertCallOrder(t *testing.T, callOrder, expected []string) {
	t.Helper()

	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)

			break
		}
	}
}
