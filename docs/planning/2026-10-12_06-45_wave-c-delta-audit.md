# Wave-C Engine Delta Audit (N04) — 2026-10-12

Method: per-module `git diff <newest-tag>..HEAD` with nested-submodule paths
attributed to their own module and `go.mod`/`go.sum` churn excluded, then each
remaining content delta classified. F129 (`tag-release.sh --audit --baseline
scripts/audit-tag-baseline.txt`): 1446 tags checked, **0 NEW violations**.

## Class 1 — content on TAGGABLE trains (wave III seed)

| Module | Δ files | First-read verdict | Wave III disposition |
|--------|---------|--------------------|----------------------|
| middleware | 28 | errorfamily import regrouping + deadletter/flightrecorder additions | inspect → minor tag |
| encryption | 15 | M25 (ADR-0139 completion) territory | fold into M25 |
| command | 14 | AsRecord/BDD-era test + aggregate_ref work | inspect → minor tag |
| query | 13 | AsRecord/capability test work | inspect → minor tag |
| stack (root) | 13 | v5-deletion surface (M09/M10) | DEFER — tags would immortalize a surface scheduled for deletion; deprecation annotations ride M09 |
| storage (root) | 9 | test-only (sqlite_helpers, snapshot_migration_edge) | patch tag or defer with tests-only note |
| otel | 7 | inspect (11 total incl. pins) | inspect → minor tag |
| id | 6 | inspect | inspect → minor tag |
| scheduling | 5 content (14 total) | store.go +1 line, test tweaks | patch tag |
| record/metadata/prometheus/commandlifecycle | 3 each | small; likely import regrouping | blanket patch sweep |
| signing/scenario/projection/listing/kv | 2 each | small | blanket patch sweep |
| dedup | 1 | post-wave residue | blanket patch sweep |

**Wave III shape (deferred consciously):** one inspection pass over the ≥5-file
rows to separate real API from import-regrouping, then a single minor/patch
sweep in dependency order. NOT urgent — none of these rows carries a
consumer-blocking fix; the trust-critical fixes (#62/#63/#64) already shipped
in wave II.

## Class 2 — untagged trains (ADR-0152 policy; deltas ride integration suites)

`example/*` (taskmanager 27, getting-started 8, goal-shaped-app 5,
scheduler-otel-status 4, metaengine-quickstart 4, readme-quickstart 2),
`systemtest` 16, `integration` 15, `event/v4/eventtest` 20,
`cmd/cqrs-upgrade` 11, `cmd/cqrs-bench` 10, `cmd/doc-check` 8,
`cmd/api-stability` 8, `benchkit` 4, `cmd/cqrs-gen` 2, `testutil/*`,
`transport/{grpc 12, http 6}`, `queue/{mysql, postgres}`,
`metaengine/{irohengine 21, badgerengine, bboltengine, bigtableengine,
dgraphengine, duckdbengine, mysqlengine, otelobserver, bench, graphadapter}`,
`storage/{pebble, backuptest}`, `otel/otlp`, `idempotency/{kvstore, sqlstore}`
(ghost verdict pending T02), `graph` (consumer arrived — deriver removal
note in untagged-trains.txt covers systemscenario only; graph itself stays
untagged until its own first consumer), `stack/*` presets (removed at v5),
`stack/metaengine` 2.

No action: their call-site updates ride the next content tag each train earns
(established 10-11 wave precedent).

## Class 3 — pin-only (no content, no action)

Every module tagged in trust wave II (schema, snapshot, decider, catalog,
systemscenario, system, metaengine, sqliteengine, cqrs-lint) plus watermill,
projectionhost, queue/sqlite: post-tag deltas are exclusively go.mod/go.sum
pin settles from the wave's dependent-pin automation. Verified by file-name
diff (all pin files).
