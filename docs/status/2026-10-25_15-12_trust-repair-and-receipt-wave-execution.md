# Trust-Repair & Receipt Wave — Execution Session (2026-10-25 ~14:35–15:15)

Session: 2026-10-25 afternoon, plan
[`docs/planning/2026-10-25_14-49_SUPERB-trust-repair-and-receipt-pareto-plan.html`](../planning/2026-10-25_14-49_SUPERB-trust-repair-and-receipt-pareto-plan.html).
Scope: repair the config damage found live in HEAD, close the value-integrity
wave's missing mutation receipts (14:06 report (b)-1), and drain the
executable NEXT queue. All work verified step-by-step; receipts below.

---

## a) FULLY DONE (evidence inline)

1. **P0 CONFIG REPAIR — buildflow auto-configure damage restored.** A parallel
   session's `buildflow --fix --build-mode=full` (14:31–14:4x) stripped the
   depguard allow-list + systemscenario carve-out comments from `.golangci.yml`
   (−111 lines), reset `issues:` limits 50/10→0/0, AND stripped the
   `.buildflow.yml` `env:` disk-cache chain (the 2026-09-22 ENOSPC lesson);
   the daemon absorbed it all into HEAD at 14:35 (`3d8c1bf01`). Restored both
   byte-exact from `3d8c1bf01^`: `check-lint-config` rc=0,
   `check-golangci-hash.sh` PASS (74a068c17227…). Incident 12 recorded in
   `docs/agents/gotchas-tooling-build.md`; receipt added to the durable-guard
   TODO row. **Coverage gap identified**: `.golangci.yml` is hash-golden
   protected, `.buildflow.yml` is NOT — that extension is the concrete
   mechanism proposal for the open row.
2. **go.sum triage.** 27 modified go.sum files at session start = workspace-mode
   residue from the same buildflow run (per-module `GOWORK=off go mod tidy
   -diff` wants the added lines REMOVED). Restored all 27 to the committed
   baseline; noted HEAD itself carries extra /go.mod hashes (pre-existing,
   harmless to readonly builds — strict tidy-cleanliness is not the repo's
   go.sum gate).
3. **Mutation receipts ×3 — the wave's pin classes now have proven teeth**
   (each: mutate product code → RED with exact evidence → restore
   byte-identical → green):
   - **Planned-path/MapUpdate**: `execPlannedSet` value encoded through a
     wrapper struct → BOTH `TestStoredBytes_PlannedMapSet` and
     `TestStoredBytes_PlannedMapUpdate` RED with `{"V":{...}}` evidence
     (proves both paths route through the one seam).
   - **Turso delegation**: `sqliteengine.encode_value` code mutated →
     `TestTurso_MapSet_RejectsUnrepresentableValue` RED
     (`code = "sqliteengine.wrong_code", want "sqliteengine.encode_value"`).
   - **Bigtable adttest**: `keycodec.encode_value` code mutated →
     `TestValueEncodeRejection` RED on the bttest fake.
   Method notes: two compile-broken first attempts (unused import; comment
   split an arg list) were discarded as invalid proofs — only semantic REDs
   count; all mutations ran mutate→test→restore inside ONE command to avoid
   daemon absorption.
4. **LSM MapUpdate re-encode pins shipped** (pebble/bbolt/badger):
   `TestStoredBytes_MapUpdate_ReEncodes` per engine (Set→Update→raw read →
   literal `{"title":"updated","n":9}`), all green; pebble version
   mutation-proved (update-path wrapper-struct → RED → restore → green).
   Closes NEXT #29.
5. **json/v2 determinism blast radius recorded**: new TODO row — map-typed
   STORAGE KEYS are iteration-order dependent too (a Get can miss the row a
   Set wrote — worse than value bytes); gotchas-language-footguns entry for
   the v1→v2 sorted-map-keys change + `json.Deterministic(true)` cure; pins
   must stay struct-shaped until the Q2 wave. Closes NEXT #2 + #28.
6. **CHANGELOG honesty fix**: the schema-builder Added entry (T1a) described
   the pre-repair fluent `.Event[T]` builder method — reshaped to the shipped
   package-level `system.SchemaEvent[T](set, …)` with the legality rationale;
   `check-changelog-symbols` green (11 citations honest). Closes NEXT #12+ #49.
