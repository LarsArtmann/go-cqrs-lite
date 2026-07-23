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

// FindUser: point lookup by ID → reads from users model (Map ADT).
type FindUser struct {
	ID UserID `metaengine:"key"`
}

type FindUserResult struct {
	ID       UserID
	Name     string
	Email    string
	Status   string
	Country  string
	JoinedAt time.Time `metaengine:"sort"`
}

// CheckEmail: membership test → reads from emails model (Set ADT).
type CheckEmail struct {
	Email string `metaengine:"key"`
}

// ListByStatus: filtered scan → reads from users model (Map ADT, paginated).
type ListByStatus struct {
	Status string // domain filter — matches FindUserResult.Status
}

// CountByStatus: aggregate read → reads from counts model (Counter ADT).
type CountByStatus struct{}

// FriendsOf: graph traversal → reads from friendships model (Graph ADT).
type FriendsOf struct {
	ID    UserID `metaengine:"key"`
	Depth int
}

// ════════════ READ MODEL DEFINITIONS ════════════

func usersModel() ReadModel {
	return MustModel("users", []Fold{
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
	}...)
}

func emailsModel() ReadModel {
	return MustModel("emails", OnSet(UserCreated{}, func(e UserCreated) string {
		return e.Email
	}))
}

func countsModel() ReadModel {
	return MustModel("counts",
		OnCount(UserCreated{}, func(e UserCreated) Delta {
			return Delta{"active": +1}
		}),
		OnCount(UserSuspended{}, func(e UserSuspended) Delta {
			return Delta{"active": -1, "suspended": +1}
		}),
		OnCount(UserDeleted{}, func(e UserDeleted) Delta {
			return Delta{"suspended": -1, "deleted": +1}
		}),
	)
}

func friendshipsModel() ReadModel {
	return MustModel("friendships", OnEdge(Friendship{}, func(e Friendship) Edge {
		return Edge{From: e.From, To: e.To}
	}))
}

// ════════════ TEST: PLAN + ALL 5 QUERY TYPES ════════════

func buildAllQueries() []any {
	users := usersModel()
	emails := emailsModel()
	counts := countsModel()
	friendships := friendshipsModel()

	findUser := Query[FindUser, FindUserResult]("find_user", users, Volume(1_000_000))
	checkEmail := Query[CheckEmail, bool]("check_email", emails)
	listByStatus := Query[ListByStatus, Page[FindUserResult]]("list_by_status", users)
	countByStatus := Query[CountByStatus, map[string]int64]("count_by_status", counts)
	friendsOf := Query[FriendsOf, []any]("friends_of", friendships)

	return []any{findUser, checkEmail, listByStatus, countByStatus, friendsOf}
}

