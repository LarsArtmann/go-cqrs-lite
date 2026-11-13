#!/usr/bin/env bash
# check-module-layers.sh
# Validates that go-cqrs-lite module dependencies follow the layered architecture.
# go-arch-lint is unsuitable for multi-module Go workspaces because it treats
# inter-module imports as external vendor dependencies. This script parses go.mod
# files directly to enforce layer rules.

set -euo pipefail

# Layer definitions (higher layers may depend on lower layers only)
declare -A LAYER
LAYER["id/v4"]=0
LAYER["dispatcher/v4"]=0
LAYER["kv/v4"]=0
LAYER["dedup/v4"]=0
LAYER["event/v4"]=1
LAYER["command/v4"]=1
LAYER["query/v4"]=1
LAYER["scheduling/v4"]=1
LAYER["metadata/v4"]=1
LAYER["schema/v4"]=2
LAYER["snapshot/v4"]=2
LAYER["projection/v4"]=2
LAYER["deriver/v4"]=2
LAYER["commandlifecycle/v4"]=2
LAYER["commandlifecycle/projections/v4"]=3
LAYER["listing/v4"]=3
LAYER["decider/v4"]=3
LAYER["graph/v4"]=3
LAYER["scenario/v4"]=3
LAYER["projectionhost/v4"]=3
# claiming/v4: dialect-correct claim/lease SQL STATEMENT library — zero runtime
# deps (errorfamily only), owns NO store. Consumed downward by metaengine/v4
# claimkit (L3, ADR-0142), scheduling/sqlstore/v4 and the queue/v4 engines (L5).
# Layer 2 (contract/utility, like queue/v4 at L3 minus the domain tie): the
# original L4 classification predated ADR-0142, when only L5 stores used it.
LAYER["claiming/v4"]=2
# queue/v4: the durable work-queue CONTRACT (task/journal/queue packages,
# no engine). Engines implement it over claiming/v4; semantically a domain
# contract like decider/v4, and dep-light (errorfamily only).
LAYER["queue/v4"]=3
LAYER["queue/sqlite/v4"]=5
LAYER["queue/postgres/v4"]=5
LAYER["queue/mysql/v4"]=5
LAYER["signing/v4"]=4
LAYER["encryption/v4"]=4
LAYER["otel/v4"]=4
LAYER["otel/otlp/v4"]=4
LAYER["storage/memory/v4"]=4
LAYER["middleware/v4"]=5
LAYER["storage/v4"]=5
# listing/v4 is Aggregation (Tier 3) — depends on event/v4/id, not infrastructure
LAYER["watermill/v4"]=5
LAYER["prometheus/v4"]=5
LAYER["storage/pebble/v4"]=5
LAYER["storage/bbolt/v4"]=5
LAYER["storage/backuptest/v4"]=5
LAYER["storage/turso/v4"]=5
LAYER["metaengine/pebbleengine/v4"]=5
LAYER["metaengine/duckdbengine/v4"]=5
LAYER["metaengine/pgengine/v4"]=5
LAYER["metaengine/badgerengine/v4"]=5
LAYER["metaengine/dgraphengine/v4"]=5
LAYER["metaengine/graphadapter/v4"]=5
LAYER["metaengine/sqliteengine/v4"]=5
LAYER["metaengine/projectionadapter/v4"]=5
LAYER["metaengine/otelobserver/v4"]=5
LAYER["metaengine/bboltengine/v4"]=5
LAYER["metaengine/mysqlengine/v4"]=5
LAYER["metaengine/tursoengine/v4"]=5
LAYER["metaengine/bigtableengine/v4"]=5
# NOTE: metaengine/v4/{adttest,enginetest,keycodec} are PACKAGES inside the
# metaengine/v4 module (import path metaengine/v4/<name>), not separate modules —
# they have no go.mod and therefore no LAYER entries.
LAYER["metaengine/irohengine/v4"]=5
LAYER["metaengine/irohengine/loopback/v4"]=5
LAYER["metaengine/irohengine/quic/v4"]=5
LAYER["record/v4"]=0
LAYER["scheduling/sqlstore/v4"]=5
# scheduling/engine/v4: TimerStore facade over any DueClaimer engine (ADR-0142) —
# bridges scheduling/v4 (L1) to metaengine/v4 engines (L5), same tier as sqlstore.
LAYER["scheduling/engine/v4"]=5
LAYER["testutil/v4"]=5
# NOTE: testutil/v4 is also referenced as a direct dep in some lower-tier modules'
# go.mod (projectionhost/v4, transport/http, etc.) for test helpers. The
# EXCEPTIONS map handles those. See comment below for modules that intentionally
# omit testutil/v4 from LAYER.
LAYER["stack/v4"]=6
LAYER["stack/memory/v4"]=6
LAYER["stack/metaengine/v4"]=6
LAYER["stack/sqlite/v4"]=6
LAYER["stack/pebble/v4"]=6
LAYER["stack/bbolt/v4"]=6
LAYER["stack/postgres/v4"]=6
LAYER["stack/duckdb/v4"]=6
LAYER["stack/mysql/v4"]=6
LAYER["stack/turso/v4"]=6
LAYER["system/v4"]=6
# core/v5 (5): the merged v5 core module (ADR-0152) — Tier 0-3 domain plus
# the composition surface as PACKAGES of one module. Labelled at the driver
# tier (5) so engines/storage drivers (L4/L5) may depend on it as equal-or-
# lower peers in the v5 end state; during dual-support it imports NO v4
# sibling (copy-forward, no back-imports), so the downward rule is trivially
# satisfied. Layer is per-MODULE: packages that were separate layers in v4
# are same-module imports here and invisible to this gate.
LAYER["core/v5"]=5
LAYER["system/integration/v4"]=7
# systemtest/v4 (7): the Feedback-#4 split — system/v4's real-engine suites moved
# here so system/v4 consumers pull no engine implementation. Test-only
# module: no production consumers.
LAYER["systemtest/v4"]=7
# systemscenario/v4 (7): consumer-facing system-level BDD harness (ADR-0153);
# test-infrastructure tier like systemtest/v4/eventtest, but a published library
# consumers import (api-stability tracked).
LAYER["systemscenario/v4"]=7
# systemcatalog/v4 (7): the catalog×system/v4 derivation bridge
# (docs/planning/2026-10-26_systemcatalog-adapter-design.md) — derives
# architecture catalogs from system.DomainConfig + runtime registration
# records. Tooling tier beside catalog/v4; imports system/v4 (6) + catalog/v4 (7),
# downward-legal.
LAYER["systemcatalog/v4"]=7
LAYER["catalog/v4"]=7
LAYER["integration/v4"]=7
LAYER["stack/bench/v4"]=7
LAYER["metaengine/bench/v4"]=7
LAYER["benchkit/v4"]=7
LAYER["cmd/cqrs-gen/v4"]=7
LAYER["cmd/cqrs-lint/v4"]=7
LAYER["cmd/cqrs-bench/v4"]=7
LAYER["cmd/cqrs-upgrade/v4"]=7
LAYER["cmd/api-stability/v4"]=7
LAYER["cmd/doc-check/v4"]=7
LAYER["example/mesh-demo"]=7
LAYER["example/taskmanager"]=7
LAYER["example/getting-started"]=7
LAYER["example/goal-shaped-app"]=7
LAYER["example/readme-quickstart"]=7
LAYER["example/scheduler-otel-status"]=7
LAYER["example/standard-topology"]=7
LAYER["example/metaengine-quickstart"]=7
LAYER["example/graph-native"]=7
LAYER["cmd/cqrs-lint/v4/testdata/typedfixture"]=7
LAYER["cmd/cqrs-lint/v4/testdata/scanfixture"]=7
LAYER["cmd/cqrs-lint/v4/testdata/busfixture"]=7
LAYER["event/v4/eventtest"]=7
# scratch/counterkey: cross-engine repro (test-only, no product code). Rowed
# so the gate stays green while its disposition is pending — owner question
# T07.3 in docs/planning/2026-11-03_18-52_SUPERB-full-backlog-pareto-plan.md:
# keep as in-tree repro or move scratch/ out of the tree.
LAYER["scratch/counterkey"]=7
LAYER["testutil/mysqltestcontainer/v4"]=5
LAYER["testutil/pgtestcontainer/v4"]=5
# testutil/v4 is test-only infrastructure used from _test.go files across layers.
# It has LAYER[testutil/v4]=5 above, but lower-tier modules that import it from
# _test.go files get exceptions below. This avoids false layer violations while
# still budgeting its production deps.
LAYER["metaengine/v4"]=3
# idempotency/sqlstore/v4: SQL STORES over go-idempotency — storage/v4
# infrastructure by nature, and since ADR-0142 its NewFromEngine facade
# depends on metaengine/v4 (L3) legitimately (an SQL store adapting to the
# engine registry). The kvstore sibling stays L2 (pure kv/v4 abstraction).
LAYER["idempotency/sqlstore/v4"]=4

