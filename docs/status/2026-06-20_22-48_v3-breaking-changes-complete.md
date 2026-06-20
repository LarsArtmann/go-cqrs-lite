# Status Report — v3 Breaking Changes Complete

**Date:** 2026-06-20
**Base:** `c78206df` (post-projection dissolution)
**Head:** `5decab64`

---

## Summary

All five remaining v3 breaking changes are **implemented, tested, and committed**.
37/37 modules green (build + vet + full test suite). The v3 major release is now
feature-complete; only the event-opts deep-copy remains as a minor v3 item.
SSE has been moved to `transport/http/` and the wire format rewritten.

| #   | Change                                                           | Commit     | Effort  | LOC delta  |
| --- | ---------------------------------------------------------------- | ---------- | ------- | ---------- |
| 1   | Break `command/query.Metadata = event.Metadata` alias (ADR-0031) | `318abb6e` | ~45 min | +120 / −40 |
| 2   | Delete dead reactive-streams code (`event/reactive*.go`)         | `1a851c6a` | ~15 min | −343       |
| 3   | Remove `io.Closer` from 9 core interfaces (ADR-0010)             | `8fee4d0f` | ~35 min | +40 / −25  |
| 4   | Rename `Decider.Fold` → `Apply` (naming honesty)                 | `ca29a723` | ~25 min | mechanical |
| 5   | Make `event.Event` a concrete type (`= *ImmutableEvent`)         | `5decab64` | ~40 min | −60        |

**Pre-work:** committed two coherent dedup bases (`447b3ce9`, `3b59c128`) that
extracted `id/idtest` + `query/querytest` MustParse\* helpers and migrated all
module tests — these were uncommitted WIP at session start.

---

## What changed and why

### 1. Metadata alias killed (ADR-0031)

`command.Metadata` and `query.Metadata` were type aliases of `event.Metadata`,
forcing event-only concerns (Tombstone, Causation) onto commands and queries.
Each module now owns its `Metadata` struct embedding `event.Tracing` + a `Custom`
map. `storage/sql.MarshalMetadata` now accepts `any` (JSON is type-agnostic),
severing the SQL layer's dependency on `event.Metadata`'s concrete type. ADR-0031
marked Implemented.

### 2. Reactive dead code deleted

`event/reactive.go` (239 LOC) + `event/reactive_dedup.go` (104 LOC) had zero
production consumers after projection/ deletion. Removed along with the
`samber/ro` production dependency. The deployer-first stack
(`watermill.EventBus` + `bus.SubscribeAll`) is the documented replacement.

### 3. io.Closer removed from interfaces (ADR-0010)

Nine core interfaces (`event.{Bus,EventSink,EventSource,CheckpointSink,CheckpointSource}`,
`snapshot.{SnapshotSink,SnapshotSource}`, `command.{CommandSink,CommandSource,Bus}`)
no longer embed `io.Closer`. Concrete implementations keep their `Close()` method;
callers type-assert to stdlib `io.Closer` when they need cleanup. No new
`Lifecycle` type — `io.Closer` already fills that role without per-module
duplication. ADR-0010 marked Implemented.

### 4. Fold → Apply

`Decider.Fold` was the domain event applier, not a generic fold. Renamed to
`Apply` (plus `ErrNilFold`→`ErrNilApply`, `ErrFoldFailed`→`ErrApplyFailed`) across
the public API, all examples, and integration tests. The new name tells the truth.

### 5. Event made concrete

`event.Event` was an interface with exactly one implementation (`*ImmutableEvent`).
Replaced with `type Event = *ImmutableEvent`. As a type alias, all ~650 call sites
across 37 modules resolved automatically — **zero signature changes**. Deleted all
7 type-assertion sites and the obsolete fallback tests. Normalized cross-package
`*event.ImmutableEvent` return types to canonical `event.Event`.

---

## Verification

- `go build ./<all 37 modules>/...` — exit 0
- `go vet ./<all 37 modules>/...` — exit 0
- `go test ./<all 37 modules>/... -count=1` — all PASS
- API surface regenerated: 1614 → **1609** exports
- BuildFlow pre-commit hook (golangci-lint, gitleaks, gofumpt, etc.) passed on every commit

---

## Remaining v3 work (minor)

1. ~~**Move HTTP code → transport/** (SSE, healthcheck, metrics_http). ADR-0025.~~
   **DONE** — SSE moved to `transport/http/` with rewritten wire format. Healthcheck/
   metrics_http/pprof deleted (generic utilities, zero consumers).
2. **Deep-copy event opts on Clone** — currently the `*eventOptions` pointer is
   shallow-copied. Separate from the concrete-type change.

Both are tracked in `TODO_LIST.md` and `ROADMAP.md`.

---

## Commits (chronological)

```
447b3ce9 refactor: extract id/idtest MustParse* helpers, migrate event tests
3b59c128 refactor: migrate remaining module tests to idtest.MustParse* helpers
1a851c6a refactor: delete dead reactive-streams code (event/reactive*.go)
ca29a723 refactor: rename Decider.Fold → Apply (naming honesty)
8fee4d0f refactor: remove io.Closer from core interfaces (ADR-0010)
318abb6e refactor: break command/query Metadata = event.Metadata alias (ADR-0031)
5decab64 refactor: make Event a concrete type alias (remove interface)
```

Docs updated: `AGENTS.md`, `TODO_LIST.md`, `ROADMAP.md`, `docs/migration/V3_MIGRATION.md`,
ADR-0010, ADR-0031, `docs/api_surface.txt`.
