# Status Report — 2026-09-01 23:09 — TODO-Execution Session: G1 Calibration Close-Out, CI Drift Repair, Pin Sweep

> **Session scope:** executed the actionable (non-user-blocked) items from TODO_LIST.md.
> Baseline commit at session start: `e69049a07` (clean tree). All work below is in the
> working tree (95+ files modified, uncommitted — auto-commit daemon owns commits; I do
> not commit without explicit instruction).
> **Honesty note:** every "green" claim below was verified in-session with the exact
> command named. No stale GREENs. Ambient load during benches: 4.8–12 (recorded per run).

---

## a) FULLY DONE (verified green in-session)

### 1. 🔥 G1 — ReadAggregate calibration reconciliation (ADR-0133) — CLOSED

Every engine's `NsPerAggregate` now prices the actual `ReadAggregate` execution path
(`CounterBackend.CounterGet`), measured in live windows, count=5, run 1 discarded, median
of the rest, ambient load recorded (calibration protocol):

| Engine | Old | New (ns/row) | Median measured | Window | Bench |
|---|---|---|---|---|---|
| pg | 150 (SQL-SUM era) | **250** | 246.0 (242–248K ns/op / 1K rows, ±2%) | ephemeral nixpkgs PG (scripts/ephemeral-pg.sh, `go -C` + GOWORK=off) | `BenchmarkCalibration_Postgres_CounterGet` (existed) |
| mysql | 150 (DIVERGENCE) | **320** | 320.8 | userspace MariaDB 11.4.12 :33061 (fresh datadir init) | `BenchmarkCalibration_MySQL_CounterGet` (**NEW** — written this session) |
| dgraph | 950_000 (DIVERGENCE) | **2_700** | 2663.0 (cold run 3.8ms discarded) | ephemeral Dgraph 25.4.0 (`nix run .#integration-dgraph`) | `BenchmarkDgraph_CounterGet` (existed) |

The dgraph old value misused a GraphNeighbors depth-3 **per-OP** number as a **per-ROW**
constant — planner overpriced dgraph aggregates ~350×. Routing order vs SQL engines is
preserved (dgraph still 8–10× costlier per row), only the absurd factor is gone.

