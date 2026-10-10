# SUPERB — v5 Schema Evolution: Execution Plan (T2/T1 shipped → fleet value)

> **Date:** 2026-10-09 18:02 CEST · **Plan type:** Pareto execution plan (pareto-planning skill)
> **Scope:** the v5 declarative schema-evolution program — everything open after T2 + T1-core
> shipped gate-green this session (status:
> [`docs/status/2026-10-09_17-57_v5-schema-evolution-t2-t1-shipped.md`](../status/2026-10-09_17-57_v5-schema-evolution-t2-t1-shipped.md)).
> Other `TODO_LIST.md` sections (BDD harness, queue module, data-mesh, metaengine substrate…)
> belong to sibling work streams with active sessions and are deliberately OUT of scope here.
> **Customer** = the first-party fleet (bank-sync, DiscordSync, cqrs-htmx, go-appkit) + v5-us.
> **Source of truth for state:** `TODO_LIST.md` "V5 declarative schema evolution" section; this
> file is the point-in-time breakdown snapshot.

## Context (why this plan exists)

The proposal ([`2026-10-09_v5-declarative-schema-evolution.md`](2026-10-09_v5-declarative-schema-evolution.md))
was ruled on 2026-10-09 (delegated via standing loop): T2-first sequence adopted. T2 (named upcast
ops: `schema.Compile` + 7 ops + `Chain`) and T1-core (`system.DomainConfig.Schema` composing the
chain over both read seams) are implemented, tested, lint-clean, gate-green — **but nothing is
released and no consumer imports it yet**. The bank-sync pilot is proven via a preserved patch
(`2026-10-09_t2-pilot-bank-sync.patch`) that applies the moment `schema/v4 v4.6.0` tags.
The blocking dependency is repo-wide: two gates are red from the CONCURRENT session's in-flight
files (`metaengine/typed_reader_scan.go` file-size; 6 clone groups in `systemscenario/` +
`cmd/cqrs-lint`) — not from this stream.

Three owner questions from the 17:57 status report remain open (tag timing, ADR shape,
bank-sync application path). This plan encodes the recommended answer to each as its default
path with the decision point marked (D1–D3) — executing the plan without further ruling follows
the same delegated-approval pattern already used twice today.

## Pareto breakdown

### The 1% → 51% — SHIP WHAT'S BUILT (release + first adoption)

The tag wave (`schema/v4 v4.6.0` + system minor bump, co-released) plus applying the preserved
bank-sync patch. Every line of T2/T1 value is zero until a consumer imports it: the tag converts
workspace code into fleet value, unblocks system's `GOWORK=off` per-module CI (its go.mod pins
published schema v4.5.2 which lacks the new API), proves the release path end-to-end, and makes
the demand-proven consumer (bank-sync) the first adopter. Everything else in this plan compounds
on an API that nobody imports yet.

### The 4% → 64% — ONE DECLARATION, ALL CONSUMERS (T1 completion + 2nd/3rd adoption)

The declaration's promised payoff — "one list, four consumers" — is half-built: TypeDecoder
registration, catalog render (+ the semver version-identity bridge), and the cqrs-lint
undeclared-event rule still read OTHER sources. Finish those and adopt in DiscordSync (211-line
hand-rolled upcasters package → `RenameType` + `Transform`; the rename-family showcase) and
cqrs-htmx (surfaces `SchemaVersion` with zero upcasters today). After this tier the declaration
is load-bearing across the fleet and drift is machine-detected in three places — that IS the
"evolvable schema" the original ask wanted, running in production.

### The 20% → 80% — TRUST + CORRECTNESS SURFACE

- **T4 snapshot state-shape stamp** — kills today's silent stale-snapshot load (a real
  correctness bug class; bank-sync snapshots `BalanceSyncState`). Absent stamp = accept,
  mismatch = discard + counted stat + rebuild from journal (ADR-0136/0143).
- **ADRs for the shipped work** (the ruling said ADRs split out _as increments land_ — owed).
- **Benchmarks** (upcasting runs on every event load; chain vs closure cost unmeasured),
  **godoc Examples**, **fuzz + property tests** — the trust surface.
- **Docs wave**: recipes.md recipe (compile-harness classified), core.md §3, README polish.

### The other 20% → 100% — DRIFT DETECTION + POLICY + TAIL

T3 payload fingerprint ledger (declared-shape hash, metadata-carried first, advisory),
T5 persisted layout fingerprints + boot drift gate (existing `RebuildThreshold`/`ConfirmRebuild`
machinery) + completed-replay marker, T6 compat policy + additive-change lint rule, and the
housekeeping tail (kv-alias sweep, `RevisionSnapshotFilter` lead, proposal-fence convention doc,
parallel-declaration ritual, runbook note, TODO prune).

## Execution graph (mermaid)

