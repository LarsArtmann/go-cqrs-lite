package metaengine

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// costShapedEngine wraps a memory engine whose read costs for selected
// patterns are inflated, forcing deterministic routing splits between
// otherwise identical engines.
type costShapedEngine struct {
	*memoryEngine

	name      string
	expensive map[ReadPattern]float64
}

func (e *costShapedEngine) Profile() EngineProfile {
	p := e.memoryEngine.Profile()
	p.Name = e.name

	for pattern, cost := range e.expensive {
		switch pattern {
		case ReadPointLookup:
			p.ReadCosts.NsPerPointLookup = cost
		case ReadFilteredScan:
			p.ReadCosts.NsPerFilteredScan = cost
		case ReadAggregate:
			p.ReadCosts.NsPerAggregate = cost
		case ReadScan, ReadTraversal, ReadVectorSearch, ReadFullTextSearch, ReadSpatialRange:
			p.ReadCosts.NsPerScan = cost
		}
	}

	return p
}

func costShaped(name string, expensive map[ReadPattern]float64) Engine {
	return &costShapedEngine{
		memoryEngine: NewMemoryEngine().(*memoryEngine),
		name:         name,
		expensive:    expensive,
	}
}

type roleCountInput struct{}

func roleCountQuery() any {
	return Query[roleCountInput, map[string]int64](
		"role_counts",
		OnRecord(roleItemCreated{}, func(_ record.Record, e roleItemCreated) Delta {
			return Delta{"created": +1}
		}),
	)
}

// demoteTestStore builds a store whose item query routes to "items"
// (expensive aggregates) and counter query to "counts" (expensive point
// lookups). Returns the engines for direct inspection.
func demoteTestStore(t *testing.T) (store *Store, items, counts Engine) {
	t.Helper()

	items = costShaped("items", map[ReadPattern]float64{ReadAggregate: 1_000_000})
	counts = costShaped("counts", map[ReadPattern]float64{ReadPointLookup: 1_000_000})

	store, err := Plan([]Engine{items, counts}, roleItemQuery(), roleCountQuery())
	if err != nil {
		t.Fatal(err)
	}

	WithEventLog(store, NewEventLog())
	t.Cleanup(func() { _ = store.Close() })

	for _, qa := range store.Plan().Queries {
		switch qa.QueryName {
		case "role_items":
			if qa.EngineName != "items" {
				t.Fatalf("role_items routed to %q, want items", qa.EngineName)
			}
		case "role_counts":
			if qa.EngineName != "counts" {
				t.Fatalf("role_counts routed to %q, want counts", qa.EngineName)
			}
		}
	}

	return store, items, counts
}

const demoteWait = 2 * time.Second