Aligned in the same change: `pgengine/mysqlengine/dgraphengine` engine.go constants +
comments (last aggregate DIVERGENCE markers removed), pg `AggregateSum` bench comment
re-scoped to "documents the planner-bypassing typed path", calibration baseline doc
(`docs/benchmarks/calibration-2026-08-30.md`: 3 new table rows, G1 raw-runs section,
drift-job cross-check corrected from "Planned" to "Shipped"), AGENTS.md ReadCosts gotcha,
CHANGELOG [Unreleased] entry amended, FEATURES.md cost-calibration row (stale "pending
recalibration" text removed — caught during self-review, see §d).

**Gates run:** builds + full test suites + `nix run .#lint-module` ×3 engines (one godot
nit fixed), `check-changelog-symbols.sh` (143 citations), `cmd/doc-check` (956 refs),
`check-doc-links.sh` (627 targets), `gofumpt -l` clean on all touched .go files.

### 2. CI repair — four of the drift/bug classes fixed and locally verified

1. **API Stability workflow bug** (.github/workflows/ci.yml:334): `go run main.go` on a
   multi-file package (`undefined: collectExports`) → `go run -tags "goexperiment.jsonv2" .`
   (canonical form). Verified locally: 4368 exports OK + `cmd/api-stability` meta-tests ok
   + actionlint clean.
2. **shfmt drift**: `scripts/calibration-drift.sh` + `scripts/ephemeral-dgraph.sh`
   reformatted with the CI-pinned shfmt; `shfmt -d scripts/` now exits 0.
3. **Skill-doc TOC gates**: `references/core.md` (462→475 lines) and `references/faq.md`
   (301→326) gained house-style `**Contents**` blockquote TOCs matching recipes.md
   (anchors follow GitHub slug rules). CI job's exact check replicated locally: PASS.
4. **go.work↔flake sync job**: `check-workspace-sync.sh` substring-matched `../go-codec`
   etc. as `./go-codec` — now only parses `./`-prefixed use-lines; external sibling
   checkouts no longer leak into this repo's sync check. Verified OK; shfmt clean.

### 3. Version-drift consumer-pin sweep (CI version-drift job)

- All 8 drifted siblings aligned to latest tags: listing v4.3.0, sqliteengine v4.2.0,
  middleware v4.5.1, query v4.7.0, schema v4.3.1, snapshot v4.4.0,
  testutil/pgtestcontainer v4.1.0, watermill v4.5.1 — **131 pin bumps**, then
  **14 storage bumps → v4.8.1** (see §d for why), `go mod tidy` per module
  (proxy-routed, `GOPRIVATE=`).
- **The sweep surfaced a real stranding** (see §d-3) and closed it: published
  storage ≤v4.8.0 does not compile against listing v4.3.0 (`sql_aggregate_reader.go:161`
  TombstoneStatus→Status cast); `storage/v4.8.1` (2026-08-29) is the listing-compatible
  re-tag — every storage consumer now pins it, including example/getting-started whose
  stale v4.6.0 pin also carried the OLD documented TimerID stranding.
- **Verification:** 48 touched modules × `GOWORK=off go test -tags "goexperiment.jsonv2"
  -run ZZNONE -count=1 ./...` — 48/48 GREEN. `check-version-drift.sh` GREEN.
  `check-replace-directives.sh` GREEN.

### 4. Process debt + stale-TODO verification

- **`nix run .#load-sweep` GREEN** under load avg 9–12 (all middleware/scheduling/storage
  timing-assertion suites survived CPU soakers). The skipped-discipline TODO is paid.
- **Two TODO items were already shipped** by session-7 (`8e7dcddfe`) and verified by me:
  golangci-lint LSP disk-cache wrapper (installed at ~/.local/bin, home-managed) and
  ephemeral-dgraph.sh bounded health wait (`HEALTH_TIMEOUT`, DGRAPH_HEALTH_TIMEOUT
  override — exercised live in my dgraph window: "budget 60s … Alpha healthy").
  Both marked done in TODO_LIST with evidence.
- **TODO_LIST.md updated** (6 items closed/rewritten with evidence; CI item now PARTIAL
  with precise done/blocked split).
- **AGENTS.md MariaDB procedure corrected** (see §d-1): fresh-datadir init path, the
  invoking-user socket-auth fact, and the mvdan/sh `$!` trap. doc-check re-green after.

---

## b) PARTIALLY DONE

1. **Repair master CI (11 red jobs)** — fixed this session: API Stability workflow bug,
   shfmt drift, skill-doc TOC gates, version drift, go.work-sync regex. Still red-making
   (all blocked outside the repo, see §c): (a) FlakeHub unauthenticated + magic-nix-cache
   HTTP 418 (infra), (c) cqrs-lint Self-Lint go-finding credentials (auth), billing
   (user), and **29 production files >350 lines** (the CI File Size job) — a standalone
   multi-session refactor wave I deliberately did NOT rush at session end.
2. **cqrs-lint ApplyLayout rule** — untouched (needs a design pass for type-info in the
   analyzer; scoped in TODO since session 6).
3. **"Consolidate indirect dep references" TODO** — my sweep aligned INTERNAL sibling
   pins; the transitive `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}`
   EXTERNAL-module indirects in ~49 go.mods remain (they clear at the next tag wave).

---

## c) NOT STARTED (user-gated or v5-scope; not touched this session)

- The **pending v4 tag wave** (L-effort, BLOCKED on user authorization) — note my pin
  sweep changes its starting state: storage consumers already at v4.8.1, all sibling
  pins at latest published tags. The wave now only needs to tag the modules with
  untagged prod changes.
- Dead eventtest tags (needs remote tag deletion decision), go-codec F46 (uncommitted in
  ../go-codec), iroh P99 ratification, transport/http+grpc final v4.x tags,
  replace-drop sweep (after wave), GitHub Releases (gh auth), macOS PG verification,
  integration-mysql-nspawn (root), CI billing (user), the whole v5 deletion/cut train
  (stack, view, relational, tombstone API, transport, StreamRef validation, snapshot
  wire tags…), cqrs-lint exclusion-block audit tail, dgraph filtered/scan per-row
  recalibration (needs result-size-scaled benches — flagged in calibration doc).

---

## d) TOTALLY FUCKED UP (honest mistakes this session — all recovered, all lessons written down)

1. **Leaked a mariadbd process on first MariaDB attempt.** `$!` in the mvdan/sh tool
   shell returns a job id ("g1"), not a numeric PID — my `kill $MYPID` failed silently
   and the server kept running. Caught it on the next call (pgrep) and killed it.
   Also wasted a liveness-wait loop on a wrong assumption: AGENTS said "root logs in
   via the socket" — false for a FRESH userspace `mariadb-install-db` (unix_socket maps
   DB-root↔OS-root; the invoking user `lars` is the socket superuser). Two fixture
   errors in one command. Both traps now encoded in AGENTS.md.