```mermaid
flowchart TD
    subgraph P0["PHASE 0 — 1% → 51%: ship what's built"]
        M1["M1 gate pre-flight + tag-path decision D1"] --> M2["M2 implementation ADR (T2+T1) D2"]
        M2 --> M3["M3 changed-set verify + tag wave: schema v4.6.0 + system"]
        M3 --> M4["M4 proxy + clean-consumer verification"]
        M4 --> M5["M5 bank-sync: apply pilot patch D3"]
    end

    subgraph P1["PHASE 1 — 4% → 64%: one declaration, all consumers"]
        M6["M6 TypeDecoder derivation from Schema"]
        M7["M7 catalog render + semver bridge"]
        M8["M8 cqrs-lint undeclared-event rule"]
        M9["M9 DiscordSync collapse"]
        M10["M10 cqrs-htmx adoption"]
        M5 --> M9 & M10
        M6 --> M7 --> M8
    end

    subgraph P2["PHASE 2 — 20% → 80%: trust + correctness"]
        M11["M11 snapshot envelope stamp"]
        M12["M12 decider discard+rebuild path"]
        M13["M13 bank-sync consumes stamp"]
        M14["M14 chain benchmarks"]
        M15["M15 godoc examples"]
        M16["M16 fuzz + property tests"]
        M17["M17 docs wave: recipes/core/README"]
        M11 --> M12 --> M13
    end

    subgraph P3["PHASE 3 — other 20% → 100%: drift + policy + tail"]
        M18["M18 T3a fingerprint + metadata stamp"]
        M19["M19 T3b read-side warn hook"]
        M20["M20 T5a layout fingerprint persistence"]
        M21["M21 T5b boot diff → rebuild gate"]
        M22["M22 T5c completed-replay marker"]
        M23["M23 T6 compat policy + lint rule"]
        M24["M24 housekeeping A"]
        M25["M25 housekeeping B"]
        M18 --> M19
        M20 --> M21 --> M22
    end

    M5 --> M26["M26 full-repo #verify final sweep"]
    M13 & M17 & M23 --> M26
```

**Decision points:** D1 (M1) tag now vs wait for concurrent-session gates — default: re-check;
tag from changed-set testing if their gates stay red (runbook-compliant, per the 2026-10-06
lesson). D2 (M2) one combined ADR vs two — default: ONE combined ADR (ops + composition shipped
as one semantic unit). D3 (M5) who applies the bank-sync patch — default: this stream applies it
post-tag, battery-verified.

## Medium plan — 26 tasks, 30–100 min each (sorted by importance/impact/effort/customer-value)