// TestDemoteEngine_HappyPath covers the full transition: role flip, un-routing,
// mirror catch-up of never-served collections, re-routed catch-up on the
// survivor, live replication afterwards, the audit trigger, and promote-back.
func TestDemoteEngine_HappyPath(t *testing.T) {
	t.Parallel()

	store, items, _ := demoteTestStore(t)
	ctx := context.Background()

	for i := range 5 {
		if err := store.Apply(ctx, "roleItemCreated", roleItemCreated{
			ID:   "item-" + string(rune('a'+i)),
			Name: "n" + string(rune('a'+i)),
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := store.DemoteEngine(ctx, "items"); err != nil {
		t.Fatalf("demote: %v", err)
	}

	if role, _ := store.EngineRole("items"); role != RoleBackup {
		t.Fatalf("role after demote = %q, want Backup", role)
	}

	for _, qa := range store.Plan().Queries {
		if qa.EngineName == "items" {
			t.Fatalf("query %q still routed to demoted engine", qa.QueryName)
		}
	}

	// The re-routed item query must serve correct results from the survivor
	// (catch-up replay of its history).
	res, err := store.Execute(roleFindItem{ID: "item-c"})
	if err != nil {
		t.Fatal(err)
	}

	got, ok := res.(roleItem)
	if !ok || got.Name != "nc" {
		t.Fatalf("execute after demote = %#v, want name nc", res)
	}

	// The demoted engine must mirror BOTH collections: its own history plus
	// the never-served counter collection (targeted catch-up).
	mb, ok := items.(MapBackend)
	if !ok {
		t.Fatal("items engine is not a MapBackend")
	}

	if _, found, _ := mb.MapGet(ctx, "role_items", "item-c"); !found {
		t.Fatal("demoted engine lost its own served collection")
	}

	cb, ok := items.(CounterBackend)
	if !ok {
		t.Fatal("items engine is not a CounterBackend")
	}

	if m, _ := cb.CounterGet(ctx, "role_counts"); m["created"] != 5 {
		t.Fatalf("never-served collection not caught up on mirror: %#v", m)
	}

	// Live events after demotion: survivor serves, mirror replicates.
	if err := store.Apply(
		ctx,
		"roleItemCreated",
		roleItemCreated{ID: "item-z", Name: "nz"},
	); err != nil {
		t.Fatal(err)
	}

	if !waitFor(t, demoteWait, func() bool {
		v, found, _ := mb.MapGet(ctx, "role_items", "item-z")

		return found && v != nil
	}) {
		t.Fatal("post-demote event never replicated to the mirror")
	}

	if res, err = store.Execute(roleFindItem{ID: "item-z"}); err != nil {
		t.Fatal(err)
	}

	if got, ok = res.(roleItem); !ok || got.Name != "nz" {
		t.Fatalf("execute post-demote = %#v, want name nz", res)
	}

	// Audit trail records the demotion trigger.
	history := store.PlanHistory()
	if len(history) == 0 || history[len(history)-1].Trigger != triggerEngineDemote {
		t.Fatalf("last audit trigger = %#v, want %q", history, triggerEngineDemote)
	}

	// Promote back: drain + flip, queries may route home, results stay correct.
	if err := store.PromoteEngine(ctx, "items"); err != nil {
		t.Fatalf("promote after demote: %v", err)
	}

	if role, _ := store.EngineRole("items"); role != RoleActive {
		t.Fatalf("role after promote = %q, want Active", role)
	}
}

// TestDemoteEngine_Refusals proves demotion refuses the states it must never
// silently accept.
func TestDemoteEngine_Refusals(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, _, _ := demoteTestStore(t)

	if err := store.DemoteEngine(ctx, "ghost"); err == nil {
		t.Fatal("unknown engine must be refused")
	}

	if err := store.AddEngine(ctx, renamed("shadow"), WithEngineRole(RoleBackup)); err != nil {
		t.Fatal(err)
	}

	if err := store.DemoteEngine(ctx, "shadow"); err == nil {
		t.Fatal("demoting a shadow engine must be refused")
	}

	if err := store.DemoteEngine(ctx, "items", WithDemoteRole(RoleActive)); err == nil {
		t.Fatal("non-shadow target role must be refused")
	}

	single, err := Plan([]Engine{NewMemoryEngine()}, roleItemQuery())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = single.Close() })

	if err := single.DemoteEngine(ctx, "memory"); err == nil ||
		!strings.Contains(err.Error(), "only routable engine") {
		t.Fatalf("demoting the only routable engine must be refused, got %v", err)
	}

	// No EventLog: catch-up is impossible, so demotion must refuse up front.
	store.eventLog = nil

	if err := store.DemoteEngine(ctx, "items"); err == nil ||
		!strings.Contains(err.Error(), "EventLog") {
		t.Fatalf("demotion without an EventLog must be refused, got %v", err)
	}
}

// TestDemoteEngine_NonIdempotentGuard proves the re-routed catch-up refuses
// non-idempotent folds unless the operator opts in with WithDemoteForce, and
// that the forced replay lands exactly-once on the survivor.
func TestDemoteEngine_NonIdempotentGuard(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, _, counts := demoteTestStore(t)

	for range 3 {
		if err := store.Apply(
			ctx,
			"roleItemCreated",
			roleItemCreated{ID: "i", Name: "n"},
		); err != nil {
			t.Fatal(err)
		}
	}

	err := store.DemoteEngine(ctx, "counts")
	if err == nil || !strings.Contains(err.Error(), "WithDemoteForce") {
		t.Fatalf("non-idempotent re-route must demand WithDemoteForce, got %v", err)
	}

	if err = store.DemoteEngine(ctx, "counts", WithDemoteForce()); err != nil {
		t.Fatalf("forced demote: %v", err)
	}

	// Exactly-once: the counter survived with its full history, not double.
	res, err := store.Execute(roleCountInput{})
	if err != nil {
		t.Fatal(err)
	}

	if m, ok := res.(map[string]int64); !ok || m["created"] != 3 {
		t.Fatalf("counter after forced demote = %#v, want created=3", res)
	}

	// The demoted mirror also holds the never-served item collection once.
	mb, ok := counts.(MapBackend)
	if !ok {
		t.Fatal("counts engine is not a MapBackend")
	}

	if _, found, _ := mb.MapGet(ctx, "role_items", "i"); !found {
		t.Fatal("never-served collection missing on demoted mirror")
	}
}

// TestDemoteEngine_ConcurrentExactlyOnce races applies against the demotion
// transition and asserts the survivor's counter equals the number of applied
// events exactly: any dispatch/replication straddle would double-count or drop.
func TestDemoteEngine_ConcurrentExactlyOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, _, _ := demoteTestStore(t)

	const total = 200

	var (
		wg    sync.WaitGroup
		demot sync.WaitGroup
	)

	wg.Add(total)

	for i := range total {
		go func(i int) {
			defer wg.Done()

			if err := store.Apply(ctx, "roleItemCreated", roleItemCreated{
				ID:   "item-" + strconv.Itoa(i),
				Name: "n" + strconv.Itoa(i),
			}); err != nil {
				t.Errorf("apply %d: %v", i, err)
			}
		}(i)
	}

	demot.Add(1)

	go func() {
		defer demot.Done()

		if err := store.DemoteEngine(ctx, "items", WithDemoteForce()); err != nil {
			t.Errorf("demote: %v", err)
		}
	}()

	wg.Wait()
	demot.Wait()

	res, err := store.Execute(roleCountInput{})
	if err != nil {
		t.Fatal(err)
	}

	if m, ok := res.(map[string]int64); !ok || m["created"] != total {
		t.Fatalf("exactly-once violated: counter = %#v, want %d", res, total)
	}
}
