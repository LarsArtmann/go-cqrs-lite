# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- _(Effort: XS/S/M/L/XL)_ = rough size

---

## Performance — 2026-08-16 audit backlog 🔥

> From the RAM/cache-line + IO audit session
> ([status](status/2026-08-16_03-10_perf-audit-cache-line-sql-batching-deserialize-wins.md),
> [Pareto plan](planning/2026-08-16_03-18_PERF-PARETO-SAFETY-FIRST-EXECUTION.md)).
> Shipped: workloadMeter cache-line pad (−46..51% contended ops), dialect-aware SQL
> batching (99→3276 rows on PG/MySQL/DuckDB; PG integration GREEN), pebble+bbolt
> deserialize JSON round-trip elimination (−46% ns / −53% allocs) — commits
> `cdc525fd5` + `a298ea388`. Wave 3 closed 2026-08-16 (measured numbers in
> [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md)).

- [ ] [BLOCKED] **Durability tier→per-write-sync mapping** (storage/pebble
      hardcodes `pebble.Sync`; metaengine engines no NoSync path) — real win
      (fsync per append) but a behavior change for existing Normal-tier
      consumers. AWAITS USER DECISION (status §g Q3).
      _(Effort: M)_
- [ ] **Benchstat baselines for the 3 new false-sharing control benches**
      (multiSeqCounter padded/unpadded, WorkerCounters, SSEReplaySeq) —
      point measurements live in [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md);
      formal benchstat baselines still pending.
      _(Effort: XS)_

---

## Release / Tagging 🔥

> Blocked on user authorization (never tag/push without explicit instruction).
> Latest full-gate GREEN: 2026-08-16 13:15 (`#verify` run #4 EXIT=0, 591-line log,
> build+vet+test+race+lint 76/76, doc-check, api-surface; `#check-coverage` and
> `#check-duplication` also EXIT=0) — this is the release checkpoint. The
> 2026-08-16 chain (id v4.5.0, record v4.3.0, metadata v4.5.0, schema v4.3.0,
> event v4.7.0, query v4.6.0→retracted→v4.6.1, command v4.7.0→retracted→v4.7.1,
> middleware v4.5.0, metaengine v4.11.0, sqlite/pebble/pg engines v4.1.0,
> badger v4.0.2, watermill v4.5.0, mysql/bbolt/turso/iroh engines v4.0.0,
> storage v4.7.0→retracted→v4.7.1) is LIVE on the proxy.

- [ ] [BLOCKED] 🔥 **Tag the wave-4 module batch** — `event` (DecorateJournal),
      `metadata` v4.5.1+ (BrandedString), `schema`, `metaengine` (capability
      audit + iroh exports), `metaengine/irohengine`, `projectionhost`, and
      `storage` v4.7.2 (SQLite `OpenSQLiteInMemory` shared-cache DSNs). Constraint: the
      batch must tag event+metadata+schema before/with projectionhost (its
      released go.mod needs them — the release flow strips the sibling
      replaces). Via `scripts/tag-release.sh` from a clean tree.
      _(Effort: M)_
