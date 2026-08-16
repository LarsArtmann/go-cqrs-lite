package main

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// ── Graph demo: a social follow network as edges, reachability as a query ──
//
// One event type (UserFollowed) is folded into metaengine.Edge records; the
// planner routes the query to the engine's graph ADT (native on SQLite/PG,
// BFS fallback elsewhere) and returns everyone reachable within N hops.

type UserFollowed struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type FollowGraphQuery struct {
	Node  string `json:"node"`
	Depth int    `json:"depth"`
}

func runGraphDemo(ctx context.Context) error {
	query := metaengine.Query[FollowGraphQuery, []string](
		"follow_graph",
		metaengine.OnRecordTyped(
			"user.followed",
			UserFollowed{},
			func(_ record.Record, evt UserFollowed) metaengine.Edge {
				return metaengine.Edge{From: evt.From, To: evt.To}
			},
		),
	)

	store, err := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, query)
	if err != nil {
		return fmt.Errorf("plan graph: %w", err)
	}

	defer func() { _ = store.Close() }()

	follows := []UserFollowed{
		{From: "alice", To: "bob"},
		{From: "bob", To: "carol"},
		{From: "carol", To: "dave"},
		{From: "alice", To: "carol"},
	}

	for _, follow := range follows {
		if err := store.Apply(ctx, "user.followed", follow); err != nil {
			return fmt.Errorf("apply follow: %w", err)
		}
	}

	for _, depth := range []int{1, 2} {
		result, err := store.ExecuteCtx(ctx, FollowGraphQuery{Node: "alice", Depth: depth})
		if err != nil {
			return fmt.Errorf("traverse depth %d: %w", depth, err)
		}

		neighbors, ok := result.([]any)
		if !ok {
			return fmt.Errorf("traverse depth %d: unexpected result type %T", depth, result)
		}

		fmt.Printf("alice reaches %d account(s) within %d hop(s):", len(neighbors), depth)
		for _, neighbor := range neighbors {
			fmt.Printf(" %v", neighbor)
		}
		fmt.Println()
	}

	return nil
}
