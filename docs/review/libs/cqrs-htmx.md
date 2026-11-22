# cqrs-htmx — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/cqrs-htmx` @ d6189f4f (stamp 72adb563 moved same-day; latest local root tag v4.19.0, 2026-11-21) · Consumer go-cqrs-lite @ feec9bc74 · Group G5

**Verdict:** N-A-DIRECTION (companion; see §2)
**Score (inverse audit):** 6.5/10 currency + modern-surface questions answered well = 65% (mid)

cqrs-htmx is the audit subject's largest consumer, not a dependency: 30 of
its go.mod files pin go-cqrs-lite trains, and ADR-0152 credits it with 500 of
685 indirect fleet edges (73%) — the de-facto composition layer whose choices
shape the v5 cut. Importing it here would be circular, so this report audits
the inverse: pin currency, integration depth against modern vs v5-doomed
surfaces, and feedback health. The verdict is sharply two-sided. On
currency it is the fleet's best citizen — every production module pins every
train at its latest tag (528/551 resolvable pins current; the only laggards
are deliberately-stale `verify-tag` test fixtures), swept same-day through
the metaengine nested-directory restructure. On depth, its new chassis
(`systemadapter`, `gfl-usermgmt`, `dashboardui`, `setup`) is fully
`system.New`/`system.Standard`/`systemscenario` — while `usermgmt`, the
legacy module the family still ships, imports the v5-doomed `stack/v4` and
raw `storage/v4/sql` directly and accounts for the bulk of the codemod's
1,000 planned rewrites.

## 1. What the library is

An htmx/templ server chassis for CQRS apps: command/query dispatch with
authz gates, SSE delivery, sync protocol, admin/dashboard UIs, a `setup`
bundle that assembles a running service, and a growing `systemadapter` that
re-exposes go-cqrs-lite's composition root as fleet presets. It is
multi-module (30 pinning go.mod files: root, `setup`, `adminui`,
`dashboardui`, `datastar`, `e2e/server`, `systemadapter`, `usermgmt`,
`gfl-usermgmt`, `huma`, `identity-model`, `legalparty-model`, `loginpage`,
`health`, `integration_test`, and 15 `examples/*` + 2 testdata fixtures).
ADR-0152 calls it "the full-surface chassis (~30 trains, deep imports)"
(`docs/adr/0152-fleet-first-module-topology-v5-dual-support.md:24-28`) and
orders it second for the cqrs-upgrade codemod, noting
`dashboardui/system_bridge.go` is already system-aware
(`docs/adr/0152-fleet-first-module-topology-v5-dual-support.md:73`).

## 2. Usage census in go-cqrs-lite (evidence)

**None — structurally impossible.** No go-cqrs-lite module pins or imports
cqrs-htmx; the edge runs the other way at scale (30 modules, ~40 direct
train requires in `setup` alone). The consumer-side surface is process and
receipts: the `--to-v5` dry-run receipts name cqrs-htmx 29 modules / 1,413
files / 1,000 planned rewrites with 32 trains still to fold
(`CHANGELOG.md:185`); the consumer's release-wave rows plan "consumer flips
ride the tags" (`TODO_LIST.md:98`); and an open row waits on observing
cqrs-htmx's own suite green against aligned pins to close C03
(`TODO_LIST.md:540`). Prior art: the 2026-07-23 ecosystem deep-dive
(`docs/research/2026-07-23_go-cqrs-lite-cqrs-htmx-ecosystem-deep-dive.html`,
superseded by `docs/research/2026-11-13_go-cqrs-lite-deep-dive.html`).

## 2b. Inverse census (pin-vs-latest, evidence)

All 30 modules' pins were resolved against the consumer's per-module tags:
**528 of 551 resolvable pins are at latest; zero production module lags.**
Representative current pins:

