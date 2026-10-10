# Status: v5 Schema-Evolution Execution Plan — Phase 0 Complete, M7 In Flight (2026-10-10 08:45)

> **Session scope:** executing
> [`docs/planning/2026-10-09_18-02_SUPERB-v5-schema-evolution-execution-plan.md`](../planning/2026-10-09_18-02_SUPERB-v5-schema-evolution-execution-plan.md)
> (M1–M26). This report covers THIS session only (resumed ~22:30 2026-10-09 → 08:45 2026-10-10).
> Prior session state (T2+T1-core shipped, plan written): see
> [`2026-10-09_17-57_v5-schema-evolution-t2-t1-shipped.md`](2026-10-09_17-57_v5-schema-evolution-t2-t1-shipped.md).

## a) FULLY DONE (verified green this session)

| Task | What shipped | Verification |
| --- | --- | --- |
| **M1** Gate pre-flight + D1 | Re-ran `#check-file-size` + `#check-duplication`; found MY 3 file-size violations (status quo claim of "zero mine" was stale — see §d); fixed all three by file splits/extractions: `schema/chain_engine.go` 365→254 (+ `chain_payload.go` 123), `system/config_types.go` 421→302 (+ `engine_config.go` 127), `system/constructor.go` 435→352 (+ `system_lifecycle.go` 66, `projection_wiring.go` 40). D1 recorded in plan decision log | file-size gate now flags ONLY concurrent-session files (11 offenders, all theirs); system tests green after each split |
| **M2** Combined ADR (D2) | `docs/adr/0155-declarative-schema-evolution-t2-t1.md` (113 lines): op semantics, batch-vs-Upcasters bridge boundary, DomainConfig composition + Tier5→Tier2 layering, rejected alternatives ported from proposal §11, consequences + M6–M13 follow-up map. ADR index (README.md) updated through 0155 (0152–0154 were also missing — fixed) | md-go gate green (1506 blocks); doc-check green (1240 refs) |
| **M3** Tag wave — executed by the concurrent BDD session, independently VERIFIED by me | `schema/v4.6.0` + `system/v4.12.0` (+ deriver/scheduling/systemscenario) tagged 2026-10-10, riding my CHANGELOG entries. I verified: 5 tags local+origin resolved; `git tag --contains` proves schema/v4.6.0 contains the T2 code; my `system/config_types.go`/`constructor.go` splits landed AFTER the tag (unreleased, ride next wave — zero API change, pure file moves) | tag + content verification |
| **M4** Proxy + clean-consumer verification | Proxy serves schema v4.6.0 + system v4.12.0; clean-consumer smoke: fresh temp module, `go get schema/v4@v4.6.0`, compiled `RenameField`+`Transform` chain + `Upcasters()` — green with zero local replaces. Smoke-test bug found on the way: two exact ops at the same (type,version) correctly REJECTED (`ErrDuplicateOp`) — my test was wrong, not the library | `go list -m -versions` from /tmp; consumer ran |
| **M5** bank-sync adoption (D3) — SEMANTIC RE-COLLAPSE, not the patch | The preserved patch was STALE (their T18 schema-v3 work grew `upcasting.go` 116→236 lines + new v2→v3 upcaster + tests). `git apply` failed on 3 files → re-did the collapse against HEAD: `upcasting.go` 236→~160 (`UpcastChain()` with 2 `schema.Transform` ops; `upcastAmounts` domain logic preserved verbatim; `FieldRename` helper + 3 `New*Upcaster` constructors deleted); both `infrastructure.go` wiring sites + 4 test files migrated to chain semantics (error-code expectations `upcast.unmarshal`→`op_decode_failed`; corruption tables unchanged — inner codes survive wrapping) | bank-sync battery: `go build ./...`, `internal/cqrs` tests + `-race`, `golangci-lint` 0 issues, `erraudit` 0 violations. Committed via daemon commits `cfb8bfe1`+`cd224ca8` (my authored commit lost a hook race to a transient vulnix timeout + daemon — content IS in history) |
| **M6** T1a: typed declarations + decoder derivation | `schema/typed_declaration.go`: `TypedEventSchema[T]` + `EventOf[T]` (sample-based, no reflect). `system/schema_declarations.go`: `Schemas()` fluent builder — `.Event[T](type, ver, ops...)` accumulates declarations AND `projectionadapter.Register` registrations; `.Build()` (dup-reject, `system.duplicate_schema_declaration`) returns `SchemaDeclarations` with `.Declarations()` + `.TypeDecoder()` — ONE list feeds `DomainConfig.Schema` + `ProjectionTypeDecoder`. Roundtrip test: stored v1 event → rename upcast → typed fold sees current payload with stream-ID key | system tests green; api golden regenerated; changelog-symbols 19 citations verified; doc-check 1265 refs; treefmt clean; mnd finding fixed |
| **M7 (3 of 6 subtasks)** catalog bridge core | `catalog/schema_declaration.go`: `FromTypedSchema[T](decl, direction, opts...)` renders a typed declaration as a catalog event (wire name = ID, payload type = schema, version derived) + `SemverFromWire(int) "N.0.0"` — the §8 version-identity bridge (wire int is truth, semver is display). 3 tests green (render, WithVersion override, semver table) | `go test ./catalog/` green in workspace mode; catalog gained `schema/v4` dep (budget 7→ needs arch check — OPEN, see §b) |

