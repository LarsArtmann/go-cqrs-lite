# 2026-11-06 18:15 — Post-sweep Pareto plan EXECUTED: verify green, two release waves (10 tags), all five consumers adopted

Session: ~12:48–18:15 CET (continuation of the M01–M26 arc; plan
`docs/planning/2026-11-05_01-48_SUPERB-post-sweep-release-and-adoption-pareto-plan.md`).
Supersedes: 06:39 / 09:46 / 18:10 / 19:08 / 21:45 / 23-52 (11-05) and the
sibling 10:54 issue-sweep report for execution state.

## Headline

The parked plan is DONE. `nix run .#verify` is GREEN (run23, 16:02, full
build+vet+test+race+lint+doc legs on the exact tree that shipped). Two
dependency-ordered release waves cut **10 tags** (7 + 3). All five consumer
repos adopted the wave. The three parked owner questions resolved themselves
through evidence or machine state (recorded below).

## M04 — the verify arc (five fast-fail classes, all fixed)

The night's watcher (launch-16) died silently ~05:53 (no rc; trail stops at
iter 2010 — cause unknown, machine-level). Re-armed launches 17→23; the
window opened repeatedly (sum 12–15) and each verify exposed the NEXT latent
defect until run23 went green:

| Run | Fast-fail | Root cause | Fix |
| --- | --- | --- | --- |
| 17 | race ×40 in metaengine | ONE test-only race: `sse_ordering_test.go` polled `rec.Body` while the ServeSSE goroutine wrote it | mutex-guarded write snapshot in `headerProbingRecorder`; full metaengine `-race` suite green (155s) |
| 18 | lint ×2 | nlreturn in my new `v5_mode_test.go`; mnd magic-30 in `systemscenario/scenario.go` (overnight landing) | blank line + `raceAwaitTimeout` const |
| 19 | build: 5 stale `*_templ.go` | overnight `.templ` edits never regenerated | `templ generate` in docserver cwd (the `./...` form from catalog/ produces different output) |
| 20 | build: templ API undefined | **generator-pin vs library-pin split brain**: the sweep bumped `a-h/templ` lib to v0.3.1100 (which REMOVED `JoinURLErrs`/`ResolveAttributeValue`/`RenderAttributes`) while the pinned nixpkgs generator (labeled v0.3.1020, emitting v0.3.1070-era API) still emits calls to them | reverted templ + templ-components family to coherent pre-sweep pins (v0.3.1070 + v1.26.0); rowed the durable fix |
| 21 | build: workspace lift | five tool modules carried stuck `templ v0.3.1100 // indirect` self-requires | `go mod edit -droprequire` + tidy converged ALL to v0.3.1070 |
| 22 | stale `docs-ui.css` | generated CSS behind the downgraded templ-components | `nix run .#build-docserver-css` |

Before run17 armed, the preflight caught three MORE classes: the ADR-index
split brain (fixed 00:10 with a mutation-tested gate extension to
`check-adr-numbering.sh`), an **81-module go.sum strip** (missing `/go.mod`
hash lines; `TestEveryModuleGoSumIsTidy` now in the preflight; mass-tidied
under nix go1.27.3), and a latent ADR-0164 H1-vs-index title mismatch.

## M05/M06 — push + CI

Origin had auto-caught-up overnight; the session's pushes (fc6c23e14 →
3dca1081e) ride CI runs that surfaced and fixed two CI-only legs: shfmt
version skew (CI pins nixpkgs shfmt 3.14.1 — the workflow's own comment
documents the 3.13.1 disagreement; probe-proxy-tags.sh reformatted) and
test-runtime floor breaches by DB-backed suites that SKIP locally but RUN on
runners (pgengine 45.8s vs a 0.068s baseline row) — CI floor widened 30→60s
with the class documented in ci.yml. Fleet census re-run (53 submodules
required; evidence JSON committed).

## M07 — two waves, ten tags, four+three invocations

Wave 1 (post-sweep issue wave, 7 tags): A) id v4.7.5, kv v4.4.0, catalog
v4.13.0, cqrs-lint v4.21.0 → B) metaengine v4.21.0 → C) sqliteengine v4.7.0
(after bumping its metaengine pin — tidy never upgrades requires; the
run-C guard failure was the same-batch lesson re-learned) → D)
systemscenario v4.3.0. All smoked (proxy serves + cqrs-lint installs+runs)
and `go get`-verified from a clean module.

Wave 2 (MaxOpenConns plumbing, 3 tags): M14 exposed a REAL GAP —
`sqliteengine.WithMaxOpenConns` was constructor-only; `system.New`
deployments could not reach it. Built the plumbing (DriverConfig.MaxOpenConns
→ sqlite driver factory applies it only when raised; system EngineConfig
`max_open_conns` koanf field), pinned by a driver-factory DBStats test, then
released metaengine v4.22.0 → sqliteengine v4.7.1 + system v4.17.0.
versions.json + README manifest refreshed after each wave; CHANGELOG cut
per wave.

