# Status Report — Data-Mesh Conformance Assessment Session

**Date:** 2026-09-23 18:09 (Wednesday)
**Session scope:** ONE question — "How well does go-cqrs-lite conform to / enable data mesh?"
Analysis-only session. Zero code changes, zero commits authored (working tree untouched).
This report covers only what this session did and noticed.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Loaded the `go-cqrs-lite` skill (SKILL.md) before analysis | Skill view, session transcript |
| 2 | Verified **first-class data-product support** in `catalog/`: `DataProduct` — "a data product in a data mesh — a curated, owned dataset" (`catalog/types_resources.go:37`), `DataContract` attached to outputs (`types_resources.go:83`), `AddDataProduct` (`build.go:111`), domain↔product association (`domain_config.go:62`), rendered as `data-products/<id>/index.mdx` in the EventCatalog export (`catalog/README.md:396`) | `rg` output, session |
| 3 | Verified the **federation hub story**: `catalog/README.md:345-368` ("Feeding the federation hub (catalog.home.lan)", CI union-merge of per-service exports), headless export template `catalog/cmd/catalog-export/main.go`, and today's routing commit `6170c4e76` (T13 items → `TODO_LIST.md:66-85`, inspected via `git show`) | git log/show, TODO_LIST.md:66-85 |
| 4 | Research sweep via sub-agent: `catalog/` (AsyncAPI 3.0, EventCatalog, OpenAPI, D2 exporters), `watermill/` (broker-agnostic delivery, CatchUpSubscriber, journal-as-outbox per ADR-016), `schema/` (upcasting = contract evolution), `signing`+`encryption` (trust), examples inventory, `system/` DomainConfig-vs-DeploymentConfig operator story | Agent report + spot-checks |
| 5 | **Delivered the 4-principle assessment** (Dehghani): Strong on domain-owned data; first-class on data-as-product; strong on self-serve platform (`system.New` developer/operator split, `cqrs.yaml`); partial on federated governance — with honest gaps: federated queries explicitly rejected (`docs/planning/meta-engine-design.md:116`), governance gates are repo-local, gRPC transport deprecated (v5), no multi-bounded-context example in-repo | Final answer, this session |
| 6 | Resolved the one glitched fact: the coeffect summary filename is **`coeffects.md`** (`catalog/eventcatalog/exporter_coeffects.go:14,29`) — verified directly after the sub-agent failed to transcribe it | sed output, this session |

## b) PARTIALLY DONE

| # | Item | What remains |
|---|------|--------------|
| 1 | Sub-agent research | 6 of 7 topics returned clean; the 7th (coeffect summary filename) pathological-looped through ~40 lines of self-contradicting "corrections". Contained — the final answer cited no filename — but verified only one turn later, in this report's follow-up |
| 2 | Claim verification | Single-source (agent, unverified by me): watermill broker-README details (`watermill/README.md:95-108`), ADR-016 outbox position, `system/config_types.go:124-205` field list. Plausible, low risk, spot-checks pending |
| 3 | "No multi-bounded-context example in-repo" claim | Based on agent's `example/` listing, not verified file-by-file |

## c) NOT STARTED

- **Durable write-up of the data-mesh fit** — nothing persisted anywhere (no `docs/architecture-understanding/` mapping doc, no skill-reference section, no README note). The assessment lives only in chat.
- **HARVEST** — the data-mesh gap items from the answer are not routed into `TODO_LIST.md`/`ROADMAP.md` (the T13 items found this session were already routed by commit `6170c4e76`; my *new* gap findings were not).
- **Numeric scoring / ecosystem comparison** (vs. EventStore/Kurrent, raw Watermill, etc.) — not requested, not done.

## d) TOTALLY FUCKED UP

| # | What | Severity | Root cause | Mitigation |
|---|------|----------|-----------|------------|
| 1 | Sub-agent filename-transcription loop ("coeoffs.md" ≠ `coeffects.md`): ~40 lines of garbage, and I shipped the answer WITHOUT resolving it that same turn | Low (no wrong fact reached the user — answer omitted the filename) | Sub-agent token pathology, uncaught | Should have re-run a 5-second grep immediately; deferred to the next turn instead. Fix in (e1) |
| 2 | Missed cross-reference: the repo already has a paradigm-mapping precedent — `docs/architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md` — the natural sibling/anchor for a data-mesh assessment, referenced in AGENTS.md; I did not consult it | Low | Read AGENTS.md's metaengine section, not its architecture-understanding index | A data-mesh mapping doc should cite it (f1) |

## e) WHAT WE SHOULD IMPROVE

1. **Sub-agent glitch handling** — when a sub-agent returns uncertain/looping output on a *specific* token, verify that token immediately with a direct `grep`/`sed` in the same turn; never defer. Cost when skipped: an unverified claim floating between turns.
2. **Durable capture of architecture-fit analyses** — paradigm assessments (Cordis precedent) belong in `docs/architecture-understanding/`, not chat. This session produced one and wrote it nowhere.
3. **Cross-referencing prior paradigm docs** before answering paradigm questions — the Cordis doc would have strengthened the data-mesh answer with the "context paradigm" framing the user already blessed.
4. **Harvest discipline** — session-discovered gaps (data-mesh related) should be routed per the HARVEST rule the same session, not left in a timestamped file.

