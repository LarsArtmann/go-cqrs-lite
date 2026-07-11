# TODO List

**Updated:** 2026-07-11 (session: completed P3 polish items + CBOR→JSON SSE e2e test)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md). Raw ideas live in [ROADMAP.md § Raw Ideas](ROADMAP.md#raw-ideas-no-design-yet).

## Legend

- `[ ]` = Open
- `[x]` = Done (moved to [CHANGELOG.md](CHANGELOG.md))
- `[v4]` = Breaking change, deferred to v4
- `[BLOCKED]` = Blocked on upstream dependency

---

## P3 — Polish & Cleanup

_All items in this section have been completed and moved to [CHANGELOG.md](CHANGELOG.md)._

### SSE transform follow-ups (Gap 2)

- [x] **CBOR→JSON e2e test** — `TestSSEHandler_PayloadTransform_CBOR_ToJSON_BrowserFlow` added in `transport/http/sse_options_test.go`. Moved to CHANGELOG.

### Projectionhost observability

- [ ] **`LagPerProjection() map[string]time.Duration`** — Per-worker lag for dashboards (item 38).
- [ ] **`WorkerState.Lag` field** — Currently only available via aggregate `LagDuration()` (item 39).
- [ ] **`Reset(ctx, name)` purges DLQ** — Projection reset should optionally clear DLQ entries for that projection (item 46).

### Testing improvements

- [ ] **Race detector on ALL modules** — CI runs `-race` on changed modules; run full suite periodically (item 13).
- [ ] **`scenario.GivenProjection` test** — For VersionedSeekableJournal + projectionhost (item 48, P3).

### Documentation

- [ ] **Document two DeadLetterEntry types** — ADR-0043 Part B: dispatch-side vs projection poison, intentionally separate (item 45, P3).
- [ ] **README.md docs freshness** — Missing `encryption`, `turso`, `testutil` module sections.

### Experimental / Go-stdlib-blocked

- [BLOCKED] **jsonv2 codec experiment** — Pending Go stdlib stabilization (expected Go 1.27+).
- [BLOCKED] **Arena allocation experiment** — Pending Go arena API stabilization.
- [BLOCKED] **Turso MVCC concurrent-write support** — Blocked on upstream experimental MVCC.

### Rejected (with reasons)

- **Unify VersionedStore + VersionedSeekableJournal** (item 20) — Different interfaces (Store: Load/Save per aggregate, SeekableJournal: ReadFrom position-based). One type can't cleanly implement both. YAGNI.
- **VersionedJournal (ReadAll only)** (items 22, 49) — No consumer needs `ReadAll` with upcasters. SeekableJournal is the projectionhost interface. YAGNI.
- **Expose `SSEBroker.PayloadTransform()` accessor** (item 24) — No demand. Consumers configure the transform at construction.
- **`WithPayloadTransform` on SSEHandler** (item 27) — SSEHandler wraps the broker; adding transform there duplicates responsibility (SRP violation).
- **Auto-apply CQRS views by default** (items 35, 50) — Violates "library, not framework" principle. Consumers choose their histogram boundaries.
- **VersionedSeekableJournal implementing event.Store** (item 40) — Different scope (position-based vs aggregate-based reads). YAGNI.
- **Integration test in `integration/` module** (item 19) — Redundant with `projectionhost/versioned_journal_integration_test.go`.

### Transport

- [ ] **NATS/ValKey Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and `transport/redis/` modules.
- [ ] **Distributed event bus** — No multi-process backend for event distribution.

### Public Release Readiness

> **Release strategy decided 2026-07-11:** Public release (license swap + history scrub) happens **AFTER v4 cut**, not before. Not a current priority — v4 breaking changes (deprecated API removal, Storage/ split) come first.

- [ ] **License swap (PROPRIETARY → Apache-2.0)** — Hard blocker for public adoption. **Needs user approval (irreversible).** After v4.
- [ ] **Git history scrub for internal docs** — AGENTS.md, docs/planning/\* contain internal strategy. **Needs user approval (irreversible).** After v4.
- [ ] **Postgres CI coverage matrix** — Add CI Postgres service or label experimental.
- [ ] **README polish to "sales page" standard** — Per AGENTS.md rule.

---

## v4 Breaking Changes (deferred)

> **Partially unblocked.** Codec default flip is DONE (see CHANGELOG). Remaining items await final v4 cut.

- [v4] **Remove deprecated APIs** — 8 aliases in event/ + schema/ + query.Handler.
- [v4] **Storage/ split execution** — Proposal at `docs/planning/2026-07-09_STORAGE-SPLIT-PROPOSAL.md`. Awaits approval.
- [v4] **Event/ god module decomposition** — Explicitly decided: DO NOT SPLIT (27 importers, cohesion is real).
- [v4] **BackfillHandler taking \*SSEBroker** — Cleaner architecture; backfill can access broker's transform (item 47).

---

_All completed items have been moved to [CHANGELOG.md](CHANGELOG.md) under `[Unreleased]`._