| #   | Task                                                                                                                                               | Tier | Impact (1–5) | Effort (min) | Customer value                                | Depends    |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ---- | ------------ | ------------ | --------------------------------------------- | ---------- |
| M1  | Gate pre-flight: re-run `#check-file-size` + `#check-duplication`, assess concurrent-session state, decide + record tag path (D1)                  | 1%   | 5            | 30           | Unblocks the entire release chain             | —          |
| M2  | Implementation ADR for T2+T1 (one combined; D2): ops semantics, batch-vs-bridge, DomainConfig composition, layering, rejected alternatives         | 20%  | 5            | 60           | Decision archaeology ships WITH the wave      | M1         |
| M3  | Changed-set release: `GOWORK=off` per-module tests (schema+system), CHANGELOG cuts, tag `schema/v4.6.0` + system minor via release scripts (flock) | 1%   | 5            | 90           | Fleet can import; system CI unblocked         | M2         |
| M4  | Post-tag verify: proxy propagation, clean-consumer `go get` smoke, release note                                                                    | 1%   | 4            | 30           | Trust the tag is real                         | M3         |
| M5  | bank-sync: apply `2026-10-09_t2-pilot-bank-sync.patch`, resolve drift vs their HEAD, full battery, commit (D3)                                     | 1%   | 5            | 45           | First real consumer live on ops               | M4         |
| M6  | T1a: `EventSchema` Go-type binding (`Event[T]`) + `projectionadapter.TypeDecoder` derivation + system wiring when `ProjectionTypeDecoder` nil      | 4%   | 5            | 90           | Declaration replaces hand-registered decoders | M3         |
| M7  | T1b: catalog render from `Schema` + version-identity bridge (semver derived from wire int)                                                         | 4%   | 4            | 90           | One declaration → governance export           | M6         |
| M8  | T1c: cqrs-lint undeclared-event rule reading `Schema`                                                                                              | 4%   | 4            | 90           | Typo-proof declarations, third lockstep tier  | M7         |
| M9  | DiscordSync pilot: collapse 211-line upcasters onto `RenameType`/`Transform`; battery; adopt or preserve patch per tag state — **DONE 2026-10-10** (found + fixed the CBOR nested-map library bug, D7) | 4%   | 4            | 60           | Second consumer; rename-family showcase       | M5         |
| M10 | cqrs-htmx: adopt `DomainConfig.Schema` (today: SchemaVersion surfaced, zero upcasters) — **DONE 2026-10-10** (tagged surface only, D8)                                                                            | 4%   | 3            | 45           | Third consumer; zero-upcaster → declared      | M5         |
| M11 | T4a: snapshot envelope optional state-shape stamp + accessors + roundtrip tests (absent = accept)                                                  | 20%  | 4            | 60           | Kills silent stale-snapshot loads (class)     | —          |
| M12 | T4b: `decider.WithSnapshotStateVersion` + mismatch → discard + counted stat + rebuild-from-journal + conformance test                              | 20%  | 5            | 90           | Snapshot correctness, journal-is-truth        | M11        |
| M13 | T4c: bank-sync stamps `BalanceSyncState` (real-consumer proof) + battery                                                                           | 20%  | 3            | 30           | Proven on the snapshot user                   | M12, M5    |
| M14 | Benchmarks: chain `SourceTransform` vs hand-rolled closure (ns/op + allocs), record baseline, perf note                                            | 20%  | 3            | 60           | Read-path cost known, not guessed             | —          |
| M15 | Godoc `Example` functions: `Compile`, `Chain.SourceTransform`, `Declare`+system                                                                    | 20%  | 3            | 45           | Discoverability on pkg.go.dev                 | —          |
| M16 | Fuzz (arbitrary payloads × chains; random op sets) + rapid properties (version monotonicity, identity preservation)                                | 20%  | 4            | 90           | Hostile-input trust for every-load path       | —          |
| M17 | Docs wave: recipes.md recipe (compile-harness classified), core.md §3 conventions, README/faq cross-links                                          | 20%  | 3            | 90           | The sanctioned-form documentation             | M6         |
| M18 | T3a: declared-shape fingerprint helper (stable hash, never ciphertext) + opt-in metadata stamp on write                                            | tail | 3            | 90           | Drift becomes detectable                      | M6         |
| M19 | T3b: read-side compare + warn hook + opt-in hard mode + burn-in criterion wiring                                                                   | tail | 3            | 60           | Advisory ledger live in fleet                 | M18        |
| M20 | T5a: `LayoutPlan` stable fingerprint, engine-side persistence (memory+sqlite), reset clears/journal exempt, absent=accept                          | tail | 3            | 90           | Layout drift detectable at boot               | —          |
| M21 | T5b: boot diff → `LayoutDiff` → `RebuildThreshold` auto / `ConfirmRebuild` gate + tests                                                            | tail | 4            | 90           | LiveStore-style rematerialization             | M20        |
| M22 | T5c: completed-replay marker (crash-mid-rebuild safety) + Doctor render                                                                            | tail | 3            | 60           | Never serve half-rebuilt state                | M21        |
| M23 | T6: compat policy doc (additive vs version-bump triggers) + cqrs-lint additive-change rule + fixtures                                              | tail | 3            | 90           | Policy enforced, not remembered               | M8         |
| M24 | Housekeeping A: kv-alias sweep (recipes.md:65, core.md id), proposal-fence convention → gotchas doc, parallel-declaration ritual → AGENTS          | tail | 2            | 60           | Future sessions stop re-learning              | M17        |
| M25 | Housekeeping B: verify/drop `RevisionSnapshotFilter` lead, migrate-journal runbook note, TODO prune, status annotate                               | tail | 2            | 45           | Debt recorded honestly                        | M5         |
| M26 | Full-repo `nix run .#verify` + fast integration set once concurrent session is green; triage anything of mine                                      | 1%   | 4            | 60           | The one gate run this stream never did        | M5, others |

**Order = priority.** Total ≈ 23.5 h of planned work. Phases 0–1 (M1–M10, ≈ 8.5 h) deliver 64%;

- Phase 2 (M11–M17, ≈ 7.5 h) = 80%; Phase 3 (M18–M25, ≈ 8.5 h) closes to 100%; M26 is the
  closing gate.

## Fine plan — 134 subtasks, ≤12 min each