- [ ] [BLOCKED] 🔥 **Land the stranded tag-chain repair commits on master** —
      cherry-pick `092b5e8a8` (command/query `retract` directives + metadata
      v4.5.0 pins + hardened `tag-release.sh` standalone-build gate) and
      `4907b6afc` (metaengine/bench pseudo-version tidy) from the tag worktree
      (`git merge-base --is-ancestor` confirms both NOT on master, verified
      2026-08-16). Master's `command`/`query` go.mod still pin `metadata/v4
      v4.4.0` — any future tag cut from raw master re-breaks consumers (the
      v4.7.0/v4.6.0 incident class). Regen the api-stability golden fresh on
      master instead of cherry-picking `d25e8a959`.
      _(Effort: S)_
- [ ] [BLOCKED] **go-codec F46: commit + tag the `UnwrapDecode` sniff** —
      the first-byte fast path (fallback 181ns/6 allocs → 1.6ns/0 allocs) sits
      UNCOMMITTED in `../go-codec` (no auto-commit daemon there); GOWORK=off
      consumers get nothing until it is tagged.
      _(Effort: XS)_
- [ ] [BLOCKED] **Ratify one shipped judgment call** — iroh latency P99
      bound 50→150ms (worst-of-30 sample inflates under gate load). Shipped +
      gated green; keep or revisit. (The `OpenSQLiteInMemory` pool-pin
      judgment call was resolved: replaced with shared-cache DSNs.)
      _(Effort: XS)_
- [ ] **Replace-drop sweep (after the wave-4 tags)** — system ×6,
      cqrs-bench ×7, event ×2, schema ×2, projectionhost ×2, integration ×2
      local `replace` directives exist only because wave-3/4 code is untagged;
      drop + tidy + GOWORK=off re-verify each module. Every one is documented
      droppable-on-tag; the sweep cost compounds until then.
      _(Effort: M)_
- [ ] **Create GitHub Releases for the 2026-08-16 tags** — 20 tags, none have
      releases (only storage/v4.7.1 got one). `gh` auth never verified from
      this environment. Optionally curated notes for the 8 core modules.
      _(Effort: S)_
- [ ] **Document the retract-and-republish pattern** in CONTRIBUTING.md
      Release Process (what happened to command/v4.7.0, query/v4.6.0,
      storage/v4.7.0 and the exact remedy), and audit recently published tags
      with the hardened script's standalone-build gate once `092b5e8a8` lands.
      _(Effort: S)_
- [ ] [BLOCKED] **Tag final v4.x patches of `transport/http` +
      `transport/grpc`** (deprecation notices included) — prerequisite for
      the v5 deletion (ADR-0127).
      _(Effort: S)_
- [ ] **Create GitHub Releases + pkg.go.dev fetch triggers** — folded into
      the 2026-08-16 GitHub Releases item above.
- [ ] **Consolidate indirect dep references** — after new module tags are
      published, the transitive `go-cqrs-lite/{codec,retry,idempotency,
      flightrecorder}/v4` indirect deps in ~49 consumer go.mod files clean
      up. Track and verify.
      _(Effort: M)_
- [ ] **Run the pre-tag checklist** — `nix run .#vulncheck` +
      `#check-arch` (verify covered the rest) + GOWORK=off `go test ./...`
      on the module AND its test subpackages (the command/v4.6.0
      commandtest failure class).
      _(Effort: S)_
- [ ] **Run calibration benchmarks against baseline** — verify
      `calibration-baseline.md` accuracy; add CI regression check.
      _(Effort: M)_

---

## Pin & Standalone-Build Hygiene

> Root cause class found twice in two sessions (system replaces; benchkit
> stale pins → SQLITE_BUSY standalone failures). The workspace gate masks
> it: `#verify` resolves local modules, CI runs GOWORK=off per-module — and
> the CI "Benchmarks" job is currently RED.

- [ ] 🔥 **Repo-wide stale-pin sweep** — benchkit still pins
      `sqliteengine v4.0.1` (pre-JournalReadFrom-fix), `decider v4.3.0`,
      `event v4.6.0`… Mechanical bump of ~50 go.mod files, gate-verified.
      (Needs user sign-off on policy — see ROADMAP Open Questions.)
      _(Effort: M)_
- [ ] 🔥🔥 **storage/pebble + storage/bbolt standalone builds RED** (verified
      2026-08-16 14:20, `GOWORK=off go build` fails): both pin
      `event/v4 v4.6.0` but `serialization.go` calls
      `event.ReconstructEventWithAdoptedPayload` (shipped after v4.6.0; needs
      ≥ v4.7.0 + the unreleased adopt API or a local `../event` replace until
      tagged). Same workspace-masking class as the command/v4.7.0 incident.
      Fix: bump pins (or add sibling replaces) + add both to the GOWORK=off
      standalone gate.
      _(Effort: S)_
- [ ] **`#verify-standalone` nix app (GOWORK=off per module) or explicit
      decision that CI owns that signal** — then CHECK CI after gates.
      Investigate how long the CI Benchmarks job has been red (`gh run list`)
      to size the blind window.
      _(Effort: M)_
- [ ] **Add CI leg for GOWORK=off standalone builds of leaf modules**
      (integration/, examples/, benchkit/) to catch pin rot early.
      _(Effort: S)_

---

## Metaengine — Universal ADT Coverage (Phase 7)

