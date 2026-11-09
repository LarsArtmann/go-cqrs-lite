package metaengine_test

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ── Fixtures: social graph with follow/unfollow tombstones ──

type UserFollowed struct {
	Follower string
	Followee string
}

type UserUnfollowed struct {
	Follower string
	Followee string
}

type SocialFollowees struct {
	User       string
	Depth      int
	Undirected bool
}

func socialFolloweesQuery() metaengine.QueryDecl[SocialFollowees, []any] {
	return metaengine.Query[SocialFollowees, []any](
		"social_followees",
		metaengine.OnRecord(UserFollowed{}, func(_ record.Record, e UserFollowed) metaengine.Edge {
			return metaengine.Edge{From: e.Follower, To: e.Followee}
		}),
		metaengine.OnRecord(
			UserUnfollowed{},
			func(_ record.Record, e UserUnfollowed) metaengine.EdgeRemoval {
				return metaengine.EdgeRemoval{From: e.Follower, To: e.Followee}
			},
		),
	)
}

// directedOnlyEngine exposes graph add/read but NOT removal or undirected
// traversal — the negative-path probe for the optional capabilities.
type directedOnlyEngine struct {
	metaengine.Engine
}

func (d directedOnlyEngine) GraphAddEdge(ctx context.Context, col string, e metaengine.Edge) error {
	return d.Engine.(graphBackend).GraphAddEdge(ctx, col, e)
}

func (d directedOnlyEngine) GraphNeighbors(
	ctx context.Context, col string, node any, depth int,
) ([]any, error) {
	return d.Engine.(graphBackend).GraphNeighbors(ctx, col, node, depth)
}

// vectorOnlyEngine exposes plain VectorBackend but NOT the filter extension.
type vectorOnlyEngine struct {
	metaengine.Engine
}

func (v vectorOnlyEngine) VectorInsert(
	ctx context.Context, col string, emb metaengine.Embedding,
) error {
	return v.Engine.(metaengine.VectorBackend).VectorInsert(ctx, col, emb)
}

func (v vectorOnlyEngine) VectorSearch(
	ctx context.Context, col string, query []float32, k int, metric string,
) ([]metaengine.VectorResult, error) {
	return v.Engine.(metaengine.VectorBackend).VectorSearch(ctx, col, query, k, metric)
}

func neighborsOf(t *testing.T, store *metaengine.Store, input SocialFollowees) []string {
	t.Helper()

	raw, err := store.ExecuteCtx(context.Background(), input)
	if err != nil {
		t.Fatalf("ExecuteCtx(%+v): %v", input, err)
	}

	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("traversal result type %T, want []any", raw)
	}

	out := make([]string, len(items))
	for i, v := range items {
		out[i], _ = v.(string)
	}

	return out
}

// contains reports whether any element of list contains want as a
// substring — used both for exact neighbor membership and for asserting
// that error text names a missing capability.
func contains(list []string, want string) bool {
	for _, v := range list {
		if strings.Contains(v, want) {
			return true
		}
	}

	return false
}

// ── EdgeRemoval fold (ADR-0114 style tombstones) ──

func TestEdgeRemovalFold_Classification(t *testing.T) {
	t.Parallel()

	fold := metaengine.OnRecord(
		UserUnfollowed{},
		func(_ record.Record, e UserUnfollowed) metaengine.EdgeRemoval {
			return metaengine.EdgeRemoval{From: e.Follower, To: e.Followee}
		},
	)

	if fold.Kind() != metaengine.FoldEdgeRemove {
		t.Errorf("Kind() = %q, want %q", fold.Kind(), metaengine.FoldEdgeRemove)
	}

	if fold.EventType() != "UserUnfollowed" {
		t.Errorf("EventType() = %q, want UserUnfollowed", fold.EventType())
	}
}

