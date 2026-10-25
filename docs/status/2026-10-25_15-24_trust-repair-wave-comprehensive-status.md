# Trust-Repair & Receipt Wave — Comprehensive Status (2026-10-25 15:24)

Session: 2026-10-25 ~14:35–15:24 CET. Plan:
[`docs/planning/2026-10-25_14-49_SUPERB-trust-repair-and-receipt-pareto-plan.html`](../planning/2026-10-25_14-49_SUPERB-trust-repair-and-receipt-pareto-plan.html).
Predecessor: [`…_15-12_trust-repair-and-receipt-wave-execution.md`](2026-10-25_15-12_trust-repair-and-receipt-wave-execution.md)
(same session, mid-run snapshot). Scope: repair the config damage found live
in HEAD, close the value-integrity wave's missing mutation receipts, drain the
executable NEXT queue from the 14:06 report. Full context: a parallel session
ran `buildflow --fix --build-mode=full` against THIS repo 14:31–14:4x (exit 1,
1122/1447 steps failed) while five crush sessions and fleet services kept
load1 between 14 and 103 the entire time.

---

## a) FULLY DONE (evidence in parentheses)

1. **P0 CONFIG REPAIR — buildflow auto-configure damage found in HEAD and
   restored.** The parallel `buildflow --fix` stripped the depguard
   allow-list + systemscenario carve-out comments from `.golangci.yml`
   (−111 lines), reset `issues:` limits 50/10→0/0, AND stripped the
   `.buildflow.yml` `env:` disk-cache chain (the 2026-09-22 /mnt/buildcache
   ENOSPC lesson); the daemon absorbed all of it into HEAD at 14:35
   (`3d8c1bf01`). Restored both files byte-exact from `3d8c1bf01^`:
   `check-lint-config` rc=0, `check-golangci-hash.sh` PASS (74a068c17227… —
   proving the golden still pinned the curated bytes and my restore matches
   it). Incident 12 recorded in `docs/agents/gotchas-tooling-build.md`;
   dated receipt added to the durable-guard TODO row (CI section). The
   coverage gap is named: `.golangci.yml` is hash-golden protected;
   **`.buildflow.yml` has NO tripwire** — that extension is the concrete
   mechanism proposal now sitting on the open row.
