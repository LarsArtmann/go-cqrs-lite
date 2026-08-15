# Status Report: Transport Deprecation Close-Out (Loose Ends + Broker Roundtrips)

**Date:** 2026-08-14 18:25 · **Session:** close-out of the 5 loose ends from
`2026-08-14_16-00_transport-deprecation-review-and-lint-rewrite.md` (which is
now annotated with this session's resolutions).
**Scope:** all 5 known loose ends + verification gate. No unrelated research.

---

## Verdict up front

All five loose ends are closed and verified. Two extra real bugs were found
and fixed on the way (C015 void-Close false positive; broken test-helper
FileSet). The final gate run surfaced two more blockers — one self-inflicted
(depguard, fixed), one inherited (unsynchronized test helper in system,
fixed) — plus a pre-existing api-surface drift (regenerated) — after which
every leg my changes touch is GREEN. The one remaining red is the documented
pre-existing `check-arch` catalog-coverage gap that fails on master (TODO_LIST
owns it; ~47 modules lack LAYER/DEP_BUDGET entries — unrelated to this
session's changes).

---

## a) FULLY DONE

1. **taskmanager migrated off `transport/http`** — `/events` now serves
   `metaengine.ServeSSE` over a `Watcher[TaskView]` on the `task_views`
   collection with `WithReplay(256)` (Last-Event-ID reconnection) and a 30s
   heartbeat. `setup.go` wires the watcher (closed in `Stop()`); `http.go`
   routes `GET /events`. `transport/http` dropped from `go.mod` after
   `go mod tidy`. Decision rationale: the example already runs metaengine
   projections → watcher path adds zero deps and ships replay for free.
2. **SSE integration test (new)** — `example/taskmanager/sse_test.go`:
   subscribes first, dispatches `CreateTaskCmd`, asserts the projected
   `TaskView` arrives as a JSON `data:` event. PASS (0.05s). This also proves
   the metaengine watcher fires for auto-projections — previously untested
   end-to-end. Full taskmanager suite green (`go test ./...`, workspace mode).
3. **cqrs-lint F030 (new rule)** — `deprecated-transport-import`, warning /
   high confidence: flags any deprecated `transport/http|grpc` import per
   module scope with the ADR-0127 migration path (go-sse / metaengine.ServeSSE
   / watermill bridge / cqrs-htmx, plus ADR pointer). 4 table tests (http,
   grpc, both=2 findings, sanctioned paths=0). Registered in `register.go`,
   catalog entry in `catalog_extra.go`, `meta_test` 202→203 detectors,
   README rule table + counts updated. Companion helper `singleWarningFinding`
   added (36 precedent warning rules).
4. **F012 + catalog_extra audit** — F012 body is deriver-only (clean).
   `catalog_extra.go` E009 + F013 descriptions re-aligned with ADR-0127
   (no more "no HTTP/gRPC transport" coaching language).
5. **watermill/README.md** — canonical-delivery-path banner (ADR-0127 link) +
   new "External Brokers" section: working `WithBackend`/`WithCommandBackend`
   Redis Streams recipe, ephemeral-script run command, Kafka/RabbitMQ note,
   and an honest NATS JetStream status paragraph.
6. **Broker corpse tests replaced** — `watermill/broker_integration_test.go`
   rewritten as a REAL `TestRedisStreamRoundtrip`: EventBus + CommandBus over
   Redis Streams via the official `watermill-redisstream` plugin, env-gated on
   `REDIS_URL`. **PASS against a real broker** (ephemeral-redis.sh, 0.51s) —
   event/command type + payload verified through the broker. The NATS corpse
   stub (unconditional skip) is deleted with documented reason:
   `watermill-nats` is deprecated NATS-Streaming tech built against watermill
   v1.2-rc — no maintained JetStream adapter exists. Deps:
   `watermill-redisstream v1.4.5` + `go-redis/v9 v9.12.1` (test-only usage).
7. **Bonus fix 1 — C015 false positive** (surfaced by the taskmanager golden
   profile): C015 fired on `taskWatcher.Close()` which returns nothing.
   Root-cause fix in the rule: `closeReturnsValue()` consults
   `types.Info` and skips void-returning `Close()` calls (assume-error when
   no type info). Regression test via `BuildContextWithTypes` passes.
8. **Bonus fix 2 — test-helper FileSet bug** (surfaced by that test failing
   with 0 findings): `analyzer.BuildContextWithTypes` built a fresh empty
   `token.FileSet` while ASTs belong to `pkg.Fset` → every position resolved
   to zero → finding builders silently dropped results. Fixed to reuse
   `pkgs[0].Fset`. All prior `BuildContextWithTypes` users (C027 regression
   tests) were asserting on findings that could never fire — they now run
   against honest positions.
9. **Bookkeeping** — CHANGELOG `[Unreleased]`: 5 new bullets (taskmanager,
   F030, C015, broker roundtrip, NATS honesty). TODO_LIST v5 Phase 8 item
   rewritten (taskmanager done; remaining: final v4.x tags → de-list → delete
   at v5). 16:00 status report annotated inline (struck resolved items with
   evidence) + Resolution appendix. cqrs-lint README counts/tables updated.
10. **Verification (partial, pre-final-gate)** — cqrs-lint suite 17/17 packages
    ok; taskmanager suite ok; `go vet` ok; api-stability 4132 ok (no drift);
    doc-check 797 refs ok; gofmt clean.

## b) PARTIALLY DONE

1. **`verify-fast` final gate** — RED, but not on anything this session
   touched. Sequence of the four runs: (1) failed at lint (my depguard entry
   shadowed the `watermill` prefix — `'-'` sorts before `/` in depguard's
   sorted-prefix matching; adding an allow entry DENIED 17 more imports;
   isolated empirically with 3 config variants; the entry was unnecessary —
   prefix-matched already; fixed + commented). (2) failed at api-stability:
   parallel session's new `system.LoadStream` export shipped without golden
   regen — regenerated (4132→4133), mechanical. (3) failed at the race leg:
   23 `system` tests raced in `recordingCheckpointStore` (parallel session's
   committed test helper; unsynchronized map/counter written from
   projectionhost worker goroutines) — fixed with a mutex + race-safe
   accessors; `system` now passes `-short -race`. (4) final run: build, vet,
   test-short, race-short, lint ALL GREEN; fails only at `check-arch` on the
   pre-existing catalog-coverage gaps documented in TODO_LIST (fails on
   master at `6aaca6b0e`; ~47 modules missing LAYER/DEP_BUDGET — not
   researched/fixed here per scope).

## c) NOT STARTED (from this session's own findings)

1. GOWORK=off (published-versions) build of taskmanager fails: `unknown driver
   "sqlite"` — published `sqliteengine v4.0.1` predates the parallel session's
   system/engine changes; workspace mode is the canonical green path. Needs
   re-tags (already P0 item 1 in the 16:00 report).
2. `F030` does not yet coach the OTHER deprecated modules (codec, retry
   re-export shells) — deliberately deferred to the deprecation-story
   unification item.
3. Redis roundtrip not wired into any nix app/CI job (manual script run only).
4. cqrs-lint self-lint: the repo's own lint run now reports F030-style
   coaching only for real consumers; nothing here, but `library_self_lint_test`
   wasn't re-examined for F030 interplay.

## d) TOTALLY FUCKED UP

1. **I shipped a red gate and nearly reported green.** I edited
   `.golangci.yml` (adding broker deps to the depguard allow list), then ran
   the full cqrs-lint suite, doc-check, api-stability — everything green —
   and only the last `verify-fast` run caught that my config edit broke
   watermill linting (1→18 issues). Partial gates gave false confidence; the
   lint leg was never run standalone after the config change.
2. **The depguard bug was self-inflicted and subtle:** adding
   `github.com/ThreeDotsLabs/watermill-redisstream` to the allow list DENIED
   `github.com/ThreeDotsLabs/watermill/message` everywhere (sorted-prefix
   shadowing: `-` sorts before `/`). Counterintuitive — adding an allow entry
   reduced allowance. It took 3 empirical config variants to isolate because
   reasoning alone couldn't explain it. The entry was also unnecessary
   (prefix-matched by `watermill` already).
3. **I trusted a test helper that had never been able to fail** —
   `BuildContextWithTypes` zeroed all positions, so every finding-based
   assertion built on it was vacuous. My new C015 test exposed it by accident.
   (Fixed, but: which other past fixes were "verified" through that helper?)
4. **Three gate iterations burned on inherited breakage** — the session
   claimed "all gates green so far" twice before the full gate was actually
   run end-to-end. The 16:00 report's `verify-fast exit 0` claim was stale:
   today's master has the check-arch catalog gap failing. Lesson (re-learned):
   a full gate is only proven by running the FULL gate, and GREEN claims
   expire the moment the tree moves.

## e) WHAT WE SHOULD IMPROVE

1. **Config edits need the narrowest immediate gate, not the broadest
   eventual one.** After touching `.golangci.yml`, run the affected module's
   lint immediately (seconds), not the 15-min full gate (where failure costs
   a full rerun).
2. **Don't add allow-list entries without checking they're needed** — depguard
   prefixes cover subpackages and sibling names. A meta-test could assert no
   allow entry is a prefix-shadow of another.
3. **Vacuous-test audit**: grep for tests asserting 0 findings via
   `BuildContextWithTypes` written before the FileSet fix — some may have been
   hiding real positives.
4. **Env-gated broker tests deserve CI wiring** — a passing-by-manual-script
   test is one `REDIS_URL` export away from being a real regression gate
   (`nix run .#integration-*` pattern exists).
5. **The daemon committed mid-session again** (`d8c73be0a` includes my
   in-flight work + parallel-session files). Benign this time, but
   attribution in reports must rely on content, not commits.

## f) NEXT — up to 50, Pareto order

**P0 — confirm this session's state:**
~~1. Confirm `verify-fast` exit 0 after the depguard fix (run was in flight).~~ done at 5f2198189 (full verify GREEN later that night; three GREENs since)
~~2. Commit the close-out delta (daemon already has partial: d8c73be0a).~~ done - daemon-committed as d8c73be0a + fe017c06a
~~3. Re-run cqrs-lint suite once more after config fix (config affects lint~~ done - cqrs-lint 17/17; lint 76/76 modules clean in every verify since 444be10a7
   rules globally — low risk, cheap check).

**P1 — inherited P0s (unchanged, from 16:00 report):**
4. Fix release chain: re-tag `id` v4.4.0 (missing `actor_id.go`), re-tag <- **NOT-DO - premise stale: id/v4.4.0 contains actor_id.go (verified via git tag --contains). Downstream re-tags track in TODO_LIST 'Release / Tagging'.**
   dependents, bump 66 go.mods; unblocks GOWORK=off taskmanager build (c.1).
5. Engine capability conformance test (plan-time Supports-vs-implemented). <- OPEN. TODO_LIST 'Metaengine' (Engine capability conformance test)
~~6. Dgraph CounterBackend DQL colon bug + JournalReadFrom off-by-one.~~ done at 5127039da (counter rework) + 7c0a62c98 (JournalReadFrom positional)
7. ADR-0114 tombstone/DeletePolicy reconciliation. <- OPEN. TODO_LIST 'Docs Honesty' (ADR-0114 tombstone reconciliation)
8. Remove taskmanager local `replace` directive. <- NOT-DO - only replace in taskmanager/go.mod is the intentional go-must sibling dev-replace

**P2 — this session's follow-ups:**
9. Wire Redis roundtrip into CI/nix (ephemeral script + test, like <- OPEN. TODO_LIST 'Code Quality' (Wire broker tests into CI - #integration-redis)
   integration-pg).
10. Depguard prefix-shadow meta-test (allow-list hygiene).
11. Vacuous-test audit for pre-fix `BuildContextWithTypes` users.
12. F030 extension: codec/retry deprecated re-export coaching (after <- NOT-DO - codec/retry shells deleted at 5127039da (ADR-0128); nothing left to coach on
    deprecation-policy unification).
13. Teach E005 `system.RegisterCommand`; regenerate taskmanager golden <- OPEN. TODO_LIST 'cqrs-lint' (E005 + taskmanager golden item)
    (kills 10 enshrined false positives).
14. Final v4.x tags of transport/http + transport/grpc with deprecation <- OPEN. TODO_LIST 'Release / Tagging' (transport v4.x patches) + v5 section
    notices (prereq for v5 deletion, per TODO_LIST).
15. v5 migration guide (transport section now has real recipes to link). <- OPEN. TODO_LIST 'v5 Unification Phase 8' (migration guide)

**P3 — systemic (from 16:00 report, still open):**
16. Graceful degradation fallbacks for SQL engines (Set/Log/Multimap). <- OPEN. ROADMAP (Vector/Search/Spatial backends + graceful-degradation theme)
17. Dialect-DDL ↔ migrations/*.sql drift test.
18. sqliteengine/tursoengine self-opened *sql.DB Close() leak. <- OPEN. TODO_LIST 'Correctness Defect Sweep' (Resource leaks)
19. Planner cost model fixes (branching^depth, volume default, selectivity). <- OPEN. TODO_LIST 'Correctness Defect Sweep' (Planner cost model)
20. FilterOp/column allowlists; ORDER BY quoting; DSN leaks in turso errors. <- OPEN. TODO_LIST 'Correctness Defect Sweep' (SQL injection surface)
21. Core defects: singleflight leader-ctx, per-handler command middleware, <- OPEN. TODO_LIST 'Correctness Defect Sweep' (Core defects)
    query audit fake RequestIDs, Pagination.Offset underflow, kv.Cache shared
    *T, TypedQueryStore JSON decode, ghost event.ErrBinaryNotFound.
~~22. Deprecation-story unification (codec/retry/idempotency/flightrecorder).~~ done - resolved by ADR-0128 deletion; policy documented (5127039da)
23. Bench consolidation to benchkit+cqrs-bench; CI regression breach-fail. <- OPEN. TODO_LIST 'Code Quality' (One bench system)
24. storage/backuptest wiring or deletion; bbolt backup tests. <- OPEN. TODO_LIST 'Code Quality' (storage/backuptest: wire or delete)
25. MySQL/Dgraph engine tests: stop silent CI skip.
26. check-duplication gate fails loudly when art-dupl missing.
~~27. check-arch catalog coverage (94 gaps) — noted as failing on master pre-session.~~ done at 8c384f0f5 (layer-script key convention repaired; check-arch green inside #verify)
28. Transactional outbox (ADR-0016, zero code — biggest gap). <- OPEN. ROADMAP raw ideas (Transactional outbox - ADR-0016 designed)
29. Per-module CHANGELOGs (6 of ~86). <- OPEN. TODO_LIST 'Code Quality' (Per-module CHANGELOGs)
30. metadata.ActorID omitzero; 3-metadata-model split-brain reduction. <- OPEN. omitzero shipped in the 16-44 hygiene pass; 3-model split-brain reduction OPEN - TODO_LIST 'WithActor Hardening' (rides ADR-0111)
31. record.NewStreamRef validation. <- OPEN. TODO_LIST 'Correctness Defect Sweep' (Strong types)
32. id global-mutex sharded entropy. <- OPEN. TODO_LIST 'Correctness Defect Sweep' (Strong types - sharded entropy)
33. SECURITY.md v3 table; govulncheck swallow; iroh fork pin. <- OPEN. TODO_LIST 'Correctness Defect Sweep' (Security hygiene)
34. Docs website; module compatibility matrix.
35. Distributed projection runner; event archival/compaction (designs exist). <- OPEN. ROADMAP raw ideas (Distributed projection runner; Event archival)
36. README feature-table honesty (tombstone headline). <- OPEN. TODO_LIST 'Docs Honesty' (README feature-table honesty)
37. integration/README lists 5 of ~15 suites. <- OPEN. TODO_LIST v5 section (integration/README suite enumeration)
38. Metaengine-quickstart in flake examplePaths/CI (never builds). <- OPEN. TODO_LIST 'Code Quality' (Build example/metaengine-quickstart in CI)
39. Delete junk: `t/`, `result/` (16MB root-owned), empty coverage.out. <- OPEN. TODO_LIST 'Code Quality' (Delete junk from repo root)
40. Layout-planning doc corrections; ReplanLayout→Store.Replan convergence. <- OPEN. TODO_LIST 'Metaengine' (Layout Planning doc + Converge ReplanLayout into Store.Replan)
41. DuckDB calibration tie-break; row-layout calibration from real benches. <- OPEN. TODO_LIST 'Metaengine' (calibration benchmarks)
42. Two-live-backend integration test; per-fold mutex (after soak). <- OPEN. TODO_LIST 'Metaengine' (two real backends); per-fold mutex in flight - concurrent session writing fold_locks.go now

## g) QUESTIONS (cannot determine myself)

1. **Redis roundtrip in CI?** Wire `ephemeral-redis.sh` + `TestRedisStreamRoundtrip`
   into a nix app/CI job now (mirrors `#integration-pg`), or keep it
   manual-only until the release chain is fixed (P0s may re-tag everything)?
2. **F030 severity gate?** It's warning-severity and warnings don't affect
   cqrs-lint's exit code today — sufficient for coaching, or should deprecated
   imports be a hard CI failure option (`--fail-on-deprecated` flag)?
3. **Commit granularity for the close-out delta?** The daemon already
   committed a mixed batch (`d8c73be0a`, includes parallel-session storage
   fixes). Want a single follow-up commit of the remaining delta, or
   content-split (lint-fix / broker-test / depguard-fix)?

---

## h) Verification state at write time (FINAL)

| Gate | Status |
| --- | --- |
| taskmanager suite (workspace) | ✅ ok |
| SSE integration test (real server) | ✅ pass |
| cqrs-lint suite (17 pkgs) | ✅ ok |
| Redis roundtrip vs real broker | ✅ pass |
| api-stability (4133, regen for parallel session's LoadStream) | ✅ ok |
| doc-check (797 refs) | ✅ ok |
| gofmt / vet | ✅ clean |
| system `-short -race` (after mutex fix) | ✅ ok |
| `verify-fast` legs: build/vet/test/race/lint | ✅ all pass |
| `verify-fast` `check-arch` leg | ❌ pre-existing catalog-coverage gap (TODO_LIST-owned, fails on master) |

Working tree at report time (uncommitted, all this session): `.golangci.yml`
(depguard fix), `docs/api_surface.txt` (4133 regen),
`system/system_hardening_test.go` (race fix), this report. Everything else
was daemon-committed (`d8c73be0a`, `fe017c06a`).

*Report ends. Waiting for instructions.*


## Resolution (2026-08-15, docs-health pass)

36 of 42 items carry verdicts. The P0 confirm-state items closed via the
daemon commits (`d8c73be0a`, `fe017c06a`) and the first fully-green verify
(`5f2198189`); item 4's re-tag premise was stale (id/v4.4.0 contains
actor_id.go); check-arch 94-gap closed by the layer-key repair (`8c384f0f5`).
The P3 systemic backlog is routed item-by-item into TODO_LIST "Correctness
Defect Sweep" / "Code Quality" / "Docs Honesty" / "Metaengine" and ROADMAP
ideas. Open-unrouted: 10 (depguard shadow meta-test), 11 (vacuous-test
audit), 17 (DDL drift test), 25 (MySQL/Dgraph CI legs), 26 (art-dupl
fail-loud), 34 (docs website; compat matrix). Stays active.