| Site | Pins | Status |
|---|---|---|
| `cqrs-htmx/go.mod:11-16` | command v4.14.1, dedup v4.3.0, event v4.15.2, id v4.7.5, query v4.10.1, storage/memory v4.6.2 | all current |
| `cqrs-htmx/integration_test/go.mod:20-32` | catalog v4.15.0, system v4.19.0, watermill v4.7.0 (+encryption, snapshot, storage, signing, record current) | all current |
| `cqrs-htmx/gfl-usermgmt/go.mod:10-11` | system v4.19.0, systemscenario v4.3.0 | current |
| `cqrs-htmx/dashboardui/go.mod:21` | system v4.19.0 (+metaengine v4.26.0 indirect, `:76`) | current |
| `cqrs-htmx/examples/catalog-demo/go.mod:12-14` | metaengine v4.26.0, system v4.19.0, systemscenario v4.3.0 | current |
| `cqrs-htmx/systemadapter/go.mod` | full train set incl. metaengine/bboltengine v4.3.3, systemscenario v4.3.0 | current |

The 23 stale pins all live in the two deliberately-frozen verify-tag
fixtures — `cqrs-htmx/scripts/testdata/verify-tag/real-setup-v4.8.1-poisoned/go.mod:14`
pins event v4.7.0 etc. because the fixture's job is to look like a poisoned
old resolve; `sibling-local-replace/go.mod` likewise. That is not lag; it is
test data. The companion's own treadmill claims "worst lag 0" with a
first-hand `check-upstream-treadmill` receipt
(`cqrs-htmx/TODO_LIST.md:23`), and the 2026-11-14 refresh records the
metaengine v4.24.0 nested-dir restructure resolved same-day with the
v4.26.0/sqliteengine v4.9.1/system v4.19.0 tail swept the next day
(`cqrs-htmx/TODO_LIST.md:31`). Root tag v4.19.0 (2026-11-21) is current
against its own line.

## 3. Capability inventory (inverse questions)

| # | Inverse question | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Composition-root adoption (`system.New`) | **Full** | `cqrs-htmx/systemadapter/projections.go:42` documents the pattern; `:79` and `:87` wire projections and bus around `system.New` |
| 2 | Canonical topology (`system.Standard`, ADR-0165) | **Full** | every one-engine preset returns `system.Standard(...)` (`cqrs-htmx/systemadapter/presets.go:37`, `:53`; the split preset stays explicit by design) — `cqrs-htmx/CHANGELOG.md:46`; consumer receipt `TODO_LIST.md:510` |
| 3 | systemscenario harness | **Full** | `cqrs-htmx/gfl-usermgmt/usermgmt_test.go:16` — `systemscenario.System(t, ctx, systemadapter.DomainConfig(), systemadapter.RecommendedMemoryDeployment())` across the generated BDD suite; consumer tracks a cqrs-htmx sqlite-lifecycle variant as T22 tail (`TODO_LIST.md:411`) |
| 4 | Strategic engine layer (metaengine) | Partial | bbolt direct (`cqrs-htmx/systemadapter/go.mod`), metaengine v4.26.0 current but only indirect (`cqrs-htmx/dashboardui/go.mod:76`); root module is legitimately thin (dispatch/read side) |
| 5 | Legacy stack presets avoided | Missed | `cqrs-htmx/usermgmt/es_setup_core.go:11` imports `stack/v4`; `:45` holds a `Bundle *stack.Bundle` — the direct legacy edge that drags `stack/v4 v4.5.1` indirect into nearly every module's go.mod |
| 6 | Raw `storage/*` avoided | Missed | `cqrs-htmx/usermgmt/mysql_stores.go:8-9` and `cqrs-htmx/usermgmt/es_projection_setup.go:13` import `storage/v4` + `storage/v4/sql`; this is the 68-finding `storage.SQLViewStore` V007 cluster the migration plan gates (`cqrs-htmx/TODO_LIST.md:167`); dev/test memory use (`cqrs-htmx/setup/setup.go:13`, `cqrs-htmx/systemadapter/projections.go:12`) is the benign half |
| 7 | Version currency | **Full** | §2b: every production module at latest on every train; same-day restructure sweep; stale pins confined to verify-tag fixtures |
| 8 | Feedback pushes improvements upstream | **Full** | filed go-cqrs-lite#113 (interface-level `kv.NewCache` ask; in-tree adapter deletes on delivery, `cqrs-htmx/TODO_LIST.md:30`); dispatch-boundary middleware upstream ask queued (`cqrs-htmx/TODO_LIST.md:28`); cqrs-lint fixes co-trained with the consumer's v4.22.x tag wave (`cqrs-htmx/CHANGELOG.md:30`); consumer harvests an "Upstream asks (cqrs-htmx)" ledger (`TODO_LIST.md:369`) |
| 9 | v5 codemod readiness | Missed | 29 modules / 1,413 files / 1,000 planned rewrites, 32 trains still to fold (`CHANGELOG.md:185`) — the largest bill in the fleet; concentrated in `usermgmt`'s stack + storage clusters |
| 10 | Cross-repo integration harness | **Full** | `cqrs-htmx/integration_test/go.mod:20-32` pins the consumer's whole system tier (catalog, system, watermill, encryption, snapshot, storage, signing, record) at latest and runs it as the consumer-facing composition proof |

