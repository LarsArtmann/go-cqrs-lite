# ADR-0029: Consolidate Storage Backends Under `storage/`

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-20   |
| Status  | Implemented  |
| Decider | Lars Artmann |

## Context

Storage backends are scattered across the repository root:

```
go-cqrs-lite/
├── pebble/          # embedded KV event/snapshot/checkpoint store
├── turso/           # Turso (LibSQL) connector + indexing advisor
├── memory/          # in-memory stores AND buses (mixed concerns)
├── storage/         # SQLEventStore, SQLSnapshotStore, etc. + storage/sql/
│   └── sql/         # dialect, query engine, transaction helpers
└── kv/              # generic KV abstraction (Layer 0)
```

This layout hides related implementations from each other and breaks the
"deployer discovers all backends in one place" principle. A deployer looking for
"where do I plug in Pebble?" has to know it lives at the root, not under
`storage/`.

Meanwhile `stack/pebble/`, `stack/sqlite/`, `stack/postgres/`, and
`stack/memory/` already prove that **subpath modules** (`storage/pebble/v2`)
work fine in the multi-module workspace.

## Decision

**Move all concrete storage backend modules under `storage/` as subpath
modules.**

```
storage/
├── sql/             # (unchanged) dialect, query engine, tx helpers
├── memory/          # ← moved from memory/ (stores only; buses die)
├── pebble/          # ← moved from pebble/
└── turso/           # ← moved from turso/
```

Module paths become `github.com/larsartmann/go-cqrs-lite/storage/pebble/v4`, etc.

`storage/` itself keeps the SQL facade (`SQLEventStore`, `NewSQLiteBackend`).
`memory/` keeps only the in-memory **stores**; the bus implementations are
removed (ADR-0028).

## Alternatives Considered

- **Keep them at the root.** Rejected — preserves discoverability problem and
  mixes infrastructure with domain modules (`event/`, `command/`).
- **Merge everything into `storage/` as a single module.** Rejected — loses
  per-backend dependency isolation (Pebble's `cockroachdb/pebble` dep must not
  leak into consumers who only want SQLite). Subpath modules preserve the
  dependency budget.
- **Put backends under `kv/`.** Rejected — `kv/` is the Layer-0 abstraction;
  concrete backends are Layer 5.

## Consequences

- Import paths change: `pebble/v2` → `storage/pebble/v2`, `turso/v2` →
  `storage/turso/v2`, `memory/v2` (stores) → `storage/memory/v2`. This is a **v3
  breaking change**, documented in the migration guide.
- `go.work` is updated to reference the new subpaths.
- The root directory becomes cleaner: only domain modules (`event`, `command`,
  `query`, `decider`, ...) and infrastructure facades remain.
- `stack/pebble`, `stack/sqlite`, `stack/postgres` import the new subpaths.

## Forward references

- Execution plan T16 (pebble move), T17 (turso move), T18 (indexing advisor
  move), T19 (memory split).
- ADR-0023 (pebble KV adapter) — the adapter moves with `storage/pebble/`.
