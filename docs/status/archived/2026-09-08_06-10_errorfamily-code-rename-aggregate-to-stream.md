# Status: errorfamily code rename `aggregate_*` → `stream_*` (v5 batch)

> **RESOLVED + ARCHIVED (docs-health pass 2026-09-08).** Task complete —
> see CHANGELOG `[Unreleased]` "error-family codes renamed to stream
> vocabulary" for the shipped entry, and the companion self-review
> `docs/status/archived/2026-09-08_07-48_self-review-errorfamily-code-rename.md`
> (whose open follow-ups — pin-sweep standing step, aggregate-code tripwire,
> sweep-census extensions — were harvested into TODO_LIST on 2026-09-08).
> The "Explicitly NOT in scope" items below remain open in TODO_LIST → v5
> Unification → "Rest of sweep §4".

**Date:** 2026-09-08 06:10
**Task:** TODO_LIST "errorfamily code rename `aggregate_*` → `stream_*`"
(source: session-4 retro §f30; deferred 2026-09-07 by the correctness batch)
**Artifact:** `docs/planning/v5-deprecation-sweep.md` §4 rule 3

---

## Executive Summary

All 17 `aggregate_*` error-family codes renamed to the stream vocabulary in
ONE batch, with the dashboards/consumers migration note + full mapping table
in CHANGELOG `[Unreleased]`. All touched modules build, test, and lint green;
api-stability golden unchanged (codes are string literals; sentinel symbols
untouched). Two pre-existing breaks found on the way were repaired in the
same wave (stale `storage` go.mod pins; stale T18 snapshot goldens).

## What was renamed (17 codes, 9 modules)

| Old code | New code |
| --- | --- |
| `event.nil_aggregate_id` | `event.nil_stream_id` |
| `event.empty_aggregate_type` | `event.empty_stream_type` |
| `event.aggregate_not_found` | `event.stream_not_found` |
| `command.nil_aggregate_id` | `command.nil_stream_id` |
| `command.empty_aggregate_type` | `command.empty_stream_type` |
| `memory.aggregate_not_found` | `memory.stream_not_found` |
| `storage.parse_aggregate_id` | `storage.parse_stream_id` |
| `storage.parse_aggregate_type` | `storage.parse_stream_type` |
| `storage.aggregate_type_mismatch` | `storage.stream_type_mismatch` |
| `storage.aggregate_id_mismatch` | `storage.stream_id_mismatch` |
| `storage.stream_by_aggregate` | `storage.read_stream` |
| `storage.delete_by_aggregate` | `storage.delete_by_stream` |
| `pebble.aggregate_type_mismatch` | `pebble.stream_type_mismatch` |
| `pebble.aggregate_id_mismatch` | `pebble.stream_id_mismatch` |
| `watermill.parse_aggregate_id_failed` | `watermill.parse_stream_id_failed` |
| `grpc.command.parse_aggregate_id` | `grpc.command.parse_stream_id` |
| `grpc.event_client.parse_aggregate_id` | `grpc.event_client.parse_stream_id` |

The 2026-08-22 sweep-doc census had missed the watermill + transport/grpc
codes; the re-derived census caught them. transport/grpc is deprecated
(ADR-0127, deleted at v5) but still ships, so its codes renamed too — "ONE
batch" means no surviving aggregate code anywhere.

**Deliberate deviation:** `storage.stream_by_aggregate` →
`storage.read_stream` (mechanical `stream_by_stream` is unreadable). The
backing private method `streamByAggregate` → `readStream`
(`storage/eventstore/event_store_stream.go`).

## Consumer impact (the dashboards note)

- `errors.Is` / sentinel matching is UNAFFECTED: deprecated `ErrAggregate*`
  aliases already forward to the same `ErrStream*` sentinels; only the code
  STRING carried by the error changed.
- Observability consumers (log queries, alert rules, dashboard filters)
  keying on old code strings must switch at the upgrade — mapping table in
  CHANGELOG `[Unreleased]`; the 6-family taxonomy is unchanged.

## Explicitly NOT in scope (sweep §4, still open)

- `listing.aggregate_projection` projection name (storage key, not a code)
- watermill metadata keys `aggregate_id`/`aggregate_type` (wire interop)
- events/commands SQL table columns
- pebble `slog` attribute keys `aggregate_type`/`aggregate_id` (log fields,
  not tracked in the sweep census — candidate for a future §4 entry)

## Pre-existing breaks repaired on sight (verified pre-session at 04beab982)

1. **`storage` go.mod stale pins** — today's coordinated release re-tagged 15
   modules but never re-pinned `storage` (not in the release set), so
   standalone `GOWORK=off` builds failed with "updates to go.mod needed".
   Verified at the pre-session commit in a detached worktree; fixed with
   `go mod tidy` (command/event/eventtest/id/query/snapshot pins bumped).
2. **3 stale T18 snapshot-schema goldens** (`postgres`/`sqlite`/`duckdb`
   `-snapshots.snap`) — never re-blessed after the 2026-09-06 column rename;
   re-blessed via `UPDATE_SNAPS=true` (diff is exactly
   `aggregate_*` → `stream_*` PK/columns).
3. **`.golangci.yml` daemon regression** — `gci` re-added to
   `formatters.enable` a FOURTH time; `nix run .#check-lint-config` self-heal
   removed it (that script's documented repair loop).

## Gates (per-task discipline)

| Gate | Result |
| --- | --- |
| GOWORK=off build, 9 touched modules | PASS |
| GOWORK=off tests: event, command, storage, memory, pebble, sql, eventstore, watermill, grpc | PASS (-count=1) |
| Consumer modules decider + scenario | PASS |
| Workspace `go build ./...` | PASS |
| Per-module golangci-lint (9 modules) | PASS (after gci self-heal) |
| api-stability golden | PASS — 6735 exports, no drift |
| `check-changelog-symbols.sh` | PASS — 175 citations honest |
| `cmd/doc-check` (1016 refs, 45 pkgs) | PASS |
| `nix fmt` | 0 changed |
| `check-duplication` | PASS — 0 new clone groups (baseline 133) |

## Lessons

- Re-derive censuses from source, not from planning docs: the 2026-08-22
  list missed watermill + grpc codes. Docs describe the tree at writing
  time.
- A coordinated release that does NOT include a module can still BREAK that
  module standalone (its pins age against new sibling tags). The pin-sweep
  script should run over non-release modules too, or the release wave should
  tidy every module that imports the released set.
