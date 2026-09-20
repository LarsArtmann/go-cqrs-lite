# ADR-0144: DeferClose Lives in Tier-0 record

**Date:** 2026-09-20
**Status:** Accepted
**Related:** ADR-0111 (record type extraction — record as the Tier-0 structural base), ADR-0069 (error-wrapping helpers — the per-module contrast), dogfooding self-review 2026-09-19 finding 3 (`docs/status/archived/2026-09-19_16-50_dogfooding-self-review-execution.md`)

## Context

`metaengine.DeferClose` (added 2026-09-10) replaced the verbose
`defer func() { _ = x.Close() }()` idiom across the engine modules — 72
production + 33 test sites at the time. The 2026-09-19 dogfooding self-review
swept 22 more sites (engine + queue modules) and found the tail: 29 production
sites in ~20 files across `storage`, `storage/pebble`,
`storage/turso/indexing`, `scheduling/sqlstore`, `projectionhost`, `stack`,
`kv`, and `cmd/cqrs-lint` that never converted because the helper is hard to
reach.

The primitive is fine; its address is wrong. `metaengine` is a Tier-3
aggregation module whose transitive surface (id, dedup, errorfamily, go-sse,
claiming) is exactly what a leaf storage module must not drag in to reach a
two-line lifecycle helper. `storage/pebble`, `storage/turso`,
`scheduling/sqlstore`, `projectionhost`, and `kv` do not (and should not)
depend on `metaengine`. Growing `metaengine.DeferClose`'s reach by adding that
dependency would trade one inconsistent idiom for real dependency weight.

## Decision

1. **Canonical home: `record.DeferClose(c io.Closer)`** — Tier-0, inside the
   structural base module (ADR-0111). `record` has zero external deps and is
   already imported by `metaengine`, `storage`, `storage/pebble`, and `stack`,
   so most of the remaining sweep converts with **no dependency change at
   all**. The signature accepts `io.Closer`: every target (*sql.Rows, Pebble
   iterators/batches, closer handles) satisfies it structurally.
2. **`metaengine.DeferClose` stays as a self-contained twin.** Its public API
   is load-bearing (70+ call sites); no deprecation. Forwarding to
   `record.DeferClose` was considered and rejected: every module that
   compiles `metaengine` from source via a sibling replace (the whole
   dep-isolated engine family) would need a matching `record` replace too —
   a ~15-module go.mod ripple to deduplicate a one-line body. The two
   implementations are pinned by doc comments pointing at ADR-0144; the
   release wave collapses them the next time both modules tag anyway.
3. **The first Tier-0 sibling edge (`kv` → `record`) is sanctioned.**
   `check-module-layers.sh` allows same-layer dependencies; `record` stays
   zero-dep, so the edge introduces no cycle and no bloat. `kv` is blind
   storage plumbing, and `DeferClose` is structural lifecycle plumbing — the
   match is honest.
4. **Scope guard.** `DeferClose` is for rows, iterators, batches, and handles
   where the func-wrapped discard form spread. It does NOT replace bare
   `defer x.Close()` (C015-exempt, used by queue/postgres), explicit
   rollback-defers (`_ = tx.Rollback()` — wrong helper, different intent), or
   write paths where the close error is actionable.
5. **Sweep executed in the same change:** the remaining sites convert to
   `record.DeferClose` in `storage`, `storage/pebble`,
   `storage/turso/indexing`, `scheduling/sqlstore`, `projectionhost`, `stack`,
   and `kv` (28 sites). `queue/postgres` was audited and is already clean
   (bare defers throughout). The one `cmd/cqrs-lint` site
   (`pkg/suppression/linecache.go`) converts to the bare `defer f.Close()`
   stdlib idiom instead: the tool sits exactly at its dep budget (9/9), and
   pulling the runtime `record` library into a static-analysis tool to close
   one read-only `os.File` fails the proportionality test — that site is the
   documented exception, not a precedent.

## Consequences

- Additive v4.x API: `record.DeferClose` (api golden regen + CHANGELOG in the
  same edit, per repo rule).
- Dependency deltas: `kv`, `scheduling/sqlstore`, `projectionhost`, and
  `storage/turso` gain `record/v4` as a direct require (`kv` moves to
  `go 1.27.1` with it — the published record v4.5.1 declares `go 1.27.1` —
  and `go.work` follows to `go 1.27.1`, aligning with the documented
  toolchain contract). All stay within their `DEP_BUDGET` (verified by
  `scripts/check-module-layers.sh` in the same change); `storage`,
  `storage/pebble`, `stack`, and `metaengine` already require `record`.
  Until the next release wave tags record, every touched module carries the
  standard sibling replace (`replace .../record/v4 => ../record`), stripped
  by `scripts/tag-release.sh` at cut time.
- Consumers can adopt the discard-close idiom from Tier 0 without importing
  the engine substrate — the original complaint in finding 3.
- Future close-idiom sweeps have an unambiguous target address; the
  "mis-tiered helper" class of finding is closed.
