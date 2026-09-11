# Project: go-cqrs-lite

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.**
>
> Consumers import modules (`event`, `command`, `decider`, `storage`, `memory`, `catalog`, etc.) into THEIR projects.
> There is no "main app." Every module is independently importable.
>
> | If you catch yourself thinking…              | STOP — this is a LIBRARY, not an app                                                                                                       |
> | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
> | "Nothing in this repo uses it, so delete it" | **DELETING EXTERNAL-FACING API IS BREAKING THE PRODUCT.** Consumers live outside this repo. Zero internal consumers is the EXPECTED state. |
> | "Module needs a service that uses it"        | Module needs tests + stable API, not an internal consumer                                                                                  |
> | "example/ should drive real traffic"         | example/ is a usage demo, not a deployment                                                                                                 |
> | "Unused exports are waste"                   | Public API surface IS the product                                                                                                          |
>
> **The quality gate for every module: "Would a consumer trust this enough to import it?"**

A lightweight CQRS **library/SDK** for Go with Event Sourcing support, branded IDs, and auto-documentation generation.

Consumers import what they need and compose their own stack. Not a framework — no opinionated transport, message broker, or SQL driver.

## Where to Find Things

[`SKILL.md`](SKILL.md) (symlink to `.agents/skills/go-cqrs-lite/SKILL.md`) is the canonical API reference for **all** agents — consumers AND contributors. Its `references/` contain verified, copy-paste recipes and module docs. This AGENTS.md covers internal contracts, procedures, and gotchas that only matter when working **inside** the repo.

| Topic                                                                                                                         | Reference                                                                          |
| ----------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Mental model, quickstart, decision matrix, conventions, cheat sheet                                                           | [`references/core.md`](.agents/skills/go-cqrs-lite/references/core.md)             |
| Composition recipes (ES setup, persistence, snapshots, signing, encryption, OTel, catalog, CBOR, metaengine, flight recorder) | [`references/recipes.md`](.agents/skills/go-cqrs-lite/references/recipes.md)       |
| Read models (projections, SQL views, CatchUpSubscriber, tier selection)                                                       | [`references/readmodels.md`](.agents/skills/go-cqrs-lite/references/readmodels.md) |
| Advanced patterns (tombstone, watermill, gRPC, projection host, scheduling, graph, SSE, flight recorder, scenario DSL)        | [`references/advanced.md`](.agents/skills/go-cqrs-lite/references/advanced.md)     |
| Per-module quick lookup                                                                                                       | [`references/modules.md`](.agents/skills/go-cqrs-lite/references/modules.md)       |
| Common pitfalls, error messages, debugging                                                                                    | [`references/faq.md`](.agents/skills/go-cqrs-lite/references/faq.md)               |

**Contributing to the skill:** edit the `.md` files under `.agents/skills/go-cqrs-lite/`, then verify:

```bash
cd cmd/doc-check && GOWORK=off go run -tags "goexperiment.jsonv2" . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md
```

## Quick Reference

| Item        | Value                                                                                                                                           |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Language    | Go 1.26.4                                                                                                                                       |
| Build       | `nix run .#build`                                                                                                                               |
| Test        | `nix run .#test`                                                                                                                                |
| Lint        | `nix run .#lint`                                                                                                                                |
| Format      | `nix fmt`                                                                                                                                       |
| Dev shell   | `nix develop`                                                                                                                                   |
| Verify all  | `nix run .#verify` (build + vet + test + race + lint + doc-check)                                                                               |
| Int. PG     | `nix run .#integration-pg` (ephemeral, no Docker) or `nix run .#integration-pg-vm` (QEMU VM)                                                    |
| Int. MySQL  | `nix run .#integration-mysql-nspawn` (nspawn, ~15s, needs root + uid-range) or `nix run .#integration-mysql-vm` (QEMU VM, ~131s, always works)  |
| Int. All    | `nix run .#test-integration` or `nix run .#test-all-backends` (SQLite+Pebble+bbolt+DuckDB+PG+MySQL+Dgraph)                                      |
| Int. Dgraph | `nix run .#integration-dgraph` (ephemeral nixpkgs Dgraph, full dgraphengine suite; also a CI job)                                               |
| Int. Redis  | `nix run .#integration-redis` (ephemeral nixpkgs Redis; watermill broker suite: roundtrip, Nack redelivery, group exactly-once, 2 MiB payloads) |
| Load sweep  | `nix run .#load-sweep` (timing tests `-run 'Latency\|Timer\|Deadline'` under CPU soakers — run before `#verify` after touching timing paths)    |
| Verify CI   | `nix run .#verify-ci` (GOWORK=off per-module build+test — mirrors the CI matrix job)                                                            |
| Lint config | `nix run .#check-lint-config` (golangci config verify + depguard allow-list)                                                                    |
| ErrTax      | `nix run .#check-error-taxonomy` (drift gate: errorfamily codes vs docs/error-taxonomy.md, bidirectional)                                       |
| Rel. tests  | `nix run .#check-release-scripts` (tag-release.sh + batch-release.sh smoke tests vs fixture repos; also a CI leg)                               |
| CSP check   | `nix run .#check-csp` (docserver CSP policy, browser-validated)                                                                                 |
| EventCat    | `nix run .#check-eventcatalog` (EventCatalog export render-validation)                                                                          |
| Bench       | `nix run .#bench` (full sweep) · `./scripts/benchmark-regression.sh` (gate: median ns/op, 25% threshold — CI fails on breach)                   |
| CI          | GitHub Actions: ci.yml (Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module)                                                   |