7. **schema_declarations godoc pass** (NEXT #14 + #38 + #39): the `Schemas()`
   example called a NONEXISTENT `system.Event` function (would never compile)
   — rewritten as the real `SchemaEvent` chain; `Build()` doc said
   "non-empty" while empty returns nil,nil — empty-set-now-opt-out semantics
   documented; concurrency note tightened ("one goroutine, before boot").
   New README section "Declaring Event Schemas" with the builder flow;
   system module build + schema tests green.
8. **recipes.md §2.49 "Pin Your Engine's Stored Bytes"** — full recipe
   (sqlite skeleton + LSM raw-handle twins + mutation-proof discipline),
   TOC + catalog entry added; `TestRecipes*` GREEN (catalog coverage +
   compile, 9.2 s warm). Closes NEXT #23.
9. **metaengine/README codec section** (T11.2): README had NO stale claim
   (its codec section is event-payload ingress) but also didn't document the
   stored-value rejection contract — added a paragraph: rejection-not-fallback
   per engine family + the three pin layers. Closes NEXT #10.
10. **COOKBOOK pointer** (NEXT #24): "Stored values" now lists the five pin
    files + mutation-proof receipts location. Stale-mention sweep (NEXT #26):
    clean — remaining "lenient" hits are the DSN-redaction and index-key
    identity contexts, both correct.
11. **Root-cause probes with honest verdicts**:
    - **badger first-run transient (NEXT #6)**: 100× `TestStoredBytes` loop
      under ambient load — ZERO failures; transient did not reproduce;
      standing capture order documented (gotchas-tooling-build).
    - **queue lint flake (NEXT #7)**: git-log correlation shows ZERO daemon
      commits touching `queue/*` in the 12:50–14:06 flake window — the
      daemon-race hypothesis is DISPROVEN for the sources; remaining suspects
      (host pressure at load ~100) unproven; capture protocol documented.
    - **art-dupl baseline provenance (NEXT #11)**: last touch is `0dfb8a153`
      (10-23 10:32, daemon) — a SURGICAL +2-group insert (docserver templ ×2,
      architecture/index×discordsync fixture), NOT a wholesale re-pin; nothing
      was absorbed during the 10-25 sessions. The 14:06 report's "inferred:
      baselined during the parallel session's close-out" is WRONG — the
      morning group stopped flagging because its code shape changed
      (hash-based matching), not because of a re-pin. Errata recorded here.
12. **engine-capabilities drift check (NEXT #22)**: `check-engine-capabilities`
    green — no drift after the bigtable migration.
13. **SchemaEvent fleet census (NEXT #16/#17)**: zero consumers outside the
    repo (Cyberdom/Kernovia name hits are their own local symbols); no fleet
    use of the illegal `Schemas().Event[` chain → the codemod trigger for
    this repair is EMPTY; no cqrs-upgrade work needed for SchemaEvent.

## b) IN FLIGHT

1. **Composed verify-chain** (`scripts/verify-chain.sh`, durable log
   `build/verify-chain-session-*.log`): launched at ~15:10 with the built-in
   quiet-window gate (load had dropped 103→24). The 14:06 asterisk — "every
   step green individually, composed runs flaked" — gets its continuous-green
   proof or an honest bounded attempt. Verdict lands in the job output.

## c) NOT DONE (deliberate)

- Owner-gated rows unchanged: Q1 SchemaEvent shape, Q2 `Deterministic(true)`
  wave, Q3 daemon lease, envelope spike §12, markdown-format policy, v4
  fix-only declaration, tooling product line, keycodec-v2 engine wave, presets
  retirement, metaengine split (owned by the 04-25 follow-up plan), wave-2
  kickoff. See the plan's Owner-Gated table.
- Soak suites (bbolt 489 s, systemtest real-engine) — quiet-window items,
  deliberately not started while ambient load was 3× oversubscribed.

## d) WHAT WE SHOULD IMPROVE

1. `rm` inside a splice pipeline hit the PRE-EXISTING tracked
   `docs/planning/plan-graph.svg` (deleted, restored from git within a
   minute). The temp-artifact rule exists for exactly this: cleanup globs
   must be name-specific, never `rm <generic>.<ext>` in a docs directory.
2. Mutation proofs need a compile-safe mutation FIRST (unused-import and
   arg-list-comment attempts wasted two cycles) — pick wrong-code/wrong-shape
   mutations that keep the package compiling.
3. The config-tripwire class now has three data points (gci re-add ×2,
   depguard+env strip ×1); the `.buildflow.yml` hash-golden extension is a
   30-minute fix awaiting the owner's mechanism ruling.

---

*All mutations restored byte-identical (`git status` verified after each);
every green claim above is a gate or suite rc=0 observed this session.*
