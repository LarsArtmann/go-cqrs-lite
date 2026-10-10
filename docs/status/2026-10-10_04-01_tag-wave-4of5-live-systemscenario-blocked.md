# Tag Wave 4/5 Live; systemscenario v4.0.0 Blocked on a Pin (execution status)

> **Date:** 2026-10-10 04:01 CEST · **Session:** Full Execution Mode on the adoption wave
> ([plan](../planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md)); this report
> supersedes [`2026-10-10_03-40_adoption-wave-t01-t13-execution-status.md`](2026-10-10_03-40_adoption-wave-t01-t13-execution-status.md)
> for T14 progress. **State at interrupt:** FOUR of five tags are LIVE on the module proxy;
> the fifth (systemscenario v4.0.0) failed its standalone-build gate on an unverified pin
> gap and is NOT tagged. No damage; recoverable in minutes.

## a) FULLY DONE (this session, verified)

1. **T01–T13 complete** — see the 03:40 report §a for the full inventory: deadlock evidence
   pack (live stack capture), wave-order discovery, companion baselines + schema-replace
   fixes, harvest, ADR-0154, FEATURES/SKILL rows, `deriver.WithAsyncDispatch` (+4 race-clean
   tests, api golden), fixture flip, and the FULL cqrs-htmx user-train migration with legacy
   deletion (1235→896 lines, 2.2s→0.8s).
