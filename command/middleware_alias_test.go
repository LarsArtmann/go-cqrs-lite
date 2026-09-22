package command_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
)

// TestMiddlewareAliasesAreIdentical pins the E15 signature unification:
// command.Middleware and command.PublishMiddleware are aliases of the
// shared [dispatcher.Middleware] shape, so a dispatcher-typed value is
// directly usable as a command middleware — no adapter, no wrapping.
func TestMiddlewareAliasesAreIdentical(t *testing.T) {
	t.Parallel()

	var handlerMW command.Middleware = dispatcher.Middleware[command.Handler](nil)
	var publishMW command.PublishMiddleware = dispatcher.Middleware[command.Publisher](nil)

	if handlerMW != nil || publishMW != nil {
		t.Fatal("nil dispatcher middleware values must assign cleanly to the command aliases")
	}
}