# EXCEPTIONS: each entry suppresses a layer violation that is architecturally
# legitimate. The script cannot distinguish production imports from test-only
# imports (both appear as direct requires in go.mod), so an exception is
# needed whenever a higher-layer module is a direct dependency — even if the
# import lives exclusively in _test.go files. Two categories:
#
#   (a) TEST-ONLY — the dep is imported exclusively from _test.go files.
#       go mod tidy keeps it as a direct require; the exception prevents a
#       false positive.
#   (b) PRODUCTION CROSS-CUTTING — otel/v4 is imported from production .go files
#       across all tiers. The layer model accepts this as an observability
#       carve-out (otel/v4 is L4 but legitimately used by L1/L3 modules for
#       tracing spans and metrics).
declare -A EXCEPTIONS

# event/v4 (L1) — all three test-only: schema/v4 (upcaster tests), snapshot/v4
# (errors-taxonomy test), storage/memory/v4 (stream/BDD/example tests)
EXCEPTIONS["event/v4"]="schema/v4 snapshot/v4 storage/memory/v4"

# schema/v4 (L2) — test-only: VersionedStore integration/v4 + golden tests
EXCEPTIONS["schema/v4"]="storage/memory/v4"

# snapshot/v4 (L2) — storage/memory/v4 test-only (constructor roundtrip tests);
# same test-cycle shape as event↔snapshot/v4 above.
EXCEPTIONS["snapshot/v4"]="storage/memory/v4"