2. **AGENTS.md MariaDB memory was stale before I started** ("datadir persists across
   sessions" — /tmp was wiped; binary symlink gone too). I followed it verbatim and
   hit the wall instead of pre-checking `ls /tmp/mariadb-cqrs` first. Lesson recorded;
   AGENTS corrected (persistence claim now says VOLATILE).
3. **Pin sweep executed before predicting the stranding.** I ran 131 bumps and only
   discovered the listing v4.3.0 ↔ storage ≤v4.8.0 incompatibility via 13 red modules
   in the verification pass. The playbook for EXACTLY this class sits in AGENTS
   ("codec-critical methods require a consumer-pin sweep in the same wave… always run
   the GOWORK=off build matrix"). I ran the matrix (good) but should have REASONED the
   stranding risk upfront (listing v4.3.0 changed the Status type; storage's fix is in
   the pending wave). Cost: one ~4-minute wasted verify cycle. Mitigation worked as
   designed; prediction failed.
4. **FEATURES.md nearly left lying.** The cost-calibration row still said "mysql/dgraph
   pending live recalibration (DIVERGENCE notes)" AFTER I had closed G1 — caught only
   during this self-review, not during the task's gate pass. The per-task gate list for
   doc-affecting changes should include FEATURES.md explicitly (now noted in §e).
5. **Minor:** first mysql bench comment tripped godot (missing period) — caught by
   `lint-module`, fixed immediately. First `git show storage/v4.8.1:go.mod` used the
   wrong path (tag:repo-root vs tag:storage/go.mod) — one wasted probe, self-corrected.

No user-visible breakage ships from this session: every failure above was caught by a
gate before the session ended, and the tree's final state is fully gate-verified.

---

## e) WHAT WE SHOULD IMPROVE

1. **Pin sweeps deserve a pre-flight stranding check**: before bumping sibling X to
   latest, `git diff <old-tag>..<new-tag> -- .` for API-breaking type changes and grep
   which published consumers compile against the new types. Cheaper than a red matrix.
2. **TestRealProfiles_ReadCostsPinned pins only bbolt/pebble** because pg/mysql/dgraph
   Profile() needs a live server to construct. A static `Pgv4Profile()`-style
   constructor (or pinning the exported constants) would let the routing-regression test
   pin ALL engines' ReadCosts — the protocol says constant changes ship with a pin
   update; today pg/mysql/dgraph constants are only pinned by prose in the doc.
3. **Engine README cost tables exist only for badger/bbolt/pebble** — pg/mysql/dgraph/
   sqlite/duckdb READMEs lack the "Calibrated per-pattern read costs" table. Same
   information, five places it's missing.
4. **`BenchmarkDgraph_CounterGet` asserts only "non-empty"** — on a persistent server
   with leftovers the counter count could exceed 1000 and silently skew per-row math.
   Should assert the exact seeded size (my window was DropAll-clean, so today's constant
   is honest, but the bench is one shared-persistent-server run away from drift).
5. **Per-task gate checklist should name FEATURES.md** for behavior-affecting rows (§d-4).
6. **check-version-drift.sh only scans single-level `*/go.mod`** — two-level modules
   (stack/*, example/*, metaengine/*engine, cmd/*) are invisible to the CI drift gate.
   The sweep fixed them anyway, but the gate has a blind spot worth closing (find-based
   glob).
7. **CI drift for remote-engine constants has no automated leg** (local engines only,
   nightly). A DSN-gated optional job (self-hosted runner with ephemeral services) would
   cover pg/mysql/dgraph too.
8. **The 350-line gate fights reality in exported-harness packages** (adttest/harness.go
   967, enginetest.go 935 are exported test harnesses, not production logic). Either
   refactor or consciously exempt harness dirs — a policy decision (question 3).
9. **Session pattern that worked and should stay**: per-task gates run immediately
   (lint-module, doc-check, changelog-symbols) instead of one mega-verify at the end —
   zero wasted gate cycles on the G1 diff itself.

---

## f) Up to 50 things to get done next (brainstorm-ordered, impact-first; 🔥 = Pareto)

1. 🔥 User: authorize the pending v4 tag wave (starting state improved by this session's
   pin sweep; plan: docs/planning/2026-08-27_17-30_PENDING-TAG-WAVE-PLAN.md).
2. 🔥 User: fix GitHub Actions billing (kills Release/Benchmarks/ci.yml since 2026-07-17).
3. User: cqrs-lint Self-Lint needs go-finding credentials for GOWORK=off fetches.
4. Infra: FlakeHub auth + magic-nix-cache 418 on CGo Build / Flake Check / Minimum
   Coverage / integration jobs.
5. >350-line refactor wave, tranche 1: `cmd/cqrs-lint/pkg/rules/catalog_extra.go` (1207)
   + `catalog.go` (746) — rule tables → data files or split by category.
6. Tranche 2: `metaengine/typed_reader.go` (1127) — split per ADT family.
7. Tranche 3: `metaengine/adttest/harness.go` (967) + `enginetest/enginetest.go` (935) —
   or exempt harness dirs via policy decision (see question 3).
8. Tranche 4: `metaengine/store.go` (898) + `execute.go` (767) + `engine.go` (694).
9. Tranche 5: pebbleengine/engine.go (725), duckdbengine/aggregations.go (722),
   sqliteengine/engine.go (649) — split backends from engine shell.
10. Tranche 6: cqrs-lint helpers.go (628), feature_profile.go (587), explain.go (517),
    run.go (485), filters.go (450), b022_b025.go (494).
11. Tranche 7: storage/sql/dialect.go (590), cqrs-bench/render.go (580),
    benchkit/runner.go (454), catalog/docserver/index_templ.go (437).
12. Dgraph filtered/scan per-ROW recalibration: write result-size-scaled benches
    (cost as f(k)), replace the per-OP numbers in NsPerFilteredScan/NsPerScan.
13. Harden `BenchmarkDgraph_CounterGet` to assert exact counter count (1000).
14. Static profile constructors or constant pins so routing_regression covers
    pg/mysql/dgraph ReadCosts (protocol-parity with bbolt/pebble).
15. Add "Calibrated per-pattern read costs" tables to pg/mysql/dgraph/sqlite/duckdb
    READMEs (badger/bbolt/pebble already have them).
16. Extend check-version-drift.sh to two-level go.mods (find-based) — close the blind
    spot this session's sweep exposed.
17. cqrs-lint ApplyLayout-vs-LayoutPlanApplier rule: design pass (type-info in analyzer
    or capability registry fed from api-stability scan).
18. Replace-drop sweep after the tag wave (system ×6, cqrs-bench ×7, event ×2, schema ×2,
    projectionhost ×2, integration ×2, middleware/encryption/signing sibling replaces,
    quic `=> ../`).
19. Post-wave: consolidate transitive external-module indirect deps (~49 go.mods).
20. Post-wave: GitHub Releases for all un-released tags (script exists:
    create-github-releases.sh; needs gh auth verification).
21. User decision: dead eventtest tags v4.0.0/v4.2.0 — delete remote tags or
    document-and-ignore.
22. go-codec F46: commit + tag the UnwrapDecode sniff (uncommitted in ../go-codec;
    update event alloc expectations in the same change).
23. Ratify (or revisit) the iroh latency P99 bound 50→150ms judgment call.
24. Transport/http + transport/grpc final deprecation v4.x tags (pre-v5 prerequisite).
25. v5 cut train (in order): delete stack.Materialize → RunProjections → Bundle +
    presets → storage view/relational → graph.GraphProjection → ADR-0126 compat shells →
    BuildWhereClause → tombstone metadata API (with listing type-driven status +
    taskmanager migration) → honest snapshot wire tags → E1/E7/E8/E11/E13/E15 → cut
    v5.0.0 with migration-guide expansion.
26. Introspection: Doctor/EXPLAIN could render the per-engine calibrated ReadCosts
    table live (constants are in profiles; wire into the existing Doctor sections).
27. Nightly CI: DSN-gated calibration drift legs for pg/mysql/dgraph.
28. macOS ephemeral-PG verification leg (needs macOS runner — tied to billing/org).
29. `-shuffle=on` rollout into ephemeral-pg/mysql/dgraph app invocations at next tag
    wave (decision already ADOPTED; one-variable-per-change discipline).
30. Dgraph shuffled-suite evaluation (the one live suite without a shuffle verdict).
31. Wipe `/mnt/buildcache` phantom diagnostics for good: verify the LSP wrapper is
    actually invoked by the editor (crush config path moved? — see question context).
32. Fix `~/.config/crush/crush.json` reference in AGENTS (file not found at that path
    during this session — config location drifted or was renamed).
33. system/v4 .golangci.yml exclusion-block audit (20 linters disabled) — trial re-enable.
34. cmd/cqrs-lint exclusions (17) and metaengine/ (24) — same trial.
35. example/taskmanager: migrate off Materialize OnTombstone (v5 tombstone pre-req).
36. listing: type-driven status to replace the DetectTombstone call (v5 pre-req).
37. SQL snapshot wire-tag rename: migration scripts under storage/sql/migrations +
    integration-pg over renamed schema.
38. `record.NewStreamRef` v5 breaking validation — call-site inventory now (cheap prep).
39. `TestEveryEngineSetsReadCosts`: extend roster assertions to cover the three
    engines' NEW constants semantically (source-scan for the values, not just presence).
40. Bench protocol: add a "no compile storms" pre-check to load-sweep.sh (uptime gate +
    warning) so future runs self-document contention.
41. CONTRIBUTING: document the fresh-MariaDB-init procedure (mirrors the AGENTS fix).
42. calibration-drift.sh: include the G1 constants for pg/mysql/dgraph behind a
    `--with-remote` flag that requires DSNs (fail-skip, not fail-error).
43. Add integration test pinning `estimateCost` routing flip for dgraph aggregates
    pre/post constant (regression pin for the 350× fix).
44. Sweep docs/status/archived references to "950_000" dgraph aggregate figure
    (historical docs stay frozen; active docs only — verify none active remain).
45. TODO_LIST: prune the now-closed Harvested items into CHANGELOG-adjacent archive per
    docs-health discipline (list is long; harvest pass).
46. Investigate whether `metaengine/go.mod` requiring `sqliteengine` (an ENGINE) can be
    dropped (core→engine dependency inversion smell; v5 candidate).
47. `scripts/check-version-drift.sh` could ALSO check "consumer pins latest tag" (not
    just consistency) — would have flagged storage v4.6.0 in getting-started earlier.
48. Replace hand-rolled anchor TOCs with a generated TOC check (gate accepts any TOC —
    a link-checker for intra-file anchors would catch slug drift).
49. example/getting-started: add a CI-visible standalone-build canary (it silently sat
    broken on storage v4.6.0; nothing scanned example/ go.mods).
50. Reduce calibration doc duplication: constants live in engine.go comments, baseline
    doc, drift script EXPECT, and (soon) READMEs — consider generating the doc table
    from the source constants.

---

## g) Questions I can NOT figure out myself

1. **Tag wave go/no-go.** The pending v4 patch wave is BLOCKED on your authorization,
   and this session's pin sweep pre-aligned every consumer to the latest published
   tags (storage consumers now at the listing-compatible v4.8.1). Do you authorize
   cutting the wave now (event/command/query/dedup/dispatcher/middleware/scheduling/
   kv/commandlifecycle/system/idempotency/encryption/signing + snapshot chain +
   schema/projectionhost/irohengine + engine patches, per the existing module-order
   plan)? Nothing else unblocks replace-drops, GitHub Releases, or the v5 train.
2. **CI money + credentials.** Billing is dead since ~2026-07-17 (every paid job fails
   in 3–7s), FlakeHub is unauthenticated (HTTP 418), and cqrs-lint Self-Lint needs
   go-finding credentials under GOWORK=off. All three need your accounts/settings —
   should I treat "CI stays broken, local `nix run .#verify` remains the only gate" as
   the accepted status quo for the next weeks, or is fixing CI imminent (changes what
   I prioritize in e.g. the 350-line wave)?
3. **350-line gate policy.** 29 production files exceed the CI limit; several are
   exported test HARNESSES (adttest 967, enginetest 935) and generated-adjacent rule
   tables (cqrs-lint catalog_extra.go 1207). Do you want (a) full refactor to
   compliance, (b) scoped exemptions for exported harness/rule-table directories with
   the limit kept for production logic, or (c) raise the limit? This decides whether
   items 5–11 in §f are a multi-session wave or a one-line policy change.

---

## Brutal self-review addendum (the 11 questions, answered short)

- **Forgot?** FEATURES.md row (caught late), predicted-stranding analysis, pin-test
  extension for the new constants, engine README tables for pg/mysql/dgraph.
- **Stupid we do anyway?** AGENTS memory going stale then being trusted verbatim
  (MariaDB); CI gates with blind spots (single-level go.mod glob; drift between
  "consistency" and "latest"); constants duplicated across 4+ places by hand.
- **Better?** Pre-flight stranding reasoning; one combined verification script per
  sweep instead of discover-fix-re-verify; update memory files AT the moment of
  discovery (I did it for MariaDB only after self-review flagged it).
- **Still improve?** See §e (8 items) and §f.
- **Lied?** No — but two claims deserve scrutiny: "LSP fix verified" means
  wrapper-installed, NOT editor-observed; "dgraph health timeout verified" means
  code-read + exercised-in-window, not fault-injected. Both stated at their true
  strength.
- **Ghost systems?** No new ones. Found one inversion smell: `metaengine/go.mod`
  requiring `sqliteengine` (core→engine dependency, §f-46).
- **Scope creep?** The pin sweep expanded beyond the strict CI-visible set — justified
  by the MarshalBinary lesson, and it caught a real stranding; documented.
- **Removed something useful?** No deletions this session.
- **Split brains?** No new ones; reduced two (stale FEATURES row, stale AGENTS
  procedure). One known accepted split remains: constants in prose vs code (§f-50).
- **Tests?** New bench = the only new test artifact; regression pins for the routing
  flip are listed as §f-43 (not done — honest gap).