2. **go.sum residue triage.** 27 modified go.sum files at session start were
   workspace-mode residue from the same buildflow run (per-module
   `GOWORK=off go mod tidy -diff` wants the added lines REMOVED). Restored
   all 27 to the committed baseline. Verdict worth remembering: HEAD itself
   carries extra `/go.mod` hash lines (pre-existing, harmless — readonly
   builds only fail on MISSING entries; strict tidy-cleanliness is NOT the
   repo's go.sum gate).
3. **Mutation receipts ×3 — the value-integrity wave's pin classes now have
   PROVEN teeth** (each: mutate product code → semantic RED with exact
   evidence → restore byte-identical → green):
   - **Planned-path/MapUpdate** (closes 14:06 §b-1): `execPlannedSet` value
     encoded through a wrapper struct → BOTH
     `TestStoredBytes_PlannedMapSet` and `TestStoredBytes_PlannedMapUpdate`
     RED with `{"V":{...}}` evidence — proving both planned write paths
     route through the one encoder seam.
   - **Turso delegation** (§b-1): `sqliteengine.encode_value` code mutated
     to `wrong_code` → `TestTurso_MapSet_RejectsUnrepresentableValue` RED
     (`error code = "sqliteengine.wrong_code", want "sqliteengine.encode_value"`).
   - **Bigtable adttest pin** (§b-1): `keycodec.encode_value` mutated →
     `TestValueEncodeRejection` RED on the in-process bttest fake.
4. **NEW LSM MapUpdate re-encode pins shipped** (closes 14:06 §f29):
   `TestStoredBytes_MapUpdate_ReEncodes` added to pebble/bbolt/badger
   (MapSet → MapUpdate → RAW read → literal `{"title":"updated","n":9}`),
   all three green; the pebble one is mutation-proved (update path
   wrapper-struct → RED → restore → full pebble pin suite green).
5. **json/v2 nondeterminism blast radius recorded** (closes 14:06 §f2+§f28):
   new TODO row — **map-typed STORAGE KEYS are also iteration-order
   dependent**: `EncodeKeyStr` on a map key can route one logical key to
   different bytes per run, so a Get can MISS the row a Set wrote (split
   reads, broken uniqueness, orphaned deletes — strictly worse than the
   value-bytes finding). Gotcha entries written:
   `gotchas-language-footguns.md` (v1 sorted maps, v2 doesn't,
   `json.Deterministic(true)` verified cure, pins stay struct-shaped until
   Q2) and `gotchas-testing.md` (direct-sweep-vs-gate env parity — the
   14:06 §e-2 lesson).
6. **CHANGELOG honesty fix** (closes 14:06 §b-4+§f12+§f49): the schema-builder
   Added entry (T1a) described the pre-repair fluent `.Event[T]` builder
   METHOD — an API that never legally existed. Rewritten to the shipped
   package-level `system.SchemaEvent[T](set, …)` shape with the legality
   rationale; `check-changelog-symbols.sh` green (11 citations honest).
7. **schema_declarations godoc pass** (closes 14:06 §f14+§f38+§f39): the
   `Schemas()` doc example called a **NONEXISTENT `system.Event` function**
   (would never compile) — rewritten as the real `SchemaEvent` chain;
   `Build()`'s doc said "non-empty" while the code returns nil,nil on empty —
   empty-set-opt-out semantics now documented; SchemaSet concurrency note
   tightened ("one goroutine, before boot"). New README section "Declaring
   Event Schemas" with the builder flow. system build + schema tests green.
8. **recipes.md §2.49 "Pin Your Engine's Stored Bytes"** (closes 14:06 §f23):
   full recipe — sqlite skeleton (compilable fence), LSM raw-handle twins
   (`GetRawValue`/`FromDB` bucket/txn), mutation-proof discipline paragraph;
   TOC entry + `recipes_catalog_meta4.go` catalog entry added;
   `TestRecipes*` GREEN (coverage + compile, 9.2 s warm).
9. **metaengine/README rejection contract** (closes T11.2, 14:06 §f10):
   verified NO stale claim existed (its codec section is event-payload
   ingress, not stored values) but the contract was undocumented — added the
   rejection-not-fallback paragraph (per-family codes, no `%v` fallback, the
   three pin layers).
10. **COOKBOOK pin pointers** (closes 14:06 §f24) + **stale-mention sweep**
    (§f26): "Stored values" now lists all five pin files + where mutation
    receipts live; repo-wide sweep for stale "graph encoders"/"bigtable
    lenient" claims came back clean (remaining "lenient" hits are the DSN
    redaction and index-key identity contexts, both correct).
11. **Root-cause probes with honest verdicts**:
    - **badger first-run transient** (14:06 §d-1/§f6): 100× pin-suite loop
      under ambient load — ZERO failures, transient did not reproduce;
      standing capture order documented rather than a fake root cause.
    - **queue lint flake** (§f7): git-log correlation shows ZERO daemon
      commits touching `queue/*` during the 12:50–14:06 flake window — the
      daemon-race hypothesis is DISPROVEN for the sources; remaining suspect
      (host resource pressure at load ≈100) unproven; capture protocol
      documented in gotchas-tooling-build.
    - **art-dupl baseline provenance** (§f11): last touch is `0dfb8a153`
      (10-23 10:32, daemon) — a SURGICAL +2-group insert (docserver templ ×2;
      architecture/index × discordsync fixture), NOT a wholesale re-pin;
      nothing was absorbed during the 10-25 sessions. The 14:06 report's
      "inferred: baselined during the parallel session's close-out" is WRONG
      — the morning's docserver group stopped flagging because its code shape
      changed (hash-based matching), not because of a re-pin. Errata recorded.
12. **engine-capabilities drift check** (§f22): rc=0, no drift after the
    bigtable migration.
13. **SchemaEvent fleet census** (§f16+§f17): ZERO consumers outside the repo
    (Cyberdom/Kernovia grep hits are their own local symbols); no fleet use
    of the illegal `Schemas().Event[` chain → the cqrs-upgrade codemod
    trigger for this repair is EMPTY; no migration work needed.
