# G-T03 One-Pager: cqrs-gen Fold Codegen (PARKED — evidence-gated)

> **Status: PARKED by ADR-0151 (2026-10-08).** This is documentation only, no
> code. Activation gate: the G-T16 parity-benchmark numbers OR a named consumer
> ask showing the Evolution declaration is a real adoption blocker. Until then
> this page must not drive any build work.

## The idea

`cqrs-gen` (or `go:generate`) generates fold declarations from type inspection:
AST → fold-spec → rendered, diffable, committed `Evolutions` declaration. CRUD
views would need zero handwritten fold lines while the output stays visible and
auditable in the diff — answering the auditability objection that killed
runtime `Infer` (Deprecated 2026-09-16, removed at v5) by construction.

## Why it is parked, not built

- The sanctioned surface already delivers declare-once-wire-everything for
  every row-materializing shape (`DomainConfig.Evolutions` + inheritance);
  the flagship example (`example/goal-shaped-app`) needs zero fold closures.
- Count is the one shape least expressible by convention (deltas carry intent)
  — codegen would not remove its declaration either.
- ADR-0116 explicitly rejected codegen as the sole path ("complementary, not
  the primary mechanism"); `cqrs-gen` today emits only typed
  handler-registration stubs from `//cqrs:` markers — fold generation does not
  exist (M+ to build).
- No consumer — including CV — is blocked by the Evolution keyword.

## If activated

1. Scope to CRUD-shaped views (Created/Updated/Deleted convention classes).
2. Output is a committed, tested declaration file — never runtime reflection.
3. Activation evidence and G-T16 numbers recorded here before any code lands.

Evidence base: direction memo
[`2026-09-21_direction-ruling-evidence-and-decision-memo.md`](2026-09-21_direction-ruling-evidence-and-decision-memo.md) §3(b), §4.