> StreamLog on Dgraph, native graph on SQLite/Turso (iterative BFS), degraded
> rule with latency estimates, engine test parity shipped 2026-08-11.
> `JournalReadFrom` positional fixes (Dgraph + all SQL engines) shipped
> 2026-08-15 with shared-contract regressions. MariaDB dialect + numeric-safe
> sort + CTE probe, LSM vector search, native graph on PG/MySQL, and the
> SQLite recursive CTE shipped 2026-08-15 — see CHANGELOG.
>
> Follow-ups below harvested from
> `docs/status/2026-08-15_22-04_metaengine-followup-closeout.md` §f.

- [ ] **Turso explicit CTE-probe test** — the sqliteengine probe covers
      local drivers; add a tursoengine test confirming it holds over the
      remote protocol.
      _(Effort: S)_
- [ ] **Badger engine vector + graph parity audit** — has neither; audit
      against the pebble/bbolt precedent and either implement or document
      the gap.
      _(Effort: S)_
- [ ] **Vector search at scale** — quantization/HNSW spike for LSM engines
      when collections exceed ~100K vectors (brute-force scan is O(N)).
      _(Effort: L)_
- [ ] **`VectorResult` filtered k-NN** — metadata-filtered vector search
      (filter + top-k in one query); API currently returns bare top-k.
      _(Effort: M)_
- [ ] **`GraphRemoveEdge`** — the edges table exists but nothing removes
      edges; tombstone events should drive edge removal (ADR-0114 style).
      _(Effort: M)_
- [ ] **Graph directed-vs-undirected option** — `GraphNeighbors` is
      directed-only today.
      _(Effort: S)_
- [ ] **mysqlengine upsert semantics audit** — confirm `MapSet` uses
      `INSERT ... ON DUPLICATE KEY UPDATE` consistently with pg's
      `ON CONFLICT` (atomicity + affected-rows parity).
      _(Effort: S)_
- [ ] **MariaDB functional-index alternative** — generated columns + plain
      index instead of the current `ApplyLayout` no-op.
      _(Effort: M)_
- [ ] **enginetest per-run collection suffixes** — shared-server engines
      accumulate state under `-count>1` (documented constraint that bit the
      race runs); per-run suffixes in the helpers would remove the class.
      _(Effort: M)_
- [ ] **adttest: graph depth>2 + cycle scenarios in `RunMatrix`** — current
      matrix is depth-limited; the CTE/iterative divergence only shows at
      depth>2 with cycles/diamonds.
      _(Effort: S)_
- [ ] **adttest: Vector scenario on pgengine** — parity check for the
      degraded scan path against the in-memory index.
      _(Effort: S)_
- [ ] **Convergence suite order-tolerance audit** — `sameLogTail` was the
      only order-asserting helper (fixed 2026-08-15); sweep the remaining
      `waitFor*` helpers for hidden order assumptions.
      _(Effort: S)_
- [ ] **quic pooled-stream ordering guarantee** — default `sendOp` uses one
      stream per op (no cross-op order); verify + document that pooled mode
      (`sendOpPooled`, one stream per peer) DOES order ops.
      _(Effort: S)_
- [ ] **Bench: CTE vs iterative BFS crossover** — at which depth does the
      MySQL CTE beat the iterative fallback? Feeds the planner's cost model.
      _(Effort: S)_
- [ ] **Bench: MariaDB dual-key sort cost** — measure the
      `CAST(... AS DECIMAL)` overhead vs MySQL's single JSON key.
      _(Effort: S)_
- [ ] **Run `nix run .#integration-mysql-nspawn`** (needs root) — real-env
      verification incl. `stack/mysql`; live verification so far used docker
      probes only.
      _(Effort: M)_

---

## cqrs-lint

> Per-module coaching migration complete: all adoption (F003-F029) and
> resilience (B029-B031) rules evaluate per-module. 86 per-module profiles
> verified by `integration_multimodule_test.go`. F001/F002/F005/F014 remain
> workspace-global by design (low leakage risk). F030 (deprecated transport
> imports) shipped 2026-08-14 — 203 rules total.

- [ ] **Audit `.golangci.yml` exclusion blocks** — `system/` (20 linters
      disabled), `cmd/cqrs-lint/` (17), `metaengine/` (24) have the broadest
      exclusions. Track which can be removed after migrations complete.
      _(Effort: M)_

---

## WithActor Hardening

> `event/command/query.WithActor` + `Tracing.ActorID` shipped and released
> (metadata/v4.4.0, event/v4.6.0, command/v4.6.0, query/v4.5.0 — 2026-08-13).
> id/v4.4.0 contains `actor_id.go` (verified in tag).

