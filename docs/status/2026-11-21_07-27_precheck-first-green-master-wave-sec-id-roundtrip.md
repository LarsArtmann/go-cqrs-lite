# 2026-11-21 07:27 — Precheck first-ever green, master rides the v4.16 wave, SEC ID round-trip repair (session report)

Session continued the C-list mop-up (02-27 report §f) under the standing "execute one step at a
time, don't stop" mandate. Executed §f items 1–3 (precheck diagnosis, precheck commit, 8b master)
and opened 8c (SEC). Three fleet repos materially advanced; two full test suites are green for the
first time in their history.

## a) FULLY DONE

1. **§f-1 Precheck diagnosis — root cause found and fixed by DEP-WAVE ALIGNMENT, not a test hack.**
   - `TestPasskeysDisabled_LoginErrorParamBreaksRedirect` fails because it pins loginpage
     **v4.16.0-era** behavior (loop guard + `?error=` server-side rendering), while both precheck
     AND master pinned **v4.13.0**, which has neither (guard first ships in v4.16.0 — verified by
     tag sweep `handler.go` `Query().Get("error")`; v4.13.0's `#lp-error` div is a dead stub: no
     server fill, no JS). The app was developed against the cqrs-htmx working tree; the pin was
     stale relative to its own committed comments and tests.
   - **Disproved the 02-27 templ matrix**: "ALL templ-components releases (v1.23.0–v1.27.2)
     require templ v0.3.1020" is WRONG — v1.27.0/v1.27.2 require templ v0.3.1100 (module-cache
     verified). Two coherent eras exist: 1070-era (loginpage/adminui v4.13.0, tc v1.23) and
     1100-era (loginpage/adminui v4.16.0+, tc v1.27, templ v0.3.1100+).
   - **Path A executed**: loginpage+adminui → v4.16.0 (drags usermgmt v4.18.0, identity-model
     v4.15.0, templ-components v1.27.0), templ pinned v0.3.1118, webauthn → v4.13.0
     (`FinishLogin` now returns `([]byte, error)` — two test call sites fixed), 14 `*_templ.go`
     regenerated with the v0.3.1118 CLI. **Zero production app-code changes required.**
   - My interim app-side loop guard (loginPlain twin + route divert — built first under Path B)
     was **fully reverted** once the library guard made it redundant; the original comments in
     `login_pages.go`/`middleware.go` are TRUE again at v4.16.0.
2. **§f-2 T9 tripwire fallout resolved (precheck tree).** go-codec **v0.3.3 (2026-10-29)** deliberately
   converged CBOR decoding (text strings accepted for `encoding.TextUnmarshaler` — fixed real
   pre-branding journal-replay breakage); the wave drags it in and the 2026-10-15 tripwire fired
   exactly as designed. go-branded-id v0.7.0→v0.7.1 ruled out (library files byte-identical).
   Redesigned the audit CLI to **wire-shape detection** (`tenantIDIsLegacyWireText`: generic CBOR
   decode — text-string→`string`=legacy, byte-string→`[]byte`=typed; JSON indistinguishable→false)
   and rewrote the tripwire to pin convergence + detection. All CLI tests green.
