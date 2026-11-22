# go-cqrs-lite (self-dogfood) — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library github.com/larsartmann/go-cqrs-lite/** (self) @ feec9bc74 (train tags: event v4.15.2, system v4.19.0, metaengine v4.26.0, core/v5 untagged) · Consumer = the repo itself · Group G6

**Verdict:** UNDERUSED-MODERATE
**Score:** 5.5/8 capabilities (3 Full = 3.0, 5 Partial = 2.5, 0 Missed; N/A rows: none — all 8 axes apply to a self-audit) = 69% (moderate)

The repo eats its own dog food at every surface a consumer looks at first:
7 of 9 examples boot the ADR-0123 composition root `system.New`, 6 of 9 ride
metaengine, the docs recipes actively steer away from the deprecated stack
presets, and a stack of self-referential gates (V007-silent example lint,
systemscenario adoption, canonical-facts, doc-check, api-stability)
machine-enforce the modern path. The debt hides one layer deeper: the repo's
own benchmark CLI runs exclusively on v5-doomed stack presets, one example is
a walking demo of deprecated `scheduling/sqlstore`, cqrs-gen has zero in-repo
adoption, and encryption plus the engine-path scheduler have no runnable
example at all. The template teaches the fleet the right thing — except where
the fleet's second-look surfaces (benchmarks, codegen, ops demos) are
concerned.

## 1. What the library is (one short paragraph + pointer to module map)