- [ ] **Test-coverage gaps** — golden JSON for full `event.Event`/`command.
      BasicCommand` with ActorID; watermill wire-format preservation; CBOR
      roundtrip (events default to CBOR); SQL `MarshalMetadata` scan path;
      pebble/bbolt encode/decode; e2e decider (command→events) and projection
      (events→read) propagation; `TestQuery_AllMetadata`; json/v1 fallback
      `omitempty` behavior.
      _(Effort: M)_
- [ ] **Ecosystem propagation checks** — scenario DSL actor support;
      `scheduling/`, `deriver/`, `commandlifecycle/` ActorID propagation;
      ActorID-from-context middleware; `id.ActorID.Validate()`.
      _(Effort: M)_

---

## Code Quality / Infrastructure

- [ ] **CHANGELOG honesty gate** — lint that every `pkg.Symbol` identifier
      cited in CHANGELOG Added/Changed entries exists in the api-stability
      golden. Kills the reverted-work fiction class mechanically (the
      2026-08-10/11 tombstone entries described `e406edcfb`, which was
      reverted by `a6613ef0d` before any tag — CHANGELOG never recorded the
      reversion; corrected 2026-08-16).
      _(Effort: S)_
- [ ] **api-stability: fail loudly on parse-skip** — the checker prints `skip
      <module>:` and proceeds when a file is unparseable, so a corrupted
      module looks identical to a legitimately-removed one in the golden
      (a silently-shrinking golden is the corruption tell, 07:12 report §e.2).
      Cheapest corruption tripwire available.
      _(Effort: XS)_
- [ ] **BuildFlow pre-commit: `gofmt -l` syntax gate on staged `.go` files** —
      concurrent-session mid-write corruption entered the index twice on
      2026-08-16 (`func (w *workor)`, `fojection.`); a 1s syntax check on
      staged files blocks the class.
      _(Effort: XS)_
- [ ] **Pre-gate load-sweep script** — run timing-assertion tests
      (`-run 'Latency|Timer|Deadline'`) under a CPU soaker before `#verify`;
      the 12:39 session burned two full gate cycles (~20 min each)
      discovering load-sensitive flakes one at a time.
      _(Effort: S)_
- [ ] **Guard against local-path `replace` directives** — cqrs-lint rule,
      buildflow check, or both: reject `replace … => /home/…` (broke CI
      Release on every push until `ceb88738b`; dev-against-siblings belongs
      in `go.work` `use`, never in a published go.mod). Approach is a user
      question (11:00 report §g Q2).
      _(Effort: S)_
- [ ] **Clean up registered git worktrees** — `/tmp/cqrs-tagwt` (tag chain,
      commits `092b5e8a8`/`4907b6afc` stranded there), `/home/lars/projects/
      wt-head`, `/tmp/gcl-verify`, `go-cqrs-lite-pin` still registered.
      After the stranded-commit cherry-pick lands.
      _(Effort: XS)_
- [ ] **Infrastructure polish (nix apps + shared helpers)** — add
      `#check-lint-config`, `#verify-ci` (mirror GH Actions GOWORK=off
      per-module), wire `#sweep` to pre-commit/cron, consolidate engine
      `register.go` boilerplate (7 modules), add missing devShell tools
      (dprint, go-licenses, vulnix — their absence forces `--no-verify`).
      _(Effort: M)_
- [ ] **Fix `nix fmt` vs golangci-gci import-grouping conflict at the tooling
      level** — the class re-broke 95+ files once; it WILL recur on every
      future import addition. Make the treefmt formatter config aware of gci
      section rules (or vice versa).
      _(Effort: S)_
- [ ] **Enforce the heap-measurement contract mechanically** — tests calling
      `runtime.ReadMemStats` must never `t.Parallel()` (13 files fixed
      2026-08-14; repo-wide audit came back clean, but nothing prevents
      reintroduction). cqrs-lint rule or check-script grep.
      _(Effort: S)_
- [ ] **Audit benchkit's remaining wall-clock assertions** —
      `raceEnabled` branch at `benchkit_test.go:821` may share the
      load-sensitivity mis-model already fixed for `DurationAborts`/
      `CancelledContext` (flat 30s).
      _(Effort: S)_
