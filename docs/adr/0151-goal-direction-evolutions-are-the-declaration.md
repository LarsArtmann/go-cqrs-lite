# ADR-0151: Goal Direction — "Declare ONLY" Means Evolutions + Queries

- Status: Accepted
- Date: 2026-10-08
- Deciders: owner blanket execution authorization 2026-10-08 ("GET SHIT DONE") adopting the memo's recommended option (c) HYBRID; evidence pack delivered 2026-09-21 (G-T01)
- Supersedes: the undefinable reading of the AGENTS.md Goal sentence (G-T02)
- Note: TODO rows called this "ADR-0147"; that slot was claimed by mesh-policy-enforcement-non-goal, so the ruling lands here
- Related: ADR-0116 (layered auto-projection — Layer-1 status addendum below), ADR-0114 (tombstones as domain events)

## Context

The Goal sentence — "Developers declare ONLY Commands + Events + Queries and
their relationships…" — was only literally satisfiable by runtime
`Infer`/`InferFromNamedEvents`, machinery the library Deprecated for removal at
v5 (invisible fold mappings, struct-name conventions in a wire-type world, zero
production call sites). Until ruled, Goal-closure Gate A was blocked and the
Goal undefinable. Evidence pack:
[`docs/planning/2026-09-21_direction-ruling-evidence-and-decision-memo.md`](../planning/2026-09-21_direction-ruling-evidence-and-decision-memo.md).

## Decision

**(c) HYBRID — reframe now, codegen as evidence-gated opt-in later.**

1. **The sanctioned meaning of "declare ONLY":** developers declare Commands,
   Events, Queries, their relationships, and **one Evolution per read model
   (the fold declaration)**; the system wires and routes every projection.
   `Infer` stays dead and is removed at v5 per plan. Counters and non-row
   shapes are declared directly (`Count(...).On(...)` IS the declaration).
2. **The revival door stays open but parked:** a `cqrs-gen` fold-generation
   one-pager (G-T03) is filed as documentation only, activatable exclusively
   by evidence — G-T16 parity-benchmark numbers or a named consumer ask
   showing the Evolution declaration is a real adoption blocker.
3. **AGENTS.md Goal sentence is amended** to the reframe wording (see
   Consequences) so the north star is definable at 100%.

## Consequences

- Goal-closure Gate A is unblockable (this ADR + routing integration tests
  across ≥2 engines).
- ADR-0116 Layer-1 gets a status addendum: the runtime-reflection form is
  retired; the Evolution-convention declaration (`AutoCRUDByNamedEvents` keyed
  by declared wire event types, coeffect-gated, Doctor-visible, lint-covered)
  IS the sanctioned Layer 1.
- The declaration is MORE auditable than `Infer` ever was: wire types listed,
  warn-first on partial coverage, coeffect-gated, E018-visible.
- REVIVE-as-primary was rejected: M+ effort re-solving a solved problem
  against the anti-Verschlimmbesserung clause; no consumer (incl. CV) is
  blocked by the Evolution keyword.

## Amended Goal sentence (AGENTS.md)

> "Developers declare ONLY Commands + Events + Queries, their relationships,
> and one Evolution per read model (the fold declaration). We should be able
> to build superb projections (materialized views) — wired and routed by the
> system — and developers never need to worry about anything else, while
> where data lives is up to operators at DEPLOYMENT time."
