package metaengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Types mirroring metaengine/README.md Quick Example verbatim. The fold event
// type is matched by Go type name, so the names must stay UserCreated/UserDeleted.
// UserID is shared with fixtures_test.go (branded string).
type UserCreated struct {
	ID   UserID
	Name string
	At   time.Time
}

type UserDeleted struct {
	ID UserID
}

type FindUser struct {
	ID UserID
}

type FindUserResult struct {
	ID       UserID
	Name     string
	JoinedAt time.Time
}

// TestReadmeQuickExample mirrors metaengine/README.md Quick Example verbatim.
func TestReadmeQuickExample(t *testing.T) {
	ctx := context.Background()

	findUser := metaengine.Query[FindUser, FindUserResult]("find_user",
		metaengine.On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
			return e.ID, FindUserResult{ID: e.ID, Name: e.Name, JoinedAt: e.At}
		}),
		metaengine.On(UserDeleted{}, metaengine.Remove[FindUserResult]()),
	)

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, findUser)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.Apply(
		ctx,
		"UserCreated",
		UserCreated{ID: "u1", Name: "Alice", At: time.Now()},
	); err != nil {
		t.Fatal(err)
	}

	result, err := metaengine.ExecuteTyped[FindUser, FindUserResult](
		ctx, store, FindUser{ID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "Alice" {
		t.Fatalf("want Alice, got %q", result.Name)
	}

	if err := store.Apply(ctx, "UserDeleted", UserDeleted{ID: "u1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := metaengine.ExecuteTyped[FindUser, FindUserResult](
		ctx, store, FindUser{ID: "u1"}); err == nil {
		t.Fatal("expected key-not-found after Remove fold")
	}
}
