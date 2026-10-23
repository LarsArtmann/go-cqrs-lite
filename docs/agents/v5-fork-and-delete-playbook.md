# v5 Fork & Delete-at-Zero Playbook

> Canonical procedure doc for ADR-0152 dual-support exit. Written 2026-10-23 (plan task 1, F14–F17),
> absorbing the 2026-10-17 first-application lessons
> ([report](../status/2026-10-17_08-56_delete-at-zero-first-application-and-self-review.md)).
>
> **Owner ruling (2026-10-17):** phase out v4 submodule-by-submodule; delete the moment zero
> consumers remain; full v5 cut at the end; fleet window 1–2 weeks. Standing delete-at-zero rule
> is GO'ed; the Phase A fork sprint needs its own explicit GO.
>
> Two procedures live here: **FORK** (copy a v4 package into `core/v5`, snapshot semantics) and
> **DELETE** (remove a v4 module at zero consumers). They share the gate checklist (§4).

## 1. FORK procedure — v4 package → core/v5 package

Applies to the Big-23 (plan tasks 4–24). `core/v5` is ONE module; forking adds a package, not a go.mod.

| F# | Step (≤12min) | Verify |
|----|---------------|--------|
| F1 | Inventory: file list, LOC, deps, test-couplings of the v4 source package | list written |
| F2 | Exclusion decision: drop deprecated-at-v5 surfaces (§2); record verdict line | doc note |
| F3 | Copy production files to `core/v5/<pkg>` | file count matches |
| F4 | Copy test files | ditto |
| F5 | Import rewrite: `go-cqrs-lite/<pkg>/v4` → `core/v5/<pkg>` (script, per-file scope) | rg zero old paths |
| F6 | `core/v5` go.mod dep additions + `GOWORK=off go mod tidy` | tidy clean |
| F7 | Compile cycle 1: fix residual references | build green |
| F8 | Compile cycle 2: test compile + fix | vet green |
| F9 | Registration — see the policy note below | per policy |
| F10 | Regen api golden (`cd cmd/api-stability && GOWORK=off go run . --update`) | diff = expected only |
| F11 | Module tests (`GOWORK=off go test -count=1 ./<pkg>/...` inside core/v5) | suite green |
| F12 | Arch-lint DAG + dep-budget rows + file-size ratchet (trailing `//art-dupl:accept` twin directive for baselined >350-line copies) | gates green |
| F13 | CHANGELOG row + 1-line status note | changelog gate green |

### F9 registration policy (twin policy, derived 2026-10-17 — owner confirm pending)

- Fork = **point-in-time snapshot**; the v4 side goes **fix-only** afterward (feature dev lands in
  core/v5 only). One-way drift is EXPECTED for the Big-23.
- The 9 existing primitive pairs (record, id, dedup, kv, dispatcher, metadata, event, command,
  query) keep mirror-sync discipline and stay in `mirrorPairs`.
- Big-23 forks stay **UNREGISTERED** in `mirrorPairs`. ⚠ KNOWN GATE FIGHT:
  `TestMirrorLockstep` (cmd/api-stability/mirror_lockstep_test.go) currently *forces* every new
  core/v5 package in the golden to be registered. The gate scoping (plan task 25 / 47) MUST land
  before or with the first Big-23 fork, or the gate and this policy contradict each other.
- Do NOT re-pin `.art-dupl-baseline.json` wholesale to absorb fork copies; use the trailing
  `//art-dupl:accept` twin pattern on the copy's first line.

## 2. Exclusion list — deprecated surfaces NOT copied forward

| Surface | Home of the deprecation | Fork that must drop it |
|---|---|---|
| `aggregate_projection` | storage core deprecated tier | storage core (task 4) |
| `view_aliases.go` + `relational_aliases.go` re-exports | storage core alias strips (the submodules die at the storage fork) | storage core (task 4) |
| `schema.VersionedStore` / `VersionedSeekableJournal` shells | ADR-0126 | schema (task 19) |
| `signing.Rejecting*`, `encryption.ErrInnerStoreNot*` | ADR-0126 | encryption+signing (task 23) |
| tombstone metadata API (`event.DetectTombstone`/`MarkTombstone`, `TombstoneMark`) | ADR-0114 | event-adjacent forks |
| fold DSL v1 | v5 evolution docs | metaengine core (tasks 11–12) |
| Deprecated pair-form `*Ref` methods | ADR-0111 T04 | decider fork (task 9) |

Rule: every exclusion produces a verdict line in the fork's status note (F2) — "dropped X because Y".

## 3. DELETE procedure — delete-at-zero (standing rule)

**Trigger:** a v4 module has 0 fleet requires AND 0 in-repo couplings. Run the census
(`scripts/census-fleet.sh`, §5) after every fleet migration batch (plan task 38b).

**ORDERING IS LOAD-BEARING under the auto-commit daemon** (2026-10-17 lesson: deleting first
produced three mid-flight commits with red gates). Correct order:

1. **Zero-check part 1 — fleet requires:** `scripts/census-fleet.sh` shows the module at 0 direct
   and 0 indirect consumers. Re-run if the last capture is >24h old (fleet automation moves pins
   daily).