3. **PRECHECK FULL SUITE GREEN (141s) — first time ever.** All §f-1 goals met.
4. **§f-3 / 8b MASTER: browser-history green for the first time.** Master was in a MIXED era state
   (templ v0.3.1118 + webauthn v4.13.0 + usermgmt v4.20.0 + oauth2 v4.13.0, but loginpage/adminui
   v4.13.x old-era → could not build). Applied the same Path A wave bump (loginpage+adminui
   v4.16.0, keeping master's newer usermgmt/webauthn/oauth2), regen ×14, **BUILD OK; full api
   suite GREEN (193s); the never-exercised C22 debug tests verified RUNNING and passing**
   (`TestDebug_Idempotency`, `TestDebug_Idempotency_SQLitePathReportsNoSweep`); sibling modules
   (domain/storage/projection/extraction) all build+test green.
5. **SEC (8c) root cause + partial repair.** The `ac388725` "update dependencies to latest" bump
   pulled go-branded-id v0.7.0, whose `String()` is now `"Brand:value"` (BrandNamer markers) —
   every `.String()`→`Parse*ID` round-trip leaks a `StreamMarker:` prefix and ULID parsing dies
   ("ulid: bad data size"). Fixed **5 prod sites** (`command_handler.go:92`,
   `island_handler.go:96`, `testutil.go:45`, `game_create.go:71`+`:141` — `.StreamID().String()` →
   `.StreamID().Get()`) and **5 test sites** (shared `load_test.go` ×2, app `idempotency_test`,
   `snapshot_test` `mustParseGameID` signature, `scenario_test` ×2). Packages now GREEN: app,
   shared, testutil, command, event, query.

## b) PARTIALLY DONE

1. **SEC projection package (3 ginkgo specs red).** Handler keys states by
   `evt.StreamID().String()` (prefixed) while the test looks up raw — key mismatch explains the
   `proj.Get` miss; the stored-zero-GameID half is one step from closed: the fold chain is
   `encoding/json/v2`-based, payloads are JSON via SEC's own `internal/cqrs/eventtest` (pre-marshals
   JSON precisely because the decider folds JSON), so the remaining suspect is cbid v0.7.0's
   JSON marshal/unmarshal symmetry for the `GameID` field. Uncommitted at interruption:
   `idempotency_test.go`, `scenario_test.go`, `snapshot_test.go` (daemon absorbed the 5-file prod
   batch as `059ef997`).
2. **SEC server / tests-integration / cmd-cli packages** — not yet re-run after the prod fixes;
   their failures (`TestAnalyticsPageHandler`, `TestAutoPlayHandler`, …) share the same
   prefix-leak class with high probability (same `requireParsedGameID`/handler paths).

## c) NOT STARTED (this session)

