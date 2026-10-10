package main

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// The read side: the SAME two events folded into graph edges. user.followed
// adds a directed edge follower→followee; user.unfollowed retracts exactly
// that edge. The planner classifies this query as the Graph ADT and routes
// the traversal to the deployed engine — sqlite's recursive CTE by default,
// Dgraph's native @recurse traversal when an operator deploys it.

const followGraphCollection = "follow_graph"

// ReachabilityQuery is the traversal input. Conventions (metaengine
// ReadTraversal): Node is the start node, Depth defaults to 1 when zero, and
// Undirected walks edges in both directions when true (engines without
// undirected support report the missing capability instead of guessing).
type ReachabilityQuery struct {
	Node       string `json:"node"`
	Depth      int    `json:"depth"`
	Undirected bool   `json:"undirected"`
}

// followGraphProjection declares the graph read model: Edge folds add,
// EdgeRemoval folds retract (ADR-0114). The follower comes from the event's
// stream ID (EventWithID.ID); the followee from the decoded payload.
func followGraphProjection() ([]system.ProjectionDeclaration, *projectionadapter.TypeDecoder) {
	addEdge := metaengine.OnRecordTyped(
		string(evtUserFollowed),
		projectionadapter.EventWithID[FollowedPayload]{},
		func(_ record.Record, evt projectionadapter.EventWithID[FollowedPayload]) metaengine.Edge {
			return metaengine.Edge{From: evt.ID, To: evt.Payload.Followee}
		},
	)

	removeEdge := metaengine.OnRecordTyped(
		string(evtUserUnfollowed),
		projectionadapter.EventWithID[UnfollowedPayload]{},
		func(_ record.Record, evt projectionadapter.EventWithID[UnfollowedPayload]) metaengine.EdgeRemoval {
			return metaengine.EdgeRemoval{From: evt.ID, To: evt.Payload.Followee}
		},
	)

	query := metaengine.Query[ReachabilityQuery, []string](
		followGraphCollection, addEdge, removeEdge,
	)

	decoder := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(evtUserFollowed, FollowedPayload{}),
		projectionadapter.Register(evtUserUnfollowed, UnfollowedPayload{}),
	)

	return []system.ProjectionDeclaration{system.RawQuery(query)}, decoder
}
