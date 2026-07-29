package metaengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ontyped_test.go verifies OnTyped binds a fold to an explicit event-type
// string (the CQRS wire type), decoupled from the payload struct's Go name.
// This is the bridge for events whose event.Type() (e.g. "user.created") does
// not match the struct name (e.g. "UserCreated").

type onTypedUserCreated struct {
	ID   string
	Name string
}

type onTypedInput struct{ ID string }

func TestOnTyped_BindsToExplicitEventTypeString(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	q := metaengine.Query[onTypedInput, string](
		"ontyped_lookup",
		// Bind to the wire string "user.created", NOT the struct name.
		metaengine.OnTyped("user.created", onTypedUserCreated{}, func(e onTypedUserCreated) (string, string) {
			return e.ID, e.Name
		}),
	)

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, q)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// Apply uses the SAME wire string — this is the contract OnTyped enables.
	if err := store.Apply(ctx, "user.created", onTypedUserCreated{ID: "u1", Name: "Alice"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[onTypedInput, string](ctx, store, onTypedInput{ID: "u1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got != "Alice" {
		t.Fatalf("OnTyped lookup: got %q, want %q", got, "Alice")
	}

	// Sanity: the struct name ("onTypedUserCreated") must NOT match — proving the
	// fold is keyed to the wire string, not the Go type name.
	if err := store.Apply(ctx, "onTypedUserCreated", onTypedUserCreated{ID: "u2", Name: "Bob"}); err != nil {
		t.Fatalf("Apply unrelated type should be a no-op: %v", err)
	}

	got2, _ := metaengine.ExecuteTyped[onTypedInput, string](ctx, store, onTypedInput{ID: "u2"})
	if got2 != "" {
		t.Fatalf("OnTyped must not match the struct-name string; got %q for u2", got2)
	}
}
