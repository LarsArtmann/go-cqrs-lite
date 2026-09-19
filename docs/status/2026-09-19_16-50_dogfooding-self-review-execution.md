# Dogfooding Self-Review - Execution Status

- **Date:** 2026-09-19 16:50 CEST
- **Scope:** "How can go-cqrs-lite better use itself?" - audit + execute + verify
- **Status:** PAUSED per user instruction. 2 fixes shipped and verified, report written. 4 items remain (1 owner-gated).

## TL;DR

The library's own code does not always consume the primitives it ships for
consumers. The clearest proof: two sibling transports in the same module family
implement the same op-dedup window two different ways, and the loopback one had
a documented correctness gap. Fixed that, swept the close idiom in engine
modules, and wrote the mandated brutal-self-review report. Now paused for
instructions.

## What was asked

A brutal self-review framed as dogfooding: where does go-cqrs-lite fail to use
its own abstractions? Break into steps, execute and verify one at a time,
repeat until done.

## Shipped and verified

### Fix 1 - loopback transport now uses the library's own `dedup.Ring`

- **Why:** `metaengine/irohengine/loopback` hand-rolled op-dedup as
  `map[string]struct{}` that **resets wholesale at 10k entries**, so an op ID
  seen before the reset was forgotten and re-applied on redelivery. The sibling
  `metaengine/irohengine/quic` already used `dedup.Ring` ("no reset gap").
- **Files:** `transport.go` (field + init + `DefaultDedupCapacity`),
  `conn.go` (`markSeen`), `dedup_internal_test.go` (rewritten to pin graceful
  eviction of the oldest ID + bounded growth), `go.mod` (`dedup` promoted to a
  direct require; budget 4 -> 3 used).
- **Verified:** `GOWORK=off go test ./...` green (2.0s) AND workspace-mode
  `go test ./...` green (2.1s).
- **Commit:** absorbed by the auto-commit daemon into `02a133d7e`.

### Fix 2 - `metaengine.DeferClose` sweep in engine modules

- **Why:** `DeferClose` exists specifically to replace
  `defer func() { _ = x.Close() }()`, and AGENTS.md records the 2026-09-10
  conversion - but 9 sites in engine modules (which already import
  `metaengine`) never got it.
- **Files:** `metaengine/enginetest/{restart_safety,soak}.go`,
  `metaengine/claimkit/{facts,claims}.go`,
  `metaengine/pebbleengine/{vector,reset}.go`,
  `metaengine/sqliteengine/graph.go`.
- **Verified:** build green for all four modules; `go test -short` green
  (enginetest n/a, claimkit 0.01s, pebbleengine 4.3s, sqliteengine 16.7s).
- **Commit:** absorbed into `eea1c3c66` / `02a133d7e`.

### Fix 3 - same sweep in queue engines (13 sites, 7 files)

- **Files:** `queue/{sqlite,mysql}/{facts,cancel,reads}.go` plus
  `queue/sqlite/enqueue.go`; added the `metaengine/v4` import (already a direct
  go.mod require, so no dependency change).
- **Verified:** `queue/sqlite` build + `go test ./... -short` green
  (545s, slow package). `queue/mysql` build green on re-run.
- **Note:** an initial `queue/mysql` build showed a transient
  `./register.go:11:22: undefined: errors` - re-running was green. That was the
  concurrent session mid-write on `register.go`, not my change.

## Deliverable report

`docs/reviews/2026-09-19_16-22_dogfooding-brutal-self-review.html` (staged).
Built from the `html-report-kit` Bauhaus-dark template, self-contained, 65/65
`<div>` balance.

## Audit findings not yet acted on

| # | Finding                                                                                 | Severity | Status           |
| - | --------------------------------------------------------------------------------------- | -------- | ---------------- |
| 3 | `DeferClose` is mis-tiered: 54 production sites can't reach it without a dep-budget hit | Medium   | Open, owner/ADR  |
| 4 | Duplicated `scanScoredVector` / `scanJSONValues` / `sortAndPaginate*` across engines    | Low      | Open, clean-tree |
| 5 | `queue/postgres` close-idiom sites not swept (same treatment as sqlite/mysql)           | Low      | Open             |