- [ ] **Wire broker tests into CI** — `TestRedisStreamRoundtrip` passes only
      via manual `ephemeral-redis.sh`; add a `#integration-redis` nix app
      (mirrors `#integration-pg`). Then extend to broker edges the gochannel
      tests can't catch: redelivery duplicates, consumer-group rebalance,
      message size limits.
      _(Effort: M)_
- [ ] **Doc-check 0-warning CI tripwire** — warnings are at 0 since
      2026-08-15; add a regression guard so warning spam can't creep back.
      _(Effort: XS)_
- [ ] **Skill docs: capability diagnostics recipe** — `recipes.md` section on
      `CapabilityAudit`/Doctor's `--- Capability ---` section
      (consumer-facing declared-vs-implemented diagnostics), and a
      `modules.md` metaengine row note. Shipped 2026-08-16, undocumented.
      _(Effort: XS)_
- [ ] **Duplication-baseline hygiene** — add `//art-dupl:accept` directives
      at the 9 intentional clone sites; dirty-tree guard for baseline
      re-pins; re-pin at next clean tree (current pin includes in-flight
      foreign code).
      _(Effort: S)_ 2026-08-16: wave-4 triage added 3 more directives
      (2 false-sharing bench mirrors, 1 cross-module test read-file idiom)
      and consolidated 2 groups (metaengine `enginesSnapshot`, event
      `seekableReadFrom`) — gate green, baseline untouched (annotations
      suppress live). The 9 legacy sites + dirty-tree guard remain.
- [ ] **check-coverage.sh hardening** — meta-test asserting every EXPECTED
      key resolves to a real module dir (codec-dangle class); make
      `--update` auto-stamp the "verified" date.
      _(Effort: S)_
- [ ] **duckdbengine suite split** — 76-91s observed under the (now 8m)
      ceiling; worth splitting the soak into its own budget.
      _(Effort: S)_
- [ ] **macOS verification of ephemeral PG** — `scripts/ephemeral-pg.sh` claims
      cross-platform but was never tested on Darwin.
      _(Effort: M)_
- [ ] **Delete junk from repo root** — `t/`, `result/` (16MB root-owned),
      `reports/coverage.out` (empty), `reports/jscpd-report.json`; drop the
      orphaned `stash@{0}` (WIP @ `e87be3143`, pre-recovery leftovers).
      _(Effort: XS)_
- [ ] **Per-module CHANGELOGs** — 6 of ~82 modules have one. Decide policy
      (root CHANGELOG only vs per-module) and either write them or document
      the decision.
      _(Effort: L)_

---

## Correctness Defect Sweep (brutal-review backlog)

> From `docs/reviews/2026-08-14_14-25_brutal-self-review.md`. All other
> verified findings shipped 2026-08-15/16 — see CHANGELOG. These remain:

- [x] **Correctness-sweep leftovers** — `kv.Cache` shared `*T`;
      TypedQueryStore hardcoded JSON decode (`query/typed.go`); ghost
      `event.ErrBinaryNotFound` (document or delete).
      _(Effort: M)_ — done 2026-08-16: kv.Cache copy-isolates values (Get
      returns a deep copy, Set caches a private copy); all four blind stores
      decode non-envelope data via the configured codec + JSON↔CBOR
      cross-retry (ADR-0050 addendum); ErrBinaryNotFound deleted (orphan of
      the removed event/blob.go helpers).

---

## v5 Unification (Phase 8: Deletion + Cut)

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Phases 1-7 done (type foundation, dead-code removal, self-registration,
> backend porting, record-typed folds, auto-projection, layout planning,
> universal ADT coverage). Phase 8 is the breaking cut.

- [ ] **Delete `stack.Materialize`** — auto-projection replaces it.
      _(Effort: S)_
- [ ] **Delete `storage.RelationalProjection` + `storage/view` (SQLViewStore)**
      — multi-collection batch atomicity + auto-projection replaces them.
      _(Effort: M)_
- [ ] **Delete `graph.GraphProjection`** — auto-projection + graphadapter
      replaces it.
      _(Effort: S)_
- [ ] **Delete `stack.Bundle` + all 8 stack presets** — `system.System` is the
      only composition root. `stack/` module deleted entirely.
      _(Effort: M)_
- [ ] **Delete `stack.RunProjections`** — `projectionhost.Host` is the only
      projection runner.
      _(Effort: S)_
