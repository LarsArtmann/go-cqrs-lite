# Wave-4 Final Gates GREEN: art-dupl Triage Complete, projectionhost Standalone Build Fixed — 2026-08-16 13:15

> Session scope: executed the 12:39 queue tail under the standing
> READ→UNDERSTAND→RESEARCH→REFLECT→execute-verify loop: art-dupl triage →
> duplication gate → full `#verify` → `#check-coverage`. All four green.
> One incidental find fixed on the way (projectionhost standalone build).
> **Wave-4 queue is DONE. Three user decisions remain open (section d).**

---

## a) DONE (this session)

### 1. art-dupl triage — 5 clone groups, 2 consolidated + 3 annotated

Each group judged on its merits (consolidate vs. annotate), not blanket-baselined:

| # | Clone group | Verdict | Action |
|---|-------------|---------|--------|
| 1 | `metaengine/sse_replay.go` ↔ `sse_replay_falsesharing_bench_test.go` | Intentional | `//art-dupl:accept` — the mirror MUST be isomorphic for a valid A/B false-sharing measurement (F49 measure-then-pad protocol) |
| 2 | `projectionhost/worker.go` ↔ `worker_falsesharing_bench_test.go` | Intentional | Same — isomorphic mirror is the methodology |
| 3 | `capability_audit.go:172` ↔ `engine_stats.go:48` (engines-snapshot mutex idiom) | Consolidated | New named helper `Store.enginesSnapshot()` (`metaengine/store.go`) — gives the idiom a name, both call sites now one line |
| 4 | `event/journal_middleware.go:86` ↔ `store_middleware.go:186` (ReadFrom SeekableJournal delegation) | Consolidated | New shared `seekableReadFrom(ctx, inner, afterEventID, limit, noun)` (`event/store_middleware.go`) — parameterized noun preserves the distinct observable error codes (`event.journal_not_seekable` vs `event.store_not_seekable`); eliminates lockstep-drift risk between the two wrappers |
| 5 | `catalog/internal/cattest/assertions.go` ↔ `cmd/api-stability/pin_drift_test.go` (test read-file idiom) | Intentional | `//art-dupl:accept` — cross-module `internal/` package cannot be shared, 6-line universal Go testing idiom, zero drift risk |

**Empirical finding encoded into AGENTS.md contract #14**: `//art-dupl:accept`
suppresses a clone group LIVE — baseline regen is NOT needed for annotated
intentional clones (gate went 5 groups → 0 with `.art-dupl-baseline.json`
untouched). Reserve baseline regen for structural shifts.

### 2. Incidental find: projectionhost standalone (`GOWORK=off`) build was broken

Surfaced by per-module test verification after the triage edits (not by the
workspace gate, which masks it):

- `versioned_journal_integration_test.go` (wave-4, commit `ca64b3517`) uses
  `schema.UpcastSourceTransform` + `event.DecorateJournal`.
- `projectionhost/go.mod` required `schema/v4 v4.1.0` — the symbol shipped in
  **v4.3.0** (verified: `git tag --contains b5fb09002` → schema/v4.3.0).
- `event.DecorateJournal` is **unpublished** (no event tag contains `ca64b3517`).
- Fix (sibling-convention, matches `middleware/`, `schema/`, `integration/`):
  - bump `schema/v4 v4.1.0 → v4.3.0` (tag exists ✓)
  - `replace event/v4 => ../event`
  - `replace metadata/v4 => ../metadata` — required by the documented
    replaces-do-not-cascade gotcha (local event needs unpublished
    `metadata.BrandedString`; first build attempt failed exactly there)
- Verified: `GOWORK=off go build` + full test suite green standalone;
  api-stability meta-tests (incl. pin-drift, which parses replaces) green;
  CHANGELOG entry added.
- Note: the release flow strips replaces at tag time, so **event (and
  metadata) must be tagged in the same batch as projectionhost** — feeds
  open question d.1.