Score derivation: rows N=10, none N/A → **A=10**. Full=6 (6.0), Partial=1
(0.5), Missed=3 (rows 5, 6, 9). Score = round(100 × 6.5 / 10) = **65% → mid
band**.

## 4. Gap analysis

1. **The v5 bill is one module's, and it is on the shipping path.** The
   modern chassis (`systemadapter`/`gfl-usermgmt`/`dashboardui`/`setup`) is
   clean; `usermgmt` — still the module consuming apps embed — carries the
   direct `stack/v4` import (`cqrs-htmx/usermgmt/es_setup_core.go:11`) and
   the raw `storage/v4/sql` tier (`cqrs-htmx/usermgmt/mysql_stores.go:8-9`).
   The migration plan exists, is wave-structured, and its W1 probes already
   found and fixed a real preset bug, but its hard gates (declarative
   hydration coverage, durability-pairing enforcement) are upstream-open
   (`cqrs-htmx/TODO_LIST.md:166`). Until `usermgmt` moves, the consumer
   cannot delete `stack/v4` at v5 without breaking its own 73%-edge
   companion — this is the single strongest coupling shaping the v5 cut.
2. **Currency discipline is exemplary but single-sourced.** The worst-lag-0
   treadmill runs on the companion's own tooling and receipts
   (`cqrs-htmx/TODO_LIST.md:23`, `:31`); nothing in CI enforces it, so it
   holds only as long as the ritual does. The consumer's C03 row — never
   observed cqrs-htmx's suite green against aligned pins
   (`TODO_LIST.md:540`) — is the same trust gap from the other side.
3. **The feedback loop is real but queue-bound.** Two upstream asks sit
   owner-gated (#113's constructor, the dispatch middleware offer
   `cqrs-htmx/TODO_LIST.md:28`); both are small consumer-side deliverables
   that would let the companion delete adapter code and drop a hand-rolled
   gate respectively. The consumer's TODO carries the pairing-matrix doc
   note as a foreign XS row (`TODO_LIST.md:121`) — cheap to land, still
   open.

## 5. Version currency

- All 28 production modules: every direct and indirect cqrs-lite pin at the
  consumer's latest per-module tag (spot sites in §2b; stamps cross-checked
  against the tag list). Root tag v4.19.0 (2026-11-21).
- The repo moved past the orchestrator stamp (72adb563 → d6189f4f) during
  the audit window without regressing any pin.
- Watch item: `stack/v4 v4.5.1` rides indirect in most modules solely via
  `usermgmt`; it is current for v4 but the train is v5-doomed, so the
  indirect edge mass evaporates only when `usermgmt` migrates (finding 1).
