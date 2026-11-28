# search/v4

The search contract of go-cqrs-lite: declare the searchable fields of a
record type once, call `Index.Search`, and read `Result.Via` to learn which
kind of search ran.

```go
index, err := search.New(search.Spec[Note]{
	Fields: []search.Field[Note]{
		{Name: "title", Weight: 1, Value: func(n Note) string { return n.Title }},
		{Name: "body", Weight: 0.2, Value: func(n Note) string { return n.Body }},
	},
}, search.NoEngine, source) // or an engine that offers full-text search
if err != nil {
	return err
}

result, err := index.Search(ctx, search.Request{Query: r.URL.Query().Get("q")})
if err != nil {
	return err
}
fmt.Println(result.Via, result.TotalCount, result.Truncated, result.Degraded)
```

## The ladder

| Tier | Kind            | Needs                               | `Via`             |
| ---- | --------------- | ----------------------------------- | ----------------- |
| 0    | Client, lexical | a static index (`BuildStaticIndex`) | never server-side |
| 1    | Scan            | nothing                             | `ViaScan`         |
| 2    | Full-text       | a full-text index on the engine     | `ViaFullText`     |
| 3    | Vector          | a vector index and an `Embedder`    | `ViaVector`       |
| 4    | Hybrid          | tiers 2 and 3, fused by rank        | `ViaHybrid`       |

`ModeAuto` (the zero value) climbs as far as the deployment and the query
allow. Its rules: fewer than two characters, or operator tokens only, means
facets only; identifier-like text is lexical; a single token of at most six
characters is lexical; otherwise hybrid when an embedder and a vector index
exist, else the best lexical tier. An explicit mode is strict.

## What ships in this release

- `Spec`, `Field`, `Request`, `Result`, `Hit`, `Segment`, `Mode`, `Via`.
- `New`, `Index.Search`, `Tier` (the ladder as a pure function).
- `Score`, `RankHits`, `Tokenize`, `Fold`, `IsIdentifierLike`.
- `LikePattern`, `EscapeLike`, `FTS5Query`, `Snippet`, `SnippetTerms`.
- `BuildStaticIndex`, `StaticIndex.WriteFile`, `LoadStaticIndex`.
- `Slice` (a `Source` over records in memory) and `NoEngine`.

The executors for the full-text, vector and hybrid tiers arrive with the
engine work; until then, Auto steps down to the scan and sets `Degraded`.

## Invariants

- An empty query is the unfiltered list in source order, never an error.
- Auto never fails for a missing capability; it steps down and says so.
- An explicit mode without its capability is an error.
- No function returns HTML: highlighting is `[]Segment`, the template picks the tag.
- Nothing leaves the machine unless the operator configured an embedder, and
  then only text from fields that are not `Private`.

Decision record: `docs/adr/0169-search-ladder-and-operator-embedder.md`.
Recipe: `.agents/skills/go-cqrs-lite/references/recipes.md` §2.52.

The browser matcher of templ-components replays `testdata/conformance.json`;
regenerate it with `go test ./search/v4 -update` when the ranking rule changes.