A 104-`go.mod` event-sourced CQRS toolkit workspace (98 product/fixture
modules rowed across 7 tiers — the consumer-facing map is
`docs/agents/module-map.md:1`, internals in the same file's tier notes): pure
domain primitives (event, command, query, decider, record, id), the strategic
metaengine tier (17 dep-isolated engine modules under `metaengine/` plus the
in-core memory engine, projection adapter, vector/graph engines), the blessed
composition root `system/v4` (`system.New`), consumer-facing test harnesses
(`event/v4/eventtest`, `id/v4/idtest`, `query/v4/querytest`, `scenario/v4`,
`systemscenario/v4`), and its own tooling (`cmd/cqrs-lint`,
`cmd/cqrs-gen`, `cmd/cqrs-bench`, `cmd/doc-check`, `cmd/api-stability`,
`cmd/cqrs-upgrade`, `benchkit/v4`, `catalog/v4` + docserver). The consumer is
the repo itself: the product is the library, so "utilization" means dogfooding
— do `example/`, `cmd/`, and the test tiers eat the newest blessed surfaces?

## 2. Usage census (self-dogfood evidence)

**Composition root.** `system.New` call sites in own surfaces:
`example/taskmanager/setup.go:192`, `example/getting-started/main.go:167`,
`example/goal-shaped-app/main.go:47` (plus its
`example/goal-shaped-app/main_test.go:54`,
`example/goal-shaped-app/docs_compile_test.go:60`,
`example/goal-shaped-app/postgres_e2e_test.go:43`),
`example/standard-topology/main.go:45`, `example/graph-native/composition.go:19`,
`example/mesh-demo/gate.go:113`, `example/metaengine-quickstart/configfile_demo.go:41`
— 7 of 9 examples. The two exceptions: `example/readme-quickstart/main.go:7-12`
(deliberately raw decider/command/event/storage/watermill — the README's
minimal walkthrough twin) and `example/scheduler-otel-status/main.go`
(boots no system at all; see §4). Harnesses: `systemtest/v4` boots
`system.New` 14 times (e.g. `systemtest/v4/system_sqlite_test.go:77`),
`system/v4` verifies the README snippet against the real root
(`system/v4/readme_quickstart_verify_test.go:78`), and `cmd/cqrs-lint/v4`
proves its own analyzer against a booted system (2 boots / 4 harness calls,
per `scripts/check-systemscenario-adoption.sh --report`). The one tool that
never calls `system.New`: `cmd/cqrs-bench/v4` (below).

**Metaengine.** Ridden by `example/getting-started/main.go:94`
(`metaengine.Query`), `example/taskmanager/metaengine.go:42`,
`example/graph-native`, `example/standard-topology/main.go:76`
(`system.EngineConfig` map), `example/goal-shaped-app` (naming-convention
folds, per its README), and `example/metaengine-quickstart` (configfile,
graph, vector demos: `configfile_demo.go`, `graph_demo.go`,
`vector_demo.go`). Harnesses: `integration/v4/metaengine_test.go:25`,
`systemtest/v4/system_sqlite_test.go:15`, `benchkit/v4/runner.go:14`.

**Harnesses.** `eventtest` consumers include 12 `watermill/v4` test files
(e.g. `watermill/v4/coverage_test.go:32`) and `integration/v4`
(pinned `event/v4/eventtest v0.4.0` at `integration/v4/go.mod:13`); `idtest`
spans event, storage, watermill (e.g. `event/v4/event_core_test.go`);
`querytest` spans `storage/memory/v4`, `storage/bbolt/v4`,
`storage/pebble/v4`, `integration/v4/query`. `scenario/v4` has exactly two
in-repo consumers: `example/taskmanager/decider_test.go` (13 DSL sites) and
`example/goal-shaped-app/scenario_test.go` (6). `systemscenario/v4` is used
by three examples (`example/taskmanager/systemscenario_test.go`,
`example/goal-shaped-app/systemscenario_test.go`,
`example/graph-native/scenario_test.go`) plus cqrs-lint itself.

**Tooling.** cqrs-lint: self-lints its own source in CI
(`.github/workflows/ci.yml:424`), lints every example with a hard
V007-must-be-silent gate (`.github/workflows/ci.yml:437`), and carries a
zero budget in BuildFlow (`.buildflow.yml:48`). cqrs-bench + benchkit: the
nightly benchmark workflow runs `run`/`compare`/`sweep` legs and a benchkit
per-metric noise gate (`.github/workflows/benchmarks.yml:53`,
`.github/workflows/benchmarks.yml:141`, `.github/workflows/benchmarks.yml:213`).
catalog: docserver's templ codegen drift and browser-validated CSP are flake
gates (`flake.nix:1503`, `flake.nix:1555`). doc-check compiles the recipe
fences in CI (`.github/workflows/ci.yml:210`); api-stability runs at
`.github/workflows/ci.yml:90`; `scripts/check-canonical-facts.sh:1` pins
every doc-cited number to repo-derived truth.

## 3. Capability inventory (the 8 axes)

| # | Axis                                                        | Grade       | Evidence / reason                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| - | ----------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | Modern composition root (`system.New`) in own surfaces      | **Partial** | 7/9 examples + systemtest/integration/cqrs-lint ride it; `example/readme-quickstart` is deliberately raw (fine, labeled); but `example/scheduler-otel-status` boots raw `database/sql` + sqlstore, `integration/v4/graph_projection_test.go:41` hand-wires `stack.New`, and `cmd/cqrs-bench` boots **only** stack presets                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| 2 | Metaengine adoption in own surfaces                         | **Full**    | 6/9 examples, `example/metaengine-quickstart` exists with 3 demos, integration/systemtest/benchkit import it; recipes speak engine profiles; sole asterisk: cqrs-bench's _run_ path never executes an engine (its `layout` subcommand only prints `metaengine.EngineProfile` tables, `cmd/cqrs-bench/v4/layout.go:67`)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| 3 | systemscenario harness adoption (gate honesty)              | **Full**    | Gate runs in CI (`.github/workflows/ci.yml:119`); `scripts/systemscenario-adoption-allowlist.txt` holds 7 exemptions, every row with a plausible reason (benchkit = metric assertions; system/v4 = unit tests of the root itself, 31 boots; systemtest = process death/restart; integration = real-engine legs; systemcatalog = derivation-only; doc-check = fence compilation; metaengine = engine conformance). Reasons read honest, not excuses; adopters disappear from the list (cqrs-lint did)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| 4 | Test-harness self-use (eventtest/idtest/querytest/scenario) | **Full**    | eventtest: 12 watermill test files + integration; idtest: event/storage/watermill; querytest: all four storage backends + integration; scenario/v4: 2 example consumers (its designed audience); unit-only modules (projection, dedup, queue) hand-roll legitimately — no decider to drive                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| 5 | Legacy/deprecated internal usage                            | **Partial** | Quantified: stack presets in production tooling — `cmd/cqrs-bench/v4/factory.go:126` (memory), `:162` (sqlite), `:278` (postgres), imports `stack/memory`, `stack/sqlite`, `stack/postgres` (`factory.go:18-22`); `benchkit/v4/system_factory.go:56` calls `stack.New` but only as a documented Bundle-view adapter over a live system. Tombstones: exactly 1 production site, `watermill/v4/protocol.go:305` (`event.TombstoneMark`, deprecated since `event/v4/tombstone.go:12`, ADR-0114). idempotency MemoryStore: 0 in-repo legacy hits (middleware tests use the _external_ `go-idempotency` lib, `middleware/v4/test_helpers_test.go:23`); integration still pins deprecation-windowed `idempotency/sqlstore v4.4.3` (`integration/v4/go.mod:16`, consumers at `integration/v4/idempotency_contract_test.go:14`). scheduling/sqlstore: one whole example on the deprecated path (`example/scheduler-otel-status`, pinned v4.1.4). `metadata.CustomData`: zero production use — only the compat alias `event/v4/v3_compat_aliases.go:31` and cqrs-lint rewrite rules (`cmd/cqrs-lint/v4/pkg/rules/version/v007_tables_symbols.go:140`) |
| 6 | Tooling self-use                                            | **Partial** | cqrs-lint: self-lint + examples gate + BuildFlow budget — but no CI leg lints the library modules themselves. cqrs-gen: zero in-repo adoption — `//cqrs:` tags exist only in its own fixtures (`cmd/cqrs-gen/v4/main_test.go:20`); no example or recipe fence proves the codegen loop. cqrs-bench/benchkit: nightly CI rides both — on doomed presets (axis 1/5). catalog/docserver, doc-check, api-stability, canonical-facts: superb (see §2)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| 7 | Example coverage breadth                                    | **Partial** | 9 examples map: taskmanager (system, metaengine, signing `setup.go:26`, queue `workqueue.go:13`, deriver, idempotency, systemscenario, HTTP/SSE), goal-shaped-app (operator `cqrs.yaml`, snapshot + asyncapi demos, scenario), getting-started (system + metaengine read model), standard-topology (multi-engine topology), graph-native (graph projections), mesh-demo (catalog contracts + gate), metaengine-quickstart (config/graph/vector), readme-quickstart (raw blocks), scheduler-otel-status (deprecated sqlstore + otel + prometheus). **No example**: encryption (only `integration/v4/encryption/encryption_integration_test.go`), scheduling/engine timers (promised at `example/scheduler-otel-status/main.go:7-13`, not delivered), a watermill-broker app composed with system (only raw readme-quickstart + `watermill/v4/example_test.go` godoc). kv: none, correctly (v5-removed per `docs/agents/module-map.md:27`)                                                                                                                                                                                                     |
| 8 | Fleet-template fitness                                      | **Partial** | The first-look surfaces (taskmanager, goal-shaped-app, getting-started, recipes — `recipes.md:254` says the wrapper "wraps `system.New` (NOT the deprecated stack presets)") are uniformly modern; the second-look surfaces (benchmarks via cqrs-bench, scheduler ops demo, codegen) teach doomed or missing paths — and a fleet engineer hits those on day two                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |

Score derivation: rows N=8, N/A=0. Full=3 (3.0), Partial=5 (2.5), Missed=0.
Score = round(100 × 5.5 / 8) = **69% → moderate band**.

## 4. Gap analysis

1. **The benchmark tool is the biggest v5-doomed surface in the repo.**
   `cmd/cqrs-bench/v4/factory.go` builds every backend (memory, sqlite,
   sqlite-cgo, postgres, duckdb via the same bundle shape) from
   `stack/memory`, `stack/sqlite`, `stack/postgres` presets — the exact
   presets `stack/v4/doc.go:44` marks "removed in v5 (ADR-0123)". The irony
   is structural: `benchkit/v4/system_factory.go:22` already ships
   `FactoryFromSystem`, the adapter that runs "every benchkit phase unchanged
   against `system.New` deployments" — and its only callers are benchkit's
   own tests. The repo built the migration bridge and then didn't cross it.
2. **A deprecated-path example with a promissory note.**
   `example/scheduler-otel-status/main.go:7-13` opens with a NOTE:
   scheduling/sqlstore is DEPRECATED (ADR-0142, removed at v5.0.0) and "an
   engine-path observability demo (metaengine engine instrumentation)
   replaces it before v5." The honesty is good; the state is not — this is
   the only otel/prometheus example, so the fleet's ops-copy source teaches a
   surface with a death date. The V007-silent gate
   (`cmd/cqrs-lint/v4/pkg/rules/version/v007_tables.go:30` lists
   scheduling/sqlstore with its replacement) does not fire because the
   Claiming-store hooks are deliberately allowed "through their deprecation
   window" (`cmd/cqrs-lint/v4/pkg/rules/version/v007_test.go:324`) — the
   exemption is doing real work here, and the window closes at v5.
3. **cqrs-gen is sold but not eaten.** The tag-driven codegen
   (`//cqrs:command` etc., `cmd/cqrs-gen/v4/scan.go:113`) is documented in
   `.agents/skills/go-cqrs-lite/references/advanced.md` and
   `docs/DOMAIN_LANGUAGE.md`, yet nothing in `example/` or the recipe fences
   uses it; the only tags in the tree are cqrs-gen's own test fixtures. A
   tool whose dogfood proof is its unit tests is one refactor away from
   silently drifting from the blessed composition shapes.
4. **Harness seams.** `integration/v4/graph_projection_test.go:41` hand-wires
   `stack.New` for the graph-projection integration leg, and
   `integration/v4/error_classification_test.go:39-42` asserts on stack
   sentinel errors — both defensible (the latter literally tests stack's
   error taxonomy) but each is a place where the integration tier could
   drift from the system.New root it mostly uses. `scenario/v4`'s two
   consumers are exactly its target audience, but nothing _inside_ the repo
   beyond examples exercises the decider-level DSL, so its own regression
   coverage rides on consumers.

## 5. Version currency (self-referential)

- The v4 train is byte-frozen; per-train tags are the real versioning
  (`versions.json:3` still lists the relic root tag v4.0.0 alongside real
  per-module tags: catalog v4.15.0, cqrs-lint v4.22.2, …). The repo pins
  itself per-train: `example/taskmanager/go.mod:10` event v4.15.2,
  `:14` metaengine v4.26.0, `:26` system v4.19.0 — exactly the orchestrator's
  headline tags, i.e. examples consume the latest blessed trains.
- Example pins are workspace-relative in dev (`go.work` `use`-blocks all
  modules) but CI re-tests them against _published_ pins with `GOWORK=off`
  (`.github/workflows/ci.yml:414-421`, "Build + test every example against
  published pins") — the strongest self-dogfood device in the repo: every
  example is continuously proven as a real downstream consumer.
- One stale-ish pin: `event/v4/eventtest v0.4.0` (`integration/v4/go.mod:13`)
  — the nested module whose v0.x tags are proxy-invisible per
  `docs/agents/module-map.md` (the "baselined dead paths" note); consumers
  inside the workspace can't see the rot, external ones already do.
- core/v5 sits in the free pre-first-tag window (ADR-0166/0152): the repo
  dogfoods v5 inside `core/v5` tests, but no example or cmd surface rides
  core/v5 yet — correct for the freeze discipline, and the flip trigger for
  this report's example-breadth row.

## 6. The fleet argument

This repo is not merely _in_ the fleet — it is the template the fleet clones
and the skill the fleet's agents read. That cuts both ways. On the upside,
the golden path is aggressively modern: `system.New` in 7/9 examples,
metaengine in 6/9, three examples on the systemscenario BDD harness, recipes
that name the deprecated presets only to say "NOT them"
(`.agents/skills/go-cqrs-lite/references/recipes.md:254`), and CI gates that
fail an example for touching a v5-removed API. A fleet app cloned from
taskmanager or goal-shaped-app inherits the ADR-0123 world by default. On
the downside, the fleet's day-two surfaces are the lagging ones: the first
time a fleet engineer benchmarks, they reach for cqrs-bench and are handed
stack presets the library's own docs have deprecated; the first time they
wire scheduler observability, the only example is a self-confessed dead path;
the first time they ask "does cqrs-gen work?", the repo itself never answers.
Every one of these is fleet-wide debt at clone time — one repo diff, N fleet
copies — which is why the ranked list below leads with the bench bridge.

## 7. Ranked opportunities

1. **Point cqrs-bench at `FactoryFromSystem`** (high): add a `--compose
   system` mode (or make it the default for memory/sqlite/postgres) so the
   nightly benchmarks measure the blessed root; keep the stack factories as
   a labeled legacy comparison. The adapter already exists at
   `benchkit/v4/system_factory.go:22`.
2. **Ship the engine-path scheduler example** (high): replace or sibling
   `example/scheduler-otel-status` with `scheduling/engine.NewTimerStore`
   over a `metaengine.DueClaimer` engine (the replacement its own NOTE at
   `example/scheduler-otel-status/main.go:7-13` promises) — keep the otel +
   prometheus hooks, since that demo is otherwise unique.
3. **Dogfood cqrs-gen in one example** (medium): tag taskmanager's (or a new
   minimal example's) commands with `//cqrs:` and generate the handler stubs
   in CI so the codegen loop is proven end-to-end on every push.
4. **An encryption example** (medium): `encryption/v4` is flagship-tier
   (pin v4.4.6 in `integration/v4/go.mod:11`) with only integration tests to
   its name; a 30-line demo beside signing in taskmanager closes the gap.
5. **Migrate the graph integration leg off stack.New** (low):
   `integration/v4/graph_projection_test.go:41` via system.New + engine
   config, keeping a stack-shaped case only if the taxonomy assertions
   (`integration/v4/error_classification_test.go:39`) need it.
6. **Run cqrs-lint over library modules in CI** (low): today only self-lint
   and examples legs exist; a nightly full-repo pass would make the
   `.buildflow.yml:48` zero-budget claim true in CI, not just locally.

## 8. Reconsider when

- cqrs-bench's run path rides system.New (axis 1 and 5 flip; the fleet
  argument for this report's verdict weakens to breadth-only).
- core/v5 cuts its first tag: example breadth row re-grades against a
  core/v5 example set, and the V007 window exemptions
  (`cmd/cqrs-lint/v4/pkg/rules/version/v007_test.go:324`) close by design.
- The scheduler engine-path example lands; `example/scheduler-otel-status`
  is deleted or archived rather than merely annotated.
- A fleet app clones the repo's benchmark tooling as-is (the cqrs-bench gap
  stops being internal debt and becomes an incident of teaching).

## 9. How to verify

```
rg -n "system\.New\(" example/ systemtest/v4 system/v4 --type go | wc -l
bash scripts/check-systemscenario-adoption.sh --report
sed -n 7,16p example/scheduler-otel-status/main.go
sed -n 18,22p cmd/cqrs-bench/v4/factory.go && sed -n 50,66p benchkit/v4/system_factory.go
sed -n 437,455p .github/workflows/ci.yml && sed -n 414,421p .github/workflows/ci.yml
sed -n 48p .buildflow.yml && sed -n 119,121p .github/workflows/ci.yml
rg -n "TombstoneMark" watermill/v4/protocol.go && rg -n "CustomData" event/v4/v3_compat_aliases.go
rg -n "//cqrs:" --glob '!cmd/cqrs-gen/**' --glob '!**/testdata/**' -g '!*.md' .
sed -n 254p .agents/skills/go-cqrs-lite/references/recipes.md
```

Context: `docs/research/2026-11-13_go-cqrs-lite-deep-dive.html` (health
72/100) is broader and older; this report supersedes it on the self-dogfood
axis only. Sibling reports in this directory cover the _external_-library
utilization of this repo (httputil, go-etag, …); this is the mirror image.
