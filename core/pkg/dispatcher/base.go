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
func (b *BaseDispatcher[H, M]) Lifecycle() *LifecycleMixin {
	return &b.inner.Lifecycle
}

// Register registers a handler for a type, applying middleware immediately.
func (b *BaseDispatcher[H, M]) Register(t string, handler H, wrap func(M, H) H) error {
	return b.inner.Register(t, handler, wrap)
}

// GetHandler returns the handler for a type and whether it exists.
func (b *BaseDispatcher[H, M]) GetHandler(t string) (H, bool) {
	return b.inner.GetHandler(t)
}

// Dispatch returns the wrapped handler for a type.
func (b *BaseDispatcher[H, M]) Dispatch(t string) (H, error) {
	return b.inner.Dispatch(t)
}

// NewBaseDispatcher creates and initializes a new BaseDispatcher with a new inner Dispatcher.
func NewBaseDispatcher[H, M any]() BaseDispatcher[H, M] {
	return BaseDispatcher[H, M]{
		inner: NewDispatcher[H, M](),
	}
}