func TestPlan_AllFiveQueries(t *testing.T) {
	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, buildAllQueries()...)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	plan := store.Plan()
	if len(plan.Models) != 4 {
		t.Fatalf("expected 4 models (users, emails, counts, friendships), got %d", len(plan.Models))
	}

	if len(plan.Queries) != 5 {
		t.Fatalf("expected 5 queries, got %d", len(plan.Queries))
	}

	// Verify model deduplication: find_user and list_by_status both read "users".
	usersCount := 0
	for _, q := range plan.Queries {
		if q.ModelName == "users" {
			usersCount++
		}
	}
	if usersCount != 2 {
		t.Errorf("expected 2 queries reading from users model, got %d", usersCount)
	}

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

	events := []struct {
		typeName string
		payload  any
	}{
		{"UserCreated", UserCreated{
			ID: "u1", Email: "alice@example.com", Name: "Alice", Country: "SE", At: now,
		}},
		{"UserCreated", UserCreated{
			ID: "u2", Email: "bob@example.com", Name: "Bob", Country: "US", At: now.Add(1 * time.Hour),
		}},
		{"UserCreated", UserCreated{
			ID: "u3", Email: "carol@example.com", Name: "Carol", Country: "SE", At: now.Add(2 * time.Hour),
		}},
		{"UserSuspended", UserSuspended{ID: "u2", At: now.Add(3 * time.Hour)}},
		{"Friendship", Friendship{From: "u1", To: "u2", At: now}},
		{"Friendship", Friendship{From: "u2", To: "u3", At: now}},
	}

	for _, e := range events {
		if err := store.Apply(e.typeName, e.payload); err != nil {
			t.Fatalf("Apply(%s): %v", e.typeName, err)
		}
	}

	// ══ Test FindUser (point lookup on users model) ══
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

	// ══ Test CheckEmail (membership on emails model) ══
	t.Run("CheckEmail", func(t *testing.T) {
		taken, err := ExecuteTyped[CheckEmail, bool](
			ctx, store, CheckEmail{Email: "alice@example.com"},
		)
		if err != nil {
			t.Fatalf("Execute CheckEmail: %v", err)
		}
		if !taken {
			t.Error("expected alice@example.com to be taken")
		}
		taken2, _ := ExecuteTyped[CheckEmail, bool](
			ctx, store, CheckEmail{Email: "nobody@example.com"},
		)
		if taken2 {
			t.Error("expected nobody@example.com to NOT be taken")
		}
	})

	// ══ Test ListByStatus (filtered scan on users model, paginated) ══
	t.Run("ListByStatus", func(t *testing.T) {
		page, err := ExecuteTyped[ListByStatus, Page[FindUserResult]](ctx, store,
			ListByStatus{Status: "active"}, WithLimit(10))
		if err != nil {
			t.Fatalf("Execute ListByStatus: %v", err)
		}
		if len(page.Items) != 2 {
			t.Fatalf("expected 2 active users, got %d", len(page.Items))
		}
		if page.Items[0].ID != "u1" {
			t.Errorf("expected first active user u1, got %s", page.Items[0].ID)
		}
	})

	// ══ Test CountByStatus (counter on counts model) ══
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

	// ══ Test FriendsOf (graph traversal on friendships model) ══
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

// ════════════ TEST: Write amplification — shared model ════════════

func TestSharedModel_WriteAmplification(t *testing.T) {
	// Both find_user and list_by_status read from the "users" model.
	// A UserCreated event should update "users" exactly once, not twice.
	users := usersModel()

	findUser := Query[FindUser, FindUserResult]("find_user", users)
	listByStatus := Query[ListByStatus, Page[FindUserResult]]("list_by_status", users)

	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, findUser, listByStatus)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	// Only 1 model despite 2 queries.
	plan := store.Plan()
	if len(plan.Models) != 1 {
		t.Fatalf("expected 1 model (dedup), got %d", len(plan.Models))
	}
	if len(plan.Queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(plan.Queries))
	}

	// Apply once.
	err = store.Apply("UserCreated", UserCreated{
		ID: "u1", Email: "a@b.com", Name: "A", Country: "SE", At: time.Now(),
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// Both queries should see the data.
	ctx := context.Background()
	r1, err := ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: "u1"})
	if err != nil {
		t.Fatalf("FindUser: %v", err)
	}
	if r1.Name != "A" {
		t.Errorf("expected Name=A, got %q", r1.Name)
	}

	page, err := ExecuteTyped[ListByStatus, Page[FindUserResult]](ctx, store,
		ListByStatus{Status: "active"}, WithLimit(10))
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(page.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(page.Items))
	}
}

// ════════════ TEST: Pagination — HasMore and Next cursor ════════════

func TestPagination_HasMore(t *testing.T) {
	users := usersModel()

	listByStatus := Query[ListByStatus, Page[FindUserResult]]("list_by_status", users)

	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, listByStatus)
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
	page, err := ExecuteTyped[ListByStatus, Page[FindUserResult]](ctx, store,
		ListByStatus{Status: "active"}, WithLimit(2))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(page.Items))
	}

	if !page.HasMore {
		t.Error("expected HasMore=true when there are more items")
	}

	if page.Next == nil {
		t.Error("expected Next cursor to be set when HasMore")
	}
}
