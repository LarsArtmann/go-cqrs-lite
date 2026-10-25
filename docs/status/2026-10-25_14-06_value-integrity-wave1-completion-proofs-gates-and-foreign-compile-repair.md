# Value-Integrity Wave 1 Completion — Proofs, Gate Unblock, Foreign Compile Repair (2026-10-25 14:06)

Session: 2026-10-25 ~12:50–14:06 CET (afternoon continuation of the 10:46
session). Scope: the recommended queue from the 10:46 status report — T2.1,
T3, LSM + planned byte pins, T9, T10, T12, T13, T16, T17, T24 — plus one
unplanned foreign compile repair. All work verified step-by-step; receipts in
plan §11 (`docs/planning/2026-10-24_12-09_SUPERB-value-integrity-and-gate-unblock-pareto-plan.md`).

Tree at close: clean (auto-commit daemon absorbed every phase, including the
close-out docs; zero authored commits this session — second consecutive
session with this outcome, see g) Q3).

---

## a) FULLY DONE (evidence in parentheses)

1. **T2.1 — batch all-or-nothing mutation proof** (3 mutations, baseline
   green before, green after):
   - Lazy per-row encoding in `StreamAppend` → RED
     (`stream version = 1 after a rejected batch, want 0`).
   - Same lazy encoding INSIDE `StreamAppendExpected`'s `RunInTx` → GREEN —
     honest finding, not a test gap: the transaction rollback genuinely makes
     lazy encoding safe there. Atomicity on that path is tx-backed.
   - Bypassing `RunInTx` entirely (version check + lazy inserts, no tx) → RED
     (`stream version = 1, want 0`). Tests have teeth for both mechanisms.
2. **T3 — turso delegation proof**
   (`metaengine/tursoengine/value_encode_rejection_test.go`): MapSet →
   Rejection + `sqliteengine.encode_value`; StreamAppend bad batch →
   Rejection + `metaengine.encode_stream_value` AND version 0. Both ran on
   the embedded libSQL driver (not skipped, 0.07 s each). errorfamily
   promoted indirect→direct in tursoengine go.mod (tidy-verified). Full
   turso short suite green (22 s).
3. **Byte pins — LSM engines** (`stored_bytes_pin_test.go` in
   pebbleengine/bboltengine/badgerengine): literal expectations
   (`{"title":"pin","n":7}`, `"raw-string"`, `42`) against raw reads —
   pebble via `RawValueReader.GetRawValue`, bbolt via `FromDB` bucket read
   (`cqrs_meta`, keycodec key layout pinned as on-disk contract), badger via
   `FromDB` txn read. 30× stability runs clean per engine.
4. **Byte pins — sqlite remaining write paths**
   (`stored_bytes_write_paths_pin_test.go`): planned MapSet +
   planned MapUpdate (`meta_planned_*` value column) + MapUpdate re-encode,
   all literal-expectation raw-column reads.
5. **Mutation proof of the LSM pin class**: envelope-wrap mutation
   (`json.Marshal(struct{ V any }{v})`) in `keycodec.EncodeJSONErr` → ALL
   THREE LSM byte pins RED with exact `{"V":{...}}` evidence; `%v`-fallback
   mutation → `TestValueEncodeRejection` RED ("must fail, got nil").
   Restored byte-identical; keycodec + pebble + badger + bbolt(SOAK_SKIP)
   suites green after restore.
6. **NEW DEFECT DISCOVERED + VERIFIED**: `encoding/json/v2` does NOT sort
   map keys (v1 did) — persisted `map[string]any` bytes are iteration-order
   dependent. Proven: identical 3-key map marshaled to two different orders
   in one process; pebble 2-key byte pin failed within 50 `-count` runs; Go
   source `encoding/json/v2_options.go` documents the behavior change. Fix
   form VERIFIED working: `json.Marshal(v, json.Deterministic(true))`
   (sorted, stable across runs). Recorded in TODO_LIST (new row) + plan §11.
   All pins converted to structs (declaration order = only deterministic
   shape).
