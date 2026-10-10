# graph-native — a social follow network on graph projections

A system + metaengine example where the read model is a GRAPH: the write side
is two guarded commands per user stream, and the read side folds the same
events into graph edges and answers reachability traversals
("who is within N hops of alice?").

## What It Demonstrates

```
Command → Decider → Event Store → Projection Host → metaengine Graph → Traversal
```

- **Deletion as a domain event (ADR-0114)**: `user.unfollowed` folds to
  `metaengine.EdgeRemoval` — the edge disappears from every traversal, no
  mutable metadata, no tombstone flag.
- **Payload-carried node names**: node names live in the event payload, not
  derived from the stream ID (the branded `StreamID.String` is a display
  form, not a clean node name).
- **Decider guards**: self-follow and duplicate edges are Rejections; unfollow
  of a missing edge is a Conflict.
- **Deployment-time engine choice**: in-memory by default; `--sqlite` swaps
  the source-of-truth engine, `--dgraph` routes projections to Dgraph, and
  `--config` boots the deployment from a `cqrs.yaml`.

## Run It

```bash
cd example/graph-native
GOWORK=off go run .
```

The binary builds a small follow network, traverses it, retracts an edge, and
traverses again. Every printed claim is asserted — the program exits non-zero
if any expectation fails.

## Test It

The BDD suite drives the SAME `DomainConfig` the binary boots through
`systemscenario` Given/When/Then:

```bash
cd example/graph-native
GOWORK=off go test ./... -count=1
```

## How It Works

1. **`domain.go`** — the declared domain: `FollowedPayload`/`UnfollowedPayload`
   facts, the `FollowState` fold, and the `follow`/`unfollow` decide functions
   with their guards.
2. **`projection.go`** — the graph projection: `user.followed` folds to
   `metaengine.EdgeAdd`, `user.unfollowed` to `metaengine.EdgeRemoval`, with a
   typed decoder for the payload.
3. **`composition.go`** — the only seam: `graphDomain()` is everything a
   developer declares, `buildDeployment` is everything an operator decides.
4. **`main.go`** — the runtime loop with converged-traversal polling and
   asserted output.
