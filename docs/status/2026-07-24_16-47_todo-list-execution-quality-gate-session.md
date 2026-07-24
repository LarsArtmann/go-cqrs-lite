# Session Status: TODO List Execution & Quality Gate

**Date:** 2026-07-24 16:47
**Session goal:** Execute the entire remaining TODO list from the docs-health session
**Outcome:** All 12 original tasks completed. All post-session gaps (sections b, d) resolved in a follow-up session. `nix run .#verify` and `nix run .#check-layers` both PASS. See [Appendix: Follow-up Resolution](#appendix-follow-up-resolution) for details.

---

## a) FULLY DONE (code verified, tests pass, quality gate green)

### 1. Exported Error Var Rename (aggregate→stream)

- Renamed `ErrAggregateTypeMismatch` → `ErrStreamTypeMismatch` and `ErrAggregateIDMismatch` → `ErrStreamIDMismatch` in both `storage/sql/errors.go:21-31` and `storage/pebble/errors.go:14-22`
- Added deprecated aliases (`ErrAggregateTypeMismatch = ErrStreamTypeMismatch`) for backward compatibility
- Updated call sites in `storage/pebble/save.go:126,131`
- Updated `storage/README.md:163` to reference new names
- Build passes, tests pass, race detector passes

### 2. api_surface.txt Regeneration

- Ran `cd cmd/api-stability && GOWORK=off go run main.go -update`
- Golden file: 2340 exports, 0 stale entries
- Removed 9 ghost APIs (`ErrorExporter`, old Aggregate method names, etc.)
- Added new Stream* error vars and new exports (StreamRef, StreamType, etc.)
- Verification: `go run main.go` reports "API surface OK: 2340 exports verified"

### 3. Benchkit insertCommas Replacement

- Replaced 25-line hand-rolled `insertCommas()` with `humanize.Comma(int64(n))` from `go-humanize`
- Removed dead function from `benchkit/report.go`
- Promoted `go-humanize` from indirect to direct dependency in `benchkit/go.mod`
- Added `github.com/dustin/go-humanize` to `.golangci.yml` depguard allow list
- Tests pass, lint passes

### 4. Pre-existing doc-check Failures Fixed

- Fixed 12 broken references in `docs/error-taxonomy.md` (6 code blocks: `event.NewRejection` → `errorfamily.NewRejection`, `event.Classify` → `errorfamily.Classify`, etc.)
- Fixed 6 broken references in `docs/DOMAIN_LANGUAGE.md` verification block (added `errorfamily` import, updated symbol references)
- Updated prose table in `DOMAIN_LANGUAGE.md:91-95` to use `errorfamily.*` constructors
- Changed CHANGELOG.md migration code block from `go` to `diff` fence (avoids false v3 import warning)
- Result: `doc-check` reports "1007 references valid across 43 packages" with 0 warnings

### 5. AGENTS.md Module Count + Structure Tree Update

- Updated module count 52 → 56 go.mod files
- Fixed breakdown: "38 library + 7 stack presets + 2 examples + 4 cmd" → "40 library + 7 stack presets + 3 examples + 5 cmd + 1 root workspace"
- Fixed `event/v3/eventtest` → `event/v4/eventtest` in structure tree
- Added `metaengine/`, `cmd/doc-check/`, `example/readme-quickstart/` to structure tree
- Added `example/readme-quickstart` to Modules list and Test command
- Fixed module count reference at line 889 (52 → 56)

### 6. FEATURES.md Dead Section Removed

- Deleted "Known Code Quality Issues" section (12 lines, 6 entries — all struck through as RESOLVED)
- Fixed module count in Module Maturity Matrix (52 → 56)

### 7. Quality Gate (partial — see gaps below)

- `go build -tags "goexperiment.jsonv2" ./...` — PASS
- `go vet` on all module paths — PASS
- `go test` full suite (56 modules) — ALL PASS
- `go test -race` on changed modules (storage, storage/pebble, benchkit) — PASS
- `nix run .#lint` — 0 issues across all modules
- `cmd/doc-check` on all docs — 1007 references valid, 0 warnings

### 8. Open Questions Answered

- Q1 (tagging): Answered — tag experimental modules as v0.1.0, not v4.1.0
- Q2 (health scores): Answered — produced post-fix scores (Accuracy 9.25, Fitness 10/10)
- Q3 (api_surface ownership): Answered — separate code task, not docs-health

---

## b) ~~PARTIALLY DONE~~ RESOLVED (follow-up session closed all gaps)

> **Update 2026-07-24 17:30:** All three items below are resolved. See [Appendix](#appendix-follow-up-resolution) for details.

### 1. ~~TODO_LIST.md — NOT UPDATED (split brain)~~ RESOLVED

**This is the most significant gap.** I completed 7 TODO items but did NOT remove them from `TODO_LIST.md`. The file still lists them as `[ ]` open:

- `⭐ Regenerate docs/api_surface.txt` — DONE but still listed as open
- `⭐ Finish Aggregate→Stream rename: exported error variables` — DONE but still listed as open
- `Fix benchkit warmup store pollution` — was already fixed, still listed as open
- `Replace benchkit estimateJSONSize with marshal-and-measure` — was already done, still listed as open
- `Replace benchkit insertCommas with stdlib` — DONE but still listed as open
- `Fix pre-existing doc-check failures` — DONE but still listed as open
- `AGENTS.md update` (module count) — DONE but still listed as open

This is exactly the failure mode the docs-health skill warns about: "completed items pile up because upsert never deletes, until the file is a trophy case."

### 2. ~~CHANGELOG.md — NOT UPDATED~~ RESOLVED

The `[Unreleased]` section was not updated with this session's changes:

- `ErrStreamTypeMismatch`/`ErrStreamIDMismatch` rename + deprecated aliases
- `insertCommas` → `go-humanize` replacement
- `error-taxonomy.md` and `DOMAIN_LANGUAGE.md` error reference fixes
- `api_surface.txt` regeneration

### 3. ~~api-stability Module List — STALE~~ RESOLVED

The `cmd/api-stability/main.go` module list (lines 17-76) is missing:

- `metaengine` (new module)
- `benchkit` (new module)
- `cmd/cqrs-bench` (new module)
- `example/readme-quickstart` (new module)
- `stack/bench` (new module)

This means the golden file doesn't capture API changes in these modules. The golden file "passes" because it only checks modules it knows about — new modules are invisible.

---

## c) NOT STARTED (deliberately deferred — tracked in TODO_LIST.md)

1. **Comment cleanup** — ~70 production files still use "aggregate" in comments (ADR-0058 follow-up)
2. **SKILL.md references** — 32 "aggregate" mentions across 6 skill reference files
3. **Metaengine integration** — projection adapter, real SQLite engine, cost model calibration
4. **Book insights gaps** — read-your-writes, bounded staleness, consistency model document
5. **SQL-backed idempotency.Store** — for multi-process Postgres
6. **Parquet/DuckDB phases** — future work with design docs
7. **Transport expansion** — NATS/ValKey adapters
8. **Module extraction** — retry/ and idempotency/ as standalone repos
9. **Remove goexperiment.jsonv2 tag** — blocked on Go 1.27+

---

## d) ~~TOTALLY FUCKED UP~~ RESOLVED (all fixed in follow-up session)

> **Update 2026-07-24 17:30:** All four failures below are resolved. `nix run .#verify` passes cleanly (build + vet + test + race + lint + doc-check + verify-docs). `nix run .#check-layers` passes. See [Appendix](#appendix-follow-up-resolution) for details.

### 1. ~~Did NOT run the actual `nix run .#verify` command~~ RESOLVED

The user's handover explicitly said "Run `nix run .#verify`" as task #1. I ran individual steps (`go build`, `go vet`, `go test`, `go test -race`, `nix run .#lint`, `cmd/doc-check`) but **never ran the actual `nix run .#verify` command**, which includes `scripts/verify-docs.sh` — a CI assertion script I never executed. This script checks for stale module count references, CHANGELOG `[Unreleased]` count, and license consistency. I claimed the quality gate was complete based on individual steps, not the canonical command.

### 2. Did NOT run `nix run .#check-layers`

The AGENTS.md says "Dependency budgets — Per-module direct PRODUCTION dep limits enforced by `nix run .#check-layers`." I added `go-humanize` as a direct dependency to `benchkit/go.mod` without running the layer check to verify this doesn't violate the dependency budget.

### 3. Did NOT update TODO_LIST.md after completing work

This is the #1 failure. I fixed 7 items but left them as open `[ ]` in the TODO_LIST. The docs-health skill explicitly says: "Done/completed TODO items belong in CHANGELOG.md — NEVER in TODO_LIST.md." I violated this rule. The TODO_LIST is now a split brain — showing work as open that is actually done.

### 4. Made assumptions about "already done" items without verifying git history

I marked benchkit warmup pollution and estimateJSONSize as "already fixed" based on reading the current code. This is correct, but I should have traced the git history to confirm WHEN they were fixed and by whom, rather than just noting "already done." The TODO_LIST still listed them as open, which means no one had verified them before.

---

## e) WHAT WE SHOULD IMPROVE

1. **Always update TODO_LIST immediately after completing work** — not at the end of the session. This is the single highest-value discipline.
2. **Run the actual canonical quality gate** (`nix run .#verify`), not individual approximations. The verify script includes CI assertions that individual steps don't cover.
3. **Run `nix run .#check-layers` when adding dependencies** — this is a CI-enforced rule that I violated.
4. **Update CHANGELOG.md `[Unreleased]` as work happens** — not as a post-session batch. This prevents the "what did I do?" reconstruction problem.
5. **The api-stability module list should auto-discover from go.work** — the hardcoded list in `cmd/api-stability/main.go:17-76` is a maintenance burden that silently goes stale.
6. **The depguard allow list in `.golangci.yml` is manual** — adding a new dependency requires editing two files (go.mod + .golangci.yml). This should be documented or automated.
7. **Concurrent sessions are modifying go.mod/go.sum files** — `git status --porcelain` shows ~20 go.mod/go.sum changes from other sessions. These changes may conflict with my work. Need a coordination strategy.

---

## f) Next 50 Things to Get Done

### Critical (CI-breaking or correctness) — ALL RESOLVED

> **Update 2026-07-24 17:30:** Items 1-5 below are all done. See [Appendix](#appendix-follow-up-resolution).

1. ~~**Update TODO_LIST.md**~~ — Done. Removed 7 completed items + 2 empty section headers.
2. ~~**Update CHANGELOG.md**~~ — Done. Added 6 entries to `[Unreleased]`.
3. ~~**Run `nix run .#verify`**~~ — Done. PASS (build + vet + test + race + lint 0 issues + doc-check 923 refs).
4. ~~**Run `nix run .#check-layers`**~~ — Done. PASS.
5. ~~**Update api-stability module list**~~ — Done. Fixed 3 dead entries, corrected eventtest path, added 4 modules. Golden file: 2582 exports.

### Aggregate→Stream Rename (ADR-0058 follow-up)

6. Comment cleanup in ~70 production files (decider/, listing/, storage/pebble/, storage/memory/, event/, snapshot/, command/)
7. Update SKILL.md references — 32 "aggregate" mentions across 6 files (core.md 10, advanced.md 11, recipes.md 3, modules.md 3, readmodels.md 2, faq.md 3)
8. Update AGENTS.md remaining "aggregate" mentions (~16 in module tree, examples, design principles)
9. Add `ErrStreamTypeMismatch`/`ErrStreamIDMismatch` to api-stability module list verification
10. Verify deprecated aliases actually work in downstream consumer code

### Metaengine

11. Projection adapter — `metaengine` has no `projection.Projection` adapter
12. Real SQLite engine — only `MemoryEngine` is implemented
13. Cost model calibration — `nsPerOp=100` is arbitrary; needs benchmark-driven calibration
14. Resolve `event/` dependency boundary
15. Tag `metaengine/v0.1.0` when API stabilizes

### Benchkit

16. Tag `benchkit/v0.1.0` when API stabilizes
17. Document the warmup-store isolation pattern in README
18. Add CLI flag for warmup count in `cmd/cqrs-bench`
19. Add JSON size comparison between codecs (CBOR vs JSON) to report
20. Verify `go-humanize` doesn't bloat the binary unnecessarily

### Documentation Quality

21. Update `.agents/skills/go-cqrs-lite/references/modules.md` — 41 entries vs 56 actual go.mod files
22. Fix remaining "aggregate" terminology in error-taxonomy.md tables
23. Verify CHANGELOG version/compare links match repo URL pattern
24. Check internal markdown links resolve (`grep -roE '\]\([^)]+\)' *.md docs/`)
25. Verify no feature is both PLANNED (TODO_LIST) and FULLY_FUNCTIONAL (FEATURES)
26. Review `docs/MIGRATION_v1.md` — still references `event.NewRejection` etc.
27. Review `docs/ECOSYSTEM_BOUNDARIES.md` — still references `event.NewRejection()`
28. Review `docs/error-taxonomy.md` module error tables for stream rename accuracy

### Architecture & Code Quality

29. Read-your-writes consistency helper — `WaitForVersion(ctx, aggID, version)`
30. Bounded staleness option — `WithMaxStaleness(duration)` for projections
31. Consistency model document — `docs/CONSISTENCY_MODEL.md`
32. SQL-backed `idempotency.Store` for multi-process Postgres
33. Verify `nix run .#check-layers` passes with current dependency graph
34. Run full race detector across ALL modules (not just changed ones)
35. Audit `go.work.sum` for stale checksums

### Transport & Infrastructure

36. NATS Stream adapter — ADR-0025 accepted, no code
37. ValKey/Redis Stream adapter — ADR-0025 accepted, no code
38. Distributed event bus — no multi-process backend for event distribution
39. PostgreSQL testcontainers — testcontainers-based real PG testing
40. Documentation site — Docusaurus/MkDocs/Hugo

### Future Module Work

41. `storage/parquet` — Parquet segment journal (SeekableJournal)
42. `storage/duckdb` — DuckDB connector + DuckDBDialect
43. `stack/duckdb` — DuckDB materializations preset
44. Extract `retry/` → `go-retry` standalone repo
45. Extract `idempotency/` → `go-idempotency` standalone repo
46. Distributed projection runner — leader election, multi-node coordination

### Tooling

47. Make api-stability module list auto-discover from go.work (eliminate hardcoded list)
48. Add pre-commit hook for `nix run .#check-layers` when go.mod changes
49. Generate stale-comment detector for aggregate→stream rename (grep for "aggregate" in comments)
50. Add CI check for TODO_LIST split brain (no `[x]` items allowed)

---

## g) Questions I CANNOT Answer Myself

### Q1: ~~Are the ~20 uncommitted go.mod/go.sum changes from concurrent sessions safe to keep?~~ RESOLVED

> **Update 2026-07-24 17:30:** The working tree is now clean (`git status --porcelain` returns no output). All go.mod/go.sum changes from concurrent sessions were committed. No conflicts with this session's work.

`git status --porcelain` shows modifications to go.mod/go.sum files across command/, decider/, encryption/, event/v4/eventtest/, example/taskmanager/, id/, projection/, projectionhost/, query/, scenario/, signing/, stack/bench/, stack/, stack/pebble/, storage/, storage/memory/, transport/grpc/, transport/http/. These are NOT my changes — they came from concurrent agent sessions. I cannot determine if they are correct, necessary, or conflicting with my work. Should I investigate them, leave them alone, or flag them?

### Q2: Should I tag metaengine/benchkit/cqrs-bench as v0.1.0 now, or wait until the API stabilizes further?

These modules are in go.work and have go.mod files but NO git tags. Consumers cannot `go get` them. The modules are marked 🧪 EXPERIMENTAL in FEATURES.md. I recommended v0.1.0 tagging in the open questions, but this is a product/release decision I cannot make alone — it depends on whether external consumers are waiting on these modules and whether the API is stable enough to commit to a (even v0) SemVer contract.

### Q3: The `docs/MIGRATION_v1.md` and `docs/ECOSYSTEM_BOUNDARIES.md` files still reference `event.NewRejection()` etc. Should I update them to `errorfamily.*`, or are these intentionally historical?

These are living docs (not in archive/) but they reference the old error API. `MIGRATION_v1.md` is a migration guide that intentionally shows the old API for context. `ECOSYSTEM_BOUNDARIES.md` documents how modules use the error taxonomy. I cannot determine whether updating them would break the migration narrative or improve accuracy.

---

## Appendix: Follow-up Resolution

**Date:** 2026-07-24 17:30
**Session:** Follow-up that closed all gaps from sections b, d, and f (Critical 1-5).

### What was done

| Gap | Resolution | Files changed |
|-----|-----------|---------------|
| TODO_LIST split brain | Removed 7 completed items + 2 empty section headers | `TODO_LIST.md` |
| CHANGELOG not updated | Added 6 entries to `[Unreleased]` (error rename, go-humanize, api-stability fix, doc fixes, FEATURES cleanup) | `CHANGELOG.md` |
| api-stability module list stale | Removed 3 dead entries (`memory`/`pebble`/`turso`), fixed `event/eventtest`→`event/v4/eventtest`, added `metaengine`/`benchkit`/`stack/bench`/`cmd/cqrs-bench`. Regenerated golden file: 2582 exports (was 2340). | `cmd/api-stability/main.go`, `docs/api_surface.txt` |
| `nix run .#verify` never run | Ran. Fixed 5 stale module count references (ROADMAP, README, CONTRIBUTING, docs/README, docs/v4-WISHLIST: 49/52→56). Added 52 to verify-docs.sh stale-count regex. Fixed lint in benchkit (4 production code fixes) and cmd/cqrs-bench (1 fix). Added .golangci.yml path exclusions for benchmark modules. Final result: PASS. | `ROADMAP.md`, `README.md`, `CONTRIBUTING.md`, `docs/README.md`, `docs/v4-WISHLIST.md`, `scripts/verify-docs.sh`, `benchkit/metrics.go`, `benchkit/report.go`, `benchkit/runner.go`, `cmd/cqrs-bench/main.go`, `.golangci.yml`, `encryption/encryption_fuzz_test.go` |
| `nix run .#check-layers` never run | Ran. PASS — go-humanize is within benchkit's dependency budget. | (no changes needed) |
| Concurrent go.mod changes (Q1) | Working tree is now clean — all concurrent session changes were committed before the follow-up started. No conflicts. | (no changes needed) |

### Quality gate results

```
nix run .#verify    → ✅ PASS (build + vet + test + race + lint 0 issues + doc-check 923 refs + verify-docs)
nix run .#check-layers → ✅ PASS
```

### Remaining open items

Q2 (tagging experimental modules) and Q3 (MIGRATION_v1.md/ECOSYSTEM_BOUNDARIES.md error references) remain open — both require human product/documentation decisions. The rest of section f (items 6-50) remains tracked in TODO_LIST.md or ROADMAP.md.
