# go-appkit — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-appkit` @ 5bb6134a (stamp 10a17ab moved same-day; latest local tag v0.13.0, cqrs submodule at cqrs/v0.9.0) · Consumer go-cqrs-lite @ feec9bc74 · Group G5

**Verdict:** N-A-DIRECTION (companion; see §2)
**Score (inverse audit):** 9/10 currency + modern-surface questions answered well = 90% (high)

go-appkit is not a dependency of go-cqrs-lite — it is one of the two
in-scope companions ADR-0152 defines, consuming 18% of the fleet's indirect
cqrs-lite edges (122 of 685). Importing it here would be circular, so this
report audits the inverse relationship: pin currency, integration depth
against the modern vs v5-doomed surfaces, and feedback health. The headline:
go-appkit is the model v5 cut citizen — it re-based off the deprecated
`stack/sqlite` preset onto `system.New` a major version ago, adopted the
`system.Standard` canonical topology within hours of its tag, touches no raw
`storage/*` in production, and pins no legacy `idempotency/v4`. Its only
weakness is train currency: 10 of 57 resolvable pins lag, all by trains
tagged 2026-11-11..15 that postdate its last cqrs-module release train
(2026-11-08), including the sqliteengine cap-trim deadlock fix.

## 1. What the library is

An application chassis above `appkit.Service`: config, logging, metrics,
OTel, health, security middleware, systemd integration, presets — and a
`cqrs` submodule that wraps go-cqrs-lite's composition root into a
lifecycle-managed `EventService`. Three of its modules pin go-cqrs-lite:
`cqrs` (the wrapper), `integration` (E2E composition tests), `docs` (the
module-inventory/docs generator consuming `catalog/v4`). ADR-0152's
consumed-surface analysis calls it "a thin `system` adapter" and grades it
**Modernity: Modern** (`docs/adr/0152-fleet-first-module-topology-v5-dual-support.md:26-28`,
`:96-100`). Root tag v0.13.0 (2026-11-21) shipped the listener-injection and
log-handler seams; the cqrs submodule's latest tag is cqrs/v0.9.0
(2026-11-08).

## 2. Usage census in go-cqrs-lite (evidence)

**None — structurally impossible.** No go-cqrs-lite module pins or imports
go-appkit; a require edge here would close a dependency circle (go-appkit's
`cqrs`, `integration`, and `docs` modules require nine cqrs-lite trains
directly). ADR-0152 scopes go-appkit as an in-scope companion precisely so
the v5 cut accounts for its consumption (`docs/adr/0152-fleet-first-module-topology-v5-dual-support.md:40`).
The consumer-side surface that exists is process, not code: the cqrs-upgrade
codemod dry-run receipts name go-appkit 12 modules / 308 files / 84 planned
import rewrites with 13 trains still to fold (`CHANGELOG.md:185`), and the
consumer's chassis-adoption row records go-appkit moving to `system.Standard`
the same evening the system v4.18.0 tag published (`TODO_LIST.md:510-511`).

## 2b. Inverse census (pin-vs-latest, evidence)

Three modules reference go-cqrs-lite; 57 pins resolve against the
consumer's per-module tags; **47 are current, 10 lag** — every lag a train
tagged after go-appkit's last cqrs release train (cqrs/v0.9.0, 2026-11-08):

| Module                                                    | Pin                   | Latest (consumer tag, date) | Behind                                                   |
| --------------------------------------------------------- | --------------------- | --------------------------- | -------------------------------------------------------- |
| `go-appkit/cqrs/go.mod:12`                                | metaengine v4.23.0    | v4.26.0 (2026-11-15)        | 3 minors                                                 |
| `go-appkit/cqrs/go.mod:11`                                | sqliteengine v4.7.1   | v4.9.1 (2026-11-15)         | 2 minors — **misses the v4.8.0 EnforceCap deadlock fix** |
| `go-appkit/cqrs/go.mod:18`                                | system v4.18.0        | v4.19.0 (2026-11-15)        | 1 minor                                                  |
| `go-appkit/cqrs/go.mod:19`                                | systemscenario v4.2.1 | v4.3.0 (2026-11-06)         | 1 minor                                                  |
| `go-appkit/integration/go.mod:25`                         | metaengine v4.23.0    | v4.26.0                     | 3 minors                                                 |
| `go-appkit/integration/go.mod:20`                         | catalog v4.12.0       | v4.15.0 (2026-11-11)        | 3 minors                                                 |
| `go-appkit/integration/go.mod:29`                         | system v4.18.0        | v4.19.0                     | 1 minor                                                  |
| `go-appkit/integration/go.mod:30`                         | systemscenario v4.2.1 | v4.3.0                      | 1 minor                                                  |
| `go-appkit/integration/go.mod:28`→sqliteengine (indirect) | v4.7.1                | v4.9.1                      | 2 minors                                                 |
| `go-appkit/docs/go.mod:5`                                 | catalog v4.12.0       | v4.15.0                     | 3 minors                                                 |