7. **T9 — CLOSED as verified-no-defect**: all four graph backends
   (mysql/duckdb/pg/badger) persist ONLY `(collection, from_node, to_node)`
   identity rows/keys — there is no payload-value path; the
   `encodeJSONString`/`graphNodeKeyJSON` helpers are key-identity lenient by
   the documented contract (same class as `encodeKey`/`indexKeyValue`).
   COOKBOOK "Still open" clause replaced with the verified closure.
8. **T10 — bigtableengine migrated**: cell-value writer →
   `keycodec.EncodeJSONErr` (code `keycodec.encode_value`), local `encodeJSON`
   `%v` helper deleted (its art-dupl accept comment went with it),
   `adttest.AssertValueEncodeRejection` wired on the in-process bttest fake.
   Suite green (0.089 s). Silent-loss class is now 4/4 closed
   (sqlite/turso, LSM ×3, bigtable; memory n/a; graphs n/a).
9. **FOREIGN COMPILE REPAIR — system module was UNBUILDABLE at HEAD ~4.5 h**:
   `SchemaSet.Event[T]` (a generic METHOD — illegal Go, "method must have no
   type parameters") landed uncompiled in the daemon's 08:40 absorption of
   the parallel session's work; its test + example used the same illegal
   chain. Verified no active owner (pgrep; only an unrelated oxlint LSP in
   another project; file stale 4.5 h). Repaired to package-level generic
   `system.SchemaEvent[T]` (set-first argument — the only legal form that
   preserves the typed decode and no-sample inference; `EventRegistration`
   handlers need compile-time type params). Name collision with the existing
   `system.Event` alias forced the `Schema` prefix. Test + example +
   recipes.md §2.46 fence all rewritten; all three schema specs pass; api
   golden regenerated (+1 export `system/func SchemaEvent`).
10. **T12 — file-size gate GREEN**: three NEW offenders (none baselined)
    split along ownership seams — `metaengine/layout.go` (360→164, new
    `layout_typed.go` 207), `system/adapter_event.go` (352→230, new
    `adapter_event_reads.go` 137), `scheduling/sqlstore/store.go` (382→276,
    new `envelope.go` 123). Module suites green after each split; gate:
    "no new offenders, no growth".
11. **T13 — duplication gate GREEN**: 0 new clone groups (baseline 189). The
    morning's docserver-templ signature group no longer flags (inferred:
    baselined during the parallel session's close-out — attribution NOT
    verified via git log on `.art-dupl-baseline.json`).
12. **T16 — taxonomy generator: NO DRIFT**: regenerated
    `docs/error-taxonomy.md` → byte-identical; gate green (544 codes / 20
    modules / 574 doc claims).
13. **T17 — every #verify-fast step green** (individually; see d)-3 for the
    composed-run caveat): nix-hashes, verify-docs, build, vet, short tests
    (all modules), short race (whole workspace), lint (all 100 modules ×2
    green runs), check-arch, check-lint-config, docserver-css, duplication,
    architecture-reference, turso-citations, templ, bench-gate, coverage,
    md-go (1519 blocks), api-stability (+`SchemaEvent` golden),
    TestRecipesCompile, doc-check RC=0, check-changelog-symbols (11
    citations honest).
14. **T24 — harvest**: TODO_LIST silent-loss row closed (4/4) + new
    json/v2-map-order row; CHANGELOG [Unreleased] bigtable Changed entry +
    system-compile Fixed entry; COOKBOOK de-staled; plan §11 corrected (the
    false "T1–T3 pre-shipped" overclaim now carries a dated correction) and
    extended with full session receipts; api golden regenerated.

## b) PARTIALLY DONE

1. **Mutation receipts for three pin groups NOT run**: the sqlite
   planned-table/MapUpdate pins, the turso delegation test, and the
   bigtable adttest pin have no dedicated mutation proofs this session
   (their encoder seams share the proven keycodec mutation, but e.g. a
   planned-path-specific mutation — `execPlannedSet` re-encoding through a
   different helper — was never executed).
2. **Soak coverage skipped everywhere**: all suites ran `-short`;
   bboltengine's full 489 s soak suite never ran (SOAK_SKIP_BOLT=1
   throughout); systemtest real-engine suites ran only in their short form.
