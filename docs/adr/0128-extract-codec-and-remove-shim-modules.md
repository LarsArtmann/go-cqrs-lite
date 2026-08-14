# ADR-0128: Extract codec/ to go-codec and Remove the Deprecated Shim Modules

**Date:** 2026-08-14
**Status:** Accepted
**Follows the pattern of:** ADR-0064 (retry → go-retry), ADR-0065 (idempotency → go-idempotency)
**Builds on:** ADR-0123 (v5 unification horizon)

---

## Context

Four utility packages were extracted from go-cqrs-lite into standalone repos
and left behind deprecated re-export shim modules inside the monorepo:

| Shim module | External repo (published) | Extraction ADR |
| --- | --- | --- |
| `codec/v4` | `github.com/larsartmann/go-codec` (v0.1.0) | none — this ADR |
| `retry/v4` | `github.com/larsartmann/go-retry` (v0.3.1) | ADR-0064 |
| `idempotency/v4` | `github.com/larsartmann/go-idempotency` (v0.1.2) | ADR-0065 |
| `flightrecorder/v4` | `github.com/larsartmann/go-flightrecorder` (v0.2.0) | none — this ADR |

The shims were pure alias layers (`type X = gorepo.X`, forwarding funcs). Their
costs: four extra go.mod files in the workspace, four entries in
flake `testModules`, api-stability, cqrs-lint's catalog, and the layer-check
script; plus deprecation lint exclusions in `.golangci.yml`.

At the time of removal, all *internal* source had already been migrated to the
external imports, with three exceptions this ADR's removal also fixed:
`decider`, `middleware`, `projectionhost`, and `stack` still imported the
`flightrecorder` shim; `middleware`, `idempotency/kvstore`, and
`idempotency/sqlstore` still imported the `idempotency` shim.

## Decision

1. **Migrate the remaining internal consumers to the external imports.**
   `go-flightrecorder` and `go-idempotency` are now direct production
   dependencies of the consuming modules (workspace siblings via `go.work`;
   versioned requires in each `go.mod`).
2. **Delete all four shim modules** from the monorepo: `codec/`, `retry/`,
   `idempotency/` (parent only — `idempotency/kvstore/` and
   `idempotency/sqlstore/` remain as independently versioned modules in their
   existing paths for consumer stability), and `flightrecorder/`.
3. **Remove every registry reference**: `go.work` use block, flake.nix
   `testModules` (which also feeds `#lint`) and `wasmMods`, the
   `cmd/api-stability` modules slice, `scripts/check-module-layers.sh`
   `LAYER`/`DEP_BUDGET` entries, and `.golangci.yml` path exclusions.
4. **cqrs-lint recognizes only the external packages.** ImportHints for the
   four modules point at `go-codec`/`go-retry`/`go-idempotency`/
   `go-flightrecorder` (plus the surviving `idempotency/{kvstore,sqlstore}`
   paths); E001's tier-0 list drops `codec`.
5. **Published tags remain valid.** The Go module proxy serves the tagged
   `*/v4` versions indefinitely; consumers pinned to a shim keep building.
   No further releases of the shim module paths will be cut.

## Consequences

- **Positive:** Workspace shrinks by four modules; one lint-exclusion block,
  one dep-budget entry, and one api-stability module per shim disappear.
  New development happens only in the external repos.
- **Positive:** The pre-extraction `codec.COSESign1String` /
  `codec.COSEEncrypt0String` symbols are not resurrected: they were dropped
  before extraction, and the published go-codec surface is now the only
  surface. (Backward-compat aliases would only matter for a codec/v4 patch
  release that will never happen.)
- **Negative:** Transitive `// indirect` requires of the four shim paths
  linger in consumer go.mod files until every upstream module that ever
  depended on a shim is re-tagged; they are cosmetic and will be tidied away
  by the v4.x re-tag chain tracked in TODO_LIST.md (Release/Tagging).
- **Negative:** `idempotency/{kvstore,sqlstore}` live under a namespace whose
  parent module no longer exists. Renaming them would break consumers for
  zero functional gain; kept as-is deliberately.
