package dispatcher

// BaseDispatcher provides common dispatcher functionality for composable dispatchers.
// H is the handler type, M is the middleware type.
type BaseDispatcher[H any, M any] struct {
	inner *Dispatcher[H, M]
}

// InitBaseDispatcher initializes the base dispatcher with the provided inner dispatcher.
func (b *BaseDispatcher[H, M]) InitBaseDispatcher(inner *Dispatcher[H, M]) {
	b.inner = inner
}

// Use adds middleware to the dispatcher.
func (b *BaseDispatcher[H, M]) Use(middleware ...M) {
	b.inner.Use(middleware...)
}

// Lifecycle returns the lifecycle manager for the dispatcher.
func (b *BaseDispatcher[H, M]) Lifecycle() *Lifecycle {
	return &b.inner.Lifecycle
}

// Register registers a handler for a type.
func (b *BaseDispatcher[H, M]) Register(t string, handler H) error {
	return b.inner.Register(t, handler)
}

// GetHandler returns the handler for a type and whether it exists.
//nolint:ireturn // generic interface return by design
func (b *BaseDispatcher[H, M]) GetHandler(t string) (H, bool) {
	return b.inner.GetHandler(t)
}

// Dispatch wraps the handler for a type with middleware and returns the wrapped handler.
//nolint:ireturn // generic interface return by design
func (b *BaseDispatcher[H, M]) Dispatch(t string, wrap func(M, H) H) (H, error) {
	return b.inner.Dispatch(t, wrap)
}

// NewBaseDispatcher creates and initializes a new BaseDispatcher with a new inner Dispatcher.
func NewBaseDispatcher[H, M any]() BaseDispatcher[H, M] {
	return BaseDispatcher[H, M]{
		inner: NewDispatcher[H, M](),
	}
}