3. **T11.2 unverified**: the plan rows "update metaengine/README.md codec
   section with the LSM closure" — I never checked metaengine/README.md
   this session (COOKBOOK was the verified surface).
4. **CHANGELOG asymmetry**: the schema-declaration BUILDER feature (the
   parallel session's new API) has no `[Unreleased]` **Added** entry — only
   my **Fixed** repair entry describes it. The feature's own changelog row
   is missing (it never existed; I noticed at report time).

## c) NOT STARTED (verified, not assumed)

1. Plan tasks beyond Wave 1 (T18+, incl. the §12 envelope spike / ADR-0066
   reversal) — untouched; their claimed state is inherited from the parallel
   session's §11.7/§11.8 receipts, which I did NOT re-verify.
2. json/v2 `Deterministic(true)` adoption — recorded + fix form verified,
   deliberately not executed (own wave, owner decision pending, g) Q2).
3. `queue` lint-flake root cause (one green/green re-run is circumstantial,
   not a cause).
4. badger first-run transient root cause (see d)-1).
5. Authored-commit mechanism / daemon lease (g) Q3 — carried over, second
   occurrence.

## d) TOTALLY FUCKED UP (all mine, all disclosed)

1. **Badger pin failed ONCE on its very first run and I moved on without
   the failure text.** I re-ran it 15× green + full suite green and
   continued, planning to "capture it if it recurs." It didn't recur. That
   is an unexplained transient left un-root-caused — the exact class of
   thing this repo's flake-cures doc exists for. Minimum honest treatment:
   capture output, then decide.
2. **First diagnosis of verify-fast failure #1 was WRONG.** I saw all-ok
   test lines + a bare `FAIL` (clipped by `tail -40`), ran direct workspace
   short+race sweeps (green), and told the user "flaky under load1≈20".
   Composed run #2 then revealed the REAL cause: `TestRecipesCompile` —
   recipes.md §2.46 used the illegal `Schemas().Event[T]` chain (a real
   defect, fixed). My direct sweeps missed it (different
   invocation/env than the gate — see e)-2). I labeled a real defect as
   flake. The user-visible narrative ("flake") was false for ~10 minutes.
   Note: failure #1's actual content is UNRECOVERABLE (clipped); that it
   was the recipes defect is an inference from run #2, not a fact.
3. **No single continuous green composed #verify-fast run exists from this
   session.** Three composed runs: #1 FAIL (cause clipped, inferred =
   recipes), #2 FAIL (recipes — real, fixed), #3 FAIL (queue/queue-sqlite/
   queue-postgres lint findings that vanish on immediate re-run ×2). Every
   individual step was then proven green — a legitimate decomposition, but
   "the composed gate passed" is NOT a claim I can make; "every step passed
   individually, composed runs flaked twice" is the honest claim.
4. **Shell-surgery sloppiness on splits**: I edited imports via `sed -i`
   and reconstructed-file tricks (adapter_event.go, layout.go) instead of
   the edit tool; the sed left a doubled blank line + unused import that
   needed two repair rounds. Recovered, but the edit tool would have done
   it in one pass.
5. **Hook trap hit AGAIN**: chained an `rg -n` after a `go test -run`
   command and tripped the rg-replace guard a further time — this host has
   a documented three-strike history; I added strike four at session start.

## e) WHAT WE SHOULD IMPROVE (systemic, from this session's failures)

1. **Capture FULL gate logs to files, always.** `nix run .#verify-fast >
   build/verify-<ts>.log` — the `tail -40` clip cost a 10-minute re-run and
   produced a wrong diagnosis. The repo already has
   `scripts/verify-chain.sh` with durable logs; ad-hoc invocations should
   follow the same discipline.
2. **Env parity when decomposing a failed gate.** My direct `go test ./...`
   sweeps (workspace mode, go-env.sh caches) are NOT the gate's per-module
   GOWORK=off invocation — TestRecipesCompile failed in the gate while my
   direct sweep passed. When a composed step fails, re-run THAT step via
   its nix app, not an approximation.
