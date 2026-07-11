# TODO List

**Updated:** 2026-07-11 (session: projectionhost observability + scenario tests + ADR-0043 Part B + README freshness)
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

- [x] **`LagPerProjection() map[string]time.Duration`** — Per-worker lag for dashboards (item 38). Moved to CHANGELOG.
- [x] **`WorkerState.Lag` field** — Populated in `snapshot()` via `worker.lagDuration()` (item 39). Moved to CHANGELOG.
- [x] **`Reset(ctx, name, opts...)` purges DLQ** — `WithPurgeDeadLetters()` option clears DLQ entries on reset (item 46). Moved to CHANGELOG.

### Testing improvements

- [x] **Race detector on ALL modules** — Full suite passes `-race` across all 48 modules; only `cmd/api-stability` fails (pre-existing: subprocess doesn't inherit goexperiment tags) (item 13). Moved to CHANGELOG.
- [x] **`scenario.GivenProjection` tests** — Added `ThenError`, multiple-events, and empty-events tests (item 48). Moved to CHANGELOG.

### Documentation

- [x] **Document two DeadLetterEntry types** — ADR-0043 Part B added: consumer operational guide with decision tree, code examples, and structural comparison (item 45). Moved to CHANGELOG.
- [x] **README.md docs freshness** — Fixed stale `testutil` API refs (`MustNewCmd`→`NewCmd`, removed `ParseAggID`, `NoopCommandHandler{}`→`NoopCommandHandler()`) and `v2`→`v3` paths in `testutil/README.md`. Moved to CHANGELOG.

### Experimental / Go-stdlib-blocked

- [BLOCKED] **Remove `goexperiment.jsonv2` tag** — JSON v2 is fully adopted (~25 production files import `encoding/json/v2`). The build tag remains only because Go 1.26 hasn't graduated json/v2 from experimental. Remove the tag when Go stabilizes it (expected Go 1.27+).
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