Current everywhere: command v4.14.1, event v4.15.2, id v4.7.5, query v4.10.1,
decider v4.8.0, middleware v4.7.2, projection v4.4.2, projectionhost v4.7.1,
commandlifecycle/projections v4.2.3, and all indirect floors
(`go-appkit/cqrs/go.mod:6-19`, `go-appkit/integration/go.mod:21-27`). The
lag is policy, not neglect: integration is "bumped onto cqrs's unreleased
go-cqrs-lite set … so the next cqrs train inherits a proven composition",
and "between trains, dependabot bumps the modules independently, so
mid-train drift is contractual" (`go-appkit/cqrs/CHANGELOG.md:62-64`), with
`go-appkit/integration/pin_drift_test.go:29-33` failing red on any
contract-relevant pin that drifts from the documented map.

## 3. Capability inventory (inverse questions)

| #  | Inverse question                                 | Grade    | Evidence / reason                                                                                                                                                                                                                                                         |
| -- | ------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Composition-root adoption (`system.New`)         | **Full** | `go-appkit/cqrs/eventservice.go:447` — `buildSystem` calls `system.New(ctx, mergeDomain(...), deployment)`; the wrapper is a DomainConfig/DeploymentConfig split over it                                                                                                  |
| 2  | Canonical topology (`system.Standard`, ADR-0165) | **Full** | `go-appkit/cqrs/eventservice.go:294` — `defaultDeployment` returns `system.Standard(...)`; shape pinned by `go-appkit/cqrs/deployment_test.go:13-18`; adopted same-evening as the system v4.18.0 tag (`TODO_LIST.md:511`)                                                 |
| 3  | Strategic engine layer (metaengine)              | **Full** | sqlite driver via blank-import contract (`go-appkit/cqrs/eventservice.go:22`); documented read side over `ExecuteTypedByName`, `NewWatcher`, `ServeSSE` (`go-appkit/cqrs/README.md:580`, `:603-609`)                                                                      |
| 4  | systemscenario harness                           | **Full** | `go-appkit/cqrs/scenario_pilot_test.go:24` and `go-appkit/cqrs/scenario_store_pin_test.go:29` — `systemscenario.Adopt(t, ctx, eventSvc.System())`, Given/When/Then against the wrapper                                                                                    |
| 5  | Legacy stack presets avoided                     | **Full** | zero `stack/v4` require in any of the 3 modules; the v0.5.0 BREAKING moved "from the deprecated stack/sqlite preset (removed at go-cqrs-lite v5, ADR-0123) to system.New" (`go-appkit/cqrs/eventservice.go:6-8`)                                                          |
| 6  | Raw `storage/*` avoided                          | **Full** | production code has zero storage imports; the one hit is test scaffolding (`go-appkit/integration/integration_test.go:17`)                                                                                                                                                |
| 7  | Legacy `idempotency/v4` avoided                  | **Full** | not pinned anywhere; policy documented: idempotency "deliberately NOT defaults — duplicate-command semantics belong to the consumer" (`go-appkit/cqrs/commands.go:20`; external `go-idempotency` path noted at `go-appkit/TODO_LIST.md:124`)                              |
| 8  | Version currency                                 | Partial  | 10/57 pins lag (§2b); ≤1 week of trains; the sqliteengine v4.8.0 fix for the EnforceCap-vs-folds deadlock — 60-600s stalls in a real consumer (`CHANGELOG.md:157`) — is not in the pinned set, though go-appkit declares no caps itself (no `WithCap`/`EnforceCap` usage) |
| 9  | Feedback pushes improvements upstream            | **Full** | go-cqrs-lite#67 filed and closed completed (graceful-shutdown ask, `go-appkit/TODO_LIST.md:129`); watchlist tracks cqrs-lite releases with pin-drift holds and unblock triggers (same row); pin-drift guard harness in `go-appkit/integration/pin_drift_test.go:1-33`     |
| 10 | v5 codemod readiness                             | Partial  | 84 planned import rewrites / 13 trains still to fold (`CHANGELOG.md:185`) — all mechanical core/v5 import folding; composition already v5-clean (rows 1,5,6,7)                                                                                                            |

Score derivation: rows N=10, none N/A → **A=10**. Full=8 (8.0), Partial=2
(2 × 0.5), Missed=0. Score = round(100 × 9.0 / 10) = **90% → high band**.

## 4. Gap analysis

1. **The lag row is a real exposure with a real clock.** go-appkit's
   default deployment is sqlite (`go-appkit/cqrs/README.md:52`), and
   sqliteengine v4.8.0 shipped the fold-lock/cap-trim deadlock fix that
   another fleet app observed as 60-600s stalls (`CHANGELOG.md:157`).
   go-appkit declares no caps today, so it will not hang itself — but every
   fleet service it presets inherits v4.7.1 until the next cqrs train, and
   the first consumer to declare `WithCap` hits an engine where that is a
   known-unfixed deadlock. The metaengine v4.24-v4.26 trains it misses are
   additive for it (json/v2 determinism `CHANGELOG.md:44`, Typed folds
   `CHANGELOG.md:46`, `SSEDroppedEvents` observability `CHANGELOG.md:50`).
