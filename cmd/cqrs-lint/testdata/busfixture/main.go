package main

// B029/B030 typed-path fixture: Bus() accessor call sites whose receivers
// and RESULT types have REAL static types (packages.Load with NeedTypes),
// which the source-map harness cannot provide (empty types.Info).
//
// engineShell.Bus() returns event.Bus — the engine journal-tail shape
// (system.System.Bus() and consumer passthroughs like CV's
// LibraryEngine.Bus() both return exactly that interface): the
// journal-tail skip must apply.
//
// rabbitConn.Bus() returns rabbitConn — a downstream transport exposing a
// same-named accessor: the skip must NOT apply (retry/circuit-breaker
// advice is category-correct there).

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

var handler event.Handler

var evt event.Event

type engineShell struct{ bus event.Bus }

// Bus mirrors the engine accessor: result type is the cqrs event bus.
func (e engineShell) Bus() event.Bus { return e.bus }

type rabbitConn struct{}

// Bus mirrors a downstream transport accessor: same method name, own type.
func (c rabbitConn) Bus() rabbitConn    { return c }
func (c rabbitConn) Publish(evt event.Event) error { return nil }
func (c rabbitConn) SubscribeAll(event.Handler) error { return nil }

func main() {
	engine := engineShell{}
	bus := engine.Bus()
	_ = bus.SubscribeAll(handler)

	rabbit := rabbitConn{}
	rabbitBus := rabbit.Bus()
	_ = rabbitBus.Publish(evt)

	_ = http.ListenAndServe(":8080", nil)
}
