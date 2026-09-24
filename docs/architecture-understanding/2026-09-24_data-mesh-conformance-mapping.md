# go-cqrs-lite → Data Mesh: Conformance Mapping

> **Date:** 2026-09-24
> **Kind:** Point-in-time conformance mapping (assessment made durable)
> **Sources:** Data-mesh conformance assessment session
> ([status report, 2026-09-23](../status/2026-09-23_18-09_data-mesh-conformance-assessment-session.md));
> repo state as of `master` after the 2026-09-23/24 federation-hub wave
> (catalog.index.json, `WithSkipBootstrapFiles`, `WithPlainRefIDs`,
> `Flow.Owners`). Companion paradigm mapping:
> [Cordis → go-cqrs-lite](2026-09-10_cordis-spatiotemporal-composability-mapping.md).

Data mesh (Dehghani) is a SOCIO-technical approach: four principles —
domain ownership, data as a product, self-serve data platform, federated
computational governance. go-cqrs-lite is a LIBRARY, not an organization
transformation; this doc maps how well its mechanics ENABLE each principle,
with the honest gaps recorded. Positioning in one sentence: **a
data-mesh-ready SDK per bounded context** — it powers the cells, not the
mesh itself.

---

## 1. Principle-by-principle conformance

| # | Principle                    | Verdict          | Load-bearing mechanics (verified)                                                                                                                                       |
| - | ---------------------------- | ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Domain ownership             | **Strong**       | Deciders own state per bounded context; branded stream IDs (`id.StreamID`, string-backed semantic keys, ADR-0111 d); `catalog.Domain` + ubiquitous-language sidecars; per-module versioning |
| 2 | Data as a product            | **First-class**  | Typed `catalog.DataProduct` / `catalog.DataContract` (`catalog/types_resources.go`); inputs/outputs with output contracts; `owners` on every ownable kind (incl. flows, 2026-09-24); `catalog.index.json` per-export manifest |
| 3 | Self-serve data platform     | **Strong**       | `system.New` DomainConfig-vs-DeploymentConfig split is COMPILER-ENFORCED (developers declare behavior, operators place data); `cqrs.yaml`; metaengine cost-planned routing with operator-picked engines |
| 4 | Federated computational governance | **Partial** | Three-tier coeffect validation (runtime `ErrDanglingEventSubscription`, cqrs-lint E018, docs-side `catalog.ValidateCoeffects`); EventCatalog contracts (see §3); mesh-level enforcement is an explicit non-goal (ADR-0147) |

## 2. Module mapping per principle

| Principle | Modules that carry it                                                                                                                            |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1         | `decider`, `command`, `event` (journal = the domain's own history), `id`, `deriver` (sagas keep compensation in-domain), `scheduling`               |
| 2         | `catalog` (`DataProduct`/`DataContract`/`AddDataProduct`), `catalog/eventcatalog` (renders products into EventCatalog), `metaengine` (`ServeSSE` output ports), `watermill` (`CatchUpSubscriber` input ports) |
| 3         | `system` (`DomainConfig`/`DeploymentConfig`), `metaengine` (planner, `Profile()`-declared engines, calibration), `storage/*` (engine menu), `stack` |
| 4         | `catalog` coeffects (three tiers), `cmd/cqrs-lint` (E018 et al.), `catalog/eventcatalog` governance exports (`WithPlainRefIDs` — lint-clean at full severity, enforced by `check-eventcatalog`), the external [eventcatalog-hub](https://github.com/LarsArtmann/eventcatalog-hub) |

## 3. Governance: what the exporter wave changed

Before 2026-09-24 the federation hub had to warn-suppress
`refs/resource-exists` and `best-practices/owner-required`. Direct
experiment proved the real conflict: `@eventcatalog/core` resolves
composite `<id>-<version>` Astro entry IDs (visualiser graph edges use
them), while `@eventcatalog/linter` indexes bare frontmatter IDs and can
NEVER resolve the composite form. The exporter now ships both shapes:

- default exports → what the hub renders (composite refs),
- `WithPlainRefIDs` exports → what governance lints (bare IDs, owners on
  every kind) — clean at full severity, enforced in CI by
  `nix run .#check-eventcatalog`.

That is the bilateral-contract rendering rule made mechanical: a source's
export is checkable WITHOUT trusting it, and `catalog.index.json` makes
hub-side change detection a JSON diff.

## 4. Honest gaps

| Gap                                             | Status                                                                                     |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------ |
| Cross-source federated queries                  | **Rejected by design** — replication via journal-as-outbox is the mechanism (ADR-0146)     |
| Mesh-level policy enforcement in this library   | Explicit non-goal; belongs to the hub (ADR-0147)                                            |
| gRPC sync transport                             | Deprecated at v5; mesh consumers move to HTTP/SSE/broker (migration guide, see FAQ)         |
| Multi-bounded-context example                   | `example/mesh-demo` (in flight)                                                            |
| SLA/freshness fields on data products           | Under investigation against EventCatalog's schema (ROADMAP raw idea)                       |

## 5. Relation to the Cordis mapping

The [Cordis mapping](2026-09-10_cordis-spatiotemporal-composability-mapping.md)
maps the SAME repo onto spatiotemporal composability: developers write
coeffect specifications (queries + relationships ≈ data products), operators
reconcile config (engines ≈ platform placement), the planner mediates.
Data-mesh principles are the ORGANIZATIONAL projection of that same split:
principles 1–2 are the developer/coeffect axis, principle 3 is the
operator/config axis, principle 4 is the context paradigm's global
equivalence class — here, the hub's union-merged contracts.
