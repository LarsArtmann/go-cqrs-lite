package indexing

import (
	"context"
	"testing"
)

func TestWithIndexingHooks(t *testing.T) {
	t.Parallel()

	var beforeCreateCalled, afterCreateCalled bool
	var beforeDropCalled, afterDropCalled bool

	hooks := hooks{}
	WithBeforeCreateHook(func(ctx context.Context, hctx HookContext) error {
		beforeCreateCalled = true
		return nil
	})(&hooks)
	WithAfterCreateHook(func(ctx context.Context, hctx HookContext) error {
		afterCreateCalled = true
		return nil
	})(&hooks)
	WithBeforeDropHook(func(ctx context.Context, hctx HookContext) error {
		beforeDropCalled = true
		return nil
	})(&hooks)
	WithAfterDropHook(func(ctx context.Context, hctx HookContext) error {
		afterDropCalled = true
		return nil
	})(&hooks)

	idx := Index{Name: "test_idx", Table: "events", Columns: []string{"id"}}
	ctx := context.Background()

	if err := hooks.fireBeforeCreate(ctx, idx, nil); err != nil {
		t.Fatalf("fireBeforeCreate: %v", err)
	}

	if !beforeCreateCalled {
		t.Error("beforeCreate hook was not called")
	}

	hooks.fireAfterCreate(ctx, idx, nil)
	if !afterCreateCalled {
		t.Error("afterCreate hook was not called")
	}

	if err := hooks.fireBeforeDrop(ctx, idx, nil); err != nil {
		t.Fatalf("fireBeforeDrop: %v", err)
	}

	if !beforeDropCalled {
		t.Error("beforeDrop hook was not called")
	}

	hooks.fireAfterDrop(ctx, idx, nil)
	if !afterDropCalled {
		t.Error("afterDrop hook was not called")
	}
}

func TestBeforeCreateHookVeto(t *testing.T) {
	t.Parallel()

	hooks := hooks{}
	WithBeforeCreateHook(func(ctx context.Context, hctx HookContext) error {
		return ErrTestVeto
	})(&hooks)

	err := hooks.fireBeforeCreate(context.Background(), Index{Name: "test"}, nil)
	if err != ErrTestVeto {
		t.Fatalf("expected veto error, got: %v", err)
	}
}

func TestBeforeDropHookVeto(t *testing.T) {
	t.Parallel()

	hooks := hooks{}
	WithBeforeDropHook(func(ctx context.Context, hctx HookContext) error {
		return ErrTestVeto
	})(&hooks)

	err := hooks.fireBeforeDrop(context.Background(), Index{Name: "test"}, nil)
	if err != ErrTestVeto {
		t.Fatalf("expected veto error, got: %v", err)
	}
}

func TestHookEventTypes(t *testing.T) {
	t.Parallel()

	events := map[HookEvent]bool{
		HookBeforeCreate: false,
		HookAfterCreate:  false,
		HookBeforeDrop:   false,
		HookAfterDrop:    false,
	}

	var captured HookEvent
	trackEvent := func(ctx context.Context, hctx HookContext) error {
		captured = hctx.Event
		return nil
	}

	hooks := hooks{}
	WithBeforeCreateHook(trackEvent)(&hooks)
	WithAfterCreateHook(trackEvent)(&hooks)
	WithBeforeDropHook(trackEvent)(&hooks)
	WithAfterDropHook(trackEvent)(&hooks)

	ctx := context.Background()
	idx := Index{Name: "test"}

	hooks.fireBeforeCreate(ctx, idx, nil)
	events[captured] = true

	hooks.fireAfterCreate(ctx, idx, nil)
	events[captured] = true

	hooks.fireBeforeDrop(ctx, idx, nil)
	events[captured] = true

	hooks.fireAfterDrop(ctx, idx, nil)
	events[captured] = true

	for evt, seen := range events {
		if !seen {
			t.Errorf("hook event %q was not captured", evt)
		}
	}
}

func TestMultipleHooksCalledInOrder(t *testing.T) {
	t.Parallel()

	var order []string

	hooks := hooks{}
	WithBeforeCreateHook(func(ctx context.Context, hctx HookContext) error {
		order = append(order, "first")
		return nil
	})(&hooks)
	WithBeforeCreateHook(func(ctx context.Context, hctx HookContext) error {
		order = append(order, "second")
		return nil
	})(&hooks)

	hooks.fireBeforeCreate(context.Background(), Index{Name: "test"}, nil)

	if len(order) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(order))
	}

	if order[0] != "first" || order[1] != "second" {
		t.Errorf("hooks called in wrong order: %v", order)
	}
}

var ErrTestVeto = new(testVetoError)

type testVetoError struct{}

func (e *testVetoError) Error() string { return "test veto" }