# decider/v4 (L3) — storage/memory/v4 test-only (bench/BDD/example tests);
# otel/v4 PRODUCTION (tracing in decider.go, load.go, otel.go, wait_for_version.go)
EXCEPTIONS["decider/v4"]="storage/memory/v4 otel/v4"

# query/v4 (L1) — storage/memory/v4 test-only (typed store tests).
# NOTE: snapshot/v4 appears as // indirect in query/v4/go.mod (transitive via
# event/v4) but the awk filter on line 318 skips indirect deps, so it never
# reaches this exception map. Do NOT add snapshot/v4 here — it would silently
# suppress a real violation if snapshot/v4 is ever promoted to direct.
EXCEPTIONS["query/v4"]="storage/memory/v4"

# command/v4 (L1) — storage/memory/v4 test-only (store suite + typed store tests).
# Same indirect-snapshot note as query/v4 above.
EXCEPTIONS["command/v4"]="storage/memory/v4"

# listing/v4 (L3) — test-only: middleware/v4/BDD/example/benchmark tests
EXCEPTIONS["listing/v4"]="storage/memory/v4"

# commandlifecycle/v4 (L2) — test-only: middleware/v4 (retry integration/v4 tests),
# storage/memory/v4 (recorder + middleware/v4 tests)
EXCEPTIONS["commandlifecycle/v4"]="middleware/v4 storage/memory/v4"

# projectionhost/v4 (L3) — storage/memory/v4 test-only (integration/v4/stress tests);
# storage/v4 test-only (pg_integration_test.go, sql_checkpoint_test.go);
# otel/v4 PRODUCTION (tracing in worker.go, options.go, worker_drain.go).
# (testutil/v4 + testutil/pgtestcontainer/v4 are covered by TEST_INFRA_MODULES.)
EXCEPTIONS["projectionhost/v4"]="storage/memory/v4 otel/v4 storage/v4"

# metaengine/v4 (L3) — metaengine/sqliteengine/v4 test-only (cross-engine ADT
# matrix + concurrency-gap tests reference the real SQLite engine for parity)
EXCEPTIONS["metaengine/v4"]="metaengine/sqliteengine/v4"

# scenario/v4 (L3) — metaengine/projectionadapter/v4 test-only: the
# observational-equivalence tests (fixed sequence + rapid property) assert
# the ADR-0136 property against the real 80%-path adapter, not a mock;
# metaengine/v4 itself is same-layer (L3→L3, no exception needed).
EXCEPTIONS["scenario/v4"]="metaengine/projectionadapter/v4"

# Test-only packages that don't count against production dep budgets.
# These are test infrastructure (assertions, PBT, mocking) used across all modules.
TEST_PACKAGES="github.com/onsi/gomega github.com/onsi/ginkgo/v2 pgregory.net/rapid"