3. **Mutation-proof every pin class in the same session it lands** — the
   pin-without-teeth is the silent lie this whole wave exists to kill, and
   three of my pin groups still lack their own receipts (b)-1).
4. **One-off failures get captured, not outrun** (d)-1: a single unexplained
   red is a signal, not noise, until its output is on disk.
5. **Compile gate before daemon absorption**: the system module sat
   unbuildable ~4.5 h because nothing compiles between daemon commits. A
   pre-commit or periodic `go build ./...` watcher (or making the daemon
   run a fast compile check) kills this class. The 2026-10-24 pgengine
   incident is the same family — second occurrence in two days.
6. **New-engine-API landing needs its CHANGELOG row in the same window**
   (b)-4: the builder feature shipped its Fixed entry before its Added one.

## f) NEXT — up to 50 things, rough priority order

1. Owner decision: json/v2 `Deterministic(true)` wave (g) Q2) — then sweep
   every value-encoder seam (`keycodec.EncodeJSONErr`,
   `sqliteengine.encodeValue`, `metaengine.EncodeStreamValueErr`) + update
   all byte expectations to sorted form.
2. Add TODO row: **map-typed STORAGE KEYS are also non-deterministic**
   (`EncodeKeyStr` on a `map[string]any` key → key bytes vary per run → the
   same logical key can land on different storage keys; distinct from the
   value-bytes finding, worse blast radius).
3. Mutation receipts: planned-path/MapUpdate pins (mutate `execPlannedSet`).
4. Mutation receipts: turso delegation test (mutate sqliteengine encoder,
   run turso suite).
5. Mutation receipt: bigtable adttest pin (mutate keycodec, run bigtable).
6. Root-cause badger first-run transient: run pin suite 100× under load,
   capture output on failure.
