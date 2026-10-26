# Status Report: catalog × samber/do × samber-do-auditlog Gap Analysis

**Date:** 2026-10-26 11:18 (Monday)
**Session type:** Analysis/research only — zero code changes in go-cqrs-lite this session.
**Scope:** Investigation started from the owner question: *"Many of our projects that use go-cqrs-lite/catalog claim like 1 Service, but they also use samber/do v2 — how could we be smarter? What can we learn from samber-do-auditlog?"* Everything below is about this session's run and what it surfaced. No unrelated research.

---

## Context: what this session investigated

- **`catalog/`** (go-cqrs-lite): static, declarative architecture catalog — `Service`, `Message`, `Domain`, `Flow`, `DataStore` resources; reflection-derived schemas (`Command[T]/Event[T]/Query[T]`, `catalog/message_config.go:206`); exports to EventCatalog MDX, one-file HTML architecture reference, OpenAPI/AsyncAPI/D2, docserver. **Zero runtime awareness, zero DI awareness.**
- **`samber-do-auditlog`**: runtime plugin over samber/do v2 hooks — records every registration, invocation, shutdown, health check; infers a dependency graph (`ServiceGraph`, `hooks.go:82` stack-based), tracks provider types, `IsHealthchecker`/`IsShutdowner`, scope tree; ~25 exports (JSON/NDJSON/CSV/HTML/Mermaid/PlantUML/DOT/D2) plus a live SSE viewer (`live.Server`).
- **`cqrs-htmx/auditlog`** (sibling repo module): already proves the "one-call Setup" bridge pattern — `WithAuditLog()` returns `Opts + Plugin + live Viewer` in one call.
- **Fleet reality (verified this session, module-graph intersection):** 10 projects sit in the exact gap — CV, DiscordSync, go-appkit/integration, InboxClean, invoices, PapDashboard, webphone, Zlota44, plus cqrs-htmx itself (and its ci-siblings fixtures).

---

## a) FULLY DONE

1. **Mapped the conceptual gap.** catalog = declared architecture truth (static); auditlog = actual runtime wiring truth (dynamic). Neither knows the other; consumer apps hand-write the static side and let the dynamic side go undocumented. Core recommendation produced: derive catalog content from `samber/do` reality via an auditlog→catalog bridge (`Plugin.Report()` → `catalog.Catalog`), with the inventory (registrations, eager at `Provide` time) as the trustworthy part and invocation-derived edges as bonus.
2. **Verified the duplication pattern in consumers.** Read `invoices/internal/readmodels/catalog.go` in full — the canonical "1 Service, hand-maintained flows/owners/datastores" shape. Located the equivalent hand-rolled file in 7+ repos (Zlota44, CV, PapDashboard, bank-sync, DiscordSync, github-local-sync, InboxClean).
3. **Verified `system/` has zero catalog awareness** (`rg` over `system/*.go`: no imports) while `DomainConfig` already holds the full declared message universe (`Events []event.Type`, `Schema []schema.EventSchema`, coeffect gate computing producers/consumers). This grounds the highest-leverage improvement: `catalog.FromDomainConfig(domain)`.
4. **Identified the cqrs-htmx `auditlog.Setup` module as the architectural template** for the missing catalog-side equivalent (builder + viewer + export in one call).
5. **Produced a ranked improvement list** (previous chat turn): DomainConfig derivation > shared drift-gate helper > fleet Setup module > schema-version lockstep > health modeling > stale-doc fixes.
6. **Self-corrected a false claim before writing this report.** Earlier this session I asserted `catalog/simple` has zero fleet adoption. Re-verification with the correct import path (`catalog/v4/simple`) shows **InboxClean, PapDashboard, and crm actively use it**. The claim was wrong; corrected in section d. This is also why the report leans on re-run greps rather than memory.

## b) PARTIALLY DONE

1. **Claim verification depth.** samber-do-auditlog's API surface and design docs came from a subagent report, not my own file reads (only cross-checked `plugin.go` signatures indirectly). High confidence, not firsthand-verified line-by-line. Same for the cqrs-htmx `auditlog.Setup` (I read only the first ~40 lines).
2. **Fleet census breadth.** The dual-consumer list is a module-graph intersection (go.mod greps) — accurate for "who imports both", but I did not open each repo to confirm *how* each uses them (e.g., whether webphone/crm actually register catalogs or merely vendor the deps transitively).
3. **The hub story.** `catalog.home.lan` federation hub and per-repo export mechanics (CV's `scripts/catalogexport/main.go`, invoices' `invoices catalog` subcommand) were observed but not audited for consistency.