# Test-infrastructure MODULES: exempt from the layer-ordering check. Their
# entire purpose is to be imported from _test.go files in ANY layer (golden
# assertions, race helpers, testcontainers), so an apparent upward dependency
# is always test-only usage, never production coupling. Their own production
# deps still count against their DEP_BUDGET.
# cmd/cqrs-lint/v4/testdata/typedfixture: the committed F091 typed-path fixture —
# a consumer-shaped module that exists only to be linted by cqrs-lint's own
# tests (replace-based schema/v4 dep; never built by the workspace). Budget 2
# since C044: the branded-ID fixtures import id/v4 alongside schema/v4.
TEST_INFRA_MODULES="event/v4/eventtest testutil/v4 testutil/mysqltestcontainer/v4 testutil/pgtestcontainer/v4 cmd/cqrs-lint/v4/testdata/typedfixture cmd/cqrs-lint/v4/testdata/scanfixture cmd/cqrs-lint/v4/testdata/busfixture"

# Dependency budgets: maximum direct PRODUCTION dependencies per module.
# Budgets are intentionally tight — new deps require explicit review.
# Test-only deps (gomega, rapid, ginkgo) are excluded from the count.
declare -A DEP_BUDGET
DEP_BUDGET["id/v4"]=3
DEP_BUDGET["dispatcher/v4"]=0
# kv/v4: 4 = codec + errorfamily + otter + record/v4 (ADR-0144 — record.DeferClose
# is the Tier-0 lifecycle primitive kv/v4's iter/batch close paths share;
# internal primitive, not external sprawl — same precedent as storage/pebble/v4).
DEP_BUDGET["kv/v4"]=4
DEP_BUDGET["dedup/v4"]=0
DEP_BUDGET["event/v4"]=13
DEP_BUDGET["command/v4"]=8
DEP_BUDGET["query/v4"]=8
DEP_BUDGET["scheduling/v4"]=2
# metadata/v4: 2 = id/v4 + record/v4 (the ADR-0111 structural Actor/stamp types —
# record/v4 is the Tier-0 base metadata/v4's CommonMetadata embeds since the
# 2026-08-22 extraction; the =1 predates it).
DEP_BUDGET["metadata/v4"]=2
DEP_BUDGET["schema/v4"]=4
DEP_BUDGET["snapshot/v4"]=5
DEP_BUDGET["projection/v4"]=2
DEP_BUDGET["deriver/v4"]=4
DEP_BUDGET["commandlifecycle/v4"]=6
DEP_BUDGET["commandlifecycle/projections/v4"]=4
DEP_BUDGET["decider/v4"]=10
DEP_BUDGET["graph/v4"]=3
DEP_BUDGET["scenario/v4"]=3
# projectionhost/v4: 10 = the prior 9 + record/v4 (ADR-0144 DeferClose in the DLQ
# row-scanning paths; internal primitive, not external sprawl).
DEP_BUDGET["projectionhost/v4"]=10
# claiming/v4: 1 = go-error-family only (the modernc.org/sqlite require is
# test-only, round-trip claim test).
DEP_BUDGET["claiming/v4"]=1
# queue/v4: 1 = go-error-family only (sentinel classification).
DEP_BUDGET["queue/v4"]=1
# queue/sqlite/v4: 4 = queue/v4 contract + modernc sqlite driver + claiming/v4
# (engine-owned claim SQL in the claiming/v4 SHAPE) + metaengine/v4 (NewEngine
# driver registration, ADR-0142 T09 — the engine joins the operator registry).
DEP_BUDGET["queue/sqlite/v4"]=4
# queue/postgres/v4: 4 = queue/v4 contract + pgx + claiming/v4 + metaengine/v4 (same
# ADR-0142 T09 driver-registration pair as queue/sqlite/v4; pgtestcontainer is
# test-only).
DEP_BUDGET["queue/postgres/v4"]=4
# queue/mysql/v4: 4 = queue/v4 contract + go-sql-driver + claiming/v4 + metaengine/v4
# (NewEngine driver registration, same ADR-0142 T09 pair as the sibling
# engines; DSN-gated conformance is test-only).
DEP_BUDGET["queue/mysql/v4"]=4
DEP_BUDGET["signing/v4"]=5
DEP_BUDGET["encryption/v4"]=5
DEP_BUDGET["otel/v4"]=7
DEP_BUDGET["middleware/v4"]=13
# storage/v4: 14 = the prior 12 + record/v4 promoted from indirect (via event/v4) to
# direct (DeferClose in snapshot_migration.go, ADR-0144; internal primitive)
# + go-sqlitestore (sqlite_helpers.go delegates the canonical SQLite engine
# bootstrap to the shared engine; added 2026-09-22 without its budget bump).
DEP_BUDGET["storage/v4"]=14
DEP_BUDGET["listing/v4"]=6
DEP_BUDGET["watermill/v4"]=9
# codec is required for CBORToJSONTransform (SSE CBOR->JSON adapter composes
# the codec.TranscodeToJSON primitive — ADR-0052, deletes per-consumer dupes).
DEP_BUDGET["prometheus/v4"]=5
# pebble/bbolt: +1 (11) for the direct record/v4 dep — the snapshot/v4
# encoding stamp (record.Encoding, 2026-08-22 T17/T18) promotes the Tier-0
# primitive from indirect (via event/v4) to direct. Internal primitive, not
# external sprawl; re-check if a future dep wants the same slot.
DEP_BUDGET["storage/pebble/v4"]=11
DEP_BUDGET["storage/bbolt/v4"]=11
DEP_BUDGET["storage/backuptest/v4"]=3
DEP_BUDGET["storage/turso/v4"]=10
DEP_BUDGET["storage/memory/v4"]=8
DEP_BUDGET["stack/v4"]=18
DEP_BUDGET["stack/memory/v4"]=10
DEP_BUDGET["stack/metaengine/v4"]=4 # typed bridge: stack/v4 + metaengine/v4 only (issue #36)
DEP_BUDGET["stack/sqlite/v4"]=10
DEP_BUDGET["stack/pebble/v4"]=10
DEP_BUDGET["stack/bbolt/v4"]=10
DEP_BUDGET["stack/postgres/v4"]=10
DEP_BUDGET["stack/duckdb/v4"]=8
DEP_BUDGET["stack/mysql/v4"]=10
DEP_BUDGET["stack/turso/v4"]=13
DEP_BUDGET["stack/bench/v4"]=25
# catalog/v4: 6 = go-faster/yaml, go-error-family, go-snaps (cattest golden
# helpers), a-h/templ + templ-components (approved docserver UI adoption),
# +1 templ-components/utils (generated d2view_templ.go imports it directly
# since templ-components v1.11.0 — same +1-for-a-submodule precedent as
# storage/pebble/v4 above), +1 templ-components/icons (same submodule precedent,
# v1.18 split the icon set into its own module), +1 schema/v4 (T1b of the
# v5 schema-evolution plan: FromTypedSchema renders governance exports from
# typed schema/v4 declarations — schema/v4 is the downward import, system/v4 stays out).
# catalog/v4: 9 = templ + templ-components (+ icons + utils) for the docserver
# export, go-snaps + go-faster/yaml for snapshot/v4/golden tests, schema/v4 (Tier 2),
# go-error-family, go-humanize. The 9th (templ-components/utils) became DIRECT
# when templ-components v1.22.0 started emitting an explicit utils import in
# generated *_templ.go files (2026-10-26; before that it was transitive).
DEP_BUDGET["catalog/v4"]=9
DEP_BUDGET["integration/v4"]=21
DEP_BUDGET["benchkit/v4"]=25
DEP_BUDGET["testutil/v4"]=5
# metaengine/v4: 8 = the four Tier-0-ish primitives (record/v4/id/dedup/errorfamily)
# + go-sse (ServeSSE) + claiming/v4 (ADR-0142 claimkit: the ONE database/sql
# claim+dedup/v4 runtime every SQL engine embeds; claiming/v4 is a zero-runtime
# statement library, L2) + go-humanize (engine stats/doctor output
# humanization, 2026-10-08 go-humanize-linter sweep) + go-codec (Tier -1
# substrate: encoded ingress decodes CBOR with the codec that produced it —
# schema-free JSON transcoding cannot carry time.Time).
DEP_BUDGET["metaengine/v4"]=8
DEP_BUDGET["metaengine/pebbleengine/v4"]=5
DEP_BUDGET["metaengine/duckdbengine/v4"]=5
DEP_BUDGET["metaengine/pgengine/v4"]=5
DEP_BUDGET["metaengine/badgerengine/v4"]=2
DEP_BUDGET["metaengine/dgraphengine/v4"]=3
DEP_BUDGET["metaengine/graphadapter/v4"]=2
# 4 = claiming/v4 + metaengine/v4 + error-family + modernc/sqlite. error-family became
# direct when the map-value encoder started classifying unrepresentable input
# as Rejection instead of silently storing a fmt.Sprintf("%v") placeholder.
DEP_BUDGET["metaengine/sqliteengine/v4"]=4
DEP_BUDGET["metaengine/projectionadapter/v4"]=10
DEP_BUDGET["metaengine/otelobserver/v4"]=4
DEP_BUDGET["otel/otlp/v4"]=5
DEP_BUDGET["metaengine/bboltengine/v4"]=5
DEP_BUDGET["metaengine/mysqlengine/v4"]=5
DEP_BUDGET["metaengine/tursoengine/v4"]=5
# 3 = cloud.google.com/go/bigtable + google.golang.org/api + grpc (the whole
# point of the dep-isolated module; SDK surface stops at the module boundary).
DEP_BUDGET["metaengine/bigtableengine/v4"]=3
DEP_BUDGET["idempotency/sqlstore/v4"]=5
DEP_BUDGET["scheduling/sqlstore/v4"]=7
# scheduling/engine/v4: 3 = scheduling/v4 contract + metaengine/v4 + sqliteengine
# (the facade's conformance/constructor target; memory engine is test-only).
DEP_BUDGET["scheduling/engine/v4"]=3
# system/v4: 21 = the prior 20 + schema/v4 (DomainConfig.Schema payload-contract
# declarations compiled and applied at the composition root, 2026-10-09 T1 —
# Tier 5 → Tier 2, downward and legal).
DEP_BUDGET["system/v4"]=21
DEP_BUDGET["system/integration/v4"]=7
# systemtest/v4: 6 = system/v4 + metaengine/v4 core + the four engine drivers whose
# suites it owns (goleak is test-only); the point of the module is engine
# coverage, so a high budget is the design, not sprawl.
DEP_BUDGET["systemtest/v4"]=6
# systemscenario/v4: 9 = system/v4 + event/v4 + command/v4 + query/v4 + id/v4 (all in-repo)
# + go-error-family (fleet-standard classification, ThenErrorFamily)
# + go-snaps (ThenGolden snapshots — the eventtest LAYER-7 precedent allows
#   test-tooling deps in test-infrastructure modules)
# + sqliteengine (SQLite(t) preset driver registration; zero NEW external
#   deps — modernc.org/sqlite rides indirect through sqliteengine, which
#   system/v4's PRODUCTION go.mod already requires)
# + metaengine/v4 (DelayedDriver chaos wrapper + SubscribeSSE helper import it
#   directly — RegisterDriver/LookupDriver, NewWatcher, ServeSSE; it was a
#   test-only import before those production files; in-repo Tier 3, already
#   an indirect dep of system/v4)
DEP_BUDGET["systemscenario/v4"]=9
# systemcatalog/v4: 4 = system/v4 + catalog/v4 + event/v4 (in-repo) + go-error-family
# (classified rejections for nil builder/system inputs). command/v4/query/schema
# types flow through system/v4's accessor signatures without direct imports.
DEP_BUDGET["systemcatalog/v4"]=4
# core/v5: skeleton starts at 0; each v5 move wave (T07/T09/T10) bumps this
# to the union of the merged trains' production deps. Intra-repo v4 deps do
# not count — merged trains become same-module packages.
DEP_BUDGET["core/v5"]=6
DEP_BUDGET["metaengine/irohengine/v4"]=2
DEP_BUDGET["metaengine/irohengine/loopback/v4"]=4
DEP_BUDGET["metaengine/irohengine/quic/v4"]=5
DEP_BUDGET["cmd/cqrs-gen/v4"]=2
# 9th dep = go-finding/toolsdk (v1.11.0): the cqrs-lint toolsdk Spec
# (pkg/toolspec) exposes cqrs-lint as a BuildFlow-discoverable tool;
# same author, no new transitive burden beyond go-finding itself.
DEP_BUDGET["cmd/cqrs-lint/v4"]=9
# metaengine/v4 added for the `cqrs-bench layout` planning CLI.
DEP_BUDGET["cmd/cqrs-bench/v4"]=19
# cqrs-upgrade: cqrs-lint (in-process V007 engine), go-finding (rendering),
# x/mod (semver parsing) — all three are the tool's job.
DEP_BUDGET["cmd/cqrs-upgrade/v4"]=3
DEP_BUDGET["cmd/api-stability/v4"]=3
DEP_BUDGET["cmd/doc-check/v4"]=2
# mesh-demo: decider/v4/event/command/id = 4 (the bilateral-contract example)
# +catalog/v4 (EventCatalog export demo surface) +system/v4 (composition-root boot,
# the ADR-0123 modern pattern) +go-branded-id (branded stream IDs)
# +go-error-family (classified domain errors at the example's boundaries) = 8.
DEP_BUDGET["example/mesh-demo"]=8
# taskmanager: the reference app exercises the full modern stack/v4; the 26th
# direct dep is httputil/server_timing, added with the Server-Timing HTTP
# surface (2026-10-15 repair) and pinned by its HTTP integration/v4 tests.
DEP_BUDGET["example/taskmanager"]=26
# counterkey repro: 4 engines + metaengine/v4 + record/v4, all first-party; the
# point of the scratch is cross-engine comparison, test-only.
DEP_BUDGET["scratch/counterkey"]=6
DEP_BUDGET["example/getting-started"]=10
# goal-shaped-app is the Goal story: types-only domain + system/v4 composition
# root with BOTH swap targets compiled in (sqliteengine + pgengine) —
# command/v4/decider/event/id/metaengine/query/system + 2 drivers + snapshot/v4
# (the snapshot-story demo strategy line) = 10.
DEP_BUDGET["example/goal-shaped-app"]=11 # multi-engine demo (sqlite+pg+turso) is the point of the app
DEP_BUDGET["example/readme-quickstart"]=6
# standard-topology is the ADR-0165 demo: engines-only config +
# system.Standard + the two drivers its VM-test leg runs against
# (sqlite projections + postgres primary over peer auth) —
# command/v4/decider/event/id/metaengine/query/system + 2 drivers = 9.
DEP_BUDGET["example/standard-topology"]=9
DEP_BUDGET["example/scheduler-otel-status"]=8
# metaengine-quickstart demos the full strategic surface: system/v4 + metaengine/v4
# + sqliteengine + projectionadapter + event/v4/id/record — 7 first-party deps
# is the honest count for a composition example.
DEP_BUDGET["example/metaengine-quickstart"]=7
DEP_BUDGET["example/graph-native"]=12 # system+metaengine/v4 core, projectionadapter, sqlite+dgraph engines, systemscenario/v4 (test)
DEP_BUDGET["cmd/cqrs-lint/v4/testdata/typedfixture"]=2
DEP_BUDGET["cmd/cqrs-lint/v4/testdata/scanfixture"]=1
DEP_BUDGET["cmd/cqrs-lint/v4/testdata/busfixture"]=1
DEP_BUDGET["event/v4/eventtest"]=5
DEP_BUDGET["record/v4"]=0
DEP_BUDGET["testutil/mysqltestcontainer/v4"]=2
DEP_BUDGET["testutil/pgtestcontainer/v4"]=3
DEP_BUDGET["metaengine/bench/v4"]=5

