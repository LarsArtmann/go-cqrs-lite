# Status Report: Release-Tooling Hardening + First upgrade-dogfood Sentinel CI Run

**Session date:** 2026-09-11, ~04:00–05:30 CEST
**Scope:** The two TODO_LIST items sourced from the docs-health fifth pass (01-47 §f39 and §b2/§f2). No other research was done; observations below come only from this session's own runs.

---

## 0. TL;DR

| Item                                                               | Verdict                                                                                                   |
| ------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------- |
| Audit `scripts/batch-release.sh` against hardened `tag-release.sh` | **DONE** (5 gaps found, all fixed + tested + CI-wired)                                                    |
| Watch the first nightly `upgrade-dogfood` sentinel run             | **DONE** — observed, it FAILED, root-caused, fixed locally, **not yet CI-validated** (no push)            |
| Bonus root-cause fix                                               | example/taskmanager's private `go-must` dependency blocked ALL workspace-mode go commands on CI — removed |

---

## a) FULLY DONE

### A1. batch-release.sh audited and hardened to the tag-release.sh bar

`scripts/batch-release.sh` did encode the pre-hardening flow. Five gaps, all fixed:

1. **No path-vs-tag guard** — could create proxy-invisible tags (the issue-#20 class: cmd/cqrs-lint v4.2.0–v4.7.0 shipped this way). Now: `path_matches_major` guard per triple runs before anything is touched; `--audit` delegates to `tag-release.sh --audit` (one implementation, no logic fork).
2. **Replace-strip was wrong twice** — the old regex only matched `go-cqrs-lite/` LHS paths, so sibling-repo LOCAL replaces (go-finding, go-must) survived into published go.mods; and it stripped ALL 84 go.mods when only the tagged modules' go.mods are read by the proxy. Now: tag-release.sh's target-shape rule (only `.`/`/`-prefixed targets) applied to the TAGGED modules only.
3. **No standalone compile gate** — a go.mod pinning an older sibling than the code needs shipped broken to consumers (command/v4.7.0 class). Now: every tagged module must build `GOWORK=off` (with `-tags goexperiment.jsonv2`) against its stripped go.mod before any tag is created.
4. **Restore path was broken twice** — `git checkout -- .` (banned command) restored the STRIPPED go.mods from the stale index, silently re-dirtying the tree; `HEAD~1` breaks when the auto-commit daemon commits between the temp commit and the reset. Now: `original_head` capture + `git reset --soft <original_head>` + `git restore --staged --worktree`, mirroring the hardened tag-release.sh. Both exit shapes (before/after temp commit) covered by one EXIT trap.
5. **No smoke/audit path** — post-run output now prints per-tag `tag-release.sh --smoke <module> <version>` commands and the `--audit` passthrough.

Plus:

- **Malformed-triple guard** — unquoted args previously died with a confusing `ERROR: v2.0.1/go.mod not found`; now a clear "malformed triple, quote as one string" error.
- **Tag-creation failure is fatal with cleanup** (old loop deleted the bad tag but still reported "Created N tags" success).

### A2. Latent binary-pollution bug found and fixed in BOTH release scripts

`go build ./...` writes main-package binaries into the module directory, silently dirtying the tree after a successful cut. `tag-release.sh:462` had the same bug (untested — its suite had no success-path test). Both scripts now build with `-o` into a throwaway dir; library-only modules fall back to the plain build (`-o` refuses "no main packages to build" — verified it does NOT compile in that case, so the fallback is mandatory, not cosmetic). Error ordering verified: a broken library module still aborts the gate via the fallback build.

### A3. Test suites written, extended, and wired into CI

- New `scripts/test-batch-release.sh` — 21 checks across 6 scenarios: path-guard rejection, malformed triple, `--audit` delegation, successful two-module batch (main + library) with **exact tree restore** + replace-stripped-at-tag + replace-kept-in-worktree + no-artifacts assertions, standalone-build abort, duplicate-tag rejection.
- `scripts/test-tag-release.sh` — added the missing **success-path Test 6** (the gap that let the binary bug hide) and fixed the 4 pre-existing `shellcheck SC2086` findings (`git $notag` → array form).
- Wired as flake app `nix run .#check-release-scripts` (flake.nix) + a CI leg in `ci.yml`'s `lint-scripts` job (actionlint rc=0, `nix flake check` passes).
- `scripts/test-exhaustruct-canary.sh` SC2012 fixed (`ls -d` → `find`), script re-run green.

### A4. First real CI run of upgrade-dogfood observed (the unobserved run)

- Sep 9 + Sep 10 nightly runs: both failed at `green-recency` (correct alarm — master CI red 158 days); `upgrade-dogfood` job was ABSENT from both (added to sentinel.yml in e37642adf at 23:38 UTC Sep 10, after both schedules).
- Dispatched the first execution: [run 34556835441](https://github.com/LarsArtmann/go-cqrs-lite/actions/runs/34556835441) — `upgrade-dogfood` **FAILED in ~20s** (nowhere near the 20-min timeout).

### A5. Root cause found and fixed: private go-must blocked every workspace-wide go command

- `example/taskmanager` (a `go.work` member) required `github.com/larsartmann/go-must` — a **private** repo. proxy.golang.org serves only v0.1.0 (cache froze when the repo went private; v0.1.1/v0.1.2 can never be served). Workspace mode unions all members' requires, so `go build ./...`, `go run ./cmd/cqrs-upgrade`, CI's `nix flake check`, and the sentinel all had to load go-must@v0.1.2 → proxy 404 → direct-VCS fallback → auth prompt → exit 128 on CI.
- Local passes invisibly (module cache has the version + devShell HTTPS→SSH git auth) — exactly why "passes locally, fails in CI".
- Fix: inlined the two helpers taskmanager uses (`Must`, `Check`) into `example/taskmanager/must.go`; dropped the require + go.sum entries; no call-site changes otherwise.
- **CI-fidelity verification**: clean `GOMODCACHE`, proxy-only `GOPROXY` (no `,direct`), no GOPRIVATE, no credentials → `go list -m all` over the whole workspace exits 0 (proves every module in the current union graph is proxy-servable). taskmanager builds and tests green under `GOWORK=off`. Local `nix run .#check-upgrade-dogfood` green.

### A6. Documentation updated

- CHANGELOG.md `[Unreleased]`: two full entries (private-dep fix; release-tooling hardening).
- TODO_LIST.md: both completed items removed (per file convention).
- AGENTS.md Quick Reference: new `Rel. tests` row.
- docs/agents/gotchas-module-management.md: the workspace-union private-module trap (with the `go list -m all` CI-fidelity repro recipe) + Release section now covers batch-release.sh.
- `cmd/doc-check` zero-warning gate: 1049 references valid.

---

## b) PARTIALLY DONE

1. **Sentinel CI re-validation** — the go-must fix is committed locally but **not pushed**. Tonight's 03:17 UTC sentinel (and the `Nix Flake Check` leg) should flip green, but that is unverified. Workflow_dispatch cannot validate it pre-push (dispatch runs origin/master code).
2. **Pure-proxy ZIP-level verification** — `go list -m all` proves the module GRAPH loads (all go.mods servable). A full clean-cache, proxy-only BUILD of a representative consumer (e.g. taskmanager) would additionally prove every zip fetches; not done (several minutes of downloads; the graph check covers the failure mode CI actually hit).
3. **taskmanager test depth** — `GOWORK=off go test ./...` passed in 0.080s, which smells like most integration tests skip without env markers. I did not investigate what actually ran. The new `must.go` helpers have **no tests of their own**.
4. **Master CI red status** — I fixed the module-load blocker only. Other red legs observed in this session's look at run 34552186915 (not fixed, mostly not mine): projectionhost file-size violations (353/382/388 > 350), coverage, api-stability, and verify-fast's `nix run .#build` (likely the same go-must class — should flip after push, unverified).
5. **`nix run .#verify` / `.#check-duplication` not run this session** — deliberately: the working tree carried ANOTHER session's in-flight work (metaengine/*engine reset changes, scheduling tests, calibration-gate.sh), and `#verify` runs exclusively + would test their half-done state. `must.go` duplicates two ~6-line helpers from a sibling repo art-dupl never scans together with this one, so the gate should be safe, but it is unrun.

---

## c) NOT STARTED (observed this session, not touched)

- projectionhost file-size violations (`sqlite_dlq.go` 353, `host.go` 382, `worker.go` 388 lines).
- The remaining open TODO_LIST items (smoke-probes.txt, `--audit --baseline`, GitHub Releases, indirect-dep consolidation, dead-path decisions, standing pin-sweep step).
- Any investigation of WHY master CI has been red since 2026-04-05 (157+ days) beyond the one blocker in front of me.
- A mechanical guard against future private-dep requires (advice documented in gotchas, not automated).
- Node 20 deprecation annotations on sentinel/ci actions (cosmetic, from run annotations).

---

## d) TOTALLY FUCKED UP (honest self-assessment)

Nothing destructive, but four real self-inflicted inefficiencies:

1. **Edit-tool JSON escaping burned three round trips** — the `\"\$0\"` sequences in the test harness failed to match twice because backslash-quote escaping was wrong in my old_strings. Cost: 2 failed multiedits + re-reads. Should have used single quotes around those assertions in the test design itself (like test-tag-release.sh does with heredocs) or written the file once with `write`.
2. **Debug repro used unquoted triples** — my first manual repro of Test 1 passed `dead v2.0.1 x` as three args, which muddied the diagnosis for a moment. The quoted-triple requirement was in the header I had just read.
3. **`go build -o` fallback needed two empirical detours** — I first tried `go install` with GOBIN (blocked by tool policy), then a nested `find` for SC2012 that I immediately replaced with the direct one-liner. A quick mental pass over `go build -o` semantics (multiple packages require a directory; zero main packages error out) would have gotten there in one step.
4. **Initial test 1 in the harness itself had the unquoted-triple bug** — I wrote the test wrong before the script was wrong. The suite is only as good as its first draft; should have mirrored test-tag-release.sh's exact invocation pattern from the start.

Not fucked up, worth stating: I did NOT touch the other session's in-flight files (metaengine resets, calibration scripts, their CHANGELOG/AGENTS lines), did not push, did not run banned git commands, and all my script changes were verified as daemon-absorbed with `git diff HEAD` empty.

---

## e) WHAT WE SHOULD IMPROVE

1. **Private deps in workspace members must be mechanically impossible, not advisory.** The gotcha documents the rule; a CI script (`check-private-deps`: every `github.com/larsartmann/*` require in every go.mod → visibility/proxy-servability check) would make the next recurrence fail in seconds instead of after a nightly.
2. **Release-script tests belong in the release procedure, not just CI** — CONTRIBUTING.md's release process should reference `check-release-scripts` so flow changes get a new fixture test by reflex (the tagger binary bug survived 0 tests on the success path).
3. **Sibling-repo visibility strategy needs an owner decision.** go-must went private and froze the proxy. go-retry/go-codec/go-branded-id/go-sse/go-idempotency/go-flightrecorder currently resolve (their pinned versions are servable), but if ANY of them is private, the NEXT pin bump of it hits the identical wall. This is a product-level risk for a public library with hard deps on helper repos.
4. **Sentinel signal hygiene**: with CI red, every nightly is red at green-recency, and a NEW failing job (like dogfood was) hides inside an already-red run. Consider a job-summary table or per-job annotations so one glance separates "known red" from "new red".
5. **test isolation from global git config** — fixtures must set `tag.gpgSign=false` etc. locally (my batch fixture does; test-tag-release.sh's fixture only strips signing for its OWN tag commands — its Test 6 needed the same local config). A shared fixture helper would remove this class.
6. **Batch dry-run is parse-only** — it checks guards but doesn't exercise strip/tidy/build. tag-release.sh's dry-run is full-fidelity. Acceptable tradeoff (documented), but a `--verify` flag doing the full pipeline without tagging would close the gap.

---

## f) UP TO 50 THINGS TO DO NEXT (prioritized, roughly Pareto-ordered)

**Validate + lock in this session's work (1–6):**

1. Push master → confirm `Nix Flake Check` + `verify-fast` legs flip green (go-must class gone).
2. Watch tonight's 03:17 UTC sentinel: `upgrade-dogfood` must pass in real CI (network topology + GOPROXY + 20-min timeout now actually exercised).
3. Re-dispatch sentinel after push instead of waiting for the schedule (same job definition).
4. Run `nix run .#check-duplication` once the tree settles (must.go + any other new Go files).
5. Add unit tests for `example/taskmanager/must.go` (it's a copy with zero tests).
6. Confirm what taskmanager's `go test` actually executes in 0.080s (skips? which env markers?).

**Private-dep hardening (7–12):**
7. Build `scripts/check-private-deps.sh` (+ flake app + CI leg): all go.mod requires of `github.com/larsartmann/*` must be proxy-servable (`@v/<version>.info` fetch) — mechanical recurrence guard.
8. Audit visibility of ALL sibling helper repos (gh api `.private`) and record which are public/private in module-map.md.
9. Decide + document the policy: examples may only depend on public/proxy-servable modules; helper repos must be public or vendored.
10. Add the CI-fidelity repro (`GOMODCACHE=$(mktemp -d) GOPROXY=proxy-only go list -m all` at root) as a cheap CI leg or nightly step.
11. Check future-bump ceilings: for any private-but-cached repo, the NEXT version can never be proxy-served — enumerate which pins are frozen.
12. Consider `GOPROXY=off` + explicit allow-list for example modules so private requires fail at dev time, not CI time.

**Master CI red (13–18):**
13. Fix projectionhost file-size violations (353/382/388 > 350) — split `host.go`/`worker.go`/`sqlite_dlq.go`.
14. Triage the coverage leg (is it a cascade of the go-must class or real?).
15. Triage the api-stability leg (downloading deps in CI — was it the same module-load failure?).
16. After green, confirm `green-recency` stops firing and the nightly goes fully green.
17. Investigate the 157-day red history enough to write a one-paragraph timeline in a status report (what broke when).
18. Set up a personal rule/notification so 100+ days of red can't happen silently again (the sentinel exists; it fired — who reads it?).

**Release tooling follow-ups (19–27):**
19. `tag-release.sh --audit --baseline` mode (open TODO — gates NEW violations only).
20. `scripts/smoke-probes.txt` per-binary probe commands (open TODO).
21. Create GitHub Releases for outstanding tags via `create-github-releases.sh` (open TODO; only storage/v4.7.1 ever got one).
22. Add a `--smoke-all` batch mode: after pushing N tags, one command smoke-checks each in sequence.
23. Document the batch inter-module limitation: a module cut in the same batch cannot pin its batch-sibling's NEW tag (resolves to the sibling's latest published).
24. Optional: batch dry-run `--verify` mode running strip+tidy+build without tagging.
25. CONTRIBUTING.md: reference batch-release.sh + check-release-scripts in the release process.
26. Consider extracting `path_matches_major` into a sourced lib to kill the two-copy lockstep risk (currently documented; art-dupl doesn't scan shell).
27. Decide whether `check-release-scripts` should also run in `#verify` (it's ~30s).

**Dead-path + module hygiene (28–33):**
28. Owner decision: example/taskmanager + getting-started invisible v3/v4 tags (re-path /v4, delete, or document v0-only) — now more relevant since taskmanager changed.
29. Same for `event/v4/eventtest` invisible v0.x tags (document as dead).
30. example/taskmanager: it publishes nothing but IS a module with tags — decide if it should stay a module or become a plain directory.
31. Update module-map.md internal notes for taskmanager (no go-must anymore).
32. Sweep `example/` for any other private requires (check-private-deps would cover this).
33. Consolidate indirect `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}` transitive refs after new tags publish (open TODO).

**Docs (34–39):**
34. Harvest + archive this report per docs-health flow (annotate the 01-47 sources §f39/§b2/§f2 as resolved).
35. Update the docs/status/README.md index.
36. recipes.md/core.md: no consumer-facing surface changed — confirm doc-check stays green after next edits (it is green now).
37. Record the `go build -o` + "no main packages" gotcha in gotchas-tooling-build.md (it burned real time).
38. Record the "LSP caches lie" instance (package-name split phantom errors after sed) if not already covered.
39. ADR-light note (or gotcha) on workspace-union dependency semantics for future module authors.

**Adjacent quality items noticed in passing (40–47):**
40. Node 20 deprecation annotations on checkout/nix-installer actions — bump pinned SHAs when the maintainers ship Node 24 builds.
41. gopls `infertypeargs` infos in decider_test.go (style-level; harmless, but noisy).
42. The other session's calibration-gate.sh (untracked, in-flight) — coordinate so two sessions don't write overlapping gate scripts.
43. `integration/go.mod` + metaengine/projectionhost gopls "go mod tidy" warnings (pre-existing, seen in diagnostics).
44. taskmanager golden pin (V006) in cqrs-lint — my go.mod change removed a require; confirm the golden doesn't pin taskmanager's module set (it pins versions for lint rules — verify V006 drift test still passes; taskmanager tests passed, so likely fine).
45. benchkit/dgraph/session work (other sessions) — out of scope, listed only to not lose them.
46. Consider making sentinel dispatch a post-push habit until CI is stably green.
47. Sentinel `green-recency` wording says "Triage failing legs or the runner/billing situation" — fine as-is.

**Longer-term (48–50):**
48. Release train: with the tooling hardened, actually run the pending pin sweep + tag wave the TODOs describe.
49. Evaluate moving example apps OUT of the workspace (workspace unions are the root hazard class; examples as plain checkouts can't poison the union).
50. Owner-facing decision doc: public library + private helper repos is a structural contradiction — pick the end state (publicize, vendor, or internal-only consumers).

---

## g) QUESTIONS FOR YOU (cannot determine myself)

1. **Push timing** — the go-must fix + release-tooling hardening are committed locally; the tree also carries ANOTHER session's in-flight work (metaengine resets, calibration-gate.sh). Push master now so CI + tonight's sentinel validate my fix, or wait until the in-flight session lands? (I won't push without your go-ahead either way.)
2. **Helper-repo visibility policy** — go-must is private and its proxy cache froze at v0.1.0. Should go-must be made public, or is the policy "private helpers → examples/vendor must inline"? Same question for go-retry/go-codec/go-branded-id/go-sse/go-idempotency/go-flightrecorder: if any is private, the next pin bump of it re-breaks CI and consumers identically.
3. **Sentinel alarm semantics** — while master CI is red, `green-recency` makes EVERY nightly red, which trains "red = normal". Keep it hard-failing (it did its job), or downgrade to a WARN/summary status until CI is stably green, with hard-fail only after a green baseline exists?

---

_Report ends. Waiting for instructions._
