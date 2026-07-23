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

// FindUser: point lookup by ID → Map ADT.
type FindUser struct {
	ID UserID
}

type FindUserResult struct {
	ID       UserID
	Name     string
	Email    string
	Status   string
	Country  string
	JoinedAt time.Time `metaengine:"sort"`
}

// CheckEmail: membership test → Set ADT.
type CheckEmail struct {
	Email string
}

type CheckEmailResult struct {
	Taken bool
}

// ListByStatus: filtered scan → SortedMap ADT with Page[T] result.
type ListByStatus struct {
	Status string // domain filter — every field in the input IS a filter
}

// CountByStatus: aggregate read → Counter ADT.
type CountByStatus struct{}

type CountByStatusResult struct {
	Active    int64
	Suspended int64
	Deleted   int64
}

// FriendsOf: graph traversal → Graph ADT.
type FriendsOf struct {
	ID    UserID
	Depth int
}

type FriendsOfResult struct {
	IDs []UserID
}

// ════════════ TEST: PLAN + ALL 5 QUERY TYPES ════════════

func TestPlan_AllFiveQueries(t *testing.T) {
	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, buildAllQueries()...)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	// Verify all 5 assignments exist.
	plan := store.Plan()
	if len(plan.Assignments) != 5 {
		t.Fatalf("expected 5 assignments, got %d", len(plan.Assignments))
	}

	// Print the plan for visual verification.
	t.Log(plan.Report())
}

func TestApplyAndExecute_AllFiveQueries(t *testing.T) {
	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, buildAllQueries()...)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now()

	// Apply events
	events := []struct {
		typeName string
		payload  any
	}{
		{
			"UserCreated",
			UserCreated{
				ID:      "u1",
				Email:   "alice@example.com",
				Name:    "Alice",
				Country: "SE",
				At:      now,
			},
		},
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

	// ══ Test FindUser (point lookup) ══
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

	// ══ Test CheckEmail (membership) ══
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

	// ══ Test ListByStatus (filtered scan with Page) ══
	t.Run("ListByStatus", func(t *testing.T) {
		page, err := ExecuteTyped[ListByStatus, Page[FindUserResult]](ctx, store,
			ListByStatus{Status: "active"}, WithLimit(10))
		if err != nil {
			t.Fatalf("Execute ListByStatus: %v", err)
		}
		if len(page.Items) != 2 {
			t.Fatalf("expected 2 active users, got %d", len(page.Items))
		}
		// Should be sorted by JoinedAt (u1 before u3)
		if page.Items[0].ID != "u1" {
			t.Errorf("expected first active user u1, got %s", page.Items[0].ID)
		}
	})

	// ══ Test CountByStatus (counter) ══
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

	// ══ Test FriendsOf (graph traversal) ══
	t.Run("FriendsOf", func(t *testing.T) {
		friends, err := ExecuteTyped[FriendsOf, []any](ctx, store, FriendsOf{ID: "u1", Depth: 2})
		if err != nil {
			t.Fatalf("Execute FriendsOf: %v", err)
		}
		if len(friends) != 2 {
			t.Fatalf("expected u1 to have 2 friends at depth 2 (u2, u3), got %d", len(friends))
		}
	})

	t.Log(store.Plan().Report())
}

// buildAllQueries creates all 5 query declarations from the design doc.
func buildAllQueries() []any {
	findUser := Query[FindUser, FindUserResult]("find_user", []Fold{
		OnInsert(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
			return e.ID, FindUserResult{
				ID: e.ID, Name: e.Name, Email: e.Email,
				Status: "active", Country: e.Country, JoinedAt: e.At,
			}
		}),
		OnUpdate(UserSuspended{}, func(e UserSuspended) UserID { return e.ID },
			func(e UserSuspended, prev FindUserResult) FindUserResult {
				prev.Status = "suspended"

				return prev
			}),
		OnRemove[UserDeleted, UserID, FindUserResult](UserDeleted{}, func(e UserDeleted) UserID {
			return e.ID
		}),
	}, Volume(1_000_000))

	checkEmail := Query[CheckEmail, bool]("check_email", []Fold{
		OnSet(UserCreated{}, func(e UserCreated) string {
			return e.Email
		}),
	})

	listByStatus := Query[ListByStatus, Page[FindUserResult]]("list_by_status", []Fold{
		OnInsert(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
			return e.ID, FindUserResult{
				ID: e.ID, Name: e.Name, Email: e.Email,
				Status: "active", Country: e.Country, JoinedAt: e.At,
			}
		}),
		OnUpdate(UserSuspended{}, func(e UserSuspended) UserID { return e.ID },
			func(e UserSuspended, prev FindUserResult) FindUserResult {
				prev.Status = "suspended"

				return prev
			}),
		OnRemove[UserDeleted, UserID, FindUserResult](UserDeleted{}, func(e UserDeleted) UserID {
			return e.ID
		}),
	})

	countByStatus := Query[CountByStatus, map[string]int64]("count_by_status", []Fold{
		OnCount(UserCreated{}, func(e UserCreated) Delta {
			return Delta{"active": +1}
		}),
		OnCount(UserSuspended{}, func(e UserSuspended) Delta {
			return Delta{"active": -1, "suspended": +1}
		}),
		OnCount(UserDeleted{}, func(e UserDeleted) Delta {
			return Delta{"suspended": -1, "deleted": +1}
		}),
	})

	friendsOf := Query[FriendsOf, []any]("friends_of", []Fold{
		OnEdge(Friendship{}, func(e Friendship) Edge {
			return Edge{From: e.From, To: e.To}
		}),
	})

	return []any{findUser, checkEmail, listByStatus, countByStatus, friendsOf}
}
