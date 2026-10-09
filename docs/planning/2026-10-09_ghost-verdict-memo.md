# Ghost-Verdict Memo — 9 zero-consumer modules (+1 adjacency)

**Date:** 2026-10-09 · **For:** owner decision gate (T02 of the v5 consolidation plan) · **Records into:** ADR-0152 addendum
**Consumer evidence:** 2026-10-08 who-uses audit Appendix B (zero consumers for all ten, in `~/projects`) + 2026-10-09 pkg.go.dev sweep (`docs/evidence/consumer-usage-2026-10-09.md`: zero non-LarsArtmann importers anywhere). In-repo importers below are ALL test/doc-only (integration suites, cqrs-lint rule tests, doc-check catalogs).

| Module | LOC | In-repo usage | Existing decision | Recommendation | One-line rationale |
|---|---|---|---|---|---|
| transport/http | 3,617 | doc-check catalog only | **ADR-0127: deprecated, removed in v5** | KILL (pre-decided) | Entire module is the deprecated SSEBroker family; go-sse/watermill replace it |
| transport/grpc | 2,510 | cqrs-lint rule tests only | **doc.go: "deprecated, removed in v5"** | KILL (pre-decided) | Zero adopters ever; gRPC dep stack retired with it |
| graph | 3,354 | integration suites (graph projection tests) | ADR-0113 deleted GraphBackend; skill marks projection tier "removed in v5" | **KILL** | Edges-only reads already live in metaengine/graphadapter; projection tier has no consumer |
| metaengine/graphadapter | 306 | NONE anywhere | — | **KILL** (with graph) | The surviving graph surface also has zero importers; resurrect both from git if a graph consumer ever appears |
| otel/otlp | 274 | doc-check catalog only | — | **KILL** | OTLP exporter wiring is consumer-side config; otel/ (core move) suffices |
| storage/pebble | 9,862 | integration/pebble_test + stack/pebble preset (preset dies at v5) | — | **KILL** | The pebble need is served by metaengine/pebbleengine (has a consumer); 9.8k LOC of event-store driver for zero users |
| deriver | 732 | systemscenario fixtures (sagas) | ADR-0136 names it the compensation rung | **ABSORB into core/v5** | The temporal-composability ladder (ADR-0136) is core doctrine; tiny, zero deps, and the brand-new systemscenario harness already demos it |
| queue/mysql | 2,635 | doc-check catalog only | — | **KEEP-DRIVER** | Family consistency with queue/sqlite + queue/postgres (each has a consumer) + shared conformance suite + live MySQL integration leg |
| idempotency/kvstore | 724 | integration contract/property tests | skill dedup matrix recommends it | **KEEP-DRIVER** | Documented in the skill's dedup decision matrix; KV-backed idempotency stays available, untagged until a consumer |
| storage/backuptest | 188 | storage/bbolt + storage/pebble backup lifecycle tests | — | **KEEP-DRIVER** | Active in-repo test consumers (backup verification family); testutil-tier, never tagged |

**Kill mechanics (all verdicts):** nothing is deleted today. Kills execute at the v5 cut/T26 window via `trash` after the fleet-zero-v4-pins gate; tags stay proxy-served; git history is the resurrection path. KEEP-DRIVER means: stays modular, untagged (already on `scripts/untagged-trains.txt`), tag on first consumer. ABSORB means: deriver copies into `core/v5/deriver` in wave B.

**Open question deferred to T26 window:** whether `example/*` dirs referencing killed modules (if any) migrate or die with them.
