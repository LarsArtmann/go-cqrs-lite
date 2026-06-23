# deployer-first-multidb

Multi-database split example using the SQLite preset.

Demonstrates the **deployer-first** pattern with **concern isolation**: events,
commands/queries, and materialized views each get their own SQLite database
file. The consumer code is identical to single-database mode — only the
deployer wiring changes.

## What This Shows

```
events.db     ← events + snapshots + checkpoints
queries.db    ← commands + queries (audit log)
views.db      ← materialized views (cqrs_kv)
```

This topology eliminates reader/writer contention: heavy event appends don't
block read-model scans, and the audit log is isolated from both.

## Run

```bash
go run .
```

Output: writes 3 counter-increment events, replays them into a materialized
view, and prints the final count — all spread across 3 database files.

## Compare with deployer-first

The `example/deployer-first/` example uses a single in-memory store via
`stack.New()`. This example uses `sqlite.New()` with `WithEventDB`,
`WithQueryDB`, and `WithViewDB` options. The domain code (domain.go, view.go)
is structurally identical — only the infrastructure wiring differs.
