# Trust Wave II — Manifest (2026-10-12 05:55 CEST)

Executes N03 of the [SUPERB Pareto Plan II](2026-10-12_03-30_SUPERB-m06-closed-to-v5-cut-pareto-plan.md):
the `[Unreleased]` pool becomes importable. **Policy consult result (F21):** the plan's
bbolt/badger/pebble/mysql-engine and cmd/api-stability trains are REFUSED by
`scripts/untagged-trains.txt` (ADR-0152: internal tools never tag; driver tier tags on
first consumer). Their deltas ride their integration suites / the repo until a consumer
arrives. `scheduling`'s delta is go.mod churn (N04 pin-only class, deferred).

## The 9 trains (dependency order)

| # | Train | From → To | Carries |
|---|-------|-----------|---------|
| 1 | schema | v4.6.0 → **v4.7.0** | Fingerprint()/FingerprintStamp/drift twins (T3a), typed declarations `EventOf[T]` (T1a), CBOR nested-map Transform fix (#62-adjacent M9) |
| 2 | snapshot | v4.6.2 → **v4.7.0** | `Snapshot.StateShape` stamp (T4a) |
| 3 | decider | v4.7.2 → **v4.8.0** | `WithSnapshotStateVersion` (T4b); STRIPS the dev-time sibling replace on snapshot/v4 |
| 4 | catalog | v4.7.1 → **v4.8.0** | `FromTypedSchema[T]` + `SemverFromWire` (T1b) — requires schema v4.7.0 pin |
| 5 | systemscenario | v4.0.0 → **v4.1.0** | DelayedDriver chaos seam, SubscribeSSE, Memory/SQLite presets, hardening pack A+B |
| 6 | system | v4.12.0 → **v4.13.0** | `Schemas()` builder (T1a), `StampSchemaFingerprints` (T3b), RawQuery coeffect-gate fix (M08) |
| 7 | metaengine | v4.18.0 → **v4.19.0** | `FilterBindValue` (#62), json-tag-first row resolution (#63), `keycodec` package (T07 KV cursors), SQL-family compound keyset cursors (T07) |
| 8 | metaengine/sqliteengine | v4.5.2 → **v4.5.3** | #62 standard+planned time-bind coercion |
| 9 | cmd/cqrs-lint | v4.15.0 → **v4.16.0** | P015 (213 rules), E020/E021/E022, V006 stopped-train ruling (M07), C009/E005 fixes (#64), `./...` path fix, scorecard credit/waivers/modernity |

## Explicitly NOT tagged (untagged-trains.txt, ADR-0152)

- `cmd/api-stability` (mirror gate, golden dedup) — internal tool; its CHANGELOG rows
  stay in `[Unreleased]` with a policy note.
- `metaengine/{bbolt,badger,mysql}engine`, `storage/pebble` (T07 KV cursors, Pebble DESC
  fix, MySQL claimkit/layout fixes) — driver tier: tag on first fleet consumer.
- `example/*`, `cmd/cqrs-upgrade`, `cmd/doc-check` — examples/tools.
- `scheduling` — 14-file delta is go.mod/go.sum churn + test tweaks (N04 pin-only).

## Discipline (per train)

1. Settle the train's own go.mod pins on already-tagged siblings (GOWORK=off resolvable).
2. `GOWORK=off go test -count=1 ./...` in the module — RED BLOCKS THE TAG (2026-10-06
   lesson: `verify=ok` builds, it does not test).
3. Clean tree → `scripts/tag-release.sh <module> <version> "<desc>"` → commit the
   manifest artifacts it leaves → **push the tag IMMEDIATELY** (unpushed tags make the
   next tidy downgrade pins — 10-12 lesson).
4. `--smoke` after push.
5. After the wave: pin-sweep settle → `--check --remote` green → dated CHANGELOG wave
   section (tag-membership verified via `git grep <tag>`) → changelog-symbols +
   `TestTagContentMatchesChangelog` → `check-versions-manifest --check` → push master.

## Verification matrix

`check-adr-numbering` before wave: ☐ · per-train changed-set tests: ☐ ·
smoke each: ☐ · pin-sweep remote: ☐ · changelog-symbols: ☐ · tag-content: ☐ ·
versions-manifest: ☐ · README matrix fresh: ☐