14. **Planning + reporting artifacts**: the Pareto plan (HTML, D2 graph
    rendered + inlined), the 15:12 mid-run report, this report, and the
    `docs/status/README.md` index row.

## b) PARTIALLY DONE

1. **Composed verify-chain green run (F17)** — launched
   (`scripts/verify-chain.sh`, durable log
   `build/verify-chain-session-*.log`) but NEVER got a green gate window:
   five attempts refused (load above ceiling 10 at attempts 1/4/5; tree
   changed during the 60 s stability window at attempts 2/3 — daemon
   absorption + an active parallel session's edits). It keeps retrying
   (max-wait 4 h) in the background. The 14:06 asterisk — "every step green
   individually, composed runs flaked" — therefore still lacks its
   continuous-green proof. Check the log for the eventual verdict.
2. **`.buildflow.yml` tripwire** — the gap is named and the proven
   hash-golden pattern is sitting right there, but I did NOT implement it
   (treated as owner-gated via the mechanism ruling). Arguably executable
   as a warn-only gate without the ruling.
3. **NEXT #40 (pin `meta_planned_` prefix as on-disk contract)** and
   **#41 (keycodec key-path doc note re map-key nondeterminism)** —
   deferred deliberately: doc comments feed the api golden
   (`api-stability` captures godoc), and editing them mid-verify-chain
   would have failed the running gate; they need their own edit window +
   `api-stability --update` + gate rerun.
4. **Authored commits** — none: I went to commit at the phase boundary, but
   the daemon had already absorbed everything into three `chore:` commits;
   I verified too late. Third consecutive session with zero authored
   commits (Q3 evidence pile grows).

## c) NOT STARTED (verified, not assumed)

1. **Soak coverage**: bbolt 489 s suite, systemtest real-engine suites —
   quiet-window items, deliberately not started while load was 3×
   oversubscribed.
2. **Owner-gated rows, all untouched**: Q1 `SchemaEvent` shape ratification;
   Q2 `json.Deterministic(true)` adoption wave; Q3 daemon pause/lease;
   envelope spike §12; markdown-format policy (I08/I09/N1); v4 fix-only
   declaration (I10); tooling product line (I27/I29); keycodec-v2 engine
   migration wave (A9) + deployed-DB census; presets retirement;
   metaengine 5-package split; cqrs-upgrade codemod (now confirmed
   trigger-empty for SchemaEvent); wave-2 kickoff.
3. **Routed plans, not mine** (no re-dispatch): 04-25 architecture-review
   follow-up T01–T24 (ledger, package-surface/cycles gates, arch-metrics,
   split plan, API-golden tiering); testing-improvements T01–T10;
   19-19 repo-scale; 19-40 fleet-audit.
4. From the 14:06 NEXT queue, not reached: #19 (compile watcher/daemon
   compile check), #20 (docserver templ residue permanent home), #21
   (bigtable → taxonomy GATED_MODULES), #25 (verify plan §11.7/§11.8
   receipts), #30/#31 (byte-LEVEL stream pins; multimap/log backends),
   #32 (turso planned-path delegation test), #33 (systemtest SchemaEvent
   spec), #34 (TestRecipesCompile hermeticity), #36 (daemon pause), #42–#49.

## d) TOTALLY FUCKED UP (all mine, all disclosed)