func TestEdgeRemovalFold_TombstoneRemovesEdge(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		socialFolloweesQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	apply := func(eventType string, payload any) {
		t.Helper()
		if err := store.Apply(ctx, eventType, payload); err != nil {
			t.Fatalf("Apply %s: %v", eventType, err)
		}
	}

	apply("UserFollowed", UserFollowed{Follower: "alice", Followee: "bob"})
	apply("UserFollowed", UserFollowed{Follower: "alice", Followee: "carol"})

	if got := neighborsOf(t, store, SocialFollowees{User: "alice", Depth: 1}); len(got) != 2 {
		t.Fatalf("before tombstone: neighbors = %v, want 2", got)
	}

	apply("UserUnfollowed", UserUnfollowed{Follower: "alice", Followee: "bob"})

	got := neighborsOf(t, store, SocialFollowees{User: "alice", Depth: 1})
	if len(got) != 1 || got[0] != "carol" {
		t.Errorf("after tombstone: neighbors = %v, want [carol]", got)
	}
}

func TestEdgeRemovalFold_EngineWithoutRemovalFailsExplicitly(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{directedOnlyEngine{Engine: metaengine.NewMemoryEngine()}},
		socialFolloweesQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(
		ctx,
		"UserFollowed",
		UserFollowed{Follower: "a", Followee: "b"},
	); err != nil {
		t.Fatalf("Apply follow: %v", err)
	}

	err = store.Apply(ctx, "UserUnfollowed", UserUnfollowed{Follower: "a", Followee: "b"})
	if err == nil {
		t.Fatal("EdgeRemoval fold against engine without GraphRemoveEdge must fail")
	}

	if !contains([]string{err.Error()}, "GraphRemoveEdge") {
		t.Errorf("error should name the missing capability, got: %v", err)
	}
}

// ── Undirected traversal ──

func TestUndirectedTraversal_ThroughStore(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		socialFolloweesQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	// Incoming edge (bob → alice): invisible to directed traversal.
	if err := store.Apply(
		ctx,
		"UserFollowed",
		UserFollowed{Follower: "bob", Followee: "alice"},
	); err != nil {
		t.Fatal(err)
	}

	if got := neighborsOf(
		t,
		store,
		SocialFollowees{User: "alice", Depth: 1},
	); contains(
		got,
		"bob",
	) {
		t.Errorf("directed traversal saw incoming edge: %v", got)
	}

	got := neighborsOf(t, store, SocialFollowees{User: "alice", Depth: 1, Undirected: true})
	if !contains(got, "bob") {
		t.Errorf("undirected traversal missed incoming edge: %v", got)
	}
}

