# Ghost-Verdict Memo — 9 zero-consumer modules (+1 adjacency)

**Date:** 2026-10-09 · **For:** owner decision gate (T02 of the v5 consolidation plan) · **Records into:** ADR-0152 addendum
**Consumer evidence:** 2026-10-08 who-uses audit Appendix B (zero consumers for all ten, in `~/projects`) + 2026-10-09 pkg.go.dev sweep (`docs/evidence/consumer-usage-2026-10-09.md`: zero non-LarsArtmann importers anywhere). In-repo importers below are ALL test/doc-only (integration suites, cqrs-lint rule tests, doc-check catalogs).

| Module                  | LOC   | In-repo usage                                                     | Existing decision                                                          | Recommendation          | One-line rationale                                                                                                                        |
| ----------------------- | ----- | ----------------------------------------------------------------- | -------------------------------------------------------------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| transport/http          | 3,617 | doc-check catalog only                                            | **ADR-0127: deprecated, removed in v5**                                    | KILL (pre-decided)      | Entire module is the deprecated SSEBroker family; go-sse/watermill replace it                                                             |
| transport/grpc          | 2,510 | cqrs-lint rule tests only                                         | **doc.go: "deprecated, removed in v5"**                                    | KILL (pre-decided)      | Zero adopters ever; gRPC dep stack retired with it                                                                                        |
| graph                   | 3,354 | integration suites (graph projection tests)                       | ADR-0113 deleted GraphBackend; skill marks projection tier "removed in v5" | **KILL**                | Edges-only reads already live in metaengine/graphadapter; projection tier has no consumer                                                 |
| metaengine/graphadapter | 306   | NONE anywhere                                                     | —                                                                          | **KILL** (with graph)   | The surviving graph surface also has zero importers; resurrect both from git if a graph consumer ever appears                             |
| otel/otlp               | 274   | doc-check catalog only                                            | —                                                                          | **KILL**                | OTLP exporter wiring is consumer-side config; otel/ (core move) suffices                                                                  |
| storage/pebble          | 9,862 | integration/pebble_test + stack/pebble preset (preset dies at v5) | —                                                                          | **KILL**                | The pebble need is served by metaengine/pebbleengine (has a consumer); 9.8k LOC of event-store driver for zero users                      |
| deriver                 | 732   | systemscenario fixtures (sagas)                                   | ADR-0136 names it the compensation rung                                    | **ABSORB into core/v5** | The temporal-composability ladder (ADR-0136) is core doctrine; tiny, zero deps, and the brand-new systemscenario harness already demos it |
| queue/mysql             | 2,635 | doc-check catalog only                                            | —                                                                          | **KEEP-DRIVER**         | Family consistency with queue/sqlite + queue/postgres (each has a consumer) + shared conformance suite + live MySQL integration leg       |
| idempotency/kvstore     | 724   | integration contract/property tests                               | skill dedup matrix recommends it                                           | **KEEP-DRIVER**         | Documented in the skill's dedup decision matrix; KV-backed idempotency stays available, untagged until a consumer                         |
| storage/backuptest      | 188   | storage/bbolt + storage/pebble backup lifecycle tests             | —                                                                          | **KEEP-DRIVER**         | Active in-repo test consumers (backup verification family); testutil-tier, never tagged                                                   |

**Kill mechanics (all verdicts):** nothing is deleted today. Kills execute at the v5 cut/T26 window via `trash` after the fleet-zero-v4-pins gate; tags stay proxy-served; git history is the resurrection path. KEEP-DRIVER means: stays modular, untagged (already on `scripts/untagged-trains.txt`), tag on first consumer. ABSORB means: deriver copies into `core/v5/deriver` in wave B.

**Open question deferred to T26 window:** whether `example/*` dirs referencing killed modules (if any) migrate or die with them.

---

## Addendum 2026-10-23 — verdicts re-based on the delete-at-zero ruling + fork-aware census