7. Root-cause queue lint flake: correlate lint timestamps with daemon
   commit times on queue/*; if daemon race confirmed, document in
   gotchas-tooling-build.md.
8. One continuous green composed #verify-fast (or `#verify`) run for the
   record — ideally via `scripts/verify-chain.sh` (durable log, retries
   only load-guard rc=12).
9. Full soak runs (non-short): bboltengine 489 s suite; systemtest
   real-engine suites.
10. Verify T11.2: metaengine/README.md codec section vs LSM closure.
11. Verify who/when re-pinned `.art-dupl-baseline.json` (187→189 groups) —
    `git log -p -- .art-dupl-baseline.json`.
12. CHANGELOG: write the missing **Added** entry for the schema-declaration
    builder feature (Schemas/SchemaEvent/SchemaDeclarations).
13. Owner decision: ratify `system.SchemaEvent` shape (g) Q1) — or spike a
    fluent alternative (per-event builder types).
14. system/schema_declarations: godoc pass + link from system README/ doc.go
    (the feature has no README presence).
15. system train: next release should carry SchemaEvent (new export;
    version bump when tagging).
16. cqrs-upgrade codemod: only if fleet census finds consumers of the
    illegal `Schemas().Event[` form (expected zero — verify first).
17. Census: `bash scripts/census-fleet.sh` for system.Schemas/SchemaEvent
    consumers.
18. Envelope spike §12 (ADR-0066 reversal, stamped engine-value envelope) —
    the parked endgame for string/JSON ambiguity.
19. Compile watcher / daemon pre-commit compile check (e)-5).
20. docserver templ clone residue (3 baselined groups): permanent-home
    decision (baseline vs restructure).
21. bigtableengine: consider adding to taxonomy GATED_MODULES (it now emits
    `keycodec.encode_value` from a non-gated module).
22. engine-capabilities.md: `--update` drift check after the bigtable
    migration (generated doc, cheap).
23. recipes.md: add a "pin your engine's stored bytes" recipe
    (GetRawValue / FromDB patterns from this session).
24. COOKBOOK: pointer from "Stored values" to the four pin files.
25. Verify plan §11.7/§11.8 receipts exist as claimed in the header note
    (inherited, unverified).
26. TODO_LIST sweep: grep for now-stale "graph encoders"/"bigtable lenient"
    mentions outside the closed row.
27. AGENTS.md gotchas: record direct-vs-gate env parity lesson (e)-2).
28. AGENTS.md gotchas (language): json/v2 map-order non-determinism +
    `json.Deterministic(true)` — cross-project Go footgun.
29. LSM pins: MapUpdate re-encode pins for pebble/bbolt/badger (only MapSet
    + stream pinned so far).
30. LSM pins: byte-LEVEL stream pins (current pebble stream pin is
    decode-level; needs raw stream-key access — internal test package or
    a RawStreamReader test hook).
31. LSM pins: multimap/log backends (SetBackend/MultimapBackend/LogBackend
    also ride EncodeJSONErr — unpinned).
32. turso: planned-path rejection delegation test (planned MapSet via
    sqliteengine delegation).
33. systemtest: end-to-end SchemaEvent spec (upcast + decoder through a
    real system.New boot).
34. TestRecipesCompile hermeticity: 56 s + proxy fetch per cold run — pin
    or vendor the snippet mirror deps.
35. Quiet-window discipline: heavy gates under `#quiet-window-run` when
    load1 > 15 (was ~20 all session).
36. Daemon pause/lease mechanism (g) Q3).
37. Review the parallel session's 08:40 502-file absorption (one file didn't
    compile; what else landed unreviewed?).
38. `system.Schemas` empty-set semantics: `Build()` on zero events returns
    empty decls + nil decoder — deliberate? Document or guard.
39. `SchemaSet` concurrency: doc says not concurrent-safe; consider asserting
    or making it safe by construction (composition-time use only).
40. Pin the `meta_planned_` table-name prefix as a documented on-disk
    contract (my pins hard-code it; a constants doc row or exported const).
41. keycodec: doc note on EncodeJSON key path re map-key non-determinism
    (ties to #2).
42. `check-engine-capabilities --self-test` after any engine change
    (routine hygiene; ran green implicitly in verify-fast, self-test cheap).
43. Load-sweep (`nix run .#load-sweep`) after touching timing paths — none
    touched this session; run before the next `#verify` anyway.
44. Status index lint: confirm the 10:46 header-year edit didn't orphan
    older rows (quick read of docs/status/README.md).
45. Next pareto re-rank over the refreshed TODO_LIST (post-wave).
46. bbolt soak determinism: the 489 s suite under race once (slow, do in a
    quiet window).
47. badger pin: add versioned/MapSetAt-equivalent surface if badger exposes
    temporal writes (capability check first).
48. Consider a tiny `metaengine.RawStreamValueReader` capability (pebble
    first) to enable #30 cleanly.
49. CHANGELOG symbol gate: add `system.SchemaEvent` citations to the missing
    Added entry (#12) so the gate covers it.
50. Wave-2 kickoff session once Q1–Q3 are answered.

## g) QUESTIONS (cannot be figured out from the repo)

1. **SchemaEvent API shape:** I repaired the parallel session's builder to
   `system.SchemaEvent[T](set, type, version, ops...)` because Go forbids
   generic methods — but that reshaped someone else's in-flight API (fluent
   chaining became inside-out nesting). Ratify this shape, or do you want a
   different design (e.g., delete the feature until its author returns, or
   per-event builder objects that stay fluent)? This is taste/ownership,
   not derivable.
2. **json/v2 deterministic map bytes:** adopt `json.Deterministic(true)` at
   the value-encoder seams (restores v1's sorted-map-bytes durability
   property; changes byte layout for map values; touches every engine +
   all byte pins), or accept non-determinism and keep pins struct-only?
   Both are defensible; it is a durability-semantics product call.
3. **Daemon vs authored history:** every close-out this session (and the
   morning's) landed as `chore: auto-commit` — the daemon won every race,
   and no authored commit exists for the wave. Is that acceptable as the
   permanent record, or should a daemon pause/lease (or a "commit now"
   escape hatch) be built before the next wave? (Carried over from the
   10:46 report, now with a second full session as evidence.)

---

*Evidence index: plan §11 entries dated 2026-10-25 (afternoon + cont.);
gate receipts inline above; all mutation artifacts restored byte-identical
(`git status` clean at close).*
