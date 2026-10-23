# F010 doesn't recognize metaengine-native graph usage and recommends migrating off the Graph ADT

## Symptom

F010 fires on a consumer that projects relations through the **metaengine Graph ADT** (`metaengine.Edge`/`metaengine.EdgeRemoval` folds + graph traversal) with "graph module is not used — recursive SQL CTEs are slow for deep traversals", suggesting `graph.NewGraphProjection`. The CRM is the inverse of the suggestion's premise: it deliberately moved OFF a graphadapter/GraphProjection design TO the native metaengine Graph ADT (traversal runs on sqlite recursive CTEs exactly because that is the supported metaengine path — `TestRelationGraphTraversal` pins two-hop reach + retraction).

Finding site (crm @ 34f26a6): `internal/app/app.go:313` (`buildAdjacency`) — an adjacency helper beside the `Network` graph query.

## Root cause (source)

`cmd/cqrs-lint/pkg/rules/adoption/f010_f011.go:23` (@ ec36eb8d2b) — the only escape hatch is importing the standalone graph module:

```go
for _, sc := range coachingScopes(ctx) {
	if importsPathIn(sc.files, "go-cqrs-lite/graph") {
		continue
	}
	pos, ok := hasTraversalPatternsIn(ctx.Fset, sc.files)
	…
```

A consumer using `go-cqrs-lite/metaengine` graph constructs (`metaengine.Edge`, `metaengine.EdgeRemoval`, a graph-typed `metaengine.Query`) satisfies the rule's intent — the traversal IS backed by a first-party graph facility — but not the import check, so the coaching fires and points at the module the consumer intentionally avoids.

## Fix

Extend the skip to metaengine-native graph usage, e.g. any of:

- scope imports `go-cqrs-lite/metaengine` **and** the files reference `Edge`/`EdgeRemoval` (selector names, same string-match budget the current check spends); or
- the registry records graph-typed folds/queries and the detector consults it.

At minimum, the suggestion text should name the metaengine Graph ADT as the native alternative so readers with the CRM's shape know the finding is advisory for them, not a migration recommendation.

## To verify

- Fixture projecting `metaengine.Edge` + a recursive traversal helper with no `go-cqrs-lite/graph` import → no F010.
- Real consumer: crm @ 34f26a6 drops the single F010 at `internal/app/app.go:313`.

💘 Generated with Crush
