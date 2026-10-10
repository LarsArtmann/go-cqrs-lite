# v5 Schema-Evolution Plan — Status (2026-10-10 09:49 CEST)

> **UPDATE 2026-10-10 ~23:00 (evening continuation):** M9–M25 are DONE —
> DiscordSync chain migration + the CBOR nested-map library fix (D7),
> cqrs-htmx Schema adoption (D8), T4 snapshot guard (M11–M13 staged for
> bank-sync), trust surface (M14–M16), docs wave (M17), T3+T5 fingerprints
> (M18–M22), compat policy + E022 (M23), housekeeping (M24–M25). The three
> OPEN questions below are still open (tag-wave ownership is now the only
> blocker for the post-tag adoption tail). Current truth:
> [the plan's decision log](../planning/2026-10-09_18-02_SUPERB-v5-schema-evolution-execution-plan.md) (D7+) + TODO_LIST prune.

Scope: executing M1–M26 of
[the v5 schema-evolution execution plan](../planning/2026-10-09_18-02_SUPERB-v5-schema-evolution-execution-plan.md)
under the standing loop. This report covers the morning continuation (M7 finish → M8 complete →
M9 research-complete, no DiscordSync edits yet). Previous report:
[08-45 Phase-0-done M7-inflight](2026-10-10_08-45_v5-schema-plan-phase0-done-m7-inflight.md).

## a) FULLY DONE (all verified green unless attributed)

### M7 — catalog render + semver bridge (T1b) — COMPLETE

- `#check-arch` green: `DEP_BUDGET[catalog]` 7→8 with rationale (`scripts/check-module-layers.sh:321`).
- `#check-eventcatalog` green — unblocked via **dev-time sibling replace**
  (`replace schema/v4 => ../schema` in catalog/go.mod; tag-release strips it at tag time, forcing
  the schema+catalog co-release already scheduled for Phase 1). Recorded as **D5**.
- API golden regenerated (8469 → 8475 → 8477 exports); `TestEvery` meta-tests GREEN.
- CHANGELOG T1b entry (catalog.FromTypedSchema / SemverFromWire + tier ruling); changelog-symbols
  gate green (30 citations).
- modules.md catalog row + new faq entry ("EventCatalog export in sync with schema declarations")
  - TOC; doc-check green (1270 refs); md-go green.

### M8 — cqrs-lint E021 (T1c) — COMPLETE

- **New rule E021 `emitted-without-schema-declaration`** (Warning, medium confidence):
  fires when a project declares ANY event schema (schema.Event/EventOf, or the
  system.Schemas() builder's `.Event[T]`) but emits an event type outside the declaration list —
  the typo class that silently skips upcasting + typed decoding.
  - Analyzer: `EventTypesInSchemaDecl` registry field + `IsEventSchemaDeclared` accessor +
    `pendingSchemaEventTypeRefs` const-resolution (same post-pass as emitted/catalog refs).
  - Scanner: schema-qualified Event/EventOf cases + BOTH builder forms (direct chain via
    receiver-walk recursion; `schemas := system.Schemas()` variables via per-file assignment
    tracking). False-positive direction is over-declare (safe, suppresses only).
  - Scope ruling (recorded as **D6**): emitters only; consumption-only types stay E018's
    provider tier; catalog.Event counts as declared; zero declarations → silence.
  - 7 fixture tests, all green first run. Detector count 210→211; README row + "**211 rules**"
    headline; RULES.md regenerated via `go run . rules --markdown` (it is GENERATED output —
    hand edits are wrong, see section d).
  - File-size offset: extracted handlerTypeFromCall/constructorHandlerText/handlerTypeFromClosure
    to scanner_calls_helpers.go — scanner_calls.go 362→338, BELOW its 360 baseline (shrank, allowed).
  - CHANGELOG entry (T1c); changelog-symbols green; api golden regen; `check-cqrs-lint-cli.sh`
    contract probes green; file-size gate fully green repo-wide.
- **Gate hygiene absorbed from concurrent sessions** (mechanical, additive-only):
  repo-wide `go mod tidy` — 28 modules carried stale go.sum/indirect-pin drift after their
  import-churn mass commits; graph-native example README authored (meta-test demanded; claims
  verified against its source); taskmanager lint goldens V003 2→3 in BOTH places
  (testdata/taskmanager_golden.txt + the inline `taskmanagerGoldenProfile` map) — their overnight
  tag wave added systemscenario v4.0.0 which V003 now flags.
- **Verified NOT mine**: `TestMultiModuleBuildContext_PartitionsProfiles` (deriver
  CommandFlow=commands expectation) fails at the 05:44 pre-edit commit too (worktree check) —
  concurrent-session-owned, on the M26 blocker list.

### Plan bookkeeping

Decision log now carries D3 (bank-sync semantic re-collapse), D4 (M7.4 tier ruling: bridge in
catalog, never system→catalog), D5 (sibling replace), D6 (M8 execution + scope ruling + gate
hygiene + partition-test attribution).

## b) PARTIALLY DONE

### M9 — DiscordSync conversion — RESEARCH COMPLETE, ZERO EDITS YET

Everything needed is read and verified (fresh, post-drift):

- Their `upcasters.go` (211 lines): 2 type renames (discord.attachment.migrated→AttachmentBackedUp,
  discord.embed_media.migrated→EmbedMediaBackedUp) + 4 typed user-kind upcasters
  (MessageCreated/`author`, MemberJoined/MemberUpdated/MemberLeft/`user`, all sourceVersion 1,
  best-effort decode).
- Mapping settled: renames → `schema.RenameType` (chain rebuild preserves payload bytes,
  encoding stamp via WithEncoding — their old retypeTo workaround comment describes behavior the
  library has since fixed; verified `event/event_new.go:69-75`); user-kind →
  `schema.Transform(type, 1, mapFn, WithDecodePolicy(PassthroughOnDecodeError))` where mapFn
  sets `kind` from `domain.ResolveUserKind(kind, bot-flag)` — JSON tags confirmed (`bot`, `kind`,
  `author`, `user`).
- Version semantics match: Transform advances 1→2 (their old registry did +1); RenameType keeps
  version (chain contract; their rename tests don't assert version).
- Wiring seam: `storage.go:146` `cqrsschema.NewVersionedSeekableJournal(ec.store, Registry()...)`
  (deprecated shell) → canonical `event.DecorateJournal(ec.store, chain.SourceTransform())`
  (forwards SeekableJournal — verified in journal_middleware.go) + Seekable type assertion.
- Their pin is schema **v4.5.0**; adoption requires `go get schema/v4@v4.6.0` (tag exists, proxy
  serves it — verified in M3/M4). Plan says: adopt (tags available).
- Test rewrite plan: keep all 10 existing test intents (default-version firing, identity
  preservation, MemberUpdated entry, idempotence, rename firing, JSON-encoding preservation,
  bot/webhook/human derivations) at chain level via DecorateJournal + fakeJournal.
- Battery: `nix run .#verify` (their full gate), `nix run .#quality` fast path; their daemon
  auto-commits AND auto-pushes within ~30-60 min — DiscordSync work goes public fast; also
  `scripts/sync-flake-pins.sh` after dep bumps (flake pins schema version).

## c) NOT STARTED

M10 (cqrs-htmx), M11–M13 (snapshot stamp trio), M14–M16 (benchmarks/godoc/fuzz+rapid),
M17 (docs wave), M18–M22 (fingerprint/layout tail), M23 (compat policy + additive-change rule),
M24–M25 (housekeeping), M26 (full verify).

## d) TOTALLY FUCKED UP (mine, this session)

1. **Hand-edited RULES.md before checking for a generator.** The rules-doc meta-test failed and
   told me it's regenerated via `go run . rules --markdown`. I should have grepped for the
   generator FIRST — the repo convention is everywhere (goldens, baselines). Cost: one wasted
   test cycle + a hand edit that regeneration silently replaced.
2. **Misread `CQRS_LINT_UPDATE_GOLDEN=1` as "writes the golden."** For the root-package test it
   DOES write the txt; for the pkg/rules test it only TLOGS an inline map for manual pasting. I
   concluded "fixed" from a pass that was really update-mode's unconditional return, then burned
   another full-suite cycle discovering the failure persisted. Read the update code path before
   trusting a green update run.
3. **`tail -n +2` on the untidy-module list skipped the first entry** (commandlifecycle/projections)
   because I assumed a header line that wasn't there. Cost: one extra TestEvery round. Trivial
   but dumb — the list had no header.
4. **`rg -rn` typo** — `-r` is "replace matches with 'n'". Garbage output for one search; caught
   immediately by the mangled field names.
5. **multiedit TOC anchor edit failed on exact-match** — I guessed the TOC line's label from the
   section heading instead of viewing it. The tool's staleness/mod-time guards saved me twice
   (registry.go, scanner.go); always re-view before edit in daemon-active repos.

None of these shipped defects; all were caught inside the same task.

## e) IMPROVEMENTS (retrospective, for the next stretch)

- **Long gates first-in-background**: I ran the ~15-min eventcatalog gate before choosing the
  sibling replace that made it pass; researching the replace seam first would have saved a wait.
- **Run lint (buildflow) on modules I touched** — cqrs-lint module tests + CLI probe + build are
  green, but I did not run golangci over my new e021.go/scanner files (their .golangci.yml is in
  flux per Q1 anyway). Do it in M9's battery window.
- **check-duplication not re-run** after the helper extraction (a pure move creates no new clone
  shapes, but the gate is cheap — schedule with M26 or next cqrs-lint touch).
- **Daemon branch-suffixed commits** ("on cqrs-lint/a014-d013-scoped-fixes") observed; repo is on
  master and clean — not investigated further. Watch it.
- **My tidy introduced indirect pin drift** (x/sync v0.23→0.24, x/exp, docker/* in 3 modules) —
  verified the fleet is already mixed on these pins, but pin-sweep/vulncheck gates were NOT run
  this session (M26).

## f) Next up to 50 (ordered, realistic; M-numbers from the plan)

1. M9.1–M9.4: DiscordSync — `go get schema/v4@v4.6.0` (+sync-flake-pins), rewrite upcasters.go
   as `UpcastChain()` (2×RenameType + 4×Transform w/ PassthroughOnDecodeError), rewire
   storage.go Journal() via DecorateJournal, rewrite both test files chain-level (keep all 10
   intents incl. JSON-encoding-preservation).
2. M9.5: battery — `go test ./internal/eventschema/... ./internal/storage/...`, then their
   `nix run .#quality` (+ full `.#verify` if time); record adoption in plan log.
3. M10.1: read cqrs-htmx `systemadapter` + `sync_pull.go:282`/`event_catalog.go:24` seams fresh.
4. M10.2: declare DomainConfig.Schema in their composition (schema.Event declarations for their
   event types).
5. M10.3: their tests + battery (their verify gate).
6. M10.4: adopt or preserve patch + record.
7. M11.1: snapshot envelope optional state-shape stamp field.
8. M11.2: accessors + validation (string version stamp decision).
9. M11.3: envelope encode/decode roundtrip tests.
10. M11.4: legacy absent-stamp accept tests (pre-change snapshots still load).
11. M11.5: snapshot golden + CHANGELOG + api golden.
12. M12.1: `decider.WithSnapshotStateVersion[State]` option.
13. M12.2: save path stamps envelope with declared version.
14. M12.3: load path mismatch → discard + counted stat.
15. M12.4: stale-snapshot → rebuild-from-journal integration test.
16. M12.5: snapshot contract conformance suite.
17. M12.6: gates (module tests + lint + arch + changelog).
18. M13.1: bank-sync stamps BalanceSyncState.
19. M13.2: discard+rebuild path test on real snapshots.
20. M13.3: bank-sync battery + record.
21. M14.1: bench fixtures (chain vs hand-rolled closure).
22. M14.2: run bench JSON+CBOR, record ns/op + allocs.
23. M14.3: comparison table + threshold decision.
24. M14.4: perf note in schema README.
25. M15.1: ExampleCompile godoc example.
26. M15.2: ExampleChain_SourceTransform.
27. M15.3: ExampleDeclare + system composition example.
28. M15.4: vet + doc gates.
29. M16.1: fuzz — arbitrary payloads through chains (no panic, version sane).
30. M16.2: fuzz — random op sets through Compile (rejects invalid).
31. M16.3: rapid — version monotonicity property.
32. M16.4: rapid — identity preservation property.
33. M17: recipes.md recipe + recipes-catalog classification (compile harness).
34. M17: core.md §3 (declarative evolution section).
35. M17: faq/modules polish pass.
36. M18: declared-shape fingerprint (schema module).
37. M19: opt-in metadata stamp on write.
38. M20: read-side mismatch warn hook.
39. M20b: opt-in hard mode (error on mismatch).
40. M21: LayoutPlan fingerprint persistence (memory+sqlite) + boot diff.
41. M22: RebuildThreshold/ConfirmRebuild + completed-replay marker + Doctor render.
42. M23: T6 compat policy doc + cqrs-lint additive-change rule.
43. M24: housekeeping A (kv-alias sweep recipes.md:69/core.md:315, fence convention,
    parallel-declaration ritual).
44. M25: housekeeping B (RevisionSnapshotFilter lead, runbook note, TODO_LIST prune,
    status-report annotate).
45. M26: `nix run .#verify` — STILL BLOCKED by concurrent-session items (see g/Q1); run what's
    runnable, document the rest.
46. Re-check `#check-lint-config` state (Q1) before M26.
47. Re-check `#check-duplication` after M9–M24 library edits.
48. Update THIS plan's progress table (M-percentage column) — it still says the morning numbers.

## g) Up to 3 questions I cannot figure out myself

(Carried from the 08:45 report — still open; I proceeded under the delegated-decision pattern.)

1. **`.golangci.yml` conflict (Q1)**: the concurrent session re-added `gci` (fights treefmt per
   contract #18), removed the depguard allow-list, loosened cyclop, without re-pinning the config
   hash golden. (a) restore documented config + re-pin, (b) accept theirs + rewrite contract #18
   - re-pin, or (c) leave it to the BDD session? Blocks a clean M26 `#verify` (lint-config gate).
2. **Tag-wave ownership (Q2)**: may this stream tag the next wave (schema v4.6.1+ with
   TypedEventSchema + catalog + cqrs-lint co-release) when Phase 1 completes, or does the BDD
   session own tag waves? (I will NOT tag without an explicit go — same for pushes.)
3. **bank-sync attribution (Q3)**: rewrite the two daemon commits into one authored commit
   (nothing built on them), or leave daemon attribution?

Additionally flagging (not a question, a fact to survive): DiscordSync's daemon AUTO-PUSHES to
origin within ~30-60 min — the M9 conversion will be public fast whether or not that is desired.