## f) Next tasks (session-derived, ranked; 1–4 pre-exist in TODO_LIST.md T13)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Write `docs/architecture-understanding/` data-mesh mapping doc (4 principles → modules, cross-link Cordis doc) | High | M | Documentation |
| 2 | HARVEST this report's new items into `TODO_LIST.md` / `ROADMAP.md` | Medium | S | Cleanup |
| 3 | T13: export `catalog.index.json` manifest (golden-tested) for cheap hub diffing | High | M | Feature |
| 4 | T13: `skip-bootstrap-files` export option (hub owns its bootstrap) | Medium | S | Feature |
| 5 | T13-related: fix `@eventcatalog/linter` ref mismatch — versioned vs unversioned `services/<id>/` dirs (`refs/resource-exists` false flags) | Medium | M | Bug |
| 6 | T13-related: emit message/container-level `owners` (currently services only) — re-arms warn-suppressed hub lint rules | Medium | M | Feature |
| 7 | Multi-bounded-context example under `example/`: two domains, bilateral `Sends`/`Receives`, hub-mergeable exports — the in-repo proof of the data-mesh story | High | L | Documentation |
| 8 | Verify + fill "declare DataProducts + DataContracts end-to-end" cookbook recipe in `references/recipes.md` (README §Entities/DataProducts exists; recipe may not) | Medium | S | Documentation |
| 9 | Hub CI: fail the union-merge build on dangling coeffect refs (mirror `Catalog.ValidateCoeffects` at mesh level) | Medium | M | Feature |
| 10 | Hub: cross-service link completeness report (matrix of both-sides-declared links) | Low | M | Feature |
| 11 | cqrs-lint rule: data product declared without an output `DataContract` → advisory | Medium | M | Feature |
| 12 | Doc: journal-as-outbox (ADR-016) *as the mesh replication mechanism* — connect CatchUpSubscriber to data-product input ports | Medium | S | Documentation |
| 13 | Doc: `schema` upcasting + `DataContract` versioning interplay (contract evolution story) | Medium | M | Documentation |
| 14 | Doc: `metaengine.ServeSSE` + query surfaces as data-product *serving ports* | Low | S | Documentation |
| 15 | Formalize "NOT a federated query engine" from `meta-engine-design.md:116` into a dated ADR (it is currently a planning-doc aside) | Medium | S | Documentation |
| 16 | ADR: mesh-level policy enforcement (privacy/access across domains) — adopt explicitly or reject explicitly; don't leave it undecided | Medium | M | Documentation |
| 17 | Verify `DataProduct` renders in `catalog/docserver` UI (agent did not confirm) | Low | S | Quality |
| 18 | Verify `DataProduct`/`DataContract` are covered by the api-stability golden | Low | S | Quality |
| 19 | gRPC (ADR-0127, v5 removal): migration guide for sync cross-service contracts → HTTP/SSE/brokers, aimed at mesh consumers | Medium | M | Documentation |
| 20 | Hub: owners/teams federation — dedupe of teams/users across per-service exports | Low | M | Feature |
| 21 | Hub: stale-source detection (repo in `sources.json` whose export command fails/missing) | Low | S | Feature |
| 22 | `example/goal-shaped-app`: demonstrate a data product materialized view via `cqrs.yaml` operator config | Low | M | Documentation |
| 23 | ROADMAP entry: "data-mesh-ready SDK" positioning (or reject the positioning deliberately) | Medium | S | Documentation |
| 24 | Cross-link this report's (f) items to the SystemNix hub plan where they are hub-owned (cross-repo routing rule) | Low | S | Cleanup |
| 25 | Spot-verify the three single-source claims from (b2) | Low | S | Quality |
| 26 | Consider `DataProduct` SLA/freshness fields if EventCatalog schema supports them (verify first) | Low | M | Feature |

## g) Questions I cannot answer myself

1. **Durable or disposable?** Is the data-mesh conformance assessment worth a permanent `docs/architecture-understanding/` mapping doc + skill-reference section, or was this a one-off question? (Taste/priority call — yours.)
2. **Mesh governance scope:** should mesh-level policy enforcement (privacy/access rules enforced across domains in the hub) be an explicit roadmap goal for this ecosystem, or permanently out of scope for a library? (Product direction — I can argue both.)
3. **Task ownership:** the cross-repo rule routed the hub's T13 here — should the *new* data-mesh gap items (f7–f26) live in go-cqrs-lite's TODO/ROADMAP or in the `eventcatalog-hub` repo's plan? (Ownership decision, not discoverable from this repo.)

---

*Report format note: written as `.md` per explicit user instruction (status-report skill canonical format is HTML — override flagged). Section (f) is HARVEST-ready; `docs-health` HARVEST routing pending user instruction. Auto-commit daemon will absorb this file.*