- [ ] **Delete deprecated compat shells from ADR-0126** — `schema.VersionedStore`
      + `NewVersionedStore`, `signing.Rejecting*` forwarders,
      `encryption.ErrInnerStoreNot*` aliases, `metadata.CustomData`. Internal
      code is already off them (compat tests pin external behavior).
      _(Effort: S)_
- [ ] **Delete `storage/sql.BuildWhereClause`** — deprecated 2026-08-15
      (interpolates column names/operators); `BuildWhereClauseChecked` is
      the validated replacement.
      _(Effort: XS)_
- [ ] **Breaking `record.NewStreamRef` validation** — v4 kept the constructor
      non-breaking and added `StreamRef.Validate()` + `ErrInvalidStreamRef`
      (2026-08-16); `Split()` accepts the empty-streamType form that
      command/query asrecord produces. At v5, change to
      `NewStreamRef(streamType, entityID string) (StreamRef, error)` rejecting
      an empty entityID at construction (empty streamType stays legal) and
      migrate the call sites. Note: `id.NewStreamRef` is a separate,
      unrelated function.
      _(Effort: M)_
- [ ] **Delete `transport/http` + `transport/grpc` modules** (ADR-0127) —
      delivery is `watermill/` + go-sse + cqrs-htmx. `example/taskmanager` is
      migrated (metaengine.ServeSSE on the task_views watcher); cqrs-lint F030
      coaches consumers off the deprecated imports. Remaining steps: tag final
      v4.x patch releases of both modules (see Release section), drop
      the modules from `go.work`/flake `testModules`/api-stability list, then
      delete at the v5 cut.
      _(Effort: M)_
- [ ] **Delete deprecated tombstone metadata API (ADR-0114 completion)** —
      remove `event.DetectTombstone`/`MarkTombstone`/`MarkRebirth`/
      `TombstoneStatus`/`Metadata.Tombstone`; make deletion purely
      event-type-driven (metaengine already is; auto-projection replaces
      `stack.Materialize`, whose `OnTombstone`/`OnRebirth` are the last
      metadata-triggered path; `listing` needs type-based status to replace
      `StatusMiddleware` + the `event.DetectTombstone` call at
      `listing/in_memory.go:155`). The 2026-08-10 attempt (`e406edcfb`) was
      reverted (`a6613ef0d`) before release; docs realigned 2026-08-16.
      Owner decision M20 (2026-08-11): full rename deferred to v5.
      _(Effort: M)_
- [ ] **Write v5 migration guide** — document the path from v4 (stack presets,
      v1 tiers, transport/*, manual RelationalProjection/view reads) to v5
      (`system.System`, auto-projection, watermill/go-sse delivery).
      Before/after examples for each v1 tier, including
      `relational → metaengine` (consumer-pulled).
      _(Effort: L)_
- [ ] **v5 decision: kvstore SA1019 exclusion** — keep the scoped
      `.golangci.yml` exclusion (`(middleware|idempotency)/.*_test\.go$`) as
      the permanent answer, or migrate the kvstore test matrices onto the
      go-idempotency contract suite before the cut.
      _(Effort: S)_
- [ ] **Cut v5.0.0** — tag all modules. Update CHANGELOG, README, SKILL.md,
      examples. Run full verify gate.
      _(Effort: M)_

---

## Declined / Rejected (do not re-litigate)

> Full rationale in the linked ADRs/reviews.

- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
- **Rewrite `check-module-layers.sh` as Go NOW** — deferred. The script is stable
  (348 lines). Revisit when complexity grows significantly.
- **Fix LogBackend same-nanosecond collision** — `time.Now().UnixNano()` could
  theoretically collide under extreme concurrency (1-in-a-billion). The
  performance cost of an atomic counter per collection is not worth the
  theoretical correctness gain. Accepted tradeoff: a counter may be off by 1.
- **Migrate F001/F002/F005/F014 to per-module coaching** — these (event count,
  schema versioning, catalog) are workspace-global by design; low leakage risk.
- **`System.WaitReady(ctx)` API** — declined in the file-renamer TOCTOU review:
  the catch-up drain closes the race; a readiness API would be a false
  guarantee. See `docs/feedback/reviewed/2026-08-13_file-renamer_drain-live-toctou-race-review.md`.
- **file-renamer circuitbreaker/dlq modules** — rejected; failsafe-go + the
  FAQ pointer cover it. See
  `docs/feedback/reviewed/2026-08-13_file-renamer_extract-circuitbreaker-and-dlq-review.md`.