func TestUndirectedTraversal_EngineWithoutCapabilityFailsExplicitly(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{directedOnlyEngine{Engine: metaengine.NewMemoryEngine()}},
		socialFolloweesQuery(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.ExecuteCtx(context.Background(),
		SocialFollowees{User: "alice", Depth: 1, Undirected: true})
	if err == nil {
		t.Fatal("undirected query against engine without GraphNeighborsUndirected must fail")
	}

	if !contains([]string{err.Error()}, "GraphNeighborsUndirected") {
		t.Errorf("error should name the missing capability, got: %v", err)
	}
}

// ── Filtered k-NN ──

type DocEmbeddedWithMeta struct {
	ID       string
	Values   []float32
	Metadata map[string]any
}

type SemanticSearchFilteredInput struct {
	Vector  []float32
	Metric  string
	K       int
	Filters []metaengine.VectorFilter
}

func TestVectorFilteredPipeline_ThroughStore(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		metaengine.Query[SemanticSearchFilteredInput, metaengine.VectorResult](
			"semantic_search_filtered",
			metaengine.OnRecord(
				DocEmbeddedWithMeta{},
				func(_ record.Record, e DocEmbeddedWithMeta) metaengine.Embedding {
					return metaengine.Embedding{ID: e.ID, Values: e.Values, Metadata: e.Metadata}
				},
			),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	docs := []DocEmbeddedWithMeta{
		{ID: "near-a", Values: []float32{1, 0}, Metadata: map[string]any{"tenant": "a"}},
		{ID: "near-b", Values: []float32{0.95, 0.05}, Metadata: map[string]any{"tenant": "b"}},
		{ID: "far-a", Values: []float32{0, 1}, Metadata: map[string]any{"tenant": "a"}},
	}
	for _, d := range docs {
		if err := store.Apply(ctx, "DocEmbeddedWithMeta", d); err != nil {
			t.Fatalf("Apply %s: %v", d.ID, err)
		}
	}

	// Filtered top-2 = the two nearest tenant-a vectors — NOT the global
	// top-2 with tenant-b rows dropped.
	results, err := metaengine.VectorExecuteTyped[SemanticSearchFilteredInput](
		ctx, store,
		SemanticSearchFilteredInput{
			Vector: []float32{1, 0},
			Metric: "cosine",
			K:      2,
			Filters: []metaengine.VectorFilter{
				{Field: "tenant", Op: metaengine.FilterEq, Value: "a"},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("filtered k-NN returned %d results, want 2: %v", len(results), results)
	}

	if results[0].ID != "near-a" || results[1].ID != "far-a" {
		t.Errorf("filtered k-NN = [%s, %s], want [near-a, far-a]",
			results[0].ID, results[1].ID)
	}
}

func TestVectorFiltered_EngineWithoutCapabilityFailsExplicitly(t *testing.T) {
	t.Parallel()

	store, err := metaengine.Plan(
		[]metaengine.Engine{vectorOnlyEngine{Engine: metaengine.NewMemoryEngine()}},
		metaengine.Query[SemanticSearchFilteredInput, metaengine.VectorResult](
			"semantic_search_unfiltered_engine",
			metaengine.OnRecord(
				DocEmbeddedWithMeta{},
				func(_ record.Record, e DocEmbeddedWithMeta) metaengine.Embedding {
					return metaengine.Embedding{ID: e.ID, Values: e.Values, Metadata: e.Metadata}
				},
			),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()

	if err := store.Apply(ctx, "DocEmbeddedWithMeta", DocEmbeddedWithMeta{
		ID: "x", Values: []float32{1, 0}, Metadata: map[string]any{"tenant": "a"},
	}); err != nil {
		t.Fatal(err)
	}

	_, err = store.ExecuteCtx(ctx, SemanticSearchFilteredInput{
		Vector:  []float32{1, 0},
		K:       1,
		Filters: []metaengine.VectorFilter{{Field: "tenant", Op: metaengine.FilterEq, Value: "a"}},
	})
	if err == nil {
		t.Fatal("filtered search against plain VectorBackend engine must fail explicitly")
	}

	if !contains([]string{err.Error()}, "VectorFilterBackend") {
		t.Errorf("error should name the missing capability, got: %v", err)
	}
}

func TestVectorMatchesFilters_Operators(t *testing.T) {
	t.Parallel()

	meta := map[string]any{"tenant": "a", "score": float64(7)}

	tests := []struct {
		name string
		f    metaengine.VectorFilter
		want bool
	}{
		{
			"eq match",
			metaengine.VectorFilter{Field: "tenant", Op: metaengine.FilterEq, Value: "a"},
			true,
		},
		{
			"eq mismatch",
			metaengine.VectorFilter{Field: "tenant", Op: metaengine.FilterEq, Value: "b"},
			false,
		},
		{
			"ne other",
			metaengine.VectorFilter{Field: "tenant", Op: metaengine.FilterNe, Value: "b"},
			true,
		},
		{
			"ne same",
			metaengine.VectorFilter{Field: "tenant", Op: metaengine.FilterNe, Value: "a"},
			false,
		},
		{
			"in match", metaengine.VectorFilter{
				Field: "tenant", Op: metaengine.FilterIn, Value: []any{"a", "c"},
			}, true,
		},
		{
			"gt below",
			metaengine.VectorFilter{Field: "score", Op: metaengine.FilterGt, Value: float64(6)},
			true,
		},
		{
			"le above",
			metaengine.VectorFilter{Field: "score", Op: metaengine.FilterLe, Value: float64(8)},
			true,
		},
		{
			"missing field eq",
			metaengine.VectorFilter{Field: "absent", Op: metaengine.FilterEq, Value: "x"},
			false,
		},
		{
			"missing field ne",
			metaengine.VectorFilter{Field: "absent", Op: metaengine.FilterNe, Value: "x"},
			true,
		},
	}

	for _, tc := range tests {
		if got := metaengine.VectorMatchesFilters(
			meta,
			[]metaengine.VectorFilter{tc.f},
		); got != tc.want {
			t.Errorf("%s: VectorMatchesFilters = %v, want %v", tc.name, got, tc.want)
		}
	}

	if !metaengine.VectorMatchesFilters(meta, nil) {
		t.Error("empty filter slice must match everything")
	}
}
