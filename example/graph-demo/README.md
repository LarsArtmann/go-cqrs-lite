# graph-demo

Example: graph projection tier with Schema validation and MemoryDriver read API.

Models a discussion forum: users author messages, messages reply to other messages.
The graph shape captures reply chains (variable-depth traversal) and authorship —
read patterns the relational tier handles poorly (recursive CTEs) and the KV tier
cannot express.

## Run

```bash
go run .
```

## What it demonstrates

1. **Schema validation** — A `graph.Schema` declares node types (User, Message),
   edge types (AUTHORED_BY, REPLY_TO), and their properties. Every write is
   validated against the schema before touching the graph.

2. **Projection** — Events are projected into the graph via `GraphProjection`
   with `WithSchema`. The handler merges nodes and edges atomically per event.

3. **Read API** — The `MemoryDriver` exposes typed reads: `Query` (filter by
   label + predicate), `Traverse` (BFS following edge type), `ShortestPath`
   (BFS with parent tracking).

## Output

```
=== All Messages ===
  m1: Hello world!
  m2: Hi Alice!
  m3: Welcome!
  m4: Thanks!

=== Reply Chain from m4 (REPLY_TO traversal) ===
Following REPLY_TO edges from m4 back to its ancestors:
  → m2: Hi Alice!
  → m1: Hello world!

=== Shortest Path: m4 → m1 (reply chain) ===
m4 → m2 → m1
```