| Sub   | Task (≤12 min)                                                            | Min | Dep     |
| ----- | ------------------------------------------------------------------------- | --- | ------- |
| M1.1  | Re-run `#check-file-size`, record verdict + owner of offender             | 6   | —       |
| M1.2  | Re-run `#check-duplication`, record group owners                          | 6   | —       |
| M1.3  | Check concurrent-session git log for gate fixes landed since 17:57        | 5   | —       |
| M1.4  | Decide + record D1 (wait vs changed-set tag) in this plan's blocker log   | 8   | M1.1–3  |
| M1.5  | If green: `nix run .#verify-fast` pre-flight; record                      | 10  | M1.4    |
| M2.1  | ADR skeleton + Context section (proposal + status links)                  | 12  | M1      |
| M2.2  | Decision: op set + matching/versioning semantics                          | 12  | M2.1    |
| M2.3  | Decision: batch SourceTransform vs Upcasters bridge boundary              | 10  | M2.2    |
| M2.4  | Decision: DomainConfig composition point + Tier5→Tier2 layering           | 12  | M2.3    |
| M2.5  | Alternatives-rejected section (port from proposal §11)                    | 10  | M2.4    |
| M2.6  | Consequences + cross-links; md-go + doc-check gates                       | 8   | M2.5    |
| M3.1  | `GOWORK=off go test ./...` schema module, record                          | 8   | M2      |
| M3.2  | Schema CHANGELOG cut for v4.6.0                                           | 10  | M3.1    |
| M3.3  | Tag schema v4.6.0 (`tag-release.sh`, flock, self-sourced env)             | 12  | M3.2    |
| M3.4  | system: bump schema pin → v4.6.0, `go mod tidy`                           | 10  | M3.3    |
| M3.5  | `GOWORK=off go test ./...` system module                                  | 10  | M3.4    |
| M3.6  | System CHANGELOG cut (minor)                                              | 8   | M3.5    |
| M3.7  | Tag system minor via release script                                       | 10  | M3.6    |
| M3.8  | Versions-manifest + README-link gates; record tag pair                    | 10  | M3.7    |
| M4.1  | Proxy verification: `go list -m -versions`, pkg.go.dev spot-check         | 10  | M3.8    |
| M4.2  | Clean-consumer smoke: temp module `go get` + compile a chain              | 12  | M4.1    |
| M4.3  | Write release note into status stream                                     | 8   | M4.2    |
| M5.1  | `git apply` pilot patch in bank-sync                                      | 8   | M4      |
| M5.2  | Resolve drift vs bank-sync HEAD (other session moved files)               | 12  | M5.1    |
| M5.3  | `go build ./...` + `internal/cqrs` tests                                  | 10  | M5.2    |
| M5.4  | `-race` + lint + erraudit on touched pkg                                  | 12  | M5.3    |
| M5.5  | Commit in bank-sync (daemon-aware) + record                               | 5   | M5.4    |
| M6.1  | Design note: `Event[T]` type binding + inference rules                    | 12  | M3      |
| M6.2  | schema: `Event[T]` generic constructor + validation                       | 12  | M6.1    |
| M6.3  | projectionadapter: derive TypeDecoder registrations from declarations     | 12  | M6.2    |
| M6.4  | system: auto-fill `ProjectionTypeDecoder` when nil + Schema present       | 12  | M6.3    |
| M6.5  | Tests: declaration → decoder roundtrip (typed fold sees upcasted payload) | 12  | M6.4    |
| M6.6  | Gates: arch + api golden + module tests + lint                            | 10  | M6.5    |
| M7.1  | Map `EventSchema` → `catalog.Event` surface (incl. WithVersion)           | 12  | M6      |
| M7.2  | Semver derivation from wire int (version-identity ruling)                 | 12  | M7.1    |
| M7.3  | Exporter bridge + render tests                                            | 12  | M7.2    |
| M7.4  | system: expose declaration list to catalog path                           | 12  | M7.3    |
| M7.5  | `#check-eventcatalog` gate green                                          | 10  | M7.4    |
| M7.6  | Docs (modules.md, faq) + api golden                                       | 10  | M7.5    |
| M8.1  | cqrs-lint: parse `DomainConfig.Schema` declarations in analyzer           | 12  | M7      |
| M8.2  | Rule logic: consumed-but-undeclared → diagnostic                          | 12  | M8.1    |
| M8.3  | Testdata fixture project (declared + dangling consumer)                   | 12  | M8.2    |
| M8.4  | Suggestion text + catalog.Event counts as provided (E018 lockstep)        | 10  | M8.3    |
| M8.5  | Analyzer self-tests + module-catalog meta-test                            | 12  | M8.4    |
| M8.6  | cqrs-lint README + CHANGELOG + golden                                     | 10  | M8.5    |
| M9.1  | Read DiscordSync upcasters fresh; add replace scaffolding                 | 10  | M5      |
| M9.2  | Map two type renames → `RenameType` ops                                   | 12  | M9.1    |
| M9.3  | Map derived-field upcasters → `Transform` ops                             | 12  | M9.2    |
| M9.4  | Wire chain + rewrite their tests                                          | 12  | M9.3    |
| M9.5  | Battery; adopt (tag available) or preserve patch + revert                 | 12  | M9.4    |
| M10.1 | Locate cqrs-htmx SchemaVersion surface + config seam                      | 10  | M5      |
| M10.2 | Declare `DomainConfig.Schema` in their composition                        | 12  | M10.1   |
| M10.3 | Tests + battery                                                           | 12  | M10.2   |
| M10.4 | Adopt or preserve patch; record                                           | 10  | M10.3   |
| M11.1 | snapshot: envelope optional stamp field                                   | 12  | —       |
| M11.2 | Accessors + validation (stamp format decision: string version)            | 12  | M11.1   |
| M11.3 | Envelope encode/decode roundtrip tests                                    | 12  | M11.2   |
| M11.4 | Legacy absent-stamp accept-path tests (pre-change snapshots)              | 12  | M11.3   |
| M11.5 | snapshot golden + CHANGELOG + api golden                                  | 10  | M11.4   |
| M12.1 | decider: `WithSnapshotStateVersion[State]` option                         | 12  | M11     |
| M12.2 | Save path: stamp envelope with declared version                           | 12  | M12.1   |
| M12.3 | Load path: mismatch → discard + counted stat                              | 12  | M12.2   |
| M12.4 | Integration test: stale snapshot → rebuild from journal correctness       | 12  | M12.3   |
| M12.5 | Conformance test (snapshot contract suite)                                | 12  | M12.4   |
| M12.6 | Gates: module tests + lint + arch + changelog                             | 10  | M12.5   |
| M13.1 | bank-sync: stamp BalanceSyncState version                                 | 8   | M12, M5 |
| M13.2 | Test the discard+rebuild path on real snapshots                           | 12  | M13.1   |
| M13.3 | bank-sync battery + record                                                | 10  | M13.2   |
| M14.1 | Bench fixtures: identical chain vs hand-rolled closure                    | 12  | —       |
| M14.2 | Run `-bench` (JSON + CBOR), record ns/op + allocs                         | 12  | M14.1   |
| M14.3 | Compare table + accept/reject threshold decision                          | 12  | M14.2   |
| M14.4 | Perf note in schema README                                                | 10  | M14.3   |
| M15.1 | `ExampleCompile` godoc example                                            | 12  | —       |
| M15.2 | `ExampleChain_SourceTransform` example                                    | 12  | M15.1   |
| M15.3 | `ExampleDeclare` + system composition example                             | 12  | M15.2   |
| M15.4 | vet + doc gates                                                           | 8   | M15.3   |
| M16.1 | Fuzz: arbitrary payload bytes through chains (no panic, version sane)     | 12  | —       |
| M16.2 | Fuzz: random op sets — Compile accepts/rejects deterministically          | 12  | M16.1   |
| M16.3 | rapid: version monotonicity (never below source, never decreasing)        | 12  | M16.2   |
| M16.4 | rapid: identity preservation under field ops                              | 12  | M16.3   |
| M16.5 | Seed corpus run + fix findings                                            | 12  | M16.4   |
| M16.6 | Wire into existing fuzz/golden suites + gates                             | 10  | M16.5   |
| M17.1 | recipes.md: named-ops recipe fence (full example)                         | 12  | M6      |
| M17.2 | Classify in `recipes_catalog*.go` maps                                    | 12  | M17.1   |
| M17.3 | Compile-harness run green (`TestRecipes*`)                                | 12  | M17.2   |
| M17.4 | core.md §3 conventions: declarative form is sanctioned                    | 12  | M17.3   |
| M17.5 | faq + modules cross-link polish                                           | 10  | M17.4   |
| M17.6 | doc-check + md-go gates                                                   | 10  | M17.5   |
| M18.1 | Fingerprint helper: declared-shape stable hash (sorted fields, typed)     | 12  | M6      |
| M18.2 | `EventSchema` → fingerprint exposure                                      | 12  | M18.1   |
| M18.3 | Write-path metadata stamp (opt-in option)                                 | 12  | M18.2   |
| M18.4 | system wiring: opt-in stamping from Schema                                | 12  | M18.3   |
| M18.5 | Tests: stable across runs/processes; differs on shape change              | 12  | M18.4   |
| M18.6 | Gates + changelog-symbols                                                 | 10  | M18.5   |
| M19.1 | Read-side compare hook (declared vs stamped)                              | 12  | M18     |
| M19.2 | Warn seam (slog/otel event, no default noise)                             | 12  | M19.1   |
| M19.3 | Opt-in hard mode (Rejection on mismatch)                                  | 12  | M19.2   |
| M19.4 | Tests + burn-in criterion doc note (named consumers, named cycle)         | 12  | M19.3   |
| M20.1 | `LayoutPlan` stable-hash function (deterministic serialization)           | 12  | —       |
| M20.2 | Engine-side persistence contract (small interface)                        | 12  | M20.1   |
| M20.3 | Implementations: memory + sqlite engines                                  | 12  | M20.2   |
| M20.4 | Reset clears stamps / journal exempt test (ADR-0143)                      | 12  | M20.3   |
| M20.5 | Absent stamp = accept (existing collections migration)                    | 12  | M20.4   |
| M20.6 | Gates + enginetest pin                                                    | 10  | M20.5   |
| M21.1 | Boot: read persisted vs declared, produce diff set                        | 12  | M20     |
| M21.2 | Route diff → `LayoutDiff`                                                 | 12  | M21.1   |
| M21.3 | `RebuildThreshold` auto-rebuild path + test                               | 12  | M21.2   |
| M21.4 | `ConfirmRebuild` operator-gate path + test                                | 12  | M21.3   |
| M21.5 | Both-paths integration tests (engine reset + replay)                      | 12  | M21.4   |
| M21.6 | Gates                                                                     | 10  | M21.5   |
| M22.1 | Completed-replay marker type + persistence                                | 12  | M21     |
| M22.2 | Write-at-completion; check-at-serve                                       | 12  | M22.1   |
| M22.3 | Crash-mid-rebuild test (kill before marker → rebuild again)               | 12  | M22.2   |
| M22.4 | Doctor/Explain render of fingerprint + marker state                       | 12  | M22.3   |
| M22.5 | Gates                                                                     | 10  | M22.4   |
| M23.1 | Policy doc: additive changes (safe without version bump)                  | 12  | M8      |
| M23.2 | Policy doc: version-bump triggers + breaking rules                        | 12  | M23.1   |
| M23.3 | cqrs-lint additive-change rule on declared events                         | 12  | M23.2   |
| M23.4 | Fixtures + mutation-verified self-test                                    | 12  | M23.3   |
| M23.5 | Wire into policy docs + README                                            | 10  | M23.4   |
| M23.6 | Gates                                                                     | 10  | M23.5   |
| M24.1 | kv-alias sweep: recipes.md:65 + core.md:315 import-scoping                | 12  | M17     |
| M24.2 | Proposal-fence convention → gotchas-tooling-build.md                      | 12  | M24.1   |
| M24.3 | Parallel-declaration check ritual → AGENTS.md notes                       | 12  | M24.2   |
| M24.4 | doc-check + md-go after doc edits                                         | 10  | M24.3   |
| M25.1 | `RevisionSnapshotFilter`: primary-source search; verify or drop lead      | 12  | M5      |
| M25.2 | bank-sync migrate-journal runbook: declarative-upcasting note             | 10  | M25.1   |
| M25.3 | TODO_LIST schema section prune against this plan                          | 12  | M25.2   |
| M25.4 | Annotate 17:57 status report with plan link                               | 8   | M25.3   |
| M26.1 | Confirm concurrent-session gates green                                    | 10  | M5      |
| M26.2 | `nix run .#verify` full                                                   | 12  | M26.1   |
| M26.3 | Fast integration set (`#test-integration` core engines)                   | 12  | M26.2   |
| M26.4 | Triage/fix anything from this stream                                      | 12  | M26.3   |
| M26.5 | Record final green snapshot + close plan                                  | 8   | M26.4   |