1. **`rm` destroyed a pre-existing tracked file.** My splice cleanup ran
   `rm plan-graph.svg` inside `docs/planning/` — which deleted the
   PRE-EXISTING tracked `docs/planning/plan-graph.svg` (a different file
   than my session's temp SVG). Caught on the next `git diff --stat`,
   restored from git within a minute. This is the exact never-generic-rm
   class my own safety rules ban; temp artifacts belong in /tmp with
   name-specific cleanup, never generic globs in tracked directories.
2. **Two compile-broken mutation attempts wasted cycles.** Mutation 1
   (silent fallback) left `errorfamily` unused → build failure, not a RED;
   my first re-try removed the wrong import line (alias form); mutation 2
   (comment placement) split a function argument list → syntax error. A
   build-failed RED proves nothing (the test never ran) and a
   `[no tests to run]` green line is vacuous, not a pass. Only mutation 3
   (wrong-code, compile-safe) was valid. Compile-safe mutation patterns
   (wrong-code string, wrong-shape wrapper) should be the default play.
3. **Ran the gated chain into a wall.** I launched verify-chain while load
   was 24 against a gate ceiling of 10, and the tree-stability assert kept
   failing on daemon absorption — a one-shot `can-run-composed-gate` probe
   would have told me both facts BEFORE the launch. Self-correcting (the
   chain retries for 4 h), but the launch was hope, not the documented
   recipe.
4. **Authored-commit race lost by inattention** (see b-4): the repo
   documents "commit at each phase boundary immediately if you need
   authored history," and I deferred the single commit attempt to the end.
5. **Foreign edits nearly committed by me.** At commit time,
   `docs/METAENGINE_DOMAIN_LANGUAGE.md`, `docs/README.md`, and
   `metaengine/tursoengine/go.{mod,sum}` showed modifications I did NOT
   author (the parallel session, still active). I staged nothing and
   touched nothing — correct outcome, but I reached for `git status` only
   when intending to commit; a foreign-diff check belongs at every phase
   boundary, not just commit time.

## e) WHAT WE SHOULD IMPROVE (systemic)

1. **Temp-artifact rule with teeth**: session temp files go to /tmp (or a
   `build/session-<ts>/`), and cleanup is name-specific — never
   `rm <generic>.<ext>` inside tracked trees.
2. **Compile-safe mutation playbook**: standardize wrong-code /
   wrong-shape-wrapper mutations (package still compiles, semantic pin
   still REDs); document the two invalid-proof shapes (build failure,
   vacuous test-name green) in gotchas-testing.
3. **Gate pre-flight**: probe `can-run-composed-gate` one-shot before
   launching any gated chain; report ceiling/load/stability facts, THEN
   launch the wait-loop version.
4. **Commit at phase boundaries or accept chore history**: pick one (Q3);
   if authored history matters, the commit must happen within minutes of
   the phase, not at session end.
5. **Foreign-diff check at every phase boundary** when parallel sessions
   are active: `git status` + `git diff --stat` scan for files outside my
   edit set before ANY tree-level action.
6. **Doc-comment edits are golden-coupled**: batch them with
   `api-stability --update` at a phase boundary — never mid-gate (bit me
   twice today: deferred #40/#41 for it).
7. **The `.buildflow.yml` hash-golden extension is a 30-minute fix with a
   proven pattern** — stop treating it as a paragraph; propose it as the
   concrete mechanism the moment the owner rules on skip-vs-tripwire.

## f) NEXT — up to 50 things (priority-ordered, this session's evidence only)

1. Claim the continuous green composed run: check
   `build/verify-chain-session-*.log` once the quiet window opens (the
   chain is still retrying; ceiling is load1<10 + 60 s stable tree).
2. Implement the `.buildflow.yml` hash-golden tripwire (clone
   `check-golangci-hash.sh`; wire into `check-lint-config` + pre-commit) —
   or its skip_steps alternative, per Q-mech ruling below.
3. Add `golangci-lint-auto-configure` to `.buildflow.yml` `skip_steps` with
   the three-strike rationale comment (gci ×2 2026-10-10/11; depguard+env
   strip 2026-10-25) — the documented durable-guard decision, now with a
   third same-class incident as evidence.
4. Triage the buildflow run's **hostile-filename formatter failures**:
   dprint choked on `--help/catalog.index.json` and oxfmt on
   `--help/eventcatalog.config.js` — locate the generator producing files
   under a `--help/` directory and rename the output (or formatter-exclude
   it); this breaks every formatter run until fixed.
5. Triage the duplicate-gate split brain the run exposed: buildflow's
   generic `file-size-check` (226 findings) and `cqrs-lint` (415 ERROR
   findings) vs this repo's own baselined ratchets — exclude with rationale
   in `.buildflow.yml` (the buildflow skill's "what the project still
   owns" split) or reconcile baselines.
6. Two-buildflow concurrency hazard: the parallel run logged 100+
   SQLITE_BUSY resume-checkpoint failures sharing one checkpoint DB —
   document "never run two buildflows concurrently in one repo" (or a
   lock) in gotchas-tooling-build.
7. BuildFlow doctor's `workspace/go-work-paths` warning (87 `use` paths
   "missing /v4") smells like a false positive — `use` entries are
   directories, not module paths. Verify at source and file upstream
   (verify-before-filing) next to the known pseudo-version false positive.
8. File/verify the BuildFlow upstream issue for auto-configure stripping
   curated blocks (depguard + env:), evidence: incidents 6-11 + 12
   (`3d8b...`), the gci re-adds, and this repo's hash-golden workaround.
9. Harvest the remaining 14:06 NEXT items into TODO_LIST rows (#19, #20,
   #21, #25, #30–#34, #40–#49) — they currently live only in a timestamped
   report (the entombment class docs-health exists to kill).
10. NEXT #40: document the `meta_planned_` table-name prefix as an on-disk
    contract (COOKBOOK/gotchas) — one edit window + golden regen.
11. NEXT #41: keycodec `EncodeKeyStr`/`EncodeJSON` doc note re map-key
    nondeterminism (same edit window as #10).
12. NEXT #21: decide bigtable → error-taxonomy GATED_MODULES (it now emits
    `keycodec.encode_value` from a non-gated module).
13. NEXT #25: verify the value-integrity plan §11.7/§11.8 receipts actually
    exist as the header claims.
14. NEXT #34: TestRecipesCompile hermeticity — the 56 s + proxy-fetch cold
    run is a gate-latency liability; pin or vendor the snippet-mirror deps.
15. NEXT #33: systemtest end-to-end SchemaEvent spec (upcast + typed decode
    through a real `system.New` boot).
16. NEXT #32: turso planned-path rejection delegation test (planned MapSet
    via sqliteengine — the delegation pin covers standard MapSet only).
17. NEXT #30/#31: byte-LEVEL stream pins (needs raw stream-key access) and
    multimap/log-backend pins (SetBackend/MultimapBackend also ride
    EncodeJSONErr — unpinned).
18. Q2 wave prep (no behavior change): inventory every value-encoder seam +
    every byte-expectation pin into a readiness checklist so the
    `Deterministic(true)` sweep is a mechanical wave when ratified.
19. Key-nondeterminism demonstration pin: a scratch-doc or failing-by-design
    example showing map-key storage-key drift (educational, feeds Q2
    urgency honestly).
20. System release train: next `system` tag must carry `SchemaEvent` (new
    export) — version the bump accordingly.
21. Dogfood `SchemaEvent`: adopt the builder in one first-party example
    (getting-started or goal-shaped-app) — census says zero consumers; the
    builder needs its first real user before Q1 ratification.
22. SKILL.md / core.md: check whether the composition quickstart mentions
    the schema builder; add a pointer to §2.46 + `schema_declarations.go`.
23. system/doc.go: add the schema-declaration pointer (README now has it;
    doc.go still doesn't).
24. Update the Pareto plan's F17 row when the chain lands (docs-health
    ANNOTATE, non-destructive).
25. Load-sweep (`nix run .#load-sweep`) before the next full `#verify`
    (routine hygiene; no timing paths touched this session).
26. Re-run `bash scripts/census-fleet.sh` to refresh the delete-at-zero
    trigger report (last run 2026-10-24; cheap, keeps §I-row honest).
27. Add the `/mnt/buildcache` "no such file or directory" link errors from
    the buildflow run (cqrs-bench link, stack/bbolt compile) as another
    instance in the 2026-10-23 disk-full runbook (same
    cache-eviction-under-concurrency family; the bumps auto-reverted, so
    no go.mod damage).
28. Triage buildflow's `errcheck fixer` stale path
    (`example/metaengine-quickstart/example/metaengine-quickstart/…` —
    doubled segment): stale module-fan-out cache entry vs a real moved
    file; `BUILDFLOW_NO_RESULT_CACHE=1` was set, so this is the OTHER
    cache.
29. Triage the `branching-flow` static warnings for real bugs:
    systemscenario `equivalence.go:79-80`, `then.go:121-128` slice-index
    panics, `storage/bbolt/store.go:225` index panic — each needs a pin or
    a guard, not a wave-off.
30. Same triage batch: `storage/memory/log_store.go:146/75` swap-hazard
    params (brand them) and `log_store.go:160` index panic.
31. Decide `interrogate` + the "9 tools unavailable" buildflow health
    failures: install, or documented-exclude in `.buildflow.yml`.
32. Investigate the `--fix` golangci-lint-auto-configure `issues:` reset
    (50/10 → 0/0): confirm the golden's 50/10 is deliberate curation and
    note it in the tripwire golden docs (it is pinned, so protected — but
    the rationale should live beside the golden).
33. lychee excludes: add the 7 archived doc dirs to `lychee.toml`
    (`exclude_path`) — 5-minute mechanical fix the preflight keeps
    demanding.
34. flake-meta-checker: add the missing `homepage` attributes to the 3
    flake.nix meta blocks (or exclude the check with rationale).
35. `go-humanize-linter` finding (`catalog/architecture/section_shared.go:52`
    manual pluralization): one-line fix or nolint with reason.
36. `canonical-helper-reimplementation` (3 sites: benchkit `Truncate`,
    deriver `Filter`, storage/turso `Filter`): dedup-audit playbook
    refactors, small.
37. Example lint debt from the run: getting-started `bc` naming, mesh-demo
    17 findings (const extraction, godoc dedup), metaengine-quickstart
    errcheck + 0o600 magic number — one example-hygiene pass.
38. scheduler-otel-status SA1019s on deprecated `scheduling/sqlstore` are
    EXPECTED (the exempt banners demo) — confirm the exemption is
    documented where the module's deprecation is tracked, not just in the
    example.
39. Record in gotchas-module-management: the go.sum verdict (extra
    `/go.mod` hashes are canonical; per-module strict tidy is NOT the
    gate) so no future session "fixes" them again.
40. Record the compile-safe mutation playbook in gotchas-testing (from §e-2).
41. Record the temp-artifact rule + today's `plan-graph.svg` near-miss in
    the junk-file gotcha section (gotchas-tooling-build).
42. Record "gate pre-flight before launch" in the composed-gate recipe
    gotcha (from §e-3).
43. AGENTS.md memory: buildflow `--fix --build-mode=full` in THIS repo
    requires a post-run config diff (incident 12 cross-link).
44. AGENTS.md memory: verify-chain ceiling facts (load1<10, 60 s stable
    tree) next to the composed-gate recipe.
45. AGENTS.md memory: SchemaEvent zero-consumer census result + codemod
    no-go for this repair.
46. Consider `SchemaSet.Build()` duplicate detection moving INTO
    `SchemaEvent` (fail at call time) — behavior change, owner-gated; row
    it if wanted.
47. `SchemaDeclarations`/`SchemaSet` should join the experimental tier when
    API-golden tiering (04-25 plan T06) lands — one-line coordination note.
48. Nightly-bench + verify-lock: unaffected this session; confirm the
    nightly timer absorbed no surprise (routine check).
49. Q1/Q2/Q3 rulings (carried; they gate the biggest remaining waves).
50. Wave-2 kickoff session once rulings land (carried from 14:06 §f50).

## g) QUESTIONS (cannot be derived from the repo)

1. **Config-guard mechanism ruling (now 3 strikes of evidence):** may I
   implement the `.buildflow.yml` hash-golden tripwire AND add
   `golangci-lint-auto-configure` to `skip_steps` with the three-incident
   rationale — or do you want the durable fix upstream in BuildFlow first
   (issue per verify-before-filing), with this repo just restoring on
   detection until then?
2. **Duplicate-gate split brain:** buildflow's generic `file-size-check`
   (226 findings) and `cqrs-lint` (415 ERROR findings) re-report what this
   repo's own baselined gates already ratchet. Exclude them via
   `skip_steps` with rationale (project owns its curation per the buildflow
   skill), or do you want the two baselines reconciled into one source of
   truth?
3. **Authored history (third consecutive zero-authored session):** should
   sessions author commits at each phase boundary (I will, going forward,
   if you say yes), or is daemon `chore:` history acceptable as the
   permanent record — i.e., is Q3's pause/lease still wanted at all?

---

*Evidence: every green claim above is a gate or suite rc=0 observed this
session; every mutation was restored byte-identical (`git status` verified
after each); foreign edits found at commit time were left untouched.*