Also done, small: `docs/adr/README.md` index drift fixed (0152–0155); plan decision log extended (D1, D2, D1-supplement tag-wave entry); CHANGELOG [Unreleased] T1a entry added.

## b) PARTIALLY DONE

- **M7 catalog render (in flight):** core bridge + tests DONE. Remaining: `#check-arch` budget verification for catalog's new schema dep; `#check-eventcatalog` gate; modules.md/faq doc rows; api golden regen (catalog module); CHANGELOG entry; **M7.4 ruling:** "system exposes declaration list to catalog path" is TIER-ILLEGAL as planned (system Tier5 cannot import catalog Tier6) — the M6 `Schemas()` builder result IS the single artifact both consumers receive; needs recording in the plan decision log.
- **M8 (cqrs-lint undeclared-event rule):** NOT STARTED (was gated on M7).
- **Housekeeping carried mid-session:** the LSP SA4023 panic on `system/schema_declarations_test.go` is a STALE CACHED replay (identical goroutine ID across replays; CLI golangci-lint on system runs clean) — restart didn't clear it; needs a fresh session or LSP cache purge. Non-blocking for CLI gates.

## c) NOT STARTED

M8 (cqrs-lint rule), M9 (DiscordSync collapse), M10 (cqrs-htmx adoption), M11–M13 (T4 snapshot stamp trio), M14 (benchmarks), M15 (godoc examples), M16 (fuzz/rapid), M17 (docs wave), M18–M19 (T3 fingerprint ledger), M20–M22 (T5 layout fingerprints + boot gate + replay marker), M23 (T6 compat policy + lint), M24–M25 (housekeeping A/B), M26 (full-repo #verify).

## d) TOTALLY FUCKED UP (self-inflicted, this session)

1. **Trusted the handoff's "Zero of my files fail any gate" — FALSE.** `schema/chain_engine.go` (365 lines, NEW offender) was over the limit when the 17:57 status was written; the earlier gate run must have predated the file's final form or its output was misread. Caught at M1 only because the plan mandated re-running gates. Lesson: gate output excerpts in status reports should carry the exact command + full error count, not a summary claim.
2. **Bank-sync patch assumed applyable — it was stale within 12 hours.** The concurrent session's T18 landed between patch-extraction and application. `git apply --check` failed on 3/5 files. Recovery: semantic re-collapse (correct, and the result is better — it adopted THEIR new v2→v3 upcaster too, which the patch never covered). Lesson: cross-repo patches rot at the speed of the other session; always `git apply --check` FIRST and budget for re-derivation.
3. **My authored bank-sync commit lost a race twice over:** the BuildFlow pre-commit hook failed on a transient vulnix timeout (43s nix build silence), and while it retried, the daemon absorbed my 6 staged files into two `chore:` commits. Content is committed; attribution/prose is not. Mitigation for next time: commit IMMEDIATELY after battery, before writing any further files the daemon can absorb.
4. **Smoke-test authored against my own API wrongly** (two exact ops at one (type,version) → ErrDuplicateOp panic trace in the consumer). Small, but it burned a cycle and briefly looked like a library bug.
5. **Inherited-but-unhandled: `.golangci.yml` corruption tripwire is RED.** The concurrent session's auto-configure re-added `gci` (default settings — no local prefix) and REMOVED the depguard allow-list + loosened cyclop 10→12, all without re-pinning `scripts/golangci-config-hash.golden.txt`. This directly contradicts AGENTS.md contract #18 (gci deliberately removed 2026-08-16 after it fought treefmt over 95+ files; depguard allow-list is part of the documented `#check-lint-config` gate). I chose NOT to revert (their committed work, possibly deliberate, mid-flight session) — but it means: gci noise findings on ~10 schema files (mine among them, grouping correct per treefmt), and `#check-lint-config` blocks M26's `#verify`. This needs an OWNER ruling (see §g).

## e) WHAT WE SHOULD IMPROVE (observed this session)

1. **Status-report claims should be gate receipts, not summaries** (see §d.1). Format: command + exit code + offending-file list, verbatim.
2. **Cross-repo adoption patches need a freshness contract:** `git apply --check` at session start of the APPLYING session, not just the authoring one; or re-derive semantically by default.
3. **Commit before celebrating:** authored commits must land the instant the battery is green; the daemon does not wait for your prose.
4. **Concurrent-session gate triage needs a standing rule I had to invent mid-flight:** "fix mine to green, leave theirs, record attribution" worked but cost three investigations (blame archaeology on constructor.go's 357→424 pre-session growth to prove my delta was only +11 of +78). A per-file owner field in the file-size baseline would make this instant.
5. **The golangci config tripwire needs a JSON mode** (`--check` emitting machine-readable expected/actual + which rule groups changed) — I had to reconstruct intent from `git log -L` archaeology.
6. **`SchemaDeclarations` two-field wiring:** the builder delivers one LIST, but the consumer still assigns two DomainConfig fields. A future additive `DomainConfig.SchemaDeclarations SchemaDeclarations` field could take the struct directly — deferred (API surface discipline; the current form is honest and shipped).
7. **api-stability TestEvery failures from OTHER modules** (cqrs-bench/cqrs-gen go.sum untidy, example/graph-native README missing) block a clean "golden meta-test green" statement for MY work — same triage problem as #4.

## f) NEXT — up to 50, in execution order (plan M-numbers)

**Finish M7 (this session's next moves):**
1. `nix run .#check-arch` — verify catalog budget with schema/v4 added (bump `DEP_BUDGET[catalog]=7` with rationale if 6→7 real)
2. Record M7.4 tier ruling in plan decision log (system→catalog illegal; builder result is the artifact)
3. `nix run .#check-eventcatalog` gate
4. api golden regen (catalog module) + `TestEvery` scoped check
5. CHANGELOG [Unreleased] T1b entry + changelog-symbols gate
6. modules.md catalog row + faq cross-link + doc-check

**M8 (cqrs-lint undeclared-event rule):**
7. Parse `DomainConfig.Schema` declarations in the analyzer
8. Rule: consumed-but-undeclared event type → diagnostic
9. Testdata fixture project (declared + dangling consumer)
10. Suggestion text; E018 lockstep check (`catalog.Event` counts as provided)
11. Analyzer self-tests + module-catalog meta-test
12. cqrs-lint README + CHANGELOG + golden

**M9 (DiscordSync collapse at `~/projects/DiscordSync/internal/eventschema/upcasters.go`, 211 lines):**
13. Read their upcasters fresh (may have drifted like bank-sync — expect it)
14. Map the two type renames → `RenameType` ops
15. Map derived-field upcasters → `Transform` ops
16. Wire chain; rewrite their tests; battery (their env/runbook)
17. Adopt on tag or preserve patch + revert

**M10 (cqrs-htmx adoption):**
18. Locate SchemaVersion surface (`sync_pull.go:282`, `event_catalog.go:24`) + config seam
19. Declare `DomainConfig.Schema`; tests + battery; adopt/preserve

**M11–M13 (T4 snapshot state-shape stamp):**
20. snapshot envelope optional stamp field + accessors + format decision
21. Encode/decode roundtrip tests; legacy absent-stamp accept tests
22. snapshot golden + CHANGELOG + api golden
23. decider `WithSnapshotStateVersion[State]` option
24. Save path stamps; load path mismatch → discard + counted stat
25. Integration test: stale snapshot → rebuild from journal
26. Conformance test in the snapshot contract suite
27. bank-sync stamps `BalanceSyncState`; test discard+rebuild on real snapshots; battery

**M14–M16 (trust surface):**
28. Bench fixtures: chain vs hand-rolled closure (JSON + CBOR)
29. Record ns/op + allocs; accept/reject threshold decision; perf note in schema README
30. Godoc Examples: `ExampleCompile`, `ExampleChain_SourceTransform`, `ExampleDeclare` + system composition; vet + doc gates
31. Fuzz: arbitrary payload bytes through chains (no panic, version sane)
32. Fuzz: random op sets — Compile deterministic accept/reject
33. rapid: version monotonicity; identity preservation under field ops
34. Seed corpus run + fix findings; wire into fuzz/golden suites

**M17 (docs wave):**
35. recipes.md named-ops recipe fence + classify in `recipes_catalog*.go`
36. Compile-harness green (`TestRecipes*`); core.md §3 declarative-form convention
37. faq + modules cross-link polish; doc-check + md-go

**M18–M22 (T3/T5 tail — drift detection):**
38. Declared-shape fingerprint helper (stable hash, sorted fields) + `EventSchema` exposure
39. Opt-in write-path metadata stamp + system wiring
40. Read-side compare + warn seam (slog/otel) + opt-in hard mode + burn-in doc note
41. `LayoutPlan` stable hash + persistence contract (memory + sqlite engines)
42. Reset-clears-stamps/journal-exempt test (ADR-0143); absent = accept
43. Boot diff → `LayoutDiff` → RebuildThreshold auto / ConfirmRebuild gate paths + tests
44. Completed-replay marker (crash-mid-rebuild safety) + Doctor render

**M23–M26 (policy + close):**
45. T6 compat policy doc (additive-safe vs version-bump triggers) + cqrs-lint additive-change rule + fixtures
46. M24 housekeeping A: kv-alias sweep (recipes.md:69, core.md:315), proposal-fence convention → gotchas, parallel-declaration ritual → AGENTS
47. M25 housekeeping B: `RevisionSnapshotFilter` lead verify/drop, migrate-journal runbook note, TODO_LIST prune, 17:57 status annotate
48. M26: concurrent-session gates must go green first (11 file-size offenders, 6 clone groups, golangci tripwire, cqrs-bench/cqrs-gen go.sum, graph-native README) — then `nix run .#verify` + fast integration set; triage anything of mine
49. Next tag wave planning: schema v4.6.1+ (ships `EventOf`/`TypedEventSchema` + file splits) MUST co-release with catalog minor + any system patch — per-module GOWORK=off CI stays red for catalog until then
50. Fleet wave-2 backlog beyond plan: go-appkit typed-declaration adoption; cqrs-upgrade codemod learns `Schemas()` migration

## g) QUESTIONS FOR THE OWNER (cannot be resolved from the repo)

1. **`.golangci.yml` conflict — who wins?** The concurrent session's committed config re-added `gci` (default groups, fights treefmt per contract #18), deleted the depguard allow-list, and loosened cyclop 10→12, WITHOUT re-pinning the corruption-tripwire golden. Options: (a) restore the documented config + re-pin (reverts their committed change), (b) accept their config, update contract #18 + re-pin golden, (c) ask their session to reconcile. This gates M26 (`#verify` runs `#check-lint-config`). I can execute any option on your word.
2. **May I tag the next schema wave when Phase 1 completes** (schema v4.6.1+ for `EventOf` + catalog minor co-release; same flock + tag-release.sh path as the BDD wave), or do you want the concurrent BDD session to own tag waves while both streams are active? The catalog module's GOWORK=off per-module CI is red-on-master until this ships, so timing matters.
3. **bank-sync attribution:** the collapse landed as daemon `chore:` commits (`cfb8bfe1`+`cd224ca8`) after a hook race. Want me to `git rebase`-rewrite those two into one authored commit (local-only, nothing else built on them yet — I verified HEAD-adjacent), or leave daemon attribution as-is?

## Session ledger

- go-cqrs-lite: M1–M6 complete + M7 core; ~14 files changed (4 new code files, 4 splits/extractions, ADR-0155, README index, plan log, CHANGELOG ×2 entries, modules.md, api golden). NOT pushed (no instruction).
- bank-sync: 6-file collapse committed (daemon). NOT pushed.
- Standing risks carried: 11 file-size + 6 duplication offenders + golangci tripwire + 2 go.sum + 1 README gap — ALL concurrent-session-owned, none mine.