## Verification criteria (per phase)

- **Phase 0 done =** tags exist + resolve from proxy; bank-sync compiles + tests green ON the tag;
  system `GOWORK=off` per-module build/test green.
- **Phase 1 done =** one declaration produces decoder registrations + catalog output + lint
  diagnostics; DiscordSync + cqrs-htmx import the ops (or patches preserved if tags lag).
- **Phase 2 done =** stale snapshot test proves discard+rebuild; benchmark table committed;
  fuzz/rapid suites green; recipes recipe compiles in the harness.
- **Phase 3 done =** fingerprint stamp survives restart and warns on drift (advisory); boot gate
  routes to rebuild; policy doc + lint rule green; housekeeping rows closed.
- **Plan closed =** `nix run .#verify` green repo-wide.

## Risks / blockers

| Risk                                         | Mitigation                                                                                                                                                                       |
| -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Concurrent session's red gates block tagging | D1 changed-set path: release scripts test the changed set; per the 2026-10-06 lesson, run real per-module tests ourselves                                                        |
| bank-sync HEAD drift vs pilot patch          | M5.2 resolution step; patch is 417 lines across 6 files, all touched only by this stream                                                                                         |
| Another session edits system/ mid-T1a/b/c    | Re-view before every edit (daemon discipline); small PR-sized increments                                                                                                         |
| `Event[T]` type-binding design churn         | Design note FIRST (M6.1); mirrors projectionadapter's existing registration shape                                                                                                |
| Verschllimmbessern                           | Every task ends at a gate; no speculative rewrites; reverts only of self-authored scaffolding; the pilot-replace pattern (prove → preserve → revert) for unreleased-API adoption |