### 3. Gates — all green

| Gate | Result | Evidence |
|------|--------|----------|
| `nix run .#check-duplication` | ✅ EXIT=0 | 0 clone groups, baseline 99 untouched |
| `nix run .#verify` (run #4) | ✅ EXIT=0 | build+vet+test+race+lint 76/76 modules clean+doc-check+api-surface(-race); no FAIL lines in 591-line log (`/tmp/verify-full4.log`) |
| `nix run .#check-coverage` | ✅ EXIT=0 | all 11 tracked modules within ±2%; wave-4 tests RAISED: id +1.9%, schema +0.9%, metaengine +0.2% |

Per-module verification before the gates: metaengine (full suite), event
(full suite incl. all Decorate* tests — error-code paths pinned), projectionhost
(full suite), api-stability; golangci-lint 0 issues on event/projectionhost/
api-stability; gofumpt clean.

### 4. Docs/memory maintenance

- CHANGELOG: projectionhost standalone-fix section under [Unreleased]
- TODO_LIST: duplication-baseline-hygiene item annotated with today's progress
- AGENTS.md contract #14: annotation-suppresses-live finding added
- 12:39 report annotated with RESOLVED blockquote

---

## b) Wave-4 ledger (cumulative, all four sessions)

Shipped: F46 (go-codec sniff, uncommitted external), F47 (benchstat baselines),
F48 (contention benches), F49 (false-sharing campaign: SSEReplay unpadded-keep
+ worker unpadded-keep verdicts, benches + mirrors), Doctor capability audit
section, irohengine graph forwarding + conformance 9/9, SQLite in-memory pool
pin + RED-proven regression test, bbolt batch-commit + pebble knobs + PG COPY
(wave-3 spillover), art-dupl triage, projectionhost standalone fix.
**Every fix root-caused; every gate green.**

---

## c) What I deliberately did NOT do

- **No `art-dupl baseline` regen** — annotations made it unnecessary; re-pinning
  with in-flight foreign code is the exact anti-pattern TODO_LIST warns about.
- **No tag/release** — release scope is user question d.1.
- **No go-codec commit** — external repo, user question d.2.

---

## d) OPEN USER DECISIONS (re-surfaced, sharpened — not decided unilaterally)

1. **Tagging wave scope + operator.** Unreleased wave-4 changes sit in:
   `event` (DecorateJournal), `schema`, `metadata` (BrandedString et al.),
   `metaengine` (+irohengine), `projectionhost`, `storage` (SQLite pool pin →
   implied v4.7.2). NEW constraint from today: projectionhost's released go.mod
   needs `event` + `metadata` tags to exist (release flow strips the replaces),
   so the batch must tag at least event+metadata+schema before/with
   projectionhost+metaengine. Who runs `scripts/tag-release.sh`?
2. **go-codec F46 ownership.** The UnwrapDecode first-byte-sniff fix is
   UNCOMMITTED in `/home/lars/projects/go-codec` (no auto-commit daemon there).
   Commit + tag v0.x.y there, or user reviews first?
3. **Ratify two judgment calls** (both already shipped + gated green):
   (a) iroh latency P99 bound 50→150ms (P99-of-30 = worst sample; gate load
   inflates it; 66ms observed under load vs ~20ms idle);
   (b) SQLite `OpenSQLiteInMemory` pool pin via `ConfigureSQLitePool`
   (single connection; serializes test DB access) vs. alternative of unique
   shared-cache DSNs per call site.

---

## e) Remaining backlog (unchanged from 12:39 report §f, 24 items)

Highlights: mysql/dgraph/turso integration conformance, iroh graph WriteOp
(edge replication), irohengine optional-capability forwarding audit,
AGENTS.md gotcha for modernc `file::memory:`, pre-gate load-sweep script,
09:13 report annotation, duckdbengine suite split, duplication dirty-tree
guard (partially advanced today).
