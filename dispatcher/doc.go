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
// Middleware is applied at Register time (not dispatch time) in reverse order,
// so the last middleware added wraps the outermost layer:
//
//	d.Use(loggingMiddleware, wrapFunc)
//	d.Use(recoveryMiddleware, wrapFunc)
//	// handler = recoveryMiddleware(loggingMiddleware(rawHandler))
package dispatcher