## Decision log (executed)

### D1 — tag timing (decided 2026-10-09 22:45): TAG NOW via changed-set

Re-check at M1 found the concurrent session's gates REDDER than at 17:57 (11 file-size offenders:
cqrs-lint ×5, metaengine ×4, projectionhost ×1, system/system.go; 6 clone groups in
systemscenario/ + cqrs-lint) and `tag-release.sh` verified to NOT run repo-wide ratchet gates
(it strips replaces → tidy → resolve-verify → tag). This stream's own files were cleaned to green
first (schema/chain_engine.go split → chain_payload.go; system/config_types.go → engine_config.go
extraction, 421→302; system/constructor.go → system_lifecycle.go + projection_wiring.go
extractions, 435→352; system tests green after each). Ruling: proceed with the tag wave on
changed-set testing (schema + system real per-module test runs) per the 2026-10-06 lesson;
concurrent-session offenders stay theirs to shrink or baseline.

### D2 — ADR shape (decided at M2, below): ONE combined implementation ADR

Executed as `docs/adr/0155-declarative-schema-evolution-t2-t1.md` (docs/adr/README.md index updated
through 0155). md-go + doc-check gates green (1240 refs).

### D1-supplement — tag wave executed by the concurrent BDD session (2026-10-10)

The 2026-10-10 "BDD harness adoption wave" (manifest:
`docs/planning/2026-10-09_tag-wave-manifest-bdd-harness.md`) cut `schema/v4.6.0` + `system/v4.12.0`
(+ deriver/v4.4.0, scheduling/v4.7.0, systemscenario/v4.0.0) riding this stream's CHANGELOG entries.
M3/M4 verification performed independently this session: all 5 tags local+origin resolved,
`git tag --contains` proves schema/v4.6.0 contains the T2 code, proxy serves schema v4.6.0 +
system v4.12.0, and a clean-consumer smoke (fresh temp module, `go get schema/v4@v4.6.0`, compile
chain + Upcasters) ran green with zero local replaces. M3+M4 therefore CLOSED as done.

