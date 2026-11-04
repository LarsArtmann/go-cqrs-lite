# T08 slice 1 — `cqrs-upgrade --to-v5` dry-run receipts (2026-11-04)

**Method:** `cqrs-upgrade --to-v5 --dry-run --workspace <repo>` per fleet repo,
binary built from this tree (`cmd/cqrs-upgrade`, commit carried by this
evidence's session). The pass rewrites folded v4 train imports
(`github.com/larsartmann/go-cqrs-lite/<train>/v4[...])` →
`github.com/larsartmann/go-cqrs-lite/core/v5/<train>[...]`) discovered by
AST import-spec ranges only — strings and comments are never touched (unit-
pinned). `go.mod` is report-only in slice 1: `core/v5` is untagged, so an
applied require would not resolve.

## Receipts

| Repo        | Modules | Go files scanned | Planned import rewrites | Distinct not-yet-folded trains |
| ----------- | ------- | ---------------- | ----------------------- | ------------------------------ |
| cqrs-htmx   | 29      | 1,413            | 1,000                   | 32                             |
| go-appkit   | 12      | 308              | 84                      | 13                             |
| bank-sync   | 5       | 459              | 137                     | 20                             |

Full per-module reports: captured in the session's `/tmp/v5-<repo>.txt`;
the numbers above are their aggregates.

## What the receipts prove

1. **The mechanical migration is a command now.** 1,221 planned rewrites
   across the three flagship consumers — every one a byte-exact import
   literal swap that a consumer applies with `cqrs-upgrade --to-v5` once
   `core/v5` tags (plus `go.mod`, slice 2).
2. **The not-yet-folded list is the fold roadmap.** cqrs-htmx (the widest
   consumer) still needs: catalog (+docserver/eventcatalog/simple),
   commandlifecycle, decider, encryption, event/v4/eventtest, listing,
   metaengine (+projectionadapter/sqliteengine), middleware, otel,
   projection, projectionhost, prometheus, scenario, schema, signing,
   snapshot, stack (+mysql/postgres/sqlite/sqlopt), storage (+memory/sql),
   system, systemscenario, watermill — 32 trains. bank-sync's 20 and
   go-appkit's 13 are subsets of the same set.
3. **Slice boundary is honest:** `--strict --to-v5` exits non-zero while any
   not-yet-folded cqrs v4 import remains (or a pass error occurs), so fleet
   CI can gate the migration the same way it gates v5-removed API usage
   today.

## Follow-ups (rowed in TODO_LIST)

- T08 slice 2: `go.mod` require rewrite (blocked on the first `core/v5.x.y`
  tag) + `go mod tidy` verification in apply mode.
- `event/v4/eventtest` has no core/v5 twin yet (test-support package); fold
  or explicitly exclude it before fleet-wide apply.