## c) NOT STARTED (all recommendations are analysis-only; nothing implemented)

1. `auditlog → catalog` bridge exporter (`Plugin.ExportCatalog()` or equivalent).
2. `catalog.FromDomainConfig(domain system.DomainConfig)` derivation in go-cqrs-lite.
3. Shared inventory drift-gate helper (`catalog.AssertInventory(t, cat, ...)`).
4. Fleet-standard `Setup`-style catalog wiring module.
5. Schema-version lockstep (`schema.EventSchema` version → catalog `Message`).
6. Health modeling in catalog/architecture export.
7. samber-do-auditlog README `GOEXPERIMENT=jsonv2` staleness fix (flag is a no-op since jsonv2 graduation; go-cqrs-lite removed all such flags 2026-09-19).
8. `samber-do-best-practices` skill verification-table refresh (auditlog rows still "Unverified" though the repo is locally inspectable).

## d) TOTALLY FUCKED UP (brutal honesty)

1. **I published a false claim and only caught it under review pressure.** The "`catalog/simple` has zero fleet adoption" finding in my second answer was based on grepping the string `catalog/simple` — which does **not** match the real import path `go-cqrs-lite/catalog/v4/simple`. Three fleet repos use it. Consequence: the recommendation "either fix the API or nobody knows it exists" was built on sand. Lesson: verify import-path claims against the actual module path, never a directory fragment.
2. **"~10 dual consumers" was stated before it was measured.** In the first two turns it was an eyeballed overlap of two `head`-truncated lists (both truncated at 30 lines — lists were cut off!). Only in this turn did I compute the real intersection. The number happened to be right (10), but the process was wrong.
3. **Findings lived only in chat.** Two full turns of analysis produced no durable artifact — the classic entombment-in-the-opposite-direction problem (findings evaporate instead of reports rotting). This report is the fix.
4. **Single-sample generalization.** The "apps claim 1 Service" premise was verified in exactly one repo (invoices) before being generalized across the fleet in my recommendations.

## e) WHAT WE SHOULD IMPROVE (process + product)

