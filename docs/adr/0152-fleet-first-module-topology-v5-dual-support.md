# ADR-0152: Fleet-First Module Topology and v5 Dual-Support Transition

- Status: Accepted
- Date: 2026-10-08
- Deciders: owner, via chat decisions 2026-10-08 (consumer scope, transition model, boundary role — all recorded same day)
- Supersedes: the implicit "independently versioned per-module trains" model the v4.x release machinery implements
- Related: ADR-0123 (v5 unification, single composition root), ADR-0151 (Goal direction), ADR-0128 (external extractions — the stranded-pin debt this ADR's sweep resolves)

## Context

The 2026-10-08 module audit (`docs/status/2026-10-08_20-23_go-modules-audit-external-usage-and-self-review.md`)
measured the multi-module reality against its only possible market:

- **101 go.mod trains**, lockstep release waves of 90+ tags, ~1,839 LOC of
  compensating release/gate scripts — while **no consumer has ever used the
  version independence** (fleet pin-skew on the same train spans v4.5.0–v4.13.1,
  purely accidental drift).
- **The market is the first-party fleet, by policy and by evidence**: the repo is
  public but PROPRIETARY (pkg.go.dev hides docs; Imported-by: 0; 1 star). In the
  `~/projects` universe: 52 trains consumed, **49 live trains + the root train
  with zero consumers**.
- **The import surface is inverted from ADR-0123's declaration**: consumers
  import event(35)/id(36)/command(29)/decider(28)/middleware(23)/query(21)
  directly; `system` has 13 direct adopters. cqrs-htmx carries 500/685 indirect
  edges (73%), go-appkit 122 (18%) — they are the de-facto composition layer.
- Consumed-surface analysis (2026-10-08): **go-appkit is a thin `system`
  wrapper** (8 production files; its 24-train require list is mostly transitive
  surface); **cqrs-htmx is the full-surface chassis** (~30 trains, deep imports
  across root/adminui/dashboardui/datastar/e2e + its own examples).

## Decisions

1. **Consumer scope: first-party fleet only.** External adoption is not a
   current goal; the PROPRIETARY license is intentional. Zero-consumer-by-fleet
   evidence is therefore a legitimate kill/integrate signal, unlike a
   public-market library.
2. **Transition model: DUAL-SUPPORT.** v4 trains keep tagging fixes; v5 lands
   alongside on new module paths; apps migrate opportunistically. No hard
   fleet-wide cutover.
3. **Boundary: cqrs-htmx and go-appkit are IN-SCOPE COMPANIONS.** They co-release
   in the same version waves, their import surface shapes the v5 core, and the
   migration codemod targets them before the app residue.

## v5 Topology (consequence of the decisions)

**One core module** `github.com/larsartmann/go-cqrs-lite/v5` — all Tier 0–3
domain + composition as packages: record, id, dedup, kv, dispatcher, metadata,
event, command, query, scheduling (+engine, +sqlstore), schema, snapshot,
projection, projectionhost, listing, scenario, deriver\*, claiming,
commandlifecycle (+projections), decider, graph\*, metaengine core,
projectionadapter, otel, prometheus, middleware, catalog, encryption, signing,
stack, storage (+memory), system.
(\* zero-consumer modules enter the core ONLY if the ghost-system verdict
[P1 #11 of the status report] keeps them; otherwise they die at v5.)

**Driver tier stays modular** — dep isolation is the one boundary with proven
payoff: metaengine engines (sqlite, pg, mysql, duckdb/CGo, dgraph, iroh,
bigtable, turso, badger, bbolt, pebble), storage/{turso,pebble,bbolt}, queue/\*,
transport/\*, watermill, testutil/\*testcontainer, eventtest. Policy:
**tag on first consumer; untagged until then.**

**Tooling:** only `cmd/cqrs-lint` keeps a release train (cross-repo consumers:
buildflow, gomend). Internal tools, examples, and test suites are untagged local
modules.

**Versioning: lockstep single version across all surviving trains** (one wave
version number; per-train semver independence is retired — the audit proved it
is unused degrees of freedom that only buy pin sweeps).

**Rollout order (dual-support):**

1. go-appkit (afternoon-scale: 8 production files + require sweep),
2. cqrs-htmx (+ its submodules; `dashboardui/system_bridge.go` is already a
   `system` consumer — start there),
3. app residue via an extended `cmd/cqrs-upgrade` v4→v5 import codemod.

## Consequences

**Positive:** tag waves drop from 90+ to <15; version soup (v0–v4 across
siblings) dies; the pin-sweep/go.sum-repair/same-batch-sibling failure classes
die with per-train independence; the v5 core is shaped by its two real
composition layers instead of a hypothetical public market.

**Negative / accepted costs:** dual surface during transition (v4 fixes on old
paths, v5 features on new); the eventual import rewrite touches every app
(codemod-mitigated); untagged drivers lose the "tagged = ready" signal —
capability is proven by integration suites only; cqrs-htmx's own submodule
 sprawl mirrors this repo's and will need the same treatment eventually.

## Evidence

Full consumer matrix, zero-consumer list, stranded-pin inventory, and the
verification log: `docs/status/2026-10-08_20-23_go-modules-audit-external-usage-and-self-review.md`.

**Ratchet evidence (2026-10-09, cqrs-lint scorecard):** the migration-order
premise is now measurable. `cqrs-lint scorecard --path ~/projects/cqrs-htmx`
grades **Modernity: Legacy** — 41 v5-removed API uses plus 1 stack-surface
import (`stack/v4` bundle, live in `usermgmt/` production code) — while its
modern side (the `systemadapter/` composition module, metaengine pushdown)
renders in the same report. `~/projects/go-appkit` grades **Modernity: Modern**
(the `cqrs/` submodule's `system.New` wiring is detected project-wide). The
first-migration-target ordering this ADR predicts is exactly what the
scorecard now surfaces per project. Snapshots live in
`docs/status/2026-10-09_17-32_scorecard-followup-wave-session-review.md` and
its follow-up addendum.