- The consumer's release-wave row explicitly couples its tags to consumer
  flips here — "cqrs-htmx snapshot_shape test + pin" rides R11-13
  (`TODO_LIST.md:98`).

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only
fix bugs all day long." cqrs-htmx is the re-integration the lens is about:
one place where htmx delivery, authz, SSE, and CQRS composition get fixed
for every fleet web app. The inverse audit says the companion pays for
itself — it absorbed `system.Standard` within a day, built the GFL BDD
suite on `systemscenario`, files real upstream asks, and keeps 551-pin
currency at zero lag. What it has not yet paid down is its own founding
debt: `usermgmt` predates the composition root and still speaks
`stack.Bundle` + raw SQL stores. That debt is why the score is mid: the
companion's modernization curve is steep and current, but the v5 cut's
cost — and the consumer's freedom to delete legacy surface — is priced
almost entirely by this one module. Under the 73% weighting, every week
`usermgmt` stays on `stack/v4` is a week the consumer's v5 deletion list
cannot shrink.

## 7. Ranked opportunities

1. **Land the usermgmt W2-W5 waves** (high): the plan is drafted and probed
   (`cqrs-htmx/TODO_LIST.md:166-167`); finishing it deletes the direct
   `stack/v4` import (`cqrs-htmx/usermgmt/es_setup_core.go:11`), retires
   the 68 V007 findings, and shrinks the codemod bill from 1,000 rewrites
   to the mechanical remainder.
2. **Publish the suite-green receipt to close consumer C03** (medium): one
   full `go test ./...` logged against the aligned pins satisfies
   `TODO_LIST.md:540` and converts the currency claim into observed
   evidence.
3. **Deliver the two upstream asks** (medium): #113's interface-level cache
   constructor and the dispatch-middleware offer (`cqrs-htmx/TODO_LIST.md:28`,
   `:30`) — both let this companion delete code it currently maintains.
4. **Wire the treadmill into CI** (low): make worst-lag-0 a gate, not a
   ritual (`cqrs-htmx/TODO_LIST.md:23`).
5. **Ship the v4.16-era pairing doc note** (low, XS): the foreign row the
   consumer is waiting on (`TODO_LIST.md:121`).

## 8. Reconsider when

- core/v5 tags its first release — cqrs-htmx is the codemod's second target
  (`docs/adr/0152-fleet-first-module-topology-v5-dual-support.md:73`) and
  its 32-unfolded-trains bill becomes actionable.
- The consumer ships the declarative Hydrator / durability-pairing gates
  the usermgmt plan waits on (`cqrs-htmx/TODO_LIST.md:166`) — then finding
  1 stops being blocked upstream.
- `setup` grows a `DispatchGate` convenience (two-consumer trigger,
  `cqrs-htmx/TODO_LIST.md:28`) — the authz chassis formalizes its coupling
  to the consumer's dispatch surface.
- templ pairing churns again — the era-matrix cost two sessions last time
  (`TODO_LIST.md:526`); the doc note in opportunity 5 is the prophylactic.

## 9. How to verify

```
rg -n 'larsartmann/go-cqrs-lite' -g go.mod /home/lars/projects/cqrs-htmx | wc -l
sed -n '11,16p' /home/lars/projects/cqrs-htmx/go.mod
sed -n '20,32p' /home/lars/projects/cqrs-htmx/integration_test/go.mod
sed -n '10,11p' /home/lars/projects/cqrs-htmx/gfl-usermgmt/go.mod
sed -n '36,56p' /home/lars/projects/cqrs-htmx/systemadapter/presets.go
sed -n '8,15p;40,48p' /home/lars/projects/cqrs-htmx/usermgmt/es_setup_core.go
sed -n '8,10p' /home/lars/projects/cqrs-htmx/usermgmt/mysql_stores.go
sed -n '14,20p' /home/lars/projects/cqrs-htmx/gfl-usermgmt/usermgmt_test.go
git -C /home/lars/projects/cqrs-htmx log -1 --format='%H' && git -C /home/lars/projects/cqrs-htmx tag --list | grep -v / | sort -V | tail -3
sed -n '185p' /home/lars/projects/go-cqrs-lite/CHANGELOG.md
sed -n '23,31p;166,167p' /home/lars/projects/cqrs-htmx/TODO_LIST.md
```

Companion twin report: `go-appkit.md` (the 18%-of-edges thin `system`
adapter) in this directory.