- 8d InboxClean: 2 labels-preservation tests + gitignored-go.sum policy routing.
- cqrs-lint 21-package full-suite rerun (post-F030-fixture).
- Fleet receipts: `scripts/fleet-dedup-receipt.sh` across all 9 repos with the fixed re-bind binary.
- Item 6 docs-health harvest into TODO_LIST + owner decision sheet.
- Final guardrail sweep + session-close ritual (CHANGELOG rows for browser-history/SEC work live in
  THOSE repos' conventions, not this one — noted for the harvest).

## d) TOTALLY FUCKED UP (honest ledger)

1. **Five phase-boundary commits stolen by the daemon** (precheck `be769f7`/`e10910f`/`065adfe`/
   `15ffa0e`, master `dfe9f9f`): zero authored commits this session. The trees are clean and the
   suites were verified green AT the committed trees, so nothing is lost — but the 2026-11-03
   authored-commit ruling produced no authored history here. Improvement below (e).
2. **Path B detour**: built the app-side loop guard (4 files) before verifying the templ-era claim
   that made it necessary. The claim was wrong (see a-1); verifying the module cache FIRST would
   have skipped the detour. It was cleanly reverted, and it did prove the redirect loop was real
   at v4.13.0 (so the work wasn't wasted as evidence).
3. **Wrong SEC candidate first**: checked `securitymd` (green, irrelevant) before grepping the
   plan doc for the repo name (`games/SEC`). Plan-doc grep should have been the first move.

## e) WHAT WE SHOULD IMPROVE

1. **Commit faster at verified-green boundaries**: pre-write the message file BEFORE the
   verification run, `git add` the exact list, commit the moment the suite reports green. The
   daemon polls on a timer; the window between "green" and "commit" was ~minutes every time.
2. **Module-cache verification before matrix claims**: the 02-27 report's templ-components claim
   was wrong and directly shaped (wasted) work. Any "all versions do X" claim gets one grep
   against the cache before it steers a decision.
3. **Plan-doc repo-name resolution before directory guessing.**
4. **T9 divergence**: precheck now carries MY wire-shape redesign while master carries ITS
   accept-convergence redesign (auto-commits `698bde4`/`3ddafe1`/`77afe71`, master is
   authoritative). The deploy-runbook property "a legacy-shaped event can never silently
   reappear" is only preserved by wire-shape detection — master's redesign drops it. Owner call
   (question 2).
5. **SEC's `.String()` round-trip pattern**: the five prod sites fixed were found by grep; the
   same shape may exist in `server/`, `e2e/`, `web/` paths not yet swept. A repo-wide
   `ParseX(...String())` sweep belongs in the SEC close-out.

## f) NEXT (up to 50, ordered)

1. Close SEC projection: read `decider.FoldEventErr` payload path vs cbid v0.7.0 JSON symmetry;
   fix keying (`Get()` not `String()`) in `game_state.go:49`+`:118`.
2. Re-run SEC `server` package; fix remaining prefix-leak sites (analytics/autoplay handlers,
   `requireParsedGameID`).
3. Re-run SEC `tests/integration` + `cmd/cli`; fix.
4. SEC full-suite green run (all packages).
5. SEC repo-wide sweep for residual `.String()`→`Parse*` round-trips (incl. `e2e/`, `web/`).
6. SEC: check whether `sec.db`/`sec.db-wal`/`test-results/` untracked junk should be gitignored.
7. Route the SEC dep-bump aftermath: was `ac388725` ("update dependencies to latest") deliberate?
   Its fallout (8+ red suites for weeks) suggests unpinned bumps need a test gate in that repo.
8. 8d InboxClean: run the 2 labels-preservation tests; diagnose; fix at root cause.
9. 8d InboxClean: route the gitignored-go.sum policy question (owner or TODO_LIST).
10. cqrs-lint full 21-package suite rerun (`cd cmd/cqrs-lint/v4 && GOWORK=off go test ./... -count=1`).
11. Fleet receipts: `bash scripts/fleet-dedup-receipt.sh` all 9 default repos; archive artifacts.
12. Verify the receipts show the re-bind fix working fleet-wide (warning + per-repo configs).
13. Item 6 harvest: load docs-health skill; harvest this + 02-27 §f remainder into TODO_LIST.
14. Harvest: dependabot owner-ping row (8+ days stale, 138/139 open).
15. Harvest: templ-era matrix CORRECTION row (the 02-27 knowledge was partly wrong).
16. Harvest: T9 design divergence row (master accept-convergence vs wire-shape detection).
17. Harvest: cqrs-htmx v4.16-wave migration row for remaining fleet consumers (loginpage/adminui
    old pins elsewhere?).
18. Harvest: SEC dep-bump gate row.
19. Harvest: evidence §1.1 tick — the cwd-preset footgun is now FIXED (re-bind shipped `8aaa19d9c`).
20. Owner decision sheet: push blessing (now 20+ unpushed commits across ≥4 repos, two suites
    first-time-green).
21. Owner decision sheet: retroactive blessing of the v4.16 wave bump on browser-history master.
22. Owner decision sheet: T9 wire-shape detection port to master (yes/no).
23. Owner decision sheet: Q1 verify strategy (C5, 6th session carrying); replay-vs-dedup (C23);
    D28–D31 gates; C26 fill timing.
24. browser-history: CHANGELOG/FEATURES rows for the wave migration + T9 story (its own repo
    conventions).
25. precheck worktree disposition: its verification purpose is served; owner may want it dropped
    (detached HEAD, unpushed) — do NOT delete without instruction.
26. `go.work`/gls disposition row (carried from 02-27 §f).
27. Check whether cqrs-htmx v4.16.0+ tags need a release-notes/CHANGELOG cross-ref in the fleet
    docs (the wave solved browser-history; document the era matrix).
28. go-cqrs-lite guardrail spot-check after all fleet work: `check-md-go`, `lint-scope` for
    touched modules (none touched in THIS repo this session — likely clean).
29. Confirm M-wave session state before any go-cqrs-lite commit (their files may still ride the
    index — cached-diff check protocol).
30. Update this repo's `docs/evidence/2026-11-20_fleet-idempotency-baseline.md` §6 ledger with
    the new commits (precheck ×4 daemon, master `dfe9f9f`, SEC `059ef997`+).
31. Update `docs/evidence/2026-11-21_fleet-dedup-bundle-receipts.md` §1 with the new repo states.
32. TODO_LIST row: browser-history precheck/master unpushed — awaiting push blessing.
33. TODO_LIST row: SEC unpushed — awaiting push blessing.
34. Re-check InboxClean "ahead 2" + Swetty "ahead 5" push states for the same blessing batch.
35. Session-close: `git status` all touched repos; ensure no foreign files staged anywhere.
36. Session-close: reconcile TODO_LIST statuses (8a done, 8b done, 8c partial, 8d pending).
37. If owner blesses T9 port: port `tenantIDIsLegacyWireText` + tests to master (small, isolated).
38. If owner blesses push: push order = browser-history master → precheck (or drop) → SEC → others.
39. Consider a cqrs-htmx doc note: "consuming the v4.16 wave requires templ ≥ v0.3.1100 regen"
    (the era matrix that cost this session hours).
40. SEC: `FoldEvents` in `shared/fold.go` still decodes with `encoding/json/v2` — confirm every
    publish path pre-marshals JSON (app init sets `event.DefaultCodec = JSONCodec`) or the
    production replay has the same CBOR/JSON mismatch the eventtest package documents.
41. Check SEC `mustParseGameID` callers all pass types with `Get()` (compile-verified, but confirm
    none pass a display-only stringer).
42. Fuzz/canary: browser-history master now runs go-codec v0.3.3 — any golden payload fixtures in
    its repo needing regeneration (check `docs/drafts/templ-components_errorpage-shell-hook…`
    class TODOs while there).
43. Race spot-check on browser-history api (`-race` on the login/passkeys subset) — first-time
    green suites deserve one race pass.
44. Race spot-check SEC app package after it goes green.
45. Confirm the `mesh-demo` untracked dir in go-cqrs-lite belongs to the M-wave session (do not
    touch; flag in harvest if still unowned).
46. Nightly versions gate (`check-versions-manifest.sh`) unaffected — verify no drift after all
    this (fleet-side bumps don't touch this repo's manifest).
47. If the M-wave session has landed more go-cqrs-lite commits by next session, rebase NOTHING —
    read their report first.
48. Consider `brandid-lint` BD-style rule for SEC's own code: flag `X.String()` feeding
    `ParseX(...)` — the exact class that broke it (may exist upstream already).
49. Capture the "daemon absorbs within minutes — pre-write commit messages" lesson in
    `docs/agents/gotchas-module-management.md` (tag-wave section) at harvest time.
50. Rest: the remaining §f owner-gated items (D28–D31, Q1, C23, C25, C26) cannot move without
    owner input — list them at the top of the next session's context.

## g) QUESTIONS (cannot resolve myself)

1. **Push blessing.** The unpushed set now includes browser-history master (`dfe9f9f`, first-ever
   green suite) and precheck, SEC (`059ef997`+uncommitted tests, mid-repair), plus the carried
   InboxClean/Swetty/gls batches (20+ commits, ≥7 repos). Push now, or hold the whole batch for a
   review pass?
2. **T9 design divergence.** Master's redesign accepts the go-codec v0.3.3 convergence (legacy
   rows = tenant-bearing events; the audit only flags corruption). My precheck redesign keeps
   wire-shape legacy detection, preserving the runbook property "a legacy-shaped event can never
   silently reappear" (a reintroduced raw-string payload would decode fine and count as typed
   under master's design). Port wire-shape detection to master as a follow-up, or accept master's
   simpler design?
3. **Retroactive blessing of the v4.16 wave.** The committed loop-guard test could not be
   satisfied honestly at the pinned loginpage v4.13.0, so I moved browser-history (master +
   precheck) to the coherent v4.16 era (loginpage+adminui v4.16.0, usermgmt v4.18.0, webauthn
   v4.13.0, templ v0.3.1118, templ-components v1.27.0) rather than weaken the test or carry
   compensating app-side code. This is a master-level dependency decision made autonomously under
   the "fix at root cause" mandate — confirm, or direct a different pin strategy?

## Appendix: receipt commands (verified this session)

```bash
source /home/lars/projects/go-cqrs-lite/scripts/go-env.sh
cd /home/lars/projects/browser-history-precheck/api && GOWORK=off go test ./... -count=1   # ok 141s
cd /home/lars/projects/browser-history/api && GOWORK=off go test ./... -count=1            # ok 193s
cd /home/lars/projects/browser-history/api && GOWORK=off go test -run 'Debug' . -count=1  # C22 tests PASS
cd /home/lars/projects/games/SEC && GOWORK=off go test ./internal/cqrs/app/... ./internal/cqrs/shared/... # ok
GOWORK=off go run github.com/a-h/templ/cmd/templ@v0.3.1118 generate                        # regen ×14
```