## M19-tail — #114 CLOSED

Closed by the 10:54 sibling session at 09:53Z with the evidence trail
(ADR-0164, recipes §2.50). No further action.

## M11–M15 — consumers

- **M13 webphone** (quiet 4h, clean): the flagship #114 adoption. dial_windows
  + dial_overrides declare `WithCapPerGroup("owner", 500)` — the 2026-11-04
  T07 ruling's reopen trigger fired exactly as written; both hand-rolled
  expunge loops deleted; views carry typed Key fields for cap addressing
  (legacy rows without the field are never wrongly trimmed — zero-key
  addressing misses). call_transcripts adopts the per-OWNER bound
  declaratively; the per-CALL floor (1500) stays a loop (metaengine declares
  one group per collection; nested owner+call grouping rowed upstream). All
  17 429 responses type Retry-After (`WithResponseHeader`); /api/transcribe
  declares its multipart body (`WithRequestBodyContent`); the empty-host
  `WithServer` hack dropped (catalog v4.13 omits servers natively);
  WithTypedParam honestly SKIPPED (no non-string param exists to type).
  Full suite green incl. their error-registry gate (regenerated).
- **M11 cqrs-htmx** (quiet 2.5h, clean): the 182-line hand-rolled otter
  view-cache adapter DELETED — kv v4.4.0's `NewCacheForViewStore` fronts any
  ViewStore; the seam is a thin option translation; all seven ViewCache
  behavior pins pass against the upstream cache. Workspace build + full
  suite green.
- **M12 + M14 crm/Kith** (stale 6h + 4 dirty files from an abandoned session
  — takeover ruled safe: disjoint files): `scorecard.modules` pins graph +
  projectionhost as USED (staying UNWAIVED — a waiver is a refusal, these
  ARE adopted); the stale prometheus waiver REMOVED (v4.21.0 hard-errors
  waive-irrelevant); scorecard renders green under v4.21.0. `MaxOpenConns: 8`
  wired at the engine config with the read-storm rationale. Pins: system
  v4.17.0, metaengine v4.22.0, sqliteengine v4.7.1. App suite green. The
  read-storm re-measure is rowed (needs the deployment running).
- **M15 BuildFlow** (quiet 2h): cqrs-lint v4.18/v4.20 → v4.21.0 across root +
  tools + execution modules (the third surfaced via test-compile). io-surface
  joined the SDK provider set: tool-name constant, fan-out list, edge
  snapshot 391→400, ProviderRegistered test, docs counts 126→127 (115+12).
  vendorHash re-pinned via `buildflow -s nix-hash-fix --fix`. THREE
  pre-existing BuildFlow bugs fixed to unblock the gates: RepoRoot resolved
  into vendor/ under -mod=vendor (all audit tests judged the vendored
  subtree); ContentCache.misses++ raced under parallel doctor checks;
  wrapcheck (enabled 11-02) flagged every errorfamily return — package
  exempted as the wrapping boundary. F032 false-positive on the toolspec
  integration import suppressed at site with a reason (upstream detector fix
  rowed). Remaining BuildFlow findings-gate red: 3 pre-existing brandid-lint
  BD003 strong-ID sites (their own adoption debt, rowed for their sessions).

## Owner questions — resolved by evidence

1. **Re-verify policy**: answered by machine state twice — at 12:48 the load
   guard would have allowed a direct run but the preflight found reds; by
   15:00 the watcher was the only mechanism (load1 19.8 ≥ 16). The watcher +
   active-polling hybrid won.
2. **cqrs-htmx/crm takeover**: cqrs-htmx went quiet 2.5h before M11 (safe);
   crm's 4 dirty files stayed untouched (disjoint sites), M12/M14 proceeded.
3. **Backlog wave**: kept as its own rowed wave (TODO_LIST) — not folded.

## What this session left behind (rows in TODO_LIST)

Split-brain gate extension (DONE), preflight ritual (codified, incl. the
templ + tidy legs), erraudit pin-bump row (4 manual taxonomy rows retire
with it), templ pin split-brain durable fix, F032 detector fix (toolspec
imports are integration, not adoption), nested per-group caps (transcripts
per-call floor), crm read-storm re-measure, BuildFlow BD003 debt, MySQL QEMU
flake watch (this cycle's failure was a VM TCP reset, not
ConcurrentClaimersDisjoint).

— Receipts: verify `build/verify-run23.log` (rc=0); release logs
`build/release-logs/batch-20261106-16*.log` ×5; tags `git tag -l --sort=-creatordate | head -10`;
webphone c95c0cf3 + daemon commits; cqrs-htmx 63c91b26; BuildFlow daemon
commits through 51d39fffe; crm working tree (daemon-absorbed).
