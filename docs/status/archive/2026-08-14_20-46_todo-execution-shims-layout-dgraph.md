# Status Report: TODO Execution — Shims, Layout Docs, Dgraph Fix, Stale-GREEN

**Date:** 2026-08-14 20:46
**Scope:** Executed the 🔥 Pareto items from TODO_LIST.md plus the stale-GREEN backlog for `system/`.

## Completed

### Deprecated Module Removal (ADR-0128)

All four shim modules deleted: `codec/`, `retry/`, `flightrecorder/`, parent
`idempotency/` (kvstore + sqlstore kept in place for consumer stability).

The TODO's "zero internal source imports" claims were STALE — the actual
migration performed:

- `decider`, `middleware`, `projectionhost`, `stack` still imported the
  flightrecorder shim → migrated to `github.com/larsartmann/go-flightrecorder`
  (v0.2.0).
- `middleware`, `idempotency/kvstore`, `idempotency/sqlstore` still imported
  the idempotency shim → migrated to `github.com/larsartmann/go-idempotency`
  (v0.1.2). kvstore/sqlstore go.mod tidy + standalone green.
- Registry cleanup: go.work use block, flake.nix `testModules` + `wasmMods`,
  `cmd/api-stability` modules slice (golden regenerated: 4031 exports),
  `scripts/check-module-layers.sh` LAYER/DEP_BUDGET entries, `.golangci.yml`
  path exclusions (5 blocks), cqrs-lint catalog ImportHints + E001 tier-0
  list + affected tests.
- Docs: ADR-0128 written; AGENTS.md module map/tiers/counts updated;
  SKILL.md + references (recipes/modules) point at external repos. doc-check
  passes (797 refs).

Transitive `// indirect` requires of the shim paths REMAIN in ~49 go.mod
files — they resolve through published tags of upstream modules and can only
be tidied after the v4.x re-tag chain. Cosmetic, builds green.

### Dgraph CounterBackend DQL colon bug (🔥, XS)

`metaengine/dgraphengine/counter.go` — `keyVarDecls` emitted `$key0 string`
instead of `$key0: string`; DQL parse error broke `CounterIncrement` for
≤20-key batches. Fixed + `counter_test.go` regression guard.

### cqrs-lint doctor.go split

568 lines → 349 (`doctor.go`) + `doctor_profile.go` (152: suggested-config
rendering + profile merge + 5 mostPermissive helpers) + `doctor_suppressions.go`
(85). All under the 350-line CI limit; suite green.

### Layout planning docs (🔥)

- `METAENGINE-LAYOUT-PLANNING-MODEL.md`: §2 on-disk calibration addendum
  (Normalize wins WriteSpeed/StorageSpace on KV/LSM), corrected the two
  "defaults to embedding" claims (§12 one-sentence, §13 audit).
- `ADR-0124`: calibration-correction addendum with the measured ratio tables
  (KV 0.5/1.0/1.3 vs 1.8/0.48/0.63; LSM 0.74/1.10/1.15 vs 1.45/0.75/0.80).

### layout_observability convergence (S)

`layoutExplainAnnotation` hand-rolled priority resolution with INVERTED
precedence (developer-first) vs the planner's `resolvePriority`
(operator-first) — EXPLAIN could show a different priority than the plan used.
Now calls `resolvePriority` directly.

### Replace-directive audit + stale-GREEN backlog (system/)

- The BLOCKED release-tag item is DONE upstream: id/v4.3.0+v4.4.0 (contains
  `actor_id.go`), record/v4.2.0, command/v4.5.0, metaengine/v4.9.0,
  commandlifecycle/v4.0.0 + projections/v4.0.0 all exist. Removed the now-dead
  `commandlifecycle` ×2 and `id` replaces from system/go.mod; removed the
  `record` replace from metaengine/go.mod. Standalone (GOWORK=off) green.
- NEW replaces added to system/go.mod as temporary workarounds for
  UNPUBLISHED fixes: sqliteengine/badgerengine/pebbleengine/pgengine (v4.0.1
  tags predate driver self-registration → `unknown driver "sqlite"` in
  GOWORK=off builds) and watermill (v4.4.0 returns on first handler error;
  local has the errors.Join handler-independence fix). Remove after tagging:
  engine v4.0.2+ ×4 and watermill/v4.5.0.
- Fixed middleware golden `.json` regeneration (published eventtest v0.3.0
  reads raw .json; local uses go-snaps .snap — both files now exist).

