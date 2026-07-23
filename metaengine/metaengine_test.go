package metaengine

import (
	"context"
	"testing"
	"time"
)

// ════════════ DOMAIN EVENT TYPES ════════════

type UserID string

type UserCreated struct {
	ID      UserID
	Email   string
	Name    string
	Country string
	At      time.Time
}

type UserSuspended struct {
	ID UserID
	At time.Time
}

type UserDeleted struct {
	ID UserID
	At time.Time
}

type Friendship struct {
	From UserID
	To   UserID
	At   time.Time
}

// ════════════ QUERY INPUT + RESULT TYPES ════════════

type FindUser struct {
	ID UserID
}

type FindUserResult struct {
	ID       UserID
	Name     string
	Email    string
	Status   string
	Country  string
	JoinedAt time.Time
}

type CheckEmail struct {
	Email string
}

type ListByStatus struct {
	Status string
	Limit  int
	After  *Cursor
}

type ListByStatusResult struct {
	Users []FindUserResult
	Next  *Cursor
}

type CountByStatus struct{}

type FriendsOf struct {
	ID    UserID
	Depth int
}

type FriendsOfResult struct {
	IDs []UserID
}

// ════════════ QUERY DECLARATIONS (matching design doc) ════════════

func findUserQuery() QueryDecl[FindUser, FindUserResult] {
	return Query[FindUser, FindUserResult](
		"find_user",
		On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
			return e.ID, FindUserResult{
				ID: e.ID, Name: e.Name, Email: e.Email,
				Status: "active", Country: e.Country, JoinedAt: e.At,
			}
		}),
		On(UserSuspended{}, func(e UserSuspended, prev FindUserResult) FindUserResult {
			prev.Status = "suspended"
			return prev
		}),
		On(UserDeleted{}, Remove[FindUserResult]()),
	)
}

func checkEmailQuery() QueryDecl[CheckEmail, bool] {
	return Query[CheckEmail, bool](
		"check_email",
		On(UserCreated{}, func(e UserCreated) string {
			return e.Email
		}),
		On(UserDeleted{}, Remove[string]()),
	)
}

func listByStatusQuery() QueryDecl[ListByStatus, ListByStatusResult] {
	return Query[ListByStatus, ListByStatusResult](
		"list_by_status",
		On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
			return e.ID, FindUserResult{
				ID: e.ID, Name: e.Name, Email: e.Email,
				Status: "active", Country: e.Country, JoinedAt: e.At,
			}
		}),
		On(UserSuspended{}, func(e UserSuspended, prev FindUserResult) FindUserResult {
			prev.Status = "suspended"
			return prev
		}),
		On(UserDeleted{}, Remove[FindUserResult]()),
		FilterOn(func(r FindUserResult) string { return r.Status }),
		SortOn(func(r FindUserResult) time.Time { return r.JoinedAt }),
	)
}

func countByStatusQuery() QueryDecl[CountByStatus, map[string]int64] {
	return Query[CountByStatus, map[string]int64](
		"count_by_status",
		On(UserCreated{}, func(e UserCreated) Delta {
			return Delta{"active": +1}
		}),
		On(UserSuspended{}, func(e UserSuspended) Delta {
			return Delta{"active": -1, "suspended": +1}
		}),
		On(UserDeleted{}, func(e UserDeleted) Delta {
			return Delta{"suspended": -1, "deleted": +1}
		}),
	)
}

func friendsOfQuery() QueryDecl[FriendsOf, FriendsOfResult] {
	return Query[FriendsOf, FriendsOfResult](
		"friends_of",
		On(Friendship{}, func(e Friendship) Edge {
			return Edge{From: e.From, To: e.To}
		}),
	)
}

func allQueries() []any {
	return []any{
		findUserQuery(),
		checkEmailQuery(),
		listByStatusQuery(),
		countByStatusQuery(),
		friendsOfQuery(),
	}
}

// ════════════ TESTS ════════════

func TestPlan_AllFiveQueries(t *testing.T) {
	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, allQueries()...)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	plan := store.Plan()
	if len(plan.Queries) != 5 {
		t.Fatalf("expected 5 queries, got %d", len(plan.Queries))
	}

	t.Log(plan.Report())
}

