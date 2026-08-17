# system/v4 Review — Routed Design Proposals

**Date:** 2026-08-17
**Source:** full code review `docs/reviews/2026-08-16_full-code-review-system.html`
**Status:** proposals for Lars — none decided; each states the problem (code-verified), options, and a recommendation.

---

## 1. Count-by-name dispatch (P1-2)

**Problem.** `metaengine.ExecuteTyped` and the typed reader dispatch by input
*type* (`metaengine/execute.go:27-36`). Every `Count()` projection uses the
same `CountInput`, so a second `Count()` registration silently shadows the
first; `system.GetCount`'s name parameter only feeds error messages. Get/Find
are unaffected (they dispatch by name via their input's `ID`/filter fields).

**Options.**
- A. metaengine gains named dispatch: `ExecuteTypedByName[Q,R](name, input)` —
  queries register in a name→handler map alongside the type map. Additive,
  cross-module release required.
- B. system wraps counters with per-name phantom input types via reflection —
  keeps metaengine untouched but adds fragile reflection in system.

**Recommendation.** A. The type-keyed map is an implementation shortcut, not a
contract; named lookup already exists for readers. Ship as metaengine minor
+ system minor.

## 2. Named-bus API

**Problem.** Fan-out buses from `Publish: [a, b]` are reachable only via the
positional `MultiBus.Publishers()` dig; publish target names are discarded
after counting entries. Tests pin the positional contract.

**Proposal.** `System.PublisherFor(target string) (event.Publisher, bool)` +
`MultiBus.PublisherAt(name)`. Backward compatible; the positional accessor
stays. Enables NATS-for-external + GoChannel-for-local topologies to be
addressed by the names the operator already writes in YAML.

## 3. Role wiring (RoleCommands / RoleQueries / RoleSnapshots)

**Problem.** Dedicated role instances are parsed but never wired — command/
query audit stores only attach to the source-of-truth instance. The dead
conditions were removed (449e0e5a7); the semantics remain unimplemented.

**Proposal.** Wire them symmetrically in `New()`: a `commands` instance binds
`NewCommandAdapter` on its engine; `queries` likewise; `snapshots` binds
`NewSnapshotAdapter`. Open question: can one engine serve two roles
(collection separation is already in place — likely yes).

## 4. Reserved-config honesty table

| Field | Status | Recommendation |
| --- | --- | --- |
| `BusConfig.Mode` | parsed, never read (README documents `mode: sync`) | implement (sync/async publish semantics) or delete field + README example |
| `InstanceConfig.Subscribe` | parsed, never read | implement (bus subscription per instance) or delete |
| `InstanceConfig.Collections` | parsed, surfaced in topology only | keep (introspection value) but document as non-behavioral |
| `CacheConfig.Engine` | parsed, never read | delete — cache wraps the event store, not an engine |

Rule going forward: any config field that is parsed must either change
behavior or carry a doc comment saying it is introspection-only.

## 5. Durability wiring

**Problem.** `DurabilityTier` (strict/normal/relaxed) is parsed and feeds one
scream rule; nothing maps it to engine behavior.

**Proposal.** Per-engine pragma mapping applied at engine construction:
sqlite → `synchronous`/`journal_mode`, pebble/badger → sync writes,
postgres → `synchronous_commit`. Relaxed = engine defaults; strict = most
durable pragmas. Fail construction on unsupported combinations rather than
silently ignoring.

## 6. EventAdapter backend contract

**Problem.** `Save` atomicity depends on the backend (memory: atomic under
lock; sqlite: transactional; others: unverified), but the contract is
undocumented — consumers cannot reason about crash windows.

**Proposal.** Document an `Atomic | Transactional | Racy` classification per
backend in `system/doc.go` + metaengine engine docs; have engines self-declare
if the interface ever needs it.

## 7. Release coordination (from the replace audit)

system/v4.4.0 (published) still contains all 5 P1 bugs. Fixes are
master-only. Required sequence: metaengine release (local is ≥12 commits past
v4.11.0: MariaDB generated-column layout, vector ADT, CBOR-default
unification) → engine adapter releases → system/v4.5.0 with replaces stripped
(go-release flow). Until then, consumers on published tags get the pre-fix
behavior — including the stack.Bundle users below.

## 8. stack.Bundle cross-check (verified 2026-08-17, read-only)

Bundle does NOT share the scream-store bugs — it has no ack/WARN machinery at
all (no CheckSafety equivalent; nothing to port). The interesting direction is
the reverse: **stack already wires Durability** — each preset translates
`DurabilityTier` to backend-native options via `WithDurability` adapters
(stack/durability.go), exactly what system's Durability proposal (§5) needs.
Implementation input: reuse the stack preset translation tables as the
per-engine pragma mapping instead of inventing a second one; they are
deletion candidates for v5 Phase 8, so lift them into metaengine (or system)
rather than importing stack.

**Dep-diet note (same audit).** system/go.mod carries sqlite + 4 engine
adapters (badger/pebble/pg/sqlite) as DIRECT requires only because
main_test.go blank-imports them for driver registration. Options: keep
(tests need drivers; consumers don't inherit the requires — they're already
indirect for importers) vs. move the blank imports into system/integration
like duckdb. Recommendation: keep — the modules are small go.mod entries,
and moving them would gut the root test suite's engine coverage.
