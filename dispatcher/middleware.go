package dispatcher

// Middleware wraps a handler of type H with cross-cutting concerns:
// Middleware[H](next)(handler) must return a handler that runs next plus the
// wrapped concern.
//
// It is THE shared middleware shape across the workspace (E15 signature
// unification): command.Middleware, command.PublishMiddleware,
// event.Middleware, event.PublishMiddleware, query.Middleware, and
// middleware.Middleware are all aliases of this generic type, so one
// function value composes with every dispatcher and bus without adapters.
type Middleware[H any] func(H) H
