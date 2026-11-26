# Research Index

Dated research reports, deep dives, and proposals. Everything here is a
point-in-time snapshot — verify claims against current code before acting on
them. Living truth lives in `AGENTS.md`, `FEATURES.md`, and the skill
references; decisions live in `docs/adr/`.

## Layout

| Path             | Contents                                                                       |
| ---------------- | ------------------------------------------------------------------------------ |
| `*.md`, `*.html` | Dated and topical research snapshots (this index covers them)                  |
| `archive/`       | Closed history — pre-2026-06 research, already routed; do not re-open casually |

## Dated deep dives (newest first)

| Report                                                                                                     | Verdict                                                                                                                                                                                                                                                               |
| ---------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [2026-11-23 Local-first competition sweep](../competition/index.html)                                      | **RESEARCH**: all 22 localfirst.fm landscape entries, one verified page each + master matrix; 17/22 CLIENT-MINTS-FACTS (no Command concept), 3 Command-shaped (Zero, Ditto, PowerSync); 7 live corrections to the landscape self-report; assembly kit for the L4 tier |
| [2026-11-22 Fleet libs × go-cqrs-lite](../review/libs/index.html)                                          | 23-library utilization audit: cmdguard 73% / templ-components 74% adopted, 11 underused; top pattern = bridges-built-never-crossed; reports live in `docs/review/libs/`                                                                                               |
| [2026-11-22 Commands vs Facts: LiveStore + DataStar](2026-11-22_commands-vs-facts-livestore-datastar.html) | **RESEARCH** (feeds D40 L4 shape + proposed D41): owner's LiveStore-no-Commands suspicion VERIFIED; industry converged on server-authoritative intents; L4 = command queue, never event sync                                                                          |
| [2026-11-21 Go→TS frontend contracts](2026-11-21_go-ts-frontend-contracts.html)                            | **PROPOSAL** (owner decision D40): generate TS contracts from the catalog, never ship a runtime; landscape verified                                                                                                                                                   |
| [2026-11-13 library deep dive](2026-11-13_go-cqrs-lite-deep-dive.html)                                     | **CURRENT**: health 72/100 (high) at HEAD `3946b7fe6`; refresh triggers in AGENTS.md §Deep-Dive Reports                                                                                                                                                               |
| [2026-10-01 templ-components deep dive](2026-10-01_templ-components-deep-dive.html)                        | Component-library utilization audit (own repo's skill carries current truth)                                                                                                                                                                                          |
| [2026-07-23 ecosystem deep dive](2026-07-23_go-cqrs-lite-cqrs-htmx-ecosystem-deep-dive.html)               | **SUPERSEDED** by the 2026-11-13 dive (banner in-file): 58/100 health verdict is historical                                                                                                                                                                           |

## Topical research

- **Event sourcing / CQRS foundations** — `derived-event.md`, `command-event-causality.md`, `scheduled-event.md`, `datomic-lessons.md`, `storage-first-principles-analysis.md`, `time-travel-options.md`
- **Storage & engines** — `database-architecture-taxonomy.md` (+ html render), `storage-environment-mapping.md`, `kv-store-abstraction-research.md`, `graph-db-event-sourcing.html`, `tigerbeetle-event-store.html`, `nats-jetstream-implementation-plan.html`
- **Graph tooling** — `2026-08-06_graph-databases-in-go-janusgraph-equivalents.md`, `domain-linter-research.md`, `event-history-visualization-tools.md`
- **UI / transport** — `datastar-client-side.html` (hypermedia path), `2026-11-21_go-ts-frontend-contracts.html` (typed-contract path), `2026-11-22_commands-vs-facts-livestore-datastar.html` (the command-boundary synthesis of both)
- **Proposals (dispositioned)** — `PROPOSAL-dissolve-core.html`, `PROPOSAL-dissolve-core-v2.html` (see `docs/adr/` for the accepted decisions)
- **Upstream issue drafts** — `2026-09-07_turso-go-ivm-commit-failure-issue-draft.md` (filed state tracked in TODO_LIST rows)