Process (from this session's own mistakes):
1. **Evidence-before-claim for import/adoption statements** — always grep the full module path.
2. **Persist analysis to a doc in the same session** it was produced.
3. **Never `head`-truncate a list you're about to count.**

Product (the real improvements, re-ranked after correction):
1. **`catalog.FromDomainConfig`** remains the Pareto pick — kills the hand-copied message lists in every consumer, and the coeffect gate already computes producers/consumers.
2. **`catalog/simple` is adopted by exactly the newer repos** (InboxClean, PapDashboard, crm) — so the improvement is *migrating the older hand-rolled repos to it* and *teaching it in the skill*, not inventing a new API. My earlier framing was inverted.
3. **Inventory drift gate as a one-liner** (InboxClean hand-rolled the only one in the fleet).
4. **auditlog→catalog bridge** as a small samber-do-auditlog-side exporter.
5. **Fleet Setup module** standardizing builder + docserver + export + hub push.
6. **Schema-version lockstep** between `schema.EventSchema` and catalog `Message`.
7. **Health modeling**: auditlog records it, go-health-dashboard aggregates it, catalog renders nothing.

## f) Next things to get done (ranked, session-derived; HARVEST-ready)

**Bridge & derivation (highest impact)**
1. `catalog.FromDomainConfig(domain system.DomainConfig) *catalog.Catalog` in go-cqrs-lite — auto Services/Messages from Commands/Queries/Events/Schema.
2. Decide service granularity rule (deployable vs bounded context per catalog Service) — needs owner input, see question 1.
3. `Plugin.ExportCatalog()` in samber-do-auditlog (registrations → Service/Entity inventory; `ServiceGraph` → Flows).
4. Golden-report CI mode: boot injector in test, dump `Report()` JSON, diff as golden (api-stability ratchet pattern).
5. Wire `system.New` to optionally stamp schema versions (from `Schema`) onto derived catalog Messages.
6. Resolve the lazy-service blind spot: derived inventory must rely on *registrations* (eager), not invocations.

**Drift gates**
7. `catalog.AssertInventory(t, cat, wantCommands, wantEvents, ...)` shared test helper in `catalog/`.
8. Migrate InboxClean's hand-rolled golden test (`internal/domain/email/catalog_test.go`) to the shared helper.
9. Add a cqrs-lint adoption rule (or scorecard breadth credit) for "catalog has an inventory drift test".
10. Fleet sweep: every catalog consumer gets the inventory gate in one pass.

**Fleet consistency**
11. Migrate hand-rolled `internal/catalog/catalog.go` repos (invoices, Zlota44, CV, bank-sync, DiscordSync, github-local-sync) onto `catalog/simple` where shape fits.
12. Build the Setup-style module: one call returning catalog (derived) + docserver handler + export command + hub push.
13. Standardize export entrypoint across repos (replace CV's bespoke `scripts/catalogexport/main.go` and invoices' subcommand with one convention).
14. Document the `catalog.home.lan` hub ingestion contract once, centrally.
15. Replace hand-maintained `CatalogVersion` constants with `debug.ReadBuildInfo()`-derived versions.

**Health & runtime truth**
16. Architecture export: badge services with health endpoint/status.
17. Cross-link auditlog live viewer URLs into catalog service pages (or docserver).
18. Fold `RecordHealthCheck` results into the auditlog HTML report service cards.

**Docs & skills hygiene**
19. Fix samber-do-auditlog README `GOEXPERIMENT=jsonv2` staleness.
20. Refresh `samber-do-best-practices` skill verification table (auditlog rows → verified, with file:line).
21. Add a catalog×DI recipe to `.agents/skills/go-cqrs-lite/references/recipes.md` once the bridge exists.
22. Teach `catalog/simple` in the skill's quickstart (it exists and is the right default for single-service apps).

**Schema split brain**
23. Make catalog `Message` carry declared schema version when `schema.EventSchema` exists for the type.
24. Fail/warn on drift: catalog reflection shape vs `Schema`-declared current payload.
25. Render upcast chains (old versions → current) in the architecture export's message sections.

**Deeper audits this session opened but didn't do**
26. Open every dual-consumer repo and classify catalog usage depth (trivial vs flows vs full).
27. Audit hub export consistency across the 10 dual consumers.
28. Check whether `DeriveProducersConsumers` output matches the runtime coeffect gate (three-tier lockstep, contract 24).
29. Review whether webphone/crm import both by intent or transitively.
30. Assess cqrs-htmx `auditlog.Setup` viewer for reuse as the catalog-side viewer chassis.

**Smaller polish**
31. `docs/status/` HARVEST: route items 1–30 into TODO_LIST.md per docs-health (owner-gated).
32. Consider `catalog` dep on nothing new — bridge must live on the auditlog side to keep catalog dep-free.
33. Bench/gate: none needed (no code changed), but any bridge module gets the standard new-module ritual (AGENTS "Add a New Module", 8 steps).

*(33 items — listed what's real rather than padding to 50.)*

## g) Questions I cannot figure out myself

1. **Service granularity:** should a catalog `Service` represent the deployable (current fleet convention: 1 per binary) or the bounded context/module (many per binary, closer to what auditlog's graph exposes)? This is an architecture-documentation philosophy call with real noise/meaning tradeoffs — yours to make, and it gates items 1–3.
2. **Who owns the bridge:** a `catalog` export method that imports samber-do-auditlog (dependency direction: catalog → auditlog), an auditlog-side exporter that imports catalog (auditlog → catalog), or a third thin module? go-cqrs-lite dep budgets and the "catalog stays pure data" stance argue for auditlog-side or a third module — but that's a repo-topology decision.
3. **Is `catalog.home.lan` still the live federation target** (invoices docs mention it), or has the one-file architecture reference + docserver already replaced it? Determines whether item 13–14 matter or are legacy cleanup.

---

*Point-in-time snapshot. Section (f) is HARVEST-ready for TODO_LIST.md per docs-health; intentionally not harvested yet — waiting for instructions per session protocol.*