failed=0

# ── Dependency count check ──

for mod in "${!DEP_BUDGET[@]}"; do
	gomod="${mod}/go.mod"
	if [ ! -f "$gomod" ]; then
		continue
	fi

	budget=${DEP_BUDGET[$mod]}
	direct=$(awk '
        /^require \(/{found=1;next}
        /^\)/{found=0}
        found && !/\/\// && !/indirect/ && /^[[:space:]]+[^[:space:]]/{count++}
        END{print count+0}
    ' "$gomod")

	# Subtract test-only packages from the count (they don't add production weight)
	test_deps=0
	for pkg in $TEST_PACKAGES; do
		if grep -q "[[:space:]]${pkg} " "$gomod" 2>/dev/null; then
			test_deps=$((test_deps + 1))
		fi
	done

	# Also detect deps that are only imported from _test.go files.
	# These are test-only in practice even if not in TEST_PACKAGES (e.g.,
	# internal CQRS modules used solely in tests). For each direct require,
	# check if any non-test .go file imports it; if not, it's test-only.
	direct_paths=$(awk '
        /^require \(/{found=1;next}
        /^\)/{found=0}
        found && !/\/\// && !/indirect/ && /^[[:space:]]+[^[:space:]]/{print $1}
    ' "$gomod")

	prod_dep_list=""
	for dep_path in $direct_paths; do
		# Skip packages already caught by TEST_PACKAGES
		skip=0
		for pkg in $TEST_PACKAGES; do
			case "$dep_path" in
			*"$pkg"*)
				skip=1
				break
				;;
			esac
		done
		[ "$skip" -eq 1 ] && continue

		# If no non-test .go file imports this dep, it's test-only.
		# Exclude files in subdirectories with their own go.mod (submodules) —
		# their imports belong to the submodule, not the parent module.
		non_test_files=$(grep -rl "\"${dep_path}" "${mod}/" --include='*.go' 2>/dev/null | grep -v '_test\.go$' || true)
		found_prod=false
		for f in $non_test_files; do
			dir=$(dirname "$f")
			in_submodule=false
			while [ "$dir" != "${mod}" ] && [ "$dir" != "." ] && [ "$dir" != "/" ]; do
				if [ -f "$dir/go.mod" ]; then
					in_submodule=true
					break
				fi
				dir=$(dirname "$dir")
			done
			if [ "$in_submodule" = "false" ]; then
				found_prod=true
				break
			fi
		done
		if [ "$found_prod" = "false" ]; then
			test_deps=$((test_deps + 1))
			continue
		fi

		# Collect production deps for diagnostic output on violation
		prod_dep_list="${prod_dep_list}  ${dep_path}"$'\n'
	done

	prod_deps=$((direct - test_deps))

	if [ "$prod_deps" -gt "$budget" ]; then
		echo "BUDGET: ${mod} has ${prod_deps} production deps (budget: ${budget}, total: ${direct}, test: ${test_deps})"
		printf "  Production deps:\n%s" "$prod_dep_list"
		failed=1
	fi
done

# ── Layer ordering check ──

for mod in "${!LAYER[@]}"; do
	gomod="${mod}/go.mod"
	if [ ! -f "$gomod" ]; then
		continue
	fi

	mod_layer=${LAYER[$mod]}

	# Parse go.mod directly for direct go-cqrs-lite dependencies.
	# Avoids GOWORK=off + go list which fails when go.sum is stale.
	deps=$(awk '
        /^require \(/{found=1;next}
        /^\)/{found=0}
        found && /go-cqrs-lite\// && !/\/\// {print $1}
    ' "$gomod" | grep "github.com/larsartmann/go-cqrs-lite/" || true)

	for req in $deps; do
		# Skip the module itself
		# Skip the module itself (match any major version suffix)
		if echo "$req" | grep -q "go-cqrs-lite/${mod}\$"; then
			continue
		fi

		# Extract module name from import path
		# github.com/larsartmann/go-cqrs-lite/EVENT/v4 -> event/v4
		# Only a TRAILING /vN is stripped: nested modules like event/v4/eventtest
		# keep their full path so the LAYER lookup matches the map key.
		# Module dirs now match their module paths exactly (the /v4 use-path
		# sweep), so dep_mod is the import path minus the repo prefix — no
		# trailing /vN strip (that strip matched the OLD layout where module
		# X/v4 lived at dir X).
		dep_mod=$(echo "$req" | sed 's|github.com/larsartmann/go-cqrs-lite/||')

		# Skip the root module (empty after the prefix strip)
		if [ -z "$dep_mod" ]; then
			continue
		fi

		# Skip root module and examples/cmd (not in layer map)
		if [ -z "${LAYER[$dep_mod]:-}" ]; then
			continue
		fi

		# Test-infrastructure modules are test-only imports by design; an
		# upward layer edge against them is not production coupling.
		case " $TEST_INFRA_MODULES " in
		*" $dep_mod "*) continue ;;
		esac

		# Check exceptions
		exc="${EXCEPTIONS[$mod]:-}"
		if echo "$exc" | grep -qw "$dep_mod"; then
			continue
		fi

		# Check layer ordering (same-layer deps are allowed; only strictly higher = violation)
		dep_layer=${LAYER[$dep_mod]}
		if [ "$dep_layer" -gt "$mod_layer" ]; then
			echo "VIOLATION: ${mod} (layer ${mod_layer}) depends on ${dep_mod} (layer ${dep_layer}) via ${req}"
			failed=1
		fi
	done
done

if [ "$failed" -eq 1 ]; then
	echo "::error::Module layer violations detected"
	exit 1
fi

# ── Coverage check: every go.mod must be in LAYER and DEP_BUDGET ──
# Prevents drift when new modules are added without updating this script.
coverage_gaps=0
for gomod in $(find . -name go.mod -not -path './vendor/*' -not -path './go.mod' | sort); do
	mod=$(dirname "$gomod" | sed 's|^\./||')
	[ -z "$mod" ] && continue

	if [ -z "${LAYER[$mod]:-}" ]; then
		echo "COVERAGE GAP: $mod has go.mod but no LAYER assignment"
		coverage_gaps=$((coverage_gaps + 1))
	fi
	if [ -z "${DEP_BUDGET[$mod]:-}" ]; then
		echo "COVERAGE GAP: $mod has go.mod but no DEP_BUDGET"
		coverage_gaps=$((coverage_gaps + 1))
	fi
done

if [ "$coverage_gaps" -gt 0 ]; then
	echo "::error::$coverage_gaps module(s) missing from LAYER or DEP_BUDGET maps"
	echo "Add them to scripts/check-module-layers.sh"
	exit 1
fi

echo "Module layer check passed"