func TestApplyAndExecute_AllFiveQueries(t *testing.T) {
	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, allQueries()...)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now()

	events := []struct {
		typeName string
		payload  any
	}{
		{"UserCreated", UserCreated{
			ID: "u1", Email: "alice@example.com", Name: "Alice", Country: "SE", At: now,
		}},
		{
			"UserCreated",
			UserCreated{
				ID:      "u2",
				Email:   "bob@example.com",
				Name:    "Bob",
				Country: "US",
				At:      now.Add(1 * time.Hour),
			},
		},
		{
			"UserCreated",
			UserCreated{
				ID:      "u3",
				Email:   "carol@example.com",
				Name:    "Carol",
				Country: "SE",
				At:      now.Add(2 * time.Hour),
			},
		},
		{"UserSuspended", UserSuspended{ID: "u2", At: now.Add(3 * time.Hour)}},
		{"Friendship", Friendship{From: "u1", To: "u2", At: now}},
		{"Friendship", Friendship{From: "u2", To: "u3", At: now}},
	}

	for _, e := range events {
		if err := store.Apply(e.typeName, e.payload); err != nil {
			t.Fatalf("Apply(%s): %v", e.typeName, err)
		}
	}

	// Map: FindUser — point lookup
	t.Run("FindUser", func(t *testing.T) {
		result, err := ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: "u1"})
		if err != nil {
			t.Fatalf("Execute FindUser: %v", err)
		}
		if result.Name != "Alice" {
			t.Errorf("expected Name=Alice, got %q", result.Name)
		}
		if result.Status != "active" {
			t.Errorf("expected Status=active, got %q", result.Status)
		}
	})

	// Set: CheckEmail — membership test
	t.Run("CheckEmail", func(t *testing.T) {
		taken, err := ExecuteTyped[CheckEmail, bool](
			ctx,
			store,
			CheckEmail{Email: "alice@example.com"},
		)
		if err != nil {
			t.Fatalf("Execute CheckEmail: %v", err)
		}
		if !taken {
			t.Error("expected alice@example.com to be taken")
		}

		taken2, _ := ExecuteTyped[CheckEmail, bool](
			ctx,
			store,
			CheckEmail{Email: "nobody@example.com"},
		)
		if taken2 {
			t.Error("expected nobody@example.com to NOT be taken")
		}
	})

	// SortedMap: ListByStatus — filtered scan with pagination
	t.Run("ListByStatus", func(t *testing.T) {
		page, err := ExecuteTyped[ListByStatus, ListByStatusResult](ctx, store,
			ListByStatus{Status: "active", Limit: 10})
		if err != nil {
			t.Fatalf("Execute ListByStatus: %v", err)
		}
		if len(page.Users) != 2 {
			t.Fatalf("expected 2 active users, got %d", len(page.Users))
		}
		// u1 (Alice) joined before u3 (Carol), so u1 should be first
		if page.Users[0].ID != "u1" {
			t.Errorf("expected first active user u1, got %s", page.Users[0].ID)
		}
	})

	// Counter: CountByStatus — aggregate read
	t.Run("CountByStatus", func(t *testing.T) {
		counts, err := ExecuteTyped[CountByStatus, map[string]int64](ctx, store, CountByStatus{})
		if err != nil {
			t.Fatalf("Execute CountByStatus: %v", err)
		}
		if counts["active"] != 2 {
			t.Errorf("expected 2 active, got %d", counts["active"])
		}
		if counts["suspended"] != 1 {
			t.Errorf("expected 1 suspended, got %d", counts["suspended"])
		}
	})

	// Graph: FriendsOf — traversal
	t.Run("FriendsOf", func(t *testing.T) {
		friends, err := ExecuteTyped[FriendsOf, FriendsOfResult](
			ctx,
			store,
			FriendsOf{ID: "u1", Depth: 2},
		)
		if err != nil {
			t.Fatalf("Execute FriendsOf: %v", err)
		}
		if len(friends.IDs) != 2 {
			t.Fatalf("expected u1 to have 2 friends at depth 2 (u2, u3), got %d", len(friends.IDs))
		}
	})
}

func TestMap_UpdateAndRemove(t *testing.T) {
	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, findUserQuery())
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now()

	// Create user
	store.Apply("UserCreated", UserCreated{
		ID: "u1", Email: "alice@example.com", Name: "Alice", Country: "SE", At: now,
	})

	// Verify initial state
	r1, _ := ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: "u1"})
	if r1.Status != "active" {
		t.Fatalf("expected active, got %s", r1.Status)
	}

	// Suspend user (FoldUpdate via On)
	store.Apply("UserSuspended", UserSuspended{ID: "u1", At: now})

	r2, _ := ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: "u1"})
	if r2.Status != "suspended" {
		t.Errorf("expected suspended after update, got %s", r2.Status)
	}

	// Delete user (Remove sentinel)
	store.Apply("UserDeleted", UserDeleted{ID: "u1", At: now})

	r3, _ := ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: "u1"})
	if r3.Name != "" {
		t.Errorf("expected empty result after delete, got Name=%q", r3.Name)
	}
}

func TestPagination_HasMore(t *testing.T) {
	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, listByStatusQuery())
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	now := time.Now()
	for i, name := range []string{"a", "b", "c"} {
		err := store.Apply("UserCreated", UserCreated{
			ID: UserID(name), Email: name + "@x.com", Name: name,
			Country: "SE", At: now.Add(time.Duration(i) * time.Hour),
		})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	ctx := context.Background()
	page, err := ExecuteTyped[ListByStatus, ListByStatusResult](ctx, store,
		ListByStatus{Status: "active", Limit: 2})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(page.Users) != 2 {
		t.Fatalf("expected 2 users on page 1, got %d", len(page.Users))
	}

	if page.Next == nil {
		t.Error("expected Next cursor to be set when HasMore")
	}
}

func TestEventTypeNames(t *testing.T) {
	store, err := Plan([]Engine{NewMemoryEngine()}, findUserQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	names := store.EventTypeNames()
	expected := []string{"UserCreated", "UserDeleted", "UserSuspended"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d event types, got %d: %v", len(expected), len(names), names)
	}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("event type[%d]: expected %q, got %q", i, want, names[i])
		}
	}
}