Multi-module Go workspace (`go.work`) with 84 `go.mod` files (incl. root). Verify: `find . -name go.mod -not -path './vendor/*' | wc -l`

Per-module isolation: `cd event && GOWORK=off go test ./... -count=1`

## Module Map

Full table (modules, internal notes): [`docs/agents/module-map.md`](docs/agents/module-map.md). Consumer-facing lookup: [`references/modules.md`](.agents/skills/go-cqrs-lite/references/modules.md).

Tier one-liner: Tier 0 primitives (id, dispatcher, kv, dedup, record) → 1 core domain (event, command, query, scheduling, metadata) → 2 domain utilities (schema, snapshot, projection, idempotency, deriver, commandlifecycle) → 3 aggregation (decider, graph, scenario, projectionhost, listing, metaengine) → 4 infrastructure (storage/_, signing, encryption, otel, middleware, transports, watermill, engines) → 5 composition (stack, system) → 6 tooling & examples (catalog, benchkit, cmd/_, example/*, eventtest).

## Internal Contracts

Non-obvious conventions that apply when editing code inside this repo. Consumer-facing conventions are in [`references/core.md`](.agents/skills/go-cqrs-lite/references/core.md) §3.

1. **Max 350 lines/file (CI-enforced), 30 lines/function.**
2. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps.
3. **Dependency budgets** — Per-module direct PRODUCTION dep limits enforced by `nix run .#check-arch`. Test-only packages (gomega, ginkgo, rapid) are excluded. Adding production deps requires explicit budget review.
4. **OTel through otel/** — Modules import `otel/` re-exports instead of `go.opentelemetry.io` directly. OTel SDK is indirect in decider, storage, middleware go.mod files. The `otel/` module re-exports: `Int64Counter`, `AddOption`, `AddSpanEvent()`, `ServiceResourceAttributes()`, `CQRSHistogramBoundaries`, `NewCQRSViews()`, `CounterAddWithAttributes()`, `Setup()`, `WithStdoutExporter()`, `TextMapPropagator()`, `Version()`. Span names follow `{component}.{action}` — see `docs/SPAN_NAMING.md`.
5. **Zero-copy internal reads** — `PayloadReadOnly(evt)` bypasses `Payload()` clone for read-only paths (Event is a concrete type alias `= *ImmutableEvent`, no assertion needed). Used by signing, pebble, storage/sql, transport/http/sse. Internal-only `payloadForDecode()` and `encodingForCopy()` for same-package paths.
6. **Defensive clone on all public accessors** — `Payload()` returns `slices.Clone`, `Metadata()` returns `.Clone()`, `EventTypes()` returns `slices.Clone`, `MultiSignature.Get()` returns a copy, `WithCommandMetadata` clones on intake.
7. **Hot-path zero-allocation discipline** — Public API clones stay, but internal hot paths eliminate allocs via: lazy map init, pre-computed middleware chains (rebuild on `Use()`/`UsePublish()` only), cached SQL templates, pre-sized result slices, batch SQL inserts (multi-VALUES with SQLite 999-param chunking).
8. **Circuit breaker uses failsafe-go** — `middleware/circuit_breaker.go` wraps `failsafe-go/circuitbreaker`. Half-open semantics differ (limits trial executions to `SuccessThreshold` count). `decider/cache.go` uses `maypok86/otter/v2` TinyLFU.
9. **Load coalescing via singleflight** — `decider.Repository[State]` uses `singleflight.Group` to coalesce concurrent `Load` calls. Events are immutable, sharing is safe. Disable via `WithLoadCoalescing[State](false)`.
10. **Go experimental build tags** — Builds use `-tags "goexperiment.jsonv2"` enabling `encoding/json/v2`. CI and `nix run .#build` apply it automatically. Tag remains until Go graduates it (expected 1.27+).
11. **Deletion as domain events (ADR-0114, direction; partial implementation)** — Deletion SHOULD be expressed as a domain event type (e.g. `user.deleted`), not mutable metadata. Today: metaengine is fully type-based (`metaengine.Remove`); `stack.Materialize.OnTombstone/OnRebirth` are still metadata-triggered (`event.TombstoneMark` — branch on `evt.Type()` in `OnUpdate` for pure domain-event style); `listing.StatusMiddleware(deleteTypes, rebirthTypes)` bridges event types → status. `event.DetectTombstone`/`MarkTombstone` are Deprecated (removal v5). No `Delete` on Store. See ADR-0114 implementation-status addendum.
12. **Strong types** — No `any` as a value type in domain/business logic. Legitimate exceptions: JSON schema serialization (`catalog/`), `recover()` return value (`middleware/recovery.go`), `database/sql` interop. Generic type constraints (`[T any]`) are standard Go and always allowed.
13. **Error-wrapping helpers** — When `if err != nil { return WrapX(err, code, msg) }; return nil` appears 3+ times in a module, extract an unexported `wrapXOrOK(err, code, msg) error` (returns nil when err is nil). Keep per-module — see [ADR-0069](docs/adr/0069-error-wrapping-helpers.md). When modules share a dependency (e.g., encryption + signing → codec), push the helper into the shared module.
14. **Dedup helper patterns** — `storage/memory` uses `withWriteLock(code, msg, fn)` + `withReadLock[T](s, code, msg, fn)` + `wrapClosed(err, code, msg)`. `metaengine.DeferClose(c Closer)` replaces `defer func() { _ = x.Close() }()` across all engine modules (72 production + 33 test sites, 2026-09-10 recount). The `.art-dupl-baseline.json` golden + `nix run .#check-duplication` gate enforce no-new-clones; run `art-dupl baseline . --threshold 3 --semantic` to update after a consolidation. A `//art-dupl:accept <reason>` comment on/above a clone region suppresses that group LIVE — annotate intentional clones instead of re-pinning the baseline; reserve baseline regen for structural shifts. Annotation is ITERATIVE: art-dupl reports one group per region pair, so suppressing the visible groups unmasks others behind them — re-run until "0 new clone groups" (2026-08-16: 12 groups took 4 rounds). The directive must sit directly on/above the region's FIRST line; placing it above the following function's doc comment does not suppress. The `#check-duplication` app refuses to run while `.art-dupl-baseline.json` has uncommitted changes (dirty-tree guard: re-pins must happen on a committed baseline).
15. **bbolt secondary index** — `storage/bbolt` uses a `cqrs_journal_idx` bucket (eventID → journalKey) as a secondary index for O(log N) Seek-based reads in `ReadStreamFrom`. Old databases without the index fall back to linear scan transparently. The `cqrs_journal` bucket holds the event journal; `cqrs_journal_idx` is the index. Both are created at DB init in `base.go`.
16. **Store wrapping goes through `event.DecorateStore`; journal wrapping through `event.DecorateJournal`** ([ADR-0126](docs/adr/0126-metadata-generic-store-transforms-wal-unification.md)) — Never hand-write Store/Journal wrapper structs: they drop optional capabilities (the old `encryptedStore` silently lost MultiSink; the old `VersionedSeekableJournal` lost StreamingJournal). Compose `SinkTransform`/`SourceTransform` instead (`encryption.EncryptSinkTransform`; `schema.UpcastSourceTransform` + `event.DecorateJournal` for journals). Deprecated shells (`schema.VersionedStore`, `schema.VersionedSeekableJournal`, `signing.Rejecting*`, `encryption.ErrInnerStoreNot*`, `metadata.CustomData`) exist for external consumers only — internal code uses the canonical forms; removal at v5.
17. **WAL cores are generic, policies injected** (ADR-0126) — `storage/memory.LogStore[T, ID]` (via `LogStoreConfig`), `storage/sql.Inserter[T]` (write-side counterpart of `JournalReader[T]`), and `system.AdapterCore[T]` own the shared mechanics. Divergent semantics (duplicate/not-found policy, missing-position replay, per-entity conflict sentinels) live in config funcs, not forked code. New stores embed the core instead of copying it.
18. **Import grouping is owned by treefmt, not gci** — `nix fmt` (treefmt goimports `-local github.com/larsartmann/go-cqrs-lite`) produces the 3-group layout; `gci` was REMOVED from `.golangci.yml` formatters (2026-08-16) because two tools fighting over the same import blocks re-broke 95+ files once. CI's `nix fmt --fail-on-change` gate enforces grouping mechanically.
19. **Engine `register.go` files are intentional clones** — each dep-isolated `metaengine/*engine` module needs its own `init()` calling `metaengine.RegisterDriver` (the database/sql pattern; Go cannot centrally register). Each carries a `//art-dupl:accept` directive; do NOT try to deduplicate across modules.
20. **Root CHANGELOG only; per-module CHANGELOGs forbidden** — nothing reads module-local changelogs and they drifted into describing shipped work as Unreleased (consolidated 2026-08-16; policy in CONTRIBUTING.md). `scripts/check-changelog-symbols.sh` (CI + `#verify`) gates every `pkg.Symbol` cited in the root `[Unreleased]` Added/Changed sections against the api-stability golden + repo source — kills the reverted-work fiction class. `cmd/doc-check` fails on ANY warning (zero-warning policy since 2026-08-15), including zero total references; `cmd/api-stability` fails loudly on unparseable modules instead of skipping them (a silently-shrinking golden is the corruption tell).
21. **Data-model conventions (ADR-0111, T04–T12, 2026-08-22)** — `record/` is the Tier-0 structural base; the v4.x surface rules: (a) `event.Type`/`command.Type`/`query.Type` are ALIASES of `record.Type` — never reintroduce per-module copies; the lockstep tests use cross-type comparison (`event.Type("x") != record.Type("x")`), which only compiles while the alias holds — do NOT write the var-decl form (ST1023 and S1021 fight over it). (b) `Record.Encoding` is the compact typed stamp (`record.EncodingJSON`/`EncodingCBOR`; zero = `EncodingUnknown` = absent/opaque/envelope-wrapped); `record.ParseEncoding`/`String()` map the canonical "json"/"cbor" names — record stays zero-dep, bridges convert at their boundary. (c) Identity: decider's `*Ref` methods take ONE `id.StreamRef`; pair forms are deprecated forwarders (removed v5). (d) Branded IDs that carry CALLER-CHOSEN semantic keys (timer IDs, stream IDs) are STRING-backed (`cbid.ID[Marker, string]`, the `id.StreamID` pattern) — ULID backing is only for system-minted IDs; semantic keys under ULID break idempotency. (e) Actor attribution is typed `id.ActorID` end-to-end; SQL envelopes keep a plain string column and convert at the boundary (`PrefixedString()` out, `ParseActorID` in). (f) AsRecord bridges populate, never drop: `ID`, `Encoding`, `Cause` (kind explicit), structural `Actor`, `Stamp`s. (g) Capability interfaces (`query.MetadataCarrier`/`PayloadCarrier`, `command.MetadataCarrier`) replace inline duck-typed assertions — growing the core `Command`/`Query` interfaces is BREAKING and waits for v5.
22. **Temporal-composability contract (ADR-0136, 2026-09-10)** — every effect sits on the invertibility ladder: replayable (derived data → `Reset` + replay; enforced by `projectionhost.Resettable` + `metaengine.EngineResetter`/`Store.Reset`, warn-first v4.x, hard v5) → compensable (external effects → `deriver` sagas, never `Close()`) → must-be-an-event (facts → tombstone/rebirth per ADR-0114). Ask "what is its inverse?" before writing an effect. Reset must be total or loud (`ResetResult.Partial()`); ALL first-party engines implement `EngineResetter` since 2026-09-11 (memory, sqlite/turso, pebble, bbolt, badger, pg, mysql, duckdb, dgraph, iroh-via-local); sequence counters (AUTOINCREMENT/sequences/in-memory) deliberately keep advancing across resets so pre-reset resumption tokens never skip replayed entries.
23. **Health-driven engine deactivation (ADR-0137, 2026-09-10)** — `metaengine` quarantines an engine after N consecutive errorfamily Infrastructure/Transient failures (default 3, `SetEngineFailureThreshold`); execution reroutes via `effectiveQueryLocked` (execution-scoped `routedQuery` override — plan untouched) and `routableLocked` never re-plans onto quarantined engines. Fold WRITES reroute identically (`dispatchFoldsCoreLocked` — same capability-aware partition rule). Reactivation requires a successful `Prober` probe (`StartAutoReprobe`) or explicit `ReactivateEngine` — never a timeout; the reprobe path prefers `CatchUpEngine` (reset + EventLog replay into exactly the quarantined engine, quarantine lifted only after a clean rebuild; `ErrCatchUpUnsupported` falls back to plain reactivation). Rejection/Conflict/Corruption/unclassified errors never count toward health. Lock ordering: `s.mu` → `healthMu` only; health methods never take `s.mu`.
24. **Coeffect validation is three-tier (2026-09-10)** — runtime `system.New` gate (`DomainConfig.Events`, opt-in, `ErrDanglingEventSubscription` hard / `coeffect.unconsumed_event` advisory), static cqrs-lint **E018** (projection-without-emitter; `catalog.Event` declarations count as provided; silent when zero emissions detected), docs-side `catalog.ValidateCoeffects` + the `coeffects.md` export summary (derivation via `Catalog.DeriveProducersConsumers`, shared by the exporter). Keep the three in lockstep when touching subscription semantics; E018's suggestion and the core.md §3.9 recipe cross-reference the runtime gate.

## Error Handling

- **Sentinel errors**: `errors.New` in `errors.go` files
- **Contextual errors**: `fmt.Errorf("failed to process %s: %w", name, err)`
- **Classified errors**: `errorfamily.NewRejection(...)`, `errorfamily.WrapConflict(...)` via [go-error-family](https://github.com/larsartmann/go-error-family) — imported directly, no facade
- **6-family taxonomy**: Rejection / Conflict / Transient / Infrastructure / Corruption / Orchestration
- **Direct import**: All modules import `errorfamily "github.com/larsartmann/go-error-family"` directly. The `event/` package retains type aliases (`event.Family`, `event.Error`) and family constants for backward compat, but error construction/classification/wrapping functions were removed. Use `go-error-family` directly.

## Codec Defaults (debugging encoding issues)

The default codec differs by layer. Events are self-describing (`evt.Encoding()` stamped on every event), so mixed JSON+CBOR event streams decode correctly via `DecodePayloadAuto`.

| Layer                           | Default codec | How to override                                                            |
| ------------------------------- | ------------- | -------------------------------------------------------------------------- |
| `stack.ReadModel`/`Materialize` | CBORCodec     | `stack.WithDefaultCodec(json)`                                             |
| `event.New()`                   | CBORCodec     | `event.DefaultCodec = codec.JSONCodec{}` or `event.WithCodec(c)` per-event |
| `kv.NewTypedStore()`            | CBORCodec     | `kv.WithTypedCodec(c)`                                                     |
| `snapshot.NewTypedStore()`      | CBORCodec     | positional arg: `NewTypedStore(store, c)`                                  |
| command typed store             | CBORCodec     | positional arg: `NewTypedCommandStore(store, c)`                           |
| query typed store               | CBORCodec     | positional arg: `NewTypedQueryStore(store, c)`                             |

Blind stores (kv/snapshot/command/query) are self-describing too via the ADR-0044 envelope: `WrapEncode`/`UnwrapDecode` stamp the codec on write and auto-detect it on read. Non-envelope data decodes via the store's configured codec with a JSON↔CBOR cross-retry, so pre-envelope rows written with either standard codec stay readable (ADR-0050 addendum; `decodeEnvelopeOrLegacy` per blind-store module).

One-call CBOR for both events AND read models: `bundle, _ := sqlite.New(dsn, stack.WithEventCodec(codec.CBORCodec{}))` (the `stack` presets are removed at v5 — ADR-0123; pass the equivalent codec option to `system.New`'s engines instead).

## Testing

Conventions, soak env vars, integration playbooks, race thresholds, flake cures: [`docs/agents/gotchas-testing.md`](docs/agents/gotchas-testing.md).

## Gotchas & Non-Obvious Behaviors (index)

Split by topic; edit the topic file, never inline here:

- [`gotchas-tooling-build.md`](docs/agents/gotchas-tooling-build.md) — nix fmt/lint gates, exit-code traps, #verify exclusivity, background jobs, LSP noise, QEMU, storage-engine visibility quirks, turso-go IVM defects (zombie-tx readback, version-citation gate, ivmrepro release check), cache env chain, workspace rules, system/v4 follow-ups.
- [`gotchas-module-management.md`](docs/agents/gotchas-module-management.md) — testModules coupling, api golden rules, tag-wave four hard mechanics, pin sweeps, sibling replaces, release process.
- [`gotchas-language-footguns.md`](docs/agents/gotchas-language-footguns.md) — pgx/CBOR/encoding traps, GOWORK positional, alloc pins, Dgraph/MariaDB/SQLite/DuckDB dialects.
- [`gotchas-testing.md`](docs/agents/gotchas-testing.md) — full testing conventions.
- [`gowork-modes.md`](docs/agents/gowork-modes.md) — THE GOWORK decision table + mandatory env chain + jsonv2 tag.
- [`module-map.md`](docs/agents/module-map.md) — full 82-module table with internal notes.

TL;DR rules (too hot to be one click away):

1. **Never `rm`/`git reset`/`git checkout`/plain `mv`** — `trash`, `git switch`/`git restore`, `git mv`.
2. **Cache env chain + `-tags "goexperiment.jsonv2"`** on every go command ([`gowork-modes.md`](docs/agents/gowork-modes.md)).
3. **`#verify` runs exclusively** — never concurrent with integration suites or heavy builds.
4. **Auto-commit daemon absorbs working-tree changes** — expect `chore: auto-commit` commits; wait for clean tree before tagging.
5. **API-surface change ⇒ api golden regen in the same edit** (`cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update`).

## Procedures

### Add a New Module

1. Create the directory with a `go.mod` (module path: `github.com/larsartmann/go-cqrs-lite/<name>/v4`)
2. Add the module path to `go.work`
3. Add the module path to `testModules` in `flake.nix` (feeds both `#test` and `#lint`)
4. Add the module path to `cmd/api-stability/main.go` `modules` slice
5. Run `go build -tags "goexperiment.jsonv2" ./...` to verify compilation
6. Run `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update` to generate golden
7. Run the meta-tests: `cd cmd/api-stability && GOWORK=off go test -tags "goexperiment.jsonv2" -run TestEvery .`

### Change an Exported Symbol

1. Make the code change
2. Immediately: `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update` (regenerate golden)
3. Update any affected skill references (`.agents/skills/go-cqrs-lite/references/*.md`)
4. Run `cd cmd/doc-check && GOWORK=off go run -tags "goexperiment.jsonv2" . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md`
5. Run `nix run .#verify` (or at minimum `nix run .#verify-fast`)

### Verify Before Release

```bash
nix run .#verify          # build + vet + test + race + lint + doc-check + doc-assertions
nix run .#vulncheck       # per-module standalone build (catches version-sequence breaks)
nix run .#check-arch      # dependency budget enforcement
nix run .#check-coverage  # coverage drift
nix run .#check-duplication  # no-new-clones gate
nix run .#check-error-taxonomy  # errorfamily codes vs docs/error-taxonomy.md drift gate
```

## Module Tiers

Seven-tier model — see [ADR-0046](docs/adr/0046-seven-tier-model.md) and [SEVEN-TIER-MODEL.md](docs/architecture-understanding/SEVEN-TIER-MODEL.md) for full mapping (78 modules across 7 tiers).

```
Tier 0 — Primitives: id/, dispatcher/, kv/, dedup/, record/ (codec, retry, flightrecorder extracted → external repos, ADR-0128)
Tier 1 — Core Domain: event/, command/, query/, scheduling/, metadata/
Tier 2 — Domain Utilities: schema/, snapshot/, projection/, idempotency/, deriver/, commandlifecycle/, idempotency/kvstore/, idempotency/sqlstore/
Tier 3 — Aggregation: decider/, graph/, scenario/, projectionhost/, listing/, metaengine/, commandlifecycle/projections/
Tier 4 — Infrastructure: storage/*, signing/, encryption/, otel/, prometheus/, middleware/, transport/*, watermill/,
                     testutil/, metaengine/*engine/, metaengine/projectionadapter/, metaengine/keycodec/, scheduling/sqlstore/
Tier 5 — Composition: stack/, stack/*presets/, system/
Tier 6 — Tooling & Examples: catalog/, integration/, benchkit/, cmd/*, example/*, event/v4/eventtest/
```

## Dependencies

Rules only — see each module's `go.mod` for the actual package list.

- **Production deps per module**: enforced by `nix run .#check-arch` (Layer 1 cross-module rules). Adding production deps requires budget review.
- **Test-only packages** (gomega, ginkgo, rapid, go-snaps, testcontainers) are excluded from dep budget counts.
- **CGo isolation**: Only `stack/duckdb` and `metaengine/duckdbengine` require CGo. Each is in its own module so consumers who don't import them never need a C compiler.
- **External extracted modules**: `go-codec`, `go-retry`, `go-idempotency`, `go-flightrecorder`. The in-repo re-export shims were deleted (ADR-0128); import the external paths directly. Workspace `use` block points at sibling checkouts.

## Metaengine

> **metaengine/ is THE STRATEGIC FUTURE of this project** (possibly a future dedicated project).

It is Tier 3 (Aggregation) — conceptually aggregates Records into query-optimized projections. The core planner depends only on `dedup/` and `record/` (both Tier 0). The bridge to the CQRS event-sourcing world lives in `metaengine/projectionadapter/` (Tier 4).

### Canonical Design Docs (read before working on metaengine)

### v2 Architecture (ADRs 0111-0117)

ES-native planner depends on the `Record` type ([ADR-0111](docs/adr/0111-record-type-extraction.md)) and understands typed records, not opaque `any` blobs. Tombstones are domain events ([ADR-0114](docs/adr/0114-tombstone-as-domain-event.md)), not mutable metadata. GraphBackend is deleted; `graph.GraphDriver` implements `metaengine.Engine` ([ADR-0113](docs/adr/0113-delete-graphbackend.md)). SQLite engine moves to `metaengine/sqliteengine/` ([ADR-0115](docs/adr/0115-sqlite-engine-extraction.md)). Auto-projection is layered ([ADR-0116](docs/adr/0116-layered-auto-projection.md)): 80% auto-generated from type inspection, 100% auto-routed. Command lifecycle (DLQ, retries) is event streams ([ADR-0117](docs/adr/0117-command-lifecycle-as-events.md)).

### Live Cost Measurement (dynamic NetworkRTT / per-op latency)

Remote engines (PG, MySQL, Dgraph, Turso) declare compile-time RTT priors; the live system replaces them with runtime observations so the cost-based planner routes on honest data. Surface: `Prober`/`TransactMeasurer` capability interfaces, `ProbeEngine` background loop + `LatencyTracker` (EWMA/P50/P95/P99), `Calibration.ApplyCalibration` (precedence: compile-time defaults -> calibration priors -> live measurement), `Store.Replan`/`CheckRouting`/`StartAutoReplan` + `WithRoutingHysteresis`/`WithRoutingMinDelta`, `NsForRead` RTT amortization (a 10K-row scan pays RTT once), and `GetEngineStats`/`Doctor`/`EXPLAIN` diagnostics.

Design doc: [`METAENGINE-LIVE-LATENCY-MODEL.md`](docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md). Recipe: `recipes.md` §2.11.

### User's vision statement (the north star)

This is the guiding intent for every metaengine decision. When design choices conflict, defer to this.

```text
"Developers declare ONLY Commands + Events + Queries and their relationships. We should be able to build
superb projections (materialized views) and developers never need to worry about anything else, while where
data lives is up to operators at DEPLOYMENT time."
```

**Paradigm framing (2026-09-10):** the vision above is the "context paradigm" from Cordis (arXiv:2608.25512 — spatiotemporal composability). Developers write coeffect specifications (queries + relationships); operators reconcile config (engines); the planner mediates as the unified context. Engines are literally `Profile() + Closer` (capability declaration fused with a disposer); layouts are revertible via gated rebuild (`RebuildThreshold`/`ConfirmRebuild`). Full mapping — including how the whole repo and the go-modularize skill project the same two axes — lives in [`docs/architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md`](docs/architecture-understanding/2026-09-10_cordis-spatiotemporal-composability-mapping.md).