2. **Zero-check part 2 — in-repo couplings:** `rg "go-cqrs-lite/<mod>" --type go -l` plus manual
   review for (a) alias re-exports in sibling modules (the `storage/view_aliases.go` class),
   (b) bench/test importers (`benchkit`, `cmd/cqrs-bench`, `stack/bench`, integration suites),
   (c) doc/recipe fences that would stop compiling. A requires-only census missed 9 of 11
   "free" deletes on 2026-10-17 — the rg is NOT optional.
3. **Rewire in-repo consumers** (tests, fixtures, catalogs) onto surviving modules.
4. **De-register everywhere** (checklist in §4, delete column).
5. **Derive counts by measurement** — `find . -name go.mod -not -path './vendor/*' | wc -l` —
   before writing any module count into AGENTS.md/module-map. The canonical-facts gate regex only
   catches un-backticked `N go.mod` forms; backticked prose counts are invisible to the gate, so
   hand-verify those. Never arithmetic-guess from a previous number (the old number may itself be
   drifted).
6. **Regen goldens:** `cd cmd/api-stability && GOWORK=off go run . --update`; eyeball the diff —
   removals must be exactly the deleted module's packages, additions must be attributable.
7. **Gate-check** (cheap set, §4) — all green BEFORE the tree loses the module.
8. **`git rm -r <mod>` LAST.** Then: root `go build ./...`, changed-module tests, census re-confirm
   (still 0), CHANGELOG `[Unreleased]/Removed` row, FEATURES/README/module-map/AGENTS honesty pass,
   docs/status/README.md index row for the session report.

## 4. Gate checklist

| # | Gate / registration point | Fork | Delete | Command |
|---|---------------------------|------|--------|---------|
| 1 | go.work use-lines | — | edit | (manual) |
| 2 | flake.nix `testModules` (+ mkApp blocks + CI lines if any) | — | edit | (manual) |
| 3 | cmd/api-stability `modules` slice | — | edit | (manual) |
| 4 | api golden regen + diff eyeball | ✔ | ✔ | `cd cmd/api-stability && GOWORK=off go run . --update` |
| 5 | api-stability meta tests (`TestEvery`, mirror lockstep) | ✔ | ✔ | `cd cmd/api-stability && GOWORK=off go test -run TestEvery .` |
| 6 | scripts/check-module-layers.sh (LAYER + DEP_BUDGET rows) | ✔ | edit | `bash scripts/check-module-layers.sh` |
| 7 | cqrs-lint module catalog + module_catalog_test maps | — | edit | `cd cmd/cqrs-lint && GOWORK=off go test ./...` |
| 8 | doc-check recipes catalog (only if recipes.md touched) | ✔ | ✔ | `cd cmd/doc-check && GOWORK=off go test -run TestRecipes .` |
| 9 | doc-check reference scan | ✔ | ✔ | `cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md` |
| 10 | integration suite (if consumer) | — | ✔ | `cd integration && GOWORK=off go test -count=1 ./...` |
| 11 | canonical-facts + count honesty (AGENTS, module-map) | — | ✔ | `bash scripts/check-canonical-facts.sh` |
| 12 | changelog-symbols | ✔ | ✔ | `bash scripts/check-changelog-symbols.sh` |
| 13 | root workspace build | ✔ | ✔ | `go build ./...` |
| 14 | module tests (forked pkg / changed set) | ✔ | ✔ | `GOWORK=off go test -count=1 ./...` per module |
| 15 | check-md-go (any docs/ fence edit) | ✔ | ✔ | `nix run .#check-md-go` |
| 16 | census re-confirm | — | ✔ | `scripts/census-fleet.sh` |

Env discipline: `source scripts/go-env.sh` before any go command; `GOWORK=off` for per-module legs.
Exit-code discipline: never pipe test output through filters that eat the exit code — capture
`go test`'s status explicitly (`go test ...; echo $?` or no pipe).

## 5. Tools

- **`scripts/census-fleet.sh`** (this repo) — fleet census → timestamped JSON artifact in
  `docs/evidence/`, `--zero-consumers` delete-trigger report, `--self-test`.
- **`depgraph who-uses`** (~/projects/project-dependency-graph) — cross-check; semantics are
  go.mod REQUIRES (direct by default), not imports; NO `--format json` support (plain tree only).

## 6. Guardrails (from the plan — DO NOT VERSCHLIMMBESSERN)

1. v4 trains are frozen (ADR-0152): fixes cherry-pick v4→v5 only; never "improve" v4 while copying.
2. Golden regen + registration edits happen in the SAME session-step as each fork/delete.
3. Never wholesale re-pin `.art-dupl-baseline.json`; surgical single-entry insert only.
4. New copies of baselined >350-line files use the trailing `//art-dupl:accept` twin pattern.
5. Never touch `real-setup-v4.8.1-poisoned`; never re-enable BuildFlow's go-version-auto-configure.
6. Tags ship only after per-module REAL test runs — never `verify=ok`-only.
7. `#verify` runs exclusively; on chronically loud hosts verify constituents individually instead
   of burning an hour on the load gate (2026-10-23 lesson).
8. Per-repo fleet migrations: baseline tests first; go.work guard on `go get/tidy`.
9. Census numbers get re-timestamped before use.
10. Deployment-sensitive repos get a production-data compat gate on persistence-touching bumps.
11. Delete-at-zero requires the coupling rg (§3.2) — always.
12. Fork = snapshot; v4 side fix-only; no feature mirroring back into v4.