2. **T14 — FOUR tags cut, pushed, proxy-served, smoke-passed** (each in its own
   batch-release invocation, dependency order per the manifest):
   | # | Tag                 | Carries                                                                                                     | Proxy       |
   | - | ------------------- | ----------------------------------------------------------------------------------------------------------- | ----------- |
   | 1 | `schema/v4.6.0`     | Event/EventSchema declaration API (concurrent session's work; their ROADMAP row asked for exactly this tag) | ✓ attempt 1 |
   | 2 | `deriver/v4.4.0`    | `WithAsyncDispatch` + `AsyncDispatchErrorHandler` (ADR-0154 D1)                                             | ✓ attempt 1 |
   | 3 | `system/v4.12.0`    | Clock seam + schema-on-every-read-path + adapter serial hardening                                           | ✓ attempt 1 |
   | 4 | `scheduling/v4.7.0` | `WithClock(now func() time.Time)`                                                                           | ✓ attempt 1 |
3. **Untag-policy removal for `deriver`** committed with rationale (tag-on-first-consumer:
   systemscenario's module graph is the consumer; owner-approved via Q1).
4. **system's schema pin bump** (`a30b3f464`): the batch script REFUSED system's first cut
   because its go.mod still pinned schema v4.5.2 (stale-sibling guard worked as designed —
   exactly the command/v4.7.0 failure class it was built for). Bumped to v4.6.0, standalone
   build verified, re-cut clean.
5. **systemscenario go.mod pins bumped** to `system/v4 v4.12.0` (committed by the daemon —
   tree is clean at report time).

## b) PARTIALLY DONE

1. **T14 tag 5/5 — systemscenario v4.0.0**: pins for system upgraded, but the
   standalone-build gate (GOWORK=off, `go test ./...`) STILL FAILS. **Cause not yet
   diagnosed — interrupted here.** Prime suspect (UNVERIFIED): the go.mod still pins
   `scheduling/v4 v4.6.2` while the timer fixture uses `scheduling.WithClock` (v4.7.0+);
   the `go get` invocation upgraded system + a transitive gjson bump but the tail-2 output
   showed no deriver/scheduling line, so the deriver v4.4.0 pin may ALSO be missing. Both
   are one `go get` + tidy + standalone-test + batch-release away.
2. **Side effect to judge:** the systemscenario pin bump dragged `tidwall/gjson`
   v1.19.0 → v1.20.0 (transitive, via tidy). Unintended passenger in the first tag's
   go.sum. Almost certainly harmless (test-graph dep of go-snaps), but it should be a
   CONSCIOUS inclusion, not an accident.

## c) NOT STARTED

T15 (drop companion replaces — rescoped to "4 files + 2 schema lines", see 03:40 §d4),
T16–T27 (hardening, pilots, examples, presets, property pack, chaos+SSE, lint rule,
codemod, fleet rollout, watermill loud-fail, final verify + retro). The post-wave
obligations: CHANGELOG version sections, versions-manifest check, tag-COUNT assert.

## d) TOTALLY FUCKED UP

1. **The first systemscenario pin bump was PARTIAL.** I ran one `go get` with two module
   paths (system + deriver) and eyeballed a truncated tail instead of grepping the go.mod;
   at least one pin (likely two: deriver, scheduling) is still stale. Sloppy exactly where
   the release script's own guard had just taught me the lesson (stale sibling pin =
   build failure). The script caught it again — zero damage, but two invocations in a row
   failed on preventable input state.
2. **Carried from 03:40, still open:** the dirty-tree invocation (fixed by discipline, not
   yet by habit); authored-commit races with the daemon (the systemscenario pin bump
   landed as a `chore:` daemon commit, not my message).

## e) WHAT WE SHOULD IMPROVE

1. **Pin-bump checklist for wave cuts:** before EVERY batch-release invocation, grep the
   module's go.mod for every sibling that gained a tag THIS wave and assert each require
   is ≥ that tag. Mechanical, 10 seconds, would have saved both failed invocations.
2. **Never trust `tail` of a `go get`** — grep the go.mod after dependency surgery.
3. The wave-order manifest proved its worth (zero mis-ordered cuts after it existed);
   future waves should START from a manifest, not grow one during the audit.

## f) Up to 50 things to get done next

**Finish T14 (minutes):**

1. Diagnose the systemscenario standalone build failure (`go build ./...` GOWORK=off, full
   output — do not tail-truncate).
2. `go get scheduling/v4@v4.7.0 deriver/v4@v4.4.0` (+ anything else the diagnosis shows);
   grep go.mod to confirm ALL wave siblings pinned.
3. Decide the gjson v1.20.0 passenger: keep (documented) or pin back (minimal first tag).
4. Commit pins; standalone test green; `batch-release.sh "systemscenario v4.0.0 '…'"`.
5. Push tag; smoke; proxy probe.
6. Remote tag COUNT assert: exactly 5 new tags from this wave on origin.
7. CHANGELOG: five version sections (symbols gate: every cited `pkg.Symbol` exists).
8. `bash scripts/check-versions-manifest.sh --check` — update manifest/markers per output.
9. Re-run systemscenario suite in-workspace (pins must not shift behavior).

**T15 (drop replaces):**
10. cqrs-htmx: remove `systemscenario/v4` replaces (go.work + systemadapter/go.mod);
`go get systemscenario/v4@v4.0.0` + tidy; full suite green.
11. go-appkit: drop the pilot trio (go.work: systemscenario + system + schema lines) +
`cqrs/go.mod` replace; `go get`; tidy; full suites green.
12. Companions: authored commits (single-call pattern).

**T16–T20 (plan fine tasks F16–F20):**
13. F16.1 `ThenCommandsSatisfy` await variant.
14. F16.2 timeout last-error surfacing (query error vs check message) — ALSO cover the
`awaitNotFound` pattern (error-as-success hides the last real error).
15. F16.3–F16.4 tests + README rows.
16. F17.1–F17.4 `WithQuietWindow` + `WithCommandCaptureFilter` (+ tests, docs).
17. F18.1–F18.4 go-appkit layer-2 pilot (integration module via `Adopt` + `testkit.Serve`)
— mind their `integration` charter (GOWORK=off hermetic).
18. F19.1–F19.4 godoc examples (System happy path, saga via WithAsyncDispatch + Await +
ThenCommands, TimeAdvances deadline); `go vet` compile check.
19. F20.1–F20.4 `Memory()` / `SQLite(t)` presets + pilots adopt them.

**T21–T26:**
20. F21.1 fold-vs-read-model invariant property (reuse the task fixture).
21. F21.2 allocs/op bench (`-benchmem` on BenchmarkScenarioBoot) + F21.3 README metrics.
22. F22.1–F22.4 DelayedJournal chaos scenario + ServeSSE assertion helper.
23. F23.1–F23.4 cqrs-lint advisory rule (system-booting test without harness) + catalog
counts test + api-stability regen.
24. F24.1–F24.4 cqrs-upgrade `eventually`→`ThenQuery` suggestion rule.
25. F25.1–F25.4 fleet rollout: example/taskmanager + example/goal-shaped-app.
26. F26.1–F26.4 watermill `ErrReentrantPublish` loud-fail — mechanism per the evidence
pack (the nested publish runs ON the event-loop goroutine, so a goroutine-local
publisher-depth flag works; verify with the repro shape).

**T27 + tail:**
27. `nix run .#verify` (my modules green; core/v5 + metaengine reds itemized as externals).
28. `nix run .#check-md-go` on the new docs (evidence pack, ADR-0154, manifest).
29. Plan addendum: DONE/PARTIAL per section + restate the 5-module wave.
30. Retro into gotchas: "commit before release scripts", "wave = consumers of untagged
API", "grep go.mod after dependency surgery".
31. TODO_LIST residue rows: strike what T14–T26 close.
32. Companion full suites post-T15 + the 14:40 §f32 root-go.mod stray-require check.

## g) Questions I CANNOT figure out myself

1. **gjson v1.20.0 passenger in systemscenario v4.0.0** (dragged by tidy, test-graph
   transitive): ship it (documented ride-along) or pin back to v1.19.0 for a minimal
   first tag? I can verify WHY it bumped, but "minimal first tag vs current graph" is a
   taste call.
2. **Companion release cadence after T15:** once cqrs-htmx/go-appkit drop replaces and
   pin published tags, should THEY cut their own version bumps immediately (dependency-
   only releases), or wait for their next feature cadence? Cross-repo ownership — their
   daemons/sessions normally decide, but the wave is mine to close coherently.
3. **`deriver` untag-removal confirmation** (carried from 03:40 g2): I re-entered deriver
   into release trains per tag-on-first-consumer + your Q1 GO; the ADR-0152 ghost-verdict
   T02 sign-off (kill/keep/absorb) is still pending elsewhere — does the re-entry stand
   as the keep-driver outcome, or should the verdict still be formally recorded?
