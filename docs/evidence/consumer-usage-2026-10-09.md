# Consumer-Usage Evidence — pkg.go.dev Imported-by Sweep (all 52 consumed trains)

**Date:** 2026-10-09
**Method:** every page fetched live from `https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/<train>` (main page for the count, `?tab=importedby` for the decomposition of every nonzero train). pkg.go.dev shows `License: UNKNOWN` on every train (docs hidden by the proprietary license) and counts only PUBLIC importers — private fleet apps are invisible here by design; the `~/projects` who-uses scan (2026-10-08 audit) covers those.
**Purpose:** closes the audit gap *"the other 47 consumed trains not individually checked against pkg.go.dev/proxy"* (`docs/status/2026-10-08_20-23_go-modules-audit-external-usage-and-self-review.md`, line 50) and completes the external-consumer evidence for [ADR-0152](../adr/0152-fleet-first-module-topology-v5-dual-support.md).

## Verdict

**ZERO non-LarsArtmann importers across all 52 consumed trains.** Every nonzero `Imported by` count decomposes 100% into:

1. intra-repo sibling modules (`github.com/larsartmann/go-cqrs-lite/*`),
2. the two in-scope companions (`github.com/larsartmann/cqrs-htmx/*`, `github.com/larsartmann/go-appkit/*`),
3. four other LarsArtmann fleet repos — `go-localsync`, `go-taskqueue` (2 packages), `vision-review-agent` (all verified present in `~/projects`, i.e. inside the who-uses universe, 2026-10-09).

The ADR-0152 "fleet-only market" claim is evidence-complete. A correction to the audit's shorthand: "Imported-by: 0" holds for most trains but NOT all — pkg.go.dev counts self-imports from this repo's own public tagged siblings. The signal that matters (external, non-LarsArtmann importers) is zero everywhere.

## Results

### Imported by: 0 (42 trains)

event, dispatcher, metadata, record, command, query, otel, projection, snapshot, watermill, dedup, metaengine, scheduling, metaengine/projectionadapter, metaengine/sqliteengine, projectionhost, middleware, storage, system, commandlifecycle, commandlifecycle/projections, storage/memory, scenario, signing, catalog, encryption, event/v4/eventtest, idempotency/sqlstore, stack/sqlite, cmd/cqrs-lint (the live `/v4` path), testutil, testutil/pgtestcontainer, queue, queue/sqlite, queue/postgres, scheduling/engine, storage/turso, metaengine/pebbleengine — plus the retired-but-served trains codec/v4, idempotency/v4, flightrecorder/v4, retry/v4.

Notable: `cmd/cqrs-lint` shows 0 public importers (buildflow/gomend consume it privately/toolside); `system` shows 0 (its 13 fleet adopters + dashboardui are private or unindexed).

### Nonzero, decomposed (10 trains)

| Train | Imported by | Decomposition (from `?tab=importedby`) |
|---|---|---|
| id/v4 | 48 | 39 intra-repo + 8 cqrs-htmx/* + go-localsync/pkg/cqrs + go-taskqueue ×2 + vision-review-agent |
| kv/v4 | 13 | 12 intra-repo + cqrs-htmx/usermgmt |
| decider/v4 | 11 (12 displayed) | 8 intra-repo + cqrs-htmx/usermgmt + go-appkit/cqrs + go-localsync + vision-review-agent |
| stack/v4 | 11 (12 displayed) | 11 intra-repo (presets, benchkit, cqrs-bench) + cqrs-htmx/usermgmt |
| claiming/v4 | 9 | 9 intra-repo (metaengine claimkit + engines + queue/* + scheduling/sqlstore) |
| listing/v4 | 4 | 3 cqrs-htmx (dashboardui ×2, examples) + 1 intra-repo (storage) |
| prometheus/v4 | 2 | cqrs-htmx/examples/observability-demo + example/scheduler-otel-status |
| schema/v4 | 1 | go-localsync/pkg/cqrs |
| storage/bbolt/v4 | 1 (2 displayed) | stack/bbolt + vision-review-agent (internal) |
| scheduling/sqlstore/v4 | 1 | example/scheduler-otel-status |

### Version cross-check

Every train's pkg.go.dev `Version:` matched the committed `versions.json` at sweep time (e.g. event v4.13.1, system v4.11.0, metaengine v4.17.0, cmd/cqrs-lint/v4 v4.15.0). `cmd/cqrs-lint` also has a LEGACY suffix-less page (v0.2.1, the deprecation-stub dead train kept so `@latest` on the old path does not go dark — see CONTRIBUTING "Per-module tagging"); the sweep used the live `/v4` path, Imported by: 0 there.

## Consequences for the plan

- T26's "fleet-zero-v4-pins" gate must include **go-localsync, go-taskqueue, vision-review-agent** alongside cqrs-htmx + go-appkit + the ~40 app residue (they are in `~/projects`, so the who-uses-driven gate already sees them — recorded here so the connection is not lost).
- `id` is the most-imported train publicly (48); its v5 package path change ripples through exactly the companions the plan migrates first.

*Point-in-time snapshot of a third-party site; re-verify before relying on it later.*
