package metaengine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ════════════ RACE CONDITION TEST ════════════

func TestConcurrentApplyAndExecute(t *testing.T) {
	users := usersModel()
	findUser := Query[FindUser, FindUserResult]("find_user", users)

	engines := []Engine{NewMemoryEngine()}
	store, err := Plan(engines, findUser)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent writers.
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = store.Apply("UserCreated", UserCreated{
				ID:      UserID(fmt.Sprintf("u%d", n)),
				Email:   fmt.Sprintf("user%d@test.com", n),
				Name:    fmt.Sprintf("User%d", n),
				Country: "SE",
				At:      time.Now(),
			})
		}(i)
	}

	// Concurrent readers.
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: "u1"})
		}()
	}

	wg.Wait()
}

// ════════════ NUMERIC SORT CORRECTNESS ════════════

type SortItem struct {
	ID    string
	Score int `metaengine:"sort"`
}

type SortInput struct {
	Category string
}

func TestNumericSortCorrectness(t *testing.T) {
	model := MustModel("items", OnInsert(SortItem{}, func(e SortItem) (string, SortItem) {
		return e.ID, e
	}))

	listQuery := Query[SortInput, Page[SortItem]]("list", model)

	store, err := Plan([]Engine{NewMemoryEngine()}, listQuery)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// Insert items with scores 1-10 in random order.
	scores := []int{5, 1, 10, 3, 8, 2, 7, 4, 9, 6}
	for i, s := range scores {
		err := store.Apply("SortItem", SortItem{
			ID:    fmt.Sprintf("item%d", i),
			Score: s,
		})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	ctx := context.Background()
	page, err := ExecuteTyped[SortInput, Page[SortItem]](ctx, store,
		SortInput{Category: ""}, WithLimit(100))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(page.Items) != 10 {
		t.Fatalf("expected 10 items, got %d", len(page.Items))
	}

	// Verify numeric ordering (not string ordering where "10" < "2").
	for i := 1; i < len(page.Items); i++ {
		prev := page.Items[i-1].Score
		curr := page.Items[i].Score
		if prev > curr {
			t.Errorf("sort violation at position %d: %d > %d (string sort would put 10 before 2)",
				i, prev, curr)
		}
	}
}

// ════════════ FILTER TYPE CORRECTNESS ════════════

type FilterItem struct {
	ID     string
	Count  int
	Active bool
}

type FilterInput struct {
	Count int
}

func TestFilterTypeCorrectness(t *testing.T) {
	model := MustModel(
		"filteritems",
		OnInsert(FilterItem{}, func(e FilterItem) (string, FilterItem) {
			return e.ID, e
		}),
	)

	query := Query[FilterInput, Page[FilterItem]]("filter_query", model)

	store, err := Plan([]Engine{NewMemoryEngine()}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	items := []FilterItem{
		{ID: "a", Count: 5, Active: true},
		{ID: "b", Count: 10, Active: true},
		{ID: "c", Count: 5, Active: false},
	}
	for _, item := range items {
		err := store.Apply("FilterItem", item)
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}

	ctx := context.Background()
	page, err := ExecuteTyped[FilterInput, Page[FilterItem]](ctx, store,
		FilterInput{Count: 5}, WithLimit(100))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Should match items "a" and "c" (Count=5), not "b" (Count=10).
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items with Count=5, got %d", len(page.Items))
	}

	for _, item := range page.Items {
		if item.Count != 5 {
			t.Errorf("expected Count=5, got %d for item %s", item.Count, item.ID)
		}
	}
}

// ════════════ APPLY ENCODED TEST ════════════

func TestApplyEncoded(t *testing.T) {
	users := usersModel()
	findUser := Query[FindUser, FindUserResult]("find_user", users)

	store, err := Plan([]Engine{NewMemoryEngine()}, findUser)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	defer store.Close()

	// Apply via JSON-encoded payload.
	payload := `{"ID":"u1","Email":"alice@test.com","Name":"Alice","Country":"SE","At":"2026-01-01T00:00:00Z"}`
	err = store.ApplyEncoded("UserCreated", []byte(payload))
	if err != nil {
		t.Fatalf("ApplyEncoded: %v", err)
	}

	ctx := context.Background()
	result, err := ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: "u1"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Name != "Alice" {
		t.Errorf("expected Name=Alice, got %q", result.Name)
	}
}

func TestEventTypeNames(t *testing.T) {
	users := usersModel()
	store, err := Plan([]Engine{NewMemoryEngine()},
		Query[FindUser, FindUserResult]("find_user", users))
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
