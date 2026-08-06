package metaengine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Test domain types for auto-projection tests.
type autoUserCreated struct {
	ID    string
	Name  string
	Email string
}

type autoUserUpdated struct {
	ID    string
	Name  string
	Email string
}

type autoUserDeleted struct {
	ID string
}

type autoUserView struct {
	ID    string
	Name  string
	Email string
}

type getAutoUser struct {
	ID string
}

func TestAutoInsert(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getAutoUser, autoUserView](
		"auto_insert",
		metaengine.AutoInsert[autoUserCreated, autoUserView]("ID"),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "autoUserCreated", autoUserCreated{
		ID: "u1", Name: "Alice", Email: "alice@example.com",
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := metaengine.ExecuteTyped[getAutoUser, autoUserView](ctx, store, getAutoUser{ID: "u1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "Alice" {
		t.Errorf("Name = %q, want %q", got.Name, "Alice")
	}

	if got.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@example.com")
	}
}

func TestAutoDelete(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getAutoUser, autoUserView](
		"auto_delete",
		metaengine.AutoInsert[autoUserCreated, autoUserView]("ID"),
		metaengine.AutoDelete[autoUserDeleted]("ID"),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "autoUserCreated", autoUserCreated{
		ID: "u1", Name: "Alice",
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "autoUserDeleted", autoUserDeleted{ID: "u1"}); err != nil {
		t.Fatal(err)
	}

	_, err = metaengine.ExecuteTyped[getAutoUser, autoUserView](ctx, store, getAutoUser{ID: "u1"})
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAutoUpdate(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[getAutoUser, autoUserView](
		"auto_update",
		metaengine.AutoInsert[autoUserCreated, autoUserView]("ID"),
		metaengine.AutoUpdate[autoUserUpdated, autoUserView]("ID"),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "autoUserCreated", autoUserCreated{
		ID: "u1", Name: "Alice", Email: "alice@old.com",
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.Apply(ctx, "autoUserUpdated", autoUserUpdated{
		ID: "u1", Name: "Alice2", Email: "alice@new.com",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := metaengine.ExecuteTyped[getAutoUser, autoUserView](ctx, store, getAutoUser{ID: "u1"})
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if got.Name != "Alice2" {
		t.Errorf("Name = %q, want %q", got.Name, "Alice2")
	}

	if got.Email != "alice@new.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@new.com")
	}
}

func TestAutoCRUD_FullLifecycle(t *testing.T) {
	t.Parallel()

	folds := metaengine.AutoCRUD[autoUserCreated, autoUserUpdated, autoUserDeleted, autoUserView]("ID")

	foldArgs := make([]any, len(folds))
	for i, f := range folds {
		foldArgs[i] = f
	}

	q := metaengine.Query[getAutoUser, autoUserView]("auto_crud", foldArgs...)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create
	if err := store.Apply(ctx, "autoUserCreated", autoUserCreated{
		ID: "u1", Name: "Alice", Email: "alice@example.com",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := metaengine.ExecuteTyped[getAutoUser, autoUserView](ctx, store, getAutoUser{ID: "u1"})
	if err != nil {
		t.Fatalf("after create: %v", err)
	}

	if got.Name != "Alice" {
		t.Fatalf("after create: Name = %q, want Alice", got.Name)
	}

	// Update (Name changes, Email not in event = preserved)
	if err := store.Apply(ctx, "autoUserUpdated", autoUserUpdated{
		ID: "u1", Name: "Bob",
	}); err != nil {
		t.Fatal(err)
	}

	got, err = metaengine.ExecuteTyped[getAutoUser, autoUserView](ctx, store, getAutoUser{ID: "u1"})
	if err != nil {
		t.Fatalf("after update: %v", err)
	}

	if got.Name != "Bob" {
		t.Errorf("after update: Name = %q, want Bob", got.Name)
	}

	if got.Email != "alice@example.com" {
		t.Errorf("after update: Email = %q, want alice@example.com (preserved)", got.Email)
	}

	// Delete
	if err := store.Apply(ctx, "autoUserDeleted", autoUserDeleted{ID: "u1"}); err != nil {
		t.Fatal(err)
	}

	_, err = metaengine.ExecuteTyped[getAutoUser, autoUserView](ctx, store, getAutoUser{ID: "u1"})
	if !errors.Is(err, metaengine.ErrNotFound) {
		t.Errorf("after delete: expected ErrNotFound, got %v", err)
	}
}

func TestAutoInsert_PartialMapping(t *testing.T) {
	t.Parallel()

	type eventWithExtra struct {
		ID    string
		Name  string
		Extra string // not in result
	}

	type resultWithMissing struct {
		ID   string
		Name string
	}

	type queryInput struct{ ID string }

	q := metaengine.Query[queryInput, resultWithMissing](
		"auto_partial",
		metaengine.AutoInsert[eventWithExtra, resultWithMissing]("ID"),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "eventWithExtra", eventWithExtra{
		ID: "e1", Name: "Test", Extra: "ignored",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := metaengine.ExecuteTyped[queryInput, resultWithMissing](
		ctx, store, queryInput{ID: "e1"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "Test" {
		t.Errorf("Name = %q, want Test", got.Name)
	}
}