### Concurrent-agent breakage repaired

The auto-commit daemon stashed/popped across in-flight work, leaving conflict
markers in `metaengine/registry.go` and unmerged system files. Resolved
toward HEAD (memory registration now lives in `metaengine/register.go`;
engines_test.go/sqlite_driver.go deletions honored).

## Verify Gate Run + Fixes (same evening, follow-up pass)

Ran `nix run .#verify` end-to-end for the first time after the shim deletion.
Test/race/api-stability/coverage/duplication/depguard/doc-check all green on
the first try. Two gates were red; both fixed:

### check-arch: 94 catalog gaps — root cause was spaced map keys

`scripts/check-module-layers.sh` LAYER/DEP_BUDGET keys used cosmetic spaces
(`LAYER[storage / memory]`, `LAYER[cmd / cqrs - gen]`). The coverage check
looks up literal directory paths, so 47 multi-segment modules (94 = 47×2
LAYER+DEP_BUDGET) were "missing" — and worse, the budget and layer-ordering
loops silently no-oped for ALL of them (`"storage / memory/go.mod"` never
exists). The spaced-key convention had quietly disabled dependency-budget and
layer enforcement for every engine, preset, storage backend, cmd, and example
module.

- All keys normalized to plain paths (`storage/memory`, `cmd/cqrs-gen`).
- `system/integration` was genuinely missing: added LAYER 7 + budget 7.
- Dep-path extraction now strips only TRAILING `/vN` so the nested
  `event/v4/eventtest` module resolves to its real path.
- With enforcement live: 15 apparent violations (L2-L6 modules depending on
  `event/v4/eventtest` L7) — all test-only usage by design. Added
  `TEST_INFRA_MODULES` exemption (eventtest, testutil, pgtestcontainer)
  instead of 15 EXCEPTIONS entries; trimmed the now-redundant testutil
  exceptions from projectionhost.
- Real budget overrun surfaced: cmd/cqrs-bench 19/18 (metaengine dep from the
  layout CLI) — budget bumped to 19 with comment.
- `check-coverage.sh` keys normalized for consistency (it stripped spaces
  itself, so it was never broken).
- AGENTS.md gotcha rewritten: plain-path keys are now the enforced rule.

### lint: shim-migration formatting fallout + SA1019

- gci/gofumpt failures in decider, projectionhost, middleware, stack (16
  files — imports rewritten during the go-flightrecorder/go-idempotency
  migration without reformatting). Fixed via `golangci-lint --fix`; module
  tests re-run green.
- 10× SA1019 `idempotency.NewMemoryStore is deprecated` in middleware tests
  (BDD + bench). The deprecation notice itself names tests as the sanctioned
  consumer. Added a scoped `.golangci.yml` exclusion (middleware `_test.go`,
  text-scoped to MemoryStore) following the tombstone/EnsureCustom precedent;
  removed the 2 now-redundant per-line nolints in test_helpers_test.go.

Result: build/vet/test/race/arch/depguard/duplication/coverage/api-stability/
doc-check + doc assertions all green. Lint is NOT yet green — 15 issues remain
in cmd/cqrs-lint (golines ×1), idempotency/kvstore (gci ×4, gofumpt ×4,
SA1019 NewMemoryStore ×3), idempotency/sqlstore (gci ×3). Same mechanical
class as the four modules already fixed; see the 2026-08-15 follow-up report.

## Not done (deliberately)

- v5 Phase 8 deletions (stack presets, RelationalProjection, transport/*)
  — transport removal needs final v4.x patch tags first (user approval).
- DuckDB/Row layout calibration benches (M each, benchmark runs).
- go-codec repo scaffolding (external repo work, not this repo).

## Resolution (2026-08-15, docs-health pass)

No numbered next-list in this report; its claims were re-verified: the shim
deletion landed at `5127039da`, the check-arch spaced-key root cause and fix
are real (87 plain-key LAYER entries, green in `#verify`), and the 15
remaining lint issues closed at `444be10a7` (first fully-green verify:
`5f2198189`). The three deliberate non-goals stay routed: v5 Phase 8
deletions -> TODO_LIST "v5 Unification Phase 8"; DuckDB/Row calibration ->
TODO_LIST "Metaengine"; go-codec scaffolding -> sibling repo. The 5 temporary
system/go.mod replaces still await the engine v4.0.2+ / watermill v4.5.0
tags (TODO_LIST "Release / Tagging", ROADMAP Open Questions #1). Archived.
