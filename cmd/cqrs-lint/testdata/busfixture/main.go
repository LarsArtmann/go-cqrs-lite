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
	"context"
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
func (c rabbitConn) Bus() rabbitConn                  { return c }
func (c rabbitConn) Publish(evt event.Event) error    { return nil }
func (c rabbitConn) SubscribeAll(event.Handler) error { return nil }

// newLocalBus is a project-CONSTRUCTED dispatch bus (the newBus() shape):
// same event.Bus interface, but not obtained from an engine accessor.
func newLocalBus() event.Bus { return shell.bus }

var shell = engineShell{}

func main() {
	engine := engineShell{}
	bus := engine.Bus()
	_ = bus.SubscribeAll(handler)

	rabbit := rabbitConn{}
	rabbitBus := rabbit.Bus()
	_ = rabbitBus.Publish(evt)

	_ = http.ListenAndServe(":8080", nil)
}

// feedTail and dispatchLocal are the two-same-named-variables shape: under
// name-keyed tracking both `bus` entries collapsed into one, and the
// accessor in feedTail masked the dispatch bus in dispatchLocal (CV
// 2026-10-05); under identity-keyed tracking they are distinct variables —
// feedTail's bus skips as the journal tail, dispatchLocal's bus draws
// retry/circuit-breaker advice.
func feedTail() {
	e := engineShell{}
	bus := e.Bus()
	_ = bus.SubscribeAll(handler)
}

func dispatchLocal() {
	bus := newLocalBus()
	_ = bus.Publish(context.Background(), evt)
}
