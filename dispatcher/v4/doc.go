// Package dispatcher provides a generic handler registry with middleware chains
// and lifecycle management.
//
// It is the foundation for the command and query dispatchers, parameterized
// on handler type [H] and middleware type [M]. Applications typically use
// the higher-level command.NewDispatcher or query.NewDispatcher rather than
// this package directly.
//
// # Basic Usage
//
//	d := dispatcher.NewDispatcher[MyHandler, MyMiddleware]()
//	d.Register("user.create", handler, wrapper)
//	h, err := d.Dispatch("user.create")
//
// # Lifecycle
//
// Each Dispatcher embeds a Lifecycle for thread-safe close detection.
// After Close, all Dispatch calls return ErrDispatcherClosed.
//
// # Middleware Chain
//
// Middleware is applied at dispatch time, so Use() can be called in any order
// relative to Register(). The chain is rebuilt on each Dispatch call via
// slices.Backward, making the first-added middleware the outermost wrapper:
//
//	d.Use(loggingMiddleware, wrapFunc)
//	d.Use(recoveryMiddleware, wrapFunc)
//	// handler = loggingMiddleware(recoveryMiddleware(rawHandler))
//
// Adding middleware after Register() takes effect on the next Dispatch.
package dispatcher