Finding 3 detail: `metaengine.DeferClose` is Tier 3, `storage/sql.CloseRows` is
a Tier-4 sibling. Tier-4 `storage/*` (16 sites) and `queue/*` import either only
with a budget hit or not at all. The primitive is fine; its address is wrong.
Recommendation: expose a Tier-0 `Closer`/`DeferClose` (or re-export from an
existing primitive); growing public API is a v5-window decision.

## Rejected on purpose (do not relitigate)

- **`queue/task` ID minter vs `id/`.** Superficially a ULID clone; it is not.
  The per-millisecond re-rolled seed + monotonic sequence is load-bearing for
  the `id ASC` claim tie-break (a documented conformance-flake fix). Unifying
  would change sort semantics and add deps to a near-zero-dep module.
- **`system.CachedEventStore` as a hand-written wrapper.** Contract #16 warns
  wrappers drop optional capabilities; this one forwards
  `ReadAll`/`ReadFrom`, and `event.DecorateStore` also implements optional
  interfaces unconditionally (`ErrInnerStoreNot*` when unsupported). Consistent.

## Concurrent-session coordination (important)

Confirmed other sessions are active. Evidence: files I never touched are
modified/staged in the same working tree, and commits interleave both sessions'
work.

- Concurrently edited areas observed: `example/goal-shaped-app/` (README.md
  added), `stack/{duckdb,mysql,postgres}/preset.go`, `queue/conformance/*`,
  `queue/mysql/{ddl,engine,open,register}.go`, `queue/sqlite/{engine,register}.go`,
  `storage/pebble/command_store.go`, `.golangci.yml`, `flake.nix`, `go.work`,
  `.agents/skills/go-cqrs-lite/references/{core,recipes}.md`,
  `cmd/doc-check/recipes_catalog_meta2.go`.
- I touched `queue/{sqlite,mysql}/*` while that session was active there.
  Overlap risk is low (my edits were additive), builds are green now, but this
  should be reconciled by whoever owns the queue work.
- The auto-commit daemon absorbed my changes into mixed `chore:` commits -
  authored history for this work does NOT exist as separate commits.

## Remaining work (ranked)

1. **Decide the Tier-0 close helper** (owner/ADR) - unlocks 54 sites.
2. **Sweep `queue/postgres`** close sites (same as sqlite/mysql).
3. **Extract duplicated scan/paginate helpers** into `metaengine` in a
   clean-tree window (the `.art-dupl` baseline dirty-tree rule requires it).
~~4. **Keep `example/goal-shaped-app` compile-gated** as the North Star dogfooding~~
~~   demo (owned by the concurrent session).~~ done 2026-09-19 — recipes §2.39 compile-gated (G-T23)

## Verification commands used

```bash
cd metaengine/irohengine/loopback && GOWORK=off go test ./... -count=1
cd metaengine/{enginetest,claimkit,pebbleengine,sqliteengine} && GOWORK=off go test ./... -count=1 -short
cd queue/sqlite && GOWORK=off go test ./... -count=1 -short
```

Env chain: `GOCACHE=/home/lars/projects/.gocache-disk GOMODCACHE=/tmp/gomod-verify
GOPATH=/tmp/gopath-verify GOTOOLCHAIN=auto GOTMPDIR=/home/lars/projects/.gotmp
TMPDIR=/home/lars/projects/.gotmp` (host go is 1.26.7; workspace needs 1.27.1,
so `GOTOOLCHAIN=auto` is mandatory).

## Honest caveats

- No `nix run .#verify` run was performed (exclusive gate, needs a quiet window;
  load on this machine has been high all day per TODO_LIST).
- No API golden regen needed: the dedup capacity const is unexported
  (`dedupCapacity`), so no exported symbol was added or changed.
- I did not commit anything myself; the auto-commit daemon did.