### D3 — M5 executed as semantic re-collapse (decided 2026-10-10 ~07:00)

The staged bank-sync patch (written against pre-T2 APIs at ~22:30) rotted against the concurrent
session's T18 changes within 12h. Ruling: re-derive semantically instead of re-applying —
`UpcastChain()` composes `schema.Transform` ops (renames + the typed money-triplet upcaster with
its Corruption codes preserved through `schema.op_transform_failed` wrapping); the four
hand-rolled upcaster types were deleted; both wiring sites go through
`event.DecorateStore(store, nil, upcastingChain.SourceTransform())`. Full battery green
(build, tests, -race, golangci, erraudit). Landed via daemon commits `cfb8bfe1`+`cd224ca8`
(attribution rewrite deferred to owner — see status report Q3).

### D4 — M7.4 tier ruling (decided 2026-10-10 ~08:30): the bridge lives in catalog, NOT system

"system exposes the declaration list to catalog" is TIER-ILLEGAL (system Tier 5 → catalog Tier 6
is an upward dependency). Ruling: the `system.Schemas()` builder RESULT is the single artifact both
consumers receive — `system.SchemaDeclarations.Declarations()` feeds `DomainConfig.Schema` (system
side) while `catalog.FromTypedSchema[T]` consumes the same `schema.TypedEventSchema[T]` values
(catalog side, importing schema downward — legal). This also forced the catalog dep budget 7→8
(`DEP_BUDGET[catalog]`, rationale in `scripts/check-module-layers.sh`).

### D5 — catalog dev-time sibling replace (decided 2026-10-10): unblock standalone gates pre-tag

`TypedEventSchema` is post-v4.6.0 (working tree only), so every GOWORK=off consumer of catalog —
`#check-eventcatalog`'s fixture builder, api-stability, CI's per-module leg — was red until the
next schema tag. Ruling: `replace github.com/larsartmann/go-cqrs-lite/schema/v4 => ../schema` in
catalog/go.mod (the repo's established dev-time pattern, 10+ modules; `tag-release.sh` strips local
replaces at tag time, forcing the schema+catalog co-release this plan already schedules for the
Phase 1 wave).

### D6 — M8 executed: E021 scopes to EMITTED types; gate-hygiene absorption (2026-10-10 ~10:00)

E021 `emitted-without-schema-declaration` (T1c) parses `schema.Event`/`schema.EventOf` calls AND
both `system.Schemas()` builder forms (direct chain via receiver-walk; local builder variable via
per-file assignment tracking). Scope ruling: consumption-only types stay E018's provider-contract
tier (imported events), catalog.Event counts as declared, zero declarations → silence. 7 fixture
tests; detector count 210→211; README/RULES (regenerated, not hand-edited — RULES.md is generated
output). Offset the scanner_calls.go growth by extracting handlerTypeFromCall et al. to
scanner_calls_helpers.go (338 < 360 baseline — the file SHRANK). Gate hygiene absorbed from the
concurrent sessions (mechanical, additive): repo-wide `go mod tidy` (28 modules with stale go.sum
from the import churn), graph-native README (meta-test demanded), taskmanager golden V003 2→3
(their tag wave added systemscenario v4.0.0). STILL THEIRS, verified pre-existing via worktree at
05:44 commit: `TestMultiModuleBuildContext_PartitionsProfiles` (deriver CommandFlow=commands
expectation vs their deriver changes) — M26 blocker list.

### D7 — M9 executed: chain migration found a REAL library bug; fix + tagged-version workaround (2026-10-10 ~17:00)

