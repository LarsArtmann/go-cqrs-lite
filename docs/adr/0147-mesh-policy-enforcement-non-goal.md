# ADR-0147: Mesh-Level Policy Enforcement Is an Explicit Non-Goal

**Date:** 2026-09-24
**Status:** Accepted
**Related:** ADR-0146 (no federated query engine), [data-mesh conformance mapping](../architecture-understanding/2026-09-24_data-mesh-conformance-mapping.md)

## Context

The data-mesh conformance assessment (2026-09-23) graded federated
computational governance as the weakest of the four principles and left a
question open: should the LIBRARY grow mesh-level policy enforcement
(cross-repo contract checks, org-wide standards, catalog-wide quality
gates)?

The library already carries strong REPO-LOCAL enforcement: the runtime
dangling-subscription gate (`ErrDanglingEventSubscription`), cqrs-lint
(including E018 projection-without-emitter), catalog validation, and —
since 2026-09-24 — a `check-eventcatalog` gate that render-validates
exports with @eventcatalog/core AND lints them with
`@eventcatalog/linter` at full severity. Those are all checks a repo runs
on ITSELF.

Mesh-level enforcement checks OTHER repos against a shared standard: "every
declared producer has a consumer somewhere in the org", "every data product
has an SLA", "contract versions never regress org-wide". The federation hub
(eventcatalog-hub) is where those checks would run — it already union-merges
per-source exports, diffs `catalog.index.json`, and renders the union.

## Decision

**Mesh-level policy enforcement is a non-goal for go-cqrs-lite.** The
division of labor:

| Layer              | Owner            | Enforces                                                                         |
| ------------------ | ---------------- | -------------------------------------------------------------------------------- |
| One repo's catalog | go-cqrs-lite     | Schema validity, coeffect completeness, lint-clean exports, render-clean exports |
| The mesh / union   | eventcatalog-hub | Cross-source contract resolution, dangling cross-repo refs, org-wide standards   |

The library's obligation to the mesh is to make each source's export
TRUSTWORTHY and machine-checkable — deterministic manifests, typed
contracts, owners on every resource, both ref formats (render + lint) —
not to adjudicate between sources.

## Consequences

- Feature requests of the form "fail the build if another team's consumer
  disappears" route to the hub, not here.
- The exporter's contract stays one-directional: it emits what THIS repo
  declared; it never reaches into other repos' exports.
- If the hub later needs richer signals (SLA fields, freshness, semantic
  diffs), the library's job is emitting the raw material (typed catalog
  fields), not the cross-source policy.
- This boundary is why T22–T24 (stale-source detection, fail-on-dangling
  coeffects at merge, owners dedupe) live in the hub repo, per decision D3
  of the 2026-09-23 data-mesh plan.