The 2026-10-17 owner ruling (phase out v4; delete at zero consumers) supersedes the "kills execute
at T26" timing above: deprecated/ghost surface deletes the moment BOTH checks pass (0 fleet
requires AND 0 in-repo couplings — playbook `docs/agents/v5-fork-and-delete-playbook.md` §3).
Evidence base: `docs/evidence/fleet-census-2026-10-23_15-13.json` — the first census that excludes
**full library forks** (`eventcatalog-hub/work/src/mesh-billing`, `mesh-orders` declare the
library's own module path; their internal requires are not fleet consumption), `.corpus-pins/`
snapshots, and testdata fixtures. tool: `scripts/census-fleet.sh --zero-consumers`.

**Superseded verdicts:** transport/http, transport/grpc — deleted 2026-10-17 (first standing-rule
application). idempotency/kvstore — the KEEP-DRIVER verdict above was WRONG under the ruling;
formally deprecated (ADR-0158) + zero consumers → deleted 2026-10-17.

**Updated verdict rows (2026-10-23 evidence):**

| Module | Fleet requires (fork-aware) | In-repo couplings | 2026-10-23 verdict | Delta vs 10-09 |
|---|---|---|---|---|
| deriver | **0** (was "systemscenario fixtures" only — the 3 "consumers" were mesh forks) | systemscenario saga fixtures | **ABSORB into core/v5** (unchanged) — but the absorb is MISSING from the Phase A task list (plan gap, rowed in TODO_LIST) | evidence corrected |
| graph | **0** (4 "consumers" were all mesh-fork internals) | integration graph projection suites | **KILL, blocked by couplings** — dies when the integration graph suites retire (Phase D) | evidence corrected |
| metaengine/graphadapter | 0 | none (registrations + generated engine-capabilities only) | **KILL — executable immediately on owner ratification** | sharpened: the only zero-zero kill left |
| otel/otlp | 0 | doc-check recipes catalog (one recipe fence) | **KILL — executable** after recipe rewire + §4 cascade | unchanged, now unblocked |
| storage/pebble | 0 (earlier 6 were forks/integration) | integration/pebble_test + stack/pebble preset | **KILL, blocked** — preset dies at task 38.0/39, integration leg rewires then | evidence corrected |
| queue/mysql | 0 | doc-check recipes catalog | **KEEP-DRIVER** (family + conformance; unchanged) | — |
| storage/backuptest | 0 | bbolt/pebble backup lifecycle tests | **KEEP-DRIVER** (active test consumers; unchanged) | — |
| metaengine/{pg,mysql,dgraph,turso,duckdb,iroh,bbolt}engine + iroh {loopback,quic} | 0 today | register.go init + conformance/CI legs | **KEEP-DRIVER — out of delete-at-zero domain**: the operator-choice driver tier (ADR-0152 vision: "where data lives is up to operators at DEPLOYMENT time"); untagged, tag on first consumer | NEW classification |
| stack presets (7) + stack/bench + metaengine/bench | 0 | benchkit tests + cmd/cqrs-bench factory (task 38.0) | **HELD by bench infra** — die at 38.0/39 | confirmed 10-17 finding |
| system/integration | 0 | leaf (it imports others) | **KEEP** — system's real-engine suite, product test infra, not deprecated surface | NEW |
| metaengine/otelobserver | 0 | doc-check recipes catalog | **KEEP** — current Tier-4 OTel observer option | NEW |
| testutil/mysqltestcontainer | 0 | none | **KEEP-DRIVER** — testutil family (pgtestcontainer has 24 direct) | NEW |

**Rule-domain boundary (recorded for the standing rule):** delete-at-zero governs
deprecated/duplicated v4 surface (ADR-0127/0158 class + memo-KILL ghosts). Never-consumed driver
modules, test infra, and untagged optionals are product surface at zero *today* — deleting them
would destroy the operator-choice tier. `census-fleet.sh --zero-consumers` is the trigger INPUT;
the playbook §3.2 coupling check + this domain boundary are the gates.