DiscordSync's 211-line hand-rolled upcaster module collapsed onto `UpcastChain() (*schema.Chain,
error)` (2× `RenameType` keeping the `//cqrs-lint:ignore(E006)` comments + 4× `Transform(type, 1,
upcastUserKindField("author"|"user"), WithDecodePolicy(PassthroughOnDecodeError))`); storage.go's
`Journal()` now compiles the chain and wraps via `event.DecorateJournal(ec.store,
chain.SourceTransform())` + SeekableJournal assertion (errkit.Rejection on compile failure — a bad
declaration is a programming error, not infrastructure). Registry test renamed to
`upcasters_chain_test.go` via git mv; all 10 original intents kept + a new
`TestChain_PassthroughOnMalformedPayload` pinning the declared policy.

**Library bug found (the reason this was not a pure mechanical migration):** fxamacker/cbor decodes
nested maps as `map[any]any` when the target is `any` — so a `Transform` touching a NESTED object
worked on JSON events and SILENTLY NO-OPPED on CBOR events (DiscordSync's journal default). All
four user-kind upcasters bumped the version (op fired) yet never set `kind`. The schema module's
own tests only exercised FLAT CBOR field maps, so T2 shipped with the gap. Fix (in-repo,
[Unreleased]): `decodeFieldMap` normalizes nested string-key maps to `map[string]any` (shared seam
under Transform/field ops/Split; non-string-key maps stay as decoded — all-or-nothing per map);
contract documented on `schema.Transform`; regression-pinned by
`TestTransformNestedMapsAreStringKeyedOnCBOR` (typed-decode assertions — re-decoding the result as
`map[string]any` would yield `map[any]any` again). Schema + system module suites green; changelog
symbols gate green (38 citations).

**Tag-state ruling (Q2 still owner-open, so NO tag from this session):** DiscordSync consumes
schema strictly via module-proxy tags (no go.work, no replaces, hermetic nix build — a filesystem
replace would break it), and the fix is unreleased. Therefore DiscordSync pins v4.6.0 AND carries a
documented `map[any]any` branch in `upcastUserKindField` (convert → derive → write back), marked
droppable when pinning schema/v4 > v4.6.0. The `go get schema/v4@v4.6.0` MVS wave also pulled
sibling pins (event v4.13.1, id v4.7.2, snapshot v4.6.2, cbor v2.9.6, …) — required by schema
v4.6.0's own go.mod; flake pin synced (`sync-flake-pins.sh`) to the covering tag commit. NOTE:
DiscordSync's daemon auto-commits AND auto-pushes — these edits go public within ~1h of writing.

### D8 — M10 executed: cqrs-htmx adopts DomainConfig.Schema via the TAGGED surface only (2026-10-10 ~17:45)

Tag-state fact (the load-bearing discovery): system/v4.12.0 + schema/v4.6.0 (cqrs-htmx's pins) DO
contain `DomainConfig.Schema` + `schema.Event`/`Declare` (T1-core + T2 tagged), but `system.Schemas()`
(the typed builder, T1a) is UNTAGGED — so cqrs-htmx CANNOT use the one-list builder via the proxy
yet. M10 therefore adopted the tagged surface: new `systemadapter/schema.go` declares all 21
identity-model event types (`EventSchemas() []schema.EventSchema` via `schema.Event(type, 1)` —
envelope version 1, zero ops, deliberately exhaustive incl. the legacy EventRolesUpdated so
projection subscriptions stay inside the coeffect gate's declared universe) and wires
`Schema: EventSchemas()` into `DomainConfig()`. `EventTypeDecoder()` stays the decoder — the
builder's `.TypeDecoder()` consolidation is a follow-up once the builder tags. Drift-pinned by
`schema_test.go`: decoder `EventTypes()` vs declared types set-equality (production-vs-production,
catches event #22 added to one list only), `schema.Declare(EventSchemas()...)` compiles, and every
declared version equals `envelopeSchemaVersion`. The existing systemscenario harness tests double
as the boot proof (applySchemaDeclaration + coeffect gate ran green through them).

Split-brain found and DOCUMENTED, not collapsed (out of M10's scope): identity-model carries its
OWN homegrown upcaster layer (`UpcasterRegistry`, payload-embedded `schema_version` field,
`CurrentSchemaVersion = 2`, global `SetUpcasterRegistry` — called by NO production code, so the
decode path is a passthrough today). That registry is the same class of migration as M9 and a
candidate follow-up task; when collapsed, the payload-embedded versioning and the envelope
`SchemaVersion()` need a one-or-the-other decision (two versioning layers on one event is the
actual disease). Also noted: root-package `EventCatalog` (Published Language registry with
per-event SchemaVersion) is a third parallel list — the M7 `catalog.FromTypedSchema` bridge is the
convergence tool once typed declarations are tag-reachable.

Battery state: workspace `#test` green (race, forEachGoModule); systemadapter lint 0 issues after
absorbing the concurrent session's mechanical debt (gci/golines reformat of their three harness
test files via `.#fmt`, plus the errname rename `fieldMismatch`→`fieldMismatchError` across both
harness files — a test-local type, zero semantic change). STILL THEIRS: identity-model (2) +
usermgmt (4) exhaustruct findings — judgment-call field additions in concurrent-session code, left
untouched (M26 triage list).
