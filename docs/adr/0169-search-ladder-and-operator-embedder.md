# ADR-0169: Search is a ladder of kinds, and embeddings belong to the operator

- Status: Accepted (2026-11-28)
- Date: 2026-11-28

## Context

The 2026-11-27 fleet census (`docs/research/2026-11-27_search-once-fleet-census-and-unification-proposal.html`
in the `docs` repository) found 889 search-named declarations in 78 repositories. 845 of them are
search code, spread over 72 repositories, and almost none of it is shared: 66 repositories match
with `strings.Contains(strings.ToLower(...))`, seven hand-copied FTS5 query sanitizers exist with
drift, three repositories hand-roll BM25 or TF-IDF, and two highlight paths emit user text
through `templ.Raw`.

Inside this library the primitives exist but do not meet:

- `FilterOp` is closed to `= != < <= > >= IN`. There is no text predicate; `FilterContains` and
  `FilterPrefix` are scoped for the v5 window because they change the operator registry.
- `SearchBackend` is implemented by memory, dgraph and iroh only. sqlite, turso, pebble, duckdb
  and postgres declare it degraded and error at execution, so nine repositories keep FTS as
  satellite SQL.
- `VectorBackend` and `VectorFilterBackend` exist on every engine, and `DueClaimer` on every
  engine, but nothing composes them into a search pipeline.

The owner's direction of 2026-11-27: search must support every kind (full text, vector
embeddings, fuzzy, client side), be smart behind a minimal SDK, and leave embeddings to the
operator.

## Decision

1. **The contract lives in a new small module, `go-cqrs-lite/search/v4`.** It depends on
   `query/v4` and `go-error-family` only. It never imports `system`: `system` imports the engine
   layer and adapts to the contract (ADR-0123 keeps `system.New` the composition root). A `/v5`
   twin follows ADR-0152's copy-forward rule when the v5 cut lands.

2. **The vocabulary has three words.** A _Filter_ narrows by closed-set facets. A _Search_ turns
   free text into ranked hits. A _Pick_ chooses or navigates. They combine and are never
   conflated.

3. **Search is a ladder, and every result says which rung ran.**

   | Tier | Kind            | Needs                               | Auto stops here when                         |
   | ---- | --------------- | ----------------------------------- | -------------------------------------------- |
   | 0    | Client, lexical | a static index or rendered rows     | the box was given an index                   |
   | 1    | Scan, lexical   | nothing                             | the engine has no full-text index            |
   | 2    | Full-text       | FTS5, tsvector or an inverted index | no embedder, or the query is lexical by rule |
   | 3    | Vector          | a vector index and an embedder      | an embedder, but no full-text index          |
   | 4    | Hybrid          | tiers 2 and 3, fused by rank        | nothing: this is the top                     |

   Facets combine with every tier. `Result.Via` names the tier that ran.

4. **Auto follows five written rules, and explicit modes are strict.**
   1. Fewer than two characters, or only operator tokens: no text search, facets only.
   2. Identifier-like text (mostly digits or punctuation): lexical.
   3. A single token of at most six characters: lexical.
   4. Otherwise, with an embedder: hybrid. Only the free text is embedded.
   5. If the embedder fails, lexical serves the query and a breaker skips embedding for 30
      seconds after 3 failures.

   An explicit mode that the deployment cannot serve is an error, never a silent step down.

5. **Embeddings are the operator's.** The SDK never embeds on its own. A nil embedder means
   lexical search only, which is a complete product. Domain code cannot tell whether semantic
   search is on.

6. **The embedding pipeline runs inside `system`**, on `DueClaimer` and `VectorBackend`, with
   the embedder supplied through `DeploymentConfig`. Lexical and vector result lists are fused
   by reciprocal rank fusion with k = 60.

7. **Full-text is a metaengine capability** (ADR-0142 gate B). `SearchBackend` becomes honest:
   an engine either serves it or reports that it cannot, and no engine returns an empty page for
   "no index".

8. **One ranking rule in memory.** A term matches when any field contains it case-folded; exact
   beats prefix, prefix beats substring, each multiplied by the field weight; every term must
   match. The browser matcher runs the same rule, pinned by a conformance corpus both runtimes
   replay.

9. **No function returns HTML.** Highlighting is typed segments (`Segment{Text, Marked}`); the
   template picks the tag.

## Consequences

- `FilterOp` stays closed in v4. `FilterContains` and `FilterPrefix` are a separate epic and
  wait for the v5 window.
- Wave A (the lexical rung) ships the contract, the templ-components pair (`SearchInput`,
  `SearchBox`, `SearchResults`) and the cqrs-htmx binding, and migrates the cqrs-htmx admin UI
  first (ADR-0001 rule 1).
- Engines that declare `SearchBackend` degraded keep that behaviour until the full-text epic
  replaces it; the new module does not call them in wave A.
- The execution plan is `docs/planning/2026-11-28_03-14_search-once-ladder-pareto-execution-plan.md`
  in the `docs` repository. Its section 5 records the owner's answers to the nine decisions.

## What this amends

- ADR-0142 (gate B): this ADR is the design the gate asks for before the full-text capability
  is built. It changes nothing in ADR-0142 itself.
- ADR-0152: the module ships as `search/v4` under the copy-forward rule, so a `/v5` twin is
  owed when the v5 cut lands.
