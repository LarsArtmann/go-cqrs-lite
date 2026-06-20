package indexing

import "context"

// HookEvent describes what triggered the hook.
type HookEvent string

const (
	// HookBeforeCreate fires before an index is created.
	HookBeforeCreate HookEvent = "before_create"
	// HookAfterCreate fires after an index is successfully created.
	HookAfterCreate HookEvent = "after_create"
	// HookBeforeDrop fires before an index is dropped.
	HookBeforeDrop HookEvent = "before_drop"
	// HookAfterDrop fires after an index is successfully dropped.
	HookAfterDrop HookEvent = "after_drop"
)

// HookContext carries information about the hook invocation.
type HookContext struct {
	Event       HookEvent
	Index       Index
	AutoIndexer *AutoIndexer
}

// Hook is a callback invoked before or after index operations.
// Return an error to abort the operation (only effective for Before hooks).
type Hook func(ctx context.Context, hctx HookContext) error

// hooks holds pre/post callbacks for index lifecycle events.
type hooks struct {
	beforeCreate []Hook
	afterCreate  []Hook
	beforeDrop   []Hook
	afterDrop    []Hook
}

// IndexingHooksOption configures hooks on an AutoIndexer.
type IndexingHooksOption func(*hooks)

// WithBeforeCreateHook registers a callback invoked before each index creation.
// Return an error to prevent the creation.
func WithBeforeCreateHook(h Hook) IndexingHooksOption {
	return func(hooks *hooks) { hooks.beforeCreate = append(hooks.beforeCreate, h) }
}

// WithAfterCreateHook registers a callback invoked after each successful index creation.
func WithAfterCreateHook(h Hook) IndexingHooksOption {
	return func(hooks *hooks) { hooks.afterCreate = append(hooks.afterCreate, h) }
}

// WithBeforeDropHook registers a callback invoked before each index drop.
func WithBeforeDropHook(h Hook) IndexingHooksOption {
	return func(hooks *hooks) { hooks.beforeDrop = append(hooks.beforeDrop, h) }
}

// WithAfterDropHook registers a callback invoked after each successful index drop.
func WithAfterDropHook(h Hook) IndexingHooksOption {
	return func(hooks *hooks) { hooks.afterDrop = append(hooks.afterDrop, h) }
}

// WithIndexingHooks configures an AutoIndexer with lifecycle hooks.
// Hooks are called in registration order. Before hooks can veto
// an operation by returning an error.
//
// Usage:
//
//	auto := indexing.NewAutoIndexer(db,
//	    indexing.WithIndexingHooks(
//	        indexing.WithBeforeCreateHook(func(ctx context.Context, hctx indexing.HookContext) error {
//	            log.Printf("creating index %s on %s", hctx.Index.Name, hctx.Index.Table)
//	            return nil
//	        }),
//	        indexing.WithAfterCreateHook(func(ctx context.Context, hctx indexing.HookContext) error {
//	            metrics.IndexCreated(hctx.Index.Name, hctx.Index.Table)
//	            return nil
//	        }),
//	    ),
//	)
func WithIndexingHooks(opts ...IndexingHooksOption) AutoIndexerOption {
	return func(a *AutoIndexer) {
		for _, opt := range opts {
			opt(&a.hooksConfig)
		}
	}
}

func (h *hooks) fireBeforeCreate(ctx context.Context, idx Index, a *AutoIndexer) error {
	hctx := HookContext{Event: HookBeforeCreate, Index: idx, AutoIndexer: a}
	for _, hook := range h.beforeCreate {
		if err := hook(ctx, hctx); err != nil {
			return err
		}
	}

	return nil
}

func (h *hooks) fireAfterCreate(ctx context.Context, idx Index, a *AutoIndexer) {
	hctx := HookContext{Event: HookAfterCreate, Index: idx, AutoIndexer: a}
	for _, hook := range h.afterCreate {
		_ = hook(ctx, hctx)
	}
}

func (h *hooks) fireBeforeDrop(ctx context.Context, idx Index, a *AutoIndexer) error {
	hctx := HookContext{Event: HookBeforeDrop, Index: idx, AutoIndexer: a}
	for _, hook := range h.beforeDrop {
		if err := hook(ctx, hctx); err != nil {
			return err
		}
	}

	return nil
}

func (h *hooks) fireAfterDrop(ctx context.Context, idx Index, a *AutoIndexer) {
	hctx := HookContext{Event: HookAfterDrop, Index: idx, AutoIndexer: a}
	for _, hook := range h.afterDrop {
		_ = hook(ctx, hctx)
	}
}
