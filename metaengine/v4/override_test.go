package metaengine_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

func TestOverride_ReplacesInferredFold(t *testing.T) {
	t.Parallel()

	type UserID string
	type UserView struct {
		ID   UserID
		Name string
		Bio  string
	}
	type GetUser struct{ ID UserID }
	type UserCreated struct {
		ID   UserID
		Name string
	}
	type UserUpdated struct {
		ID   UserID
		Name string
		Bio  string
	}
	type UserDeleted struct{ ID UserID }

	q := metaengine.Query[GetUser, UserView]("users_override",
		metaengine.Infer(UserCreated{}, UserUpdated{}, UserDeleted{}),
		metaengine.Override(metaengine.OnRecord(UserUpdated{},
			func(_ record.Record, e UserUpdated) (UserID, UserView) {
				return e.ID, UserView(e)
			})),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q, metaengine.WithDryRun())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	_, err = metaengine.ExecuteTyped[GetUser, UserView](ctx, store, GetUser{ID: "x"})
	if err == nil {
		t.Error("expected error in dry run (no data)")
	}
}

func TestOverride_WithoutInferPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when using Override without Infer")
		}
	}()

	type View struct{ ID string }
	type Event struct{ ID string }

	_ = metaengine.Query[struct{ ID string }, View](
		"test",
		metaengine.OnRecord(Event{}, func(_ record.Record, e Event) (string, View) {
			return e.ID, View(e)
		}),
		metaengine.Override(
			metaengine.OnRecord(Event{}, func(_ record.Record, e Event) (string, View) {
				return e.ID, View(e)
			}),
		),
	)
}

func TestOverride_AddsFoldForUncoveredEvent(t *testing.T) {
	t.Parallel()

	type ID string
	type View struct{ ID ID }
	type Created struct{ ID ID }
	type Deleted struct{ ID ID }
	type Archived struct{ ID ID }

	q := metaengine.Query[struct{ ID ID }, View]("archived_override",
		metaengine.Infer(Created{}, Deleted{}),
		metaengine.Override(metaengine.OnRecord(Archived{},
			metaengine.Remove[View]())),
	)

	eng := metaengine.NewMemoryEngine()
	store, err := metaengine.Plan([]metaengine.Engine{eng}, q, metaengine.WithDryRun())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	store.Apply(context.Background(), "Created", Created{ID: "x"})
	store.Apply(context.Background(), "Archived", Archived{ID: "x"})

	_, err = metaengine.ExecuteTyped[struct{ ID ID }, View](
		context.Background(), store, struct{ ID ID }{ID: "x"},
	)
	if err == nil {
		t.Error("expected ErrNotFound after Archive (Remove fold)")
	}
}
