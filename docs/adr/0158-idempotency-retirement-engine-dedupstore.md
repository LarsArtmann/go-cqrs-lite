# ADR-0158: Retire `idempotency/{kvstore,sqlstore}` — the engine `DedupStore` capability is the dedup primitive

Date: 2026-10-12 · Status: Accepted · Supersedes: nothing · Relates to: ADR-0142 (universal storage substrate / engine ADT capabilities), ADR-0128 (core extraction), ADR-0152 (fleet-first module topology, ghost verdicts), ADR-0123 (system composition root)

## Context

Three overlapping dedup layers exist in the v4 tree:

1. **`metaengine.DedupStore`** (ADR-0142): the engine-capability ADT — `DedupCheckAndRecord` (atomic CAS with TTL window), `DedupSeen`, `DedupSweep`. Semantics pinned by `adttest.AssertDedupStore`. SQL engines implement natively via `metaengine/claimkit`; map-shaped engines via `MapDedupStore` (degraded, declared in `DegradedADTs`).
2. **`idempotency/sqlstore/v4`**: proven-in-production store whose shape ADR-0142 promoted INTO the engine capability. Since `NewFromEngine(metaengine.DedupStore, collection)` it is a thin adapter OVER layer 1 — a facade with a second name.
3. **`idempotency/kvstore/v4`**: the pre-ADR-0142 shape — hand-rolled expiry windows over a raw KV backend (`KVBackend`), `Seen`/`Record`/`CheckAndRecord`/`Sweep`.

In-repo consumers of layers 2–3: only `integration` (itself an untagged test suite). The fleet's dedup consumers (middleware, commandlifecycle, taskmanager) use the **extracted external core** `github.com/larsartmann/go-idempotency` (ADR-0128) — out of scope here. ADR-0152 left `idempotency/{kvstore,sqlstore}` under "ghost verdicts pending T02"; this ADR is that verdict.

## Decision

**Layer 1 wins. `idempotency/{kvstore,sqlstore}` are deprecated at v4.x (doc-comment `Deprecated:` markers now) and deleted at v5.0.0.** No code is removed in v4; the modules keep compiling and their suites keep passing until v5.

The migration target is the engine capability, reached two ways:

- **Direct**: `var dedup metaengine.DedupStore = engine.(metaengine.DedupStore)` — capability assertion, the ADR-0142 pattern (`DegradedADTs` documents which engines degrade to `MapDedupStore`).
- **Through the composition root (the facade shape)**: `(*system.System).Dedup()` resolves the deployment's dedup engine — the engine named `"dedup"` when present, else the first configured engine — capability-asserts it, and returns the `DedupStore` (error when no engine qualifies). This mirrors the checkpoint-engine resolution (`checkpointEngine()`, named-else-first with runtime assertion) and gives `system.New` consumers a one-call migration from both retiring modules.

Migration mapping:

| Retiring call | Replacement |
|---------------|-------------|
| `sqlstore.NewFromEngine(eng, coll)` + `Seen/CheckAndRecord/Record` | `sys.Dedup()` → `DedupSeen` / `DedupCheckAndRecord` / (record is fused into the CAS) |
| `sqlstore` `Sweep` | `DedupSweep` |
| `kvstore.New(backend)` + `Seen/Record/CheckAndRecord/Sweep` | same, over the deployment's engine (map engines get `MapDedupStore` — degraded semantics declared, same TTL window contract) |

Semantic deltas consumers must review (documented on the capability): `CheckAndRecord` in the retiring modules returns `error` on duplicates; the capability's `DedupCheckAndRecord` returns `(seen bool, err)` — callers branch on the boolean. Expiry reads as unseen in both worlds; an expired window may be re-claimed in both.

## Consequences

- One dedup primitive with pinned cross-engine conformance instead of three.
- `idempotency/` leaves the v5 tree entirely (two fewer module trains to migrate).
- The go-idempotency external core (ADR-0128) is UNAFFECTED — it composes any `Store`-shaped backend and its in-fleet consumers keep working; a future note there can point engine-backed implementations at the capability.
- Deprecation is discoverable: static-analyzer candidates ride the existing deprecation report (`cmd/cqrs-upgrade`), not a new rule.

## Verification

- `(*system.System).Dedup()` pinned by unit tests (named-engine preference, fallback, no-capability error).
- `adttest.AssertDedupStore` remains the cross-engine semantic pin.
- v5 deletion: ADR-0123 cascade, slice tracking in the v5 plan (M09/M10 sequence).