2. **The "contractual drift" policy has no treadmill leg.** cqrs-htmx runs
   a strict-lag-0 upstream treadmill (`cqrs-htmx/TODO_LIST.md:23`); go-appkit
   relies on release-train moments plus dependabot. The pin-drift test pins
   the go-appkit family's own tags to newest, but nothing asserts cqrs-lite
   pin freshness between trains — the 2026-11-11..15 wave sat unpicked for a
   week and would sit longer if the next cqrs train waits.
3. **Nothing else.** No v5-doomed surface is in the production graph. This
   companion is the proof ADR-0152's "thin adapter" bet can pay: its v5
   migration is an import-path fold the codemod already enumerates.

## 5. Version currency

- 47/57 resolvable pins at the consumer's latest per-module tags; laggards
  enumerated in §2b. All lags are ≤1 week old (trains 2026-11-11..15 vs last
  cqrs train 2026-11-08).
- The consumer's own tracking agrees: the R11-13 release-wave row expects
  "consumer flips ride the tags" (`TODO_LIST.md:98`), and ADR-0152 orders
  the companions first for the cqrs-upgrade codemod with go-appkit sized
  "afternoon-scale: 8 production files + require sweep"
  (`docs/adr/0152-fleet-first-module-topology-v5-dual-support.md:72`).
- Root module v0.13.0 (2026-11-21) is one day old; the repo moved past the
  orchestrator stamp (10a17ab → 5bb6134a) during the audit window without
  touching any cqrs-lite pin.

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only
fix bugs all day long." go-appkit exists so that service-lifecycle glue is
fixed once; its cqrs submodule exists so the same is true of cqrs-lite
composition. The inverse audit says the bargain is working in both
directions: the companion absorbs the consumer's newest doctrine fast
(`system.Standard` within hours), keeps every v5-doomed surface out of its
graph so the cut stays cheap, and pushes its own seams back upstream
(#67). The one place the bargain is one-way right now is train pickup —
the consumer shipped a week of engine hardening that the companion's
dependents have not received. Under the 18%-of-edges weighting, that is a
small blast radius; it is also the cheapest fix in this report: one
release train.

## 7. Ranked opportunities

1. **Cut the next cqrs train on the 2026-11-15 pins** (high): sweep
   metaengine v4.23.0 → v4.26.0, sqliteengine v4.7.1 → v4.9.1, system
   v4.18.0 → v4.19.0, systemscenario v4.2.1 → v4.3.0 (+catalog v4.12.0 →
   v4.15.0 in integration/docs), re-run the E2E composition, tag. The
   deadlock fix rides it.
2. **Add a between-trains freshness check** (medium): extend
   `go-appkit/integration/pin_drift_test.go:29` (or its `check-pin-drift.sh`
   sibling) with a "no cqrs-lite pin older than N days vs the module's
   newest tag" leg so contractual drift has a ceiling, not just a ritual.
3. **Pre-run `cqrs-upgrade --to-v5 --dry-run` in CI** (low): 84 rewrites /
   13 trains is the whole bill (`CHANGELOG.md:185`); a dry-run gate keeps it
   from regrowing as new code lands.
4. **Document the cap posture** (low): one README line stating the preset
   declares no `WithCap` collections, so readers can weigh the sqliteengine
   v4.7.1 pin against the v4.8.0 deadlock fix knowingly.

## 8. Reconsider when

- A fleet service presets onto go-appkit AND declares capped collections —
  then the sqliteengine lag becomes a patch gate, not hygiene.
- core/v5 tags its first release — the codemod's go.mod step unblocks and
  this companion is the afternoon-scale first target
  (`docs/adr/0152-fleet-first-module-topology-v5-dual-support.md:72-73`).
- go-appkit grows a browser surface (templ-components pairing rules start
  to apply to it directly; today it has none).
- The consumer ships a metaengine breaking change — the 3-minor lag window
  is where such a change would strand the proven-composition policy.

## 9. How to verify

```
rg -n 'larsartmann/go-cqrs-lite' -g go.mod /home/lars/projects/go-appkit
sed -n '6,9p;22p;288,296p;440,452p' /home/lars/projects/go-appkit/cqrs/eventservice.go
sed -n '13,18p' /home/lars/projects/go-appkit/cqrs/deployment_test.go
sed -n '1,33p' /home/lars/projects/go-appkit/integration/pin_drift_test.go
sed -n '60,66p' /home/lars/projects/go-appkit/cqrs/CHANGELOG.md
git -C /home/lars/projects/go-appkit log -1 --format='%H' && git -C /home/lars/projects/go-appkit tag --list | sort -V | tail -3
git -C /home/lars/projects/go-cqrs-lite log -1 --format='%ad %s' --date=short metaengine/sqliteengine/v4/v4.8.0
sed -n '151,168p;185p;232p' /home/lars/projects/go-cqrs-lite/CHANGELOG.md
sed -n '24,40p;72,73p;96,100p' /home/lars/projects/go-cqrs-lite/docs/adr/0152-fleet-first-module-topology-v5-dual-support.md
```

Companion twin report: `cqrs-htmx.md` (the 73%-of-edges full-surface
chassis) in this directory.
