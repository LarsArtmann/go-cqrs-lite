# Status Report — Data-Mesh Completion Session (T22–T27 tail + §f queue)

**Date:** 2026-09-24 13:32 (Thursday)
**Session scope:** Resumed execution under the user's "READ, UNDERSTAND, RESEARCH,
REFLECT … execute and verify one step at a time, repeat until done" instruction,
picking up the
[2026-09-24 12-26 handoff](2026-09-24_12-26_data-mesh-pareto-execution-session.md)
(T01–T21 done, T22–T27 + §f queue open, 3 owner questions pending). This report
covers only what THIS session did, found, and broke. **The session was
interrupted mid-hub-phase by this report demand.**

---

## a) FULLY DONE

All verifiably complete: tested, gates green, committed (authored `8a21a0812` +
daemon waves `54684bc92`/`8d94768c2` in go-cqrs-lite; `fc95f7a` in the hub).

| #  | Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                | Evidence                                                                                                                                                                                                                                                    |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **Handoff reconciliation** — todos tool reset to true state (14 stale entries corrected); remote state discovered already externally synced (~30 commits pushed by daemon/user; only local-only commits remain unpushed)                                                                                                                                                                                                                                                                                                                                                                                            | `git log origin/master..master` at session start                                                                                                                                                                                                            |
| 2  | **T26-tail — all 3 single-source claims VERIFIED, zero drift.** Watermill matrix: `TestRedisStreamRoundtrip` exists, `ephemeral-redis.sh`/`ephemeral-nats.sh` exist, NO `transport/nats                                                                                                                                                                                                                                                                                                                                                                                                                             | redis`(only grpc/http), every documented constructor (`NewEventPublisher`,`WithBackend`,`NewCatchUpSubscriber`, …) matches, README JetStream correction in place. DeploymentConfig: struct fields match every doc citation (recipes compile-gates pin them) |
| 3  | **T25 — EventCatalog SLA/freshness schema probe → ROADMAP raw idea.** Probed `@eventcatalog/core` 4.6.3 locally: data products carry ONLY `inputs`/`outputs` pointers + base fields; zero SLA/freshness/data-quality schema anywhere; `withExtensionProperties` = `catchall(z.unknown())` (unknown frontmatter keys pass, unrendered). Raw idea written with the render-in-our-docserver path                                                                                                                                                                                                                       | ROADMAP.md Raw Ideas                                                                                                                                                                                                                                        |
| 4  | **ROADMAP duplicate-Theme-11 numbering fixed on sight** (T20's Theme 11 collided with the older persistence Theme 11) — renumbered 11→12→13 + the internal FoundationDB anchor link                                                                                                                                                                                                                                                                                                                                                                                                                                 | ROADMAP.md:419,458,651                                                                                                                                                                                                                                      |
| 5  | **CHANGELOG `[Unreleased]` receipts** for mesh-demo, E019, docserver DataProducts, the mesh cookbook (recipes §2.41 / advanced §6.21), and the gRPC v5 guide — closing the changelog-gate honesty blind spot flagged in the handoff §e2                                                                                                                                                                                                                                                                                                                                                                             | changelog-symbols gate green: "37 pkg.Symbol citation(s) … honest"                                                                                                                                                                                          |
| 6  | **Boot-smoke test for shipped config (§f15)** — `example/goal-shaped-app/config_boot_smoke_test.go`: `TestShippedConfigBoots` runs the repo's ACTUAL `cqrs.yaml` byte-for-byte (temp-cwd sandboxes its relative sqlite DSN) through full `run()` — the exact class that shipped broken in T21 can no longer recur silently                                                                                                                                                                                                                                                                                          | test green (0.22s)                                                                                                                                                                                                                                          |
| 7  | **87 MB of daemon-absorbed build binaries removed from git** — 5 tracked compiled binaries (`goal-shaped-app` 27MB, `scheduler-otel-status` 21MB, `metaengine-quickstart` 19MB, `readme-quickstart` 9MB, `example/mesh-demo/mesh-demo` 9.8MB); gitignored by exact path so re-absorption is impossible. Found while calibrating the guard threshold                                                                                                                                                                                                                                                                 | `git rm` + .gitignore; deletions in `54684bc92`                                                                                                                                                                                                             |
| 8  | **tag-release.sh pre-push zip-content guard (§f16)** — `tag_zip_content_check`: (1) any path with bytes outside printable ASCII (the tursoengine/v4.2.0 `\x06` class), (2) any blob >4 MiB (largest legit asset: scalar.js 3.7 MiB — threshold is a deliberate tripwire). Runs after tag creation with the same cleanup path as the annotation failure; header docs updated                                                                                                                                                                                                                                         | `bash -n` + self-tests                                                                                                                                                                                                                                      |
| 9  | **Three new release smoke tests** (9: control-char path kills tag + cleans up; 10: >4 MiB blob kills tag; 11: clean tree passes + tag created) — tests 9–11 green, full suite "All tag-release smoke tests passed"                                                                                                                                                                                                                                                                                                                                                                                                  | `bash scripts/test-tag-release.sh`                                                                                                                                                                                                                          |
| 10 | **catalog/README 404 hub URL fixed (§f20)** — dead `github.com/LarsArtmann/eventcatalog-hub` link replaced with the private Forgejo mirror reference (kills the nightly lychee warning)                                                                                                                                                                                                                                                                                                                                                                                                                             | catalog/README.md:347                                                                                                                                                                                                                                       |
| 11 | **SKILL.md two-export routing note (§f29)** — new "Catalog exports: render tree vs governance tree" section stating the core-vs-linter ref-format split and pointing at catalog/README + recipes §2.41                                                                                                                                                                                                                                                                                                                                                                                                              | `.agents/skills/go-cqrs-lite/SKILL.md`                                                                                                                                                                                                                      |
| 12 | **T24 — hub merge.py: owners union + ref canonicalization.** Collision merge now unions `owners` (every side's owners show) AND collapses a bare ref (`svc`) shadowed by its versioned form (`svc-1.0.0`) — the exporter emits bare refs for EXTERNAL services (no version known) and composite for own, so pre-fix the merged tree carried both forms per relationship                                                                                                                                                                                                                                             | merge.py `dedupe_refs`/`merge_message_files`                                                                                                                                                                                                                |
| 13 | **T23 — hub merge.py governance gate (strict by default, `--no-strict` escape).** Post-merge validation: message producers/consumers must resolve in the UNION (bare → any version; `svc-1.0.0` → exactly that version → **cross-source version-pin drift fails the build**); services' sends/receives + data-products' inputs/outputs pointers must resolve; each source's `coeffects.md` validated against the union (source A consuming what no source provides = fail). Plus an upgrade pass: bare refs whose union has exactly one version are rewritten to versioned so the render tree's graph edges resolve | `validate_refs`, `validate_coeffects`, `upgrade_bare_refs`                                                                                                                                                                                                  |
| 14 | **merge.py selftest rewritten end-to-end** — union, owners, dedup, ghost producer, version-pin drift, dangling pointer, coeffects ghost, upgrade pass. Selftest green; **real mesh-demo exports merged through the new gate: ideal output** (shared event: owners `orders-team`+`billing-team`, producers `orders-svc-1.0.0`, consumers `billing-svc-1.0.0`, gate green)                                                                                                                                                                                                                                            | `python3 merge.py --selftest` + live merge of both contexts                                                                                                                                                                                                 |
| 15 | **T22 — `scripts/check-stale-sources.sh`** in the hub: clone-free `git ls-remote` probe per sources.json entry vs the published dist branch's `build-stamp.json`; FRESH/STALE/NEVER-BUILT/ROT classifications; graceful no-dist handling; `--self-test` with fixture bare repos (green)                                                                                                                                                                                                                                                                                                                             | hub repo, committed by hub daemon `fc95f7a`                                                                                                                                                                                                                 |
| 16 | **mesh-demo hub-onboarding validated end-to-end (§f24 mechanics)** — fresh `git clone` of go-cqrs-lite → `cd example/mesh-demo && go build` → default export + `-plain` export all succeed from a clean tree (the `replace ../../catalog` resolves inside the clone; siblings resolve from the proxy). The sources.json entries are designed and proven, not yet written                                                                                                                                                                                                                                            | /tmp/gcl-clone simulation                                                                                                                                                                                                                                   |
| 17 | **Forgejo mirror discovery** — `lars/go-cqrs-lite` mirror EXISTS with anonymous read (prerequisite for the mesh-demo sources.json entries); bank-sync mirror requires auth anonymously                                                                                                                                                                                                                                                                                                                                                                                                                              | `git ls-remote` probes                                                                                                                                                                                                                                      |

## b) PARTIALLY DONE

| Item                                         | What works                                                                                                                                                                                                                                         | What remains                                                                                                                                                                            | Blocker                 |
| -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------- |
| **Hub governance lint wiring (§f7)**         | Mechanism designed and pattern proven (mirror of go-cqrs-lite's check-eventcatalog: write full-error `.eventcatalogrc.js` into each plain-refs lint tree, run `@eventcatalog/linter` from hub node_modules); mesh-demo's `-plain` export validated | build.yml lint step not written; optional `lint` field not added to sources.json; not run in CI                                                                                         | interrupted mid-phase   |
| **mesh-demo sources.json onboarding (§f24)** | Both entries fully designed + fresh-clone-validated (build/export/plain-export commands proven)                                                                                                                                                    | Entries not written to sources.json; hub README onboarding section not updated                                                                                                          | interrupted mid-phase   |
| **T22 probe polish**                         | Script + self-test green; no-dist-branch case handled                                                                                                                                                                                              | ROT classification conflates AUTH failure with missing repo (live run: bank-sync "ROT" was actually an anonymous-auth askpass failure — misleading report); never classified separately | small fix, not yet done |
| **Hub README + TODO_LIST sync**              | merge.py docstring fully documents new semantics                                                                                                                                                                                                   | README sections (merge semantics, probe, lint field) + TODO_LIST updates ([blocked:upstream] items 1+4 now unblocked by T01's manifest and the exporter upgrade) not written            | interrupted mid-phase   |
| **f72 — SystemNix cross-reference note**     | Target located (`docs/planning/2026-09-22_23-27_EVENTCATALOG-FEDERATION-HUB.md`)                                                                                                                                                                   | Note not written                                                                                                                                                                        | queued behind hub phase |
| **docs-health harvest**                      | All inputs known (handoff §f table + this session's outcomes)                                                                                                                                                                                      | TODO_LIST data-mesh strike-through (T01–T21 done rows), §f harvest into TODO_LIST/ROADMAP not done                                                                                      | queued last by design   |

## c) NOT STARTED

| Item                                               | Why                                                                                                                       |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Final `nix run .#verify` closing gate**          | Deliberately last; not reached before interruption                                                                        |
| **`check-md-go` vendorHash repair attempt (§f18)** | Pre-existing infra break; queued for the final-gate step                                                                  |
| **Push of anything**                               | Still owner-gated; go-cqrs-lite has local-only commits (`54684bc92`..`8a21a0812`), hub has local commits — nothing pushed |
| **Release train / tag surgery (§f11–13)**          | Still the user's open questions (see §g) — irreversible proxy actions stay owner-gated                                    |

## d) TOTALLY FUCKED UP

1. **My own smoke-test fixture lied to me (test 9).** I wrote
   `>"$dir/bad$'\x06'file"` — ANSI-C quoting does NOT expand inside double
   quotes, so the fixture filename was the literal ASCII string
   `bad$'\x06'file`: a perfectly printable path the guard correctly ignores.
   The test "should" have passed trivially while proving nothing — it only
   failed because I also asserted on the failure exit code. Fixed by building
   the name outside quotes; the corrected test drives the REAL guard path.
   Lesson: a passing test with a self-defeating fixture is worse than no test.
2. **Two merge.py selftest bugs shipped in my first draft:** the coeffects
   parser did not filter markdown alignment rows (`-----` → phantom dangling
   event), and one fixture directory was never created (`FileNotFoundError`).
   Both caught by the selftest itself before any real run — the safety net
   worked, but neither should have been in the draft.
3. **Evidence-first discipline slipped on the probe design.** I built the
   stale-source probe around the published `dist` branch's build-stamp
   baseline WITHOUT first verifying that dist exists. It does not — **the hub
   build workflow has never completed a publish** (discovered only on the
   live run). The design survives (graceful NEVER-BUILT path), but I
   validated my assumption last instead of first. Same lesson the previous
   session already wrote down (three plan assumptions died on contact); I
   re-earned it cheaply this time.
4. **My authored commit undersold itself.** The daemon absorbed 6 of my 10
   staged files mid-commit (`54684bc92`), so `8a21a0812` contains only 4 files
   while its message describes the whole wave. Content is safe in history;
   the commit/message pairing is just misleading for future archaeology.
5. **The guard function landed one indent level too deep** (edit-tool
   whitespace normalization on first insert). Bash-valid but style-wrong;
   dedented immediately. Caught by review, not by any gate.

## e) WHAT WE SHOULD IMPROVE

1. **Classify probe failures honestly:** `git ls-remote` failing on auth
   (askpass error, private mirror, no token) must read "AUTH?" not "ROT" —
   repo-gone and no-credentials are different operator actions.
2. **Verify the baseline exists before building a detector on it** (dist
   branch, stamp file, golden file — same rule as any single-source claim).
3. **Commit at phase boundaries in cross-repo work.** The interruption found
   the hub working tree mid-phase; only the hub daemon's auto-commit saved
   the state from being an uncommitted mess. Authored commits at each phase
   boundary would have kept history legible.
4. **The tag zip-content guard only protects FUTURE tags.** The two poisoned
   tags remain published until the owner decides (§g); the guard plus the
   weekly proxy probe idea (handoff §e1) is the full fix — probe not built.
5. **RC files as tree-local contracts:** go-cqrs-lite's gate writes the
   full-error rc INTO the lint tree. That pattern (config travels with the
   artifact, not the ambient directory) is worth keeping canonical for the
   hub's per-source lint step — avoid depending on linter rc-walk-up behavior.
6. **Fixture-builder review:** every fixture should be asserted to exist in
   the intended shape BEFORE the behavior under test runs (I added
   `test -e` for the control-char file after the failure; the mkdir bug and
   the literal-filename bug were both fixture-side, not code-side).

## f) Next tasks (harvest candidates, ≤50)

| #  | Task                                                                                                                                                                      | Impact   | Effort |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ |
| 1  | Finish hub build.yml: per-source plain-refs governance lint step (optional `lint` field in sources.json)                                                                  | High     | S      |
| 2  | Write mesh-demo entries into hub sources.json (validated this session) + rebuild-once to publish dist                                                                     | High     | S      |
| 3  | Probe polish: separate AUTH vs ROT classification (token-aware ls-remote)                                                                                                 | Medium   | XS     |
| 4  | Hub README: merge semantics (owners union, ref canonicalization, strict gate), probe usage, lint field                                                                    | High     | S      |
| 5  | Hub TODO_LIST: unblock the 2 [blocked:upstream] items (manifest diff available; exporter upgrade shipped) — rewrite the linter re-arm item as resolved-by-per-source-lint | Medium   | S      |
| 6  | Hub scheduled workflow for check-stale-sources (cron matching 8h mirror cadence, workflow_dispatch)                                                                       | Medium   | S      |
| 7  | f72: SystemNix plan cross-reference note (addendum linking the pareto plan + hub state)                                                                                   | Low      | XS     |
| 8  | docs-health: TODO_LIST data-mesh strike-through T01–T22 (T22–T24 hub items done/partial per this report)                                                                  | High     | S      |
| 9  | docs-health: harvest handoff §f (30 rows) + this report's §f into TODO_LIST/ROADMAP; plan doc completion addendum                                                         | High     | M      |
| 10 | Final `nix run .#verify` closing gate for the whole wave                                                                                                                  | Critical | M      |
| 11 | Attempt `check-md-go` vendorHash repair (nix hash update)                                                                                                                 | Medium   | S      |
| 12 | Investigate WHY hub build workflow never published dist (runner provisioned? secrets set? first run failed?) — may need owner/forgejo access                              | Critical | M      |
| 13 | check-architecture-changes.sh: switch to structured `catalog.index.json` diff (now shipped by exporter; hub TODO [blocked:upstream] #2)                                   | Medium   | M      |
| 14 | Release train: cut catalog/v4.6+ tag wave (owner-gated, §g1) — unblocks replace-strip + hub pin bump                                                                      | Critical | M      |
| 15 | Pin-sweep after release: strip mesh-demo `replace ../../catalog`, bump examples                                                                                           | High     | S      |
| 16 | Poisoned-tag surgery: retract+re-cut tursoengine/v4.2.0; decide storage/v4.10.0 re-cut vs retraction (owner-gated, §g3)                                                   | Critical | S      |
| 17 | Weekly "proxy resolves every module's latest tag" probe (handoff §e1)                                                                                                     | High     | M      |
| 18 | changelog gate: flag uncited consumer-visible surface (api_surface diff vs CHANGELOG) — handoff §e2                                                                       | Medium   | M      |
| 19 | goal-shaped-app: activate IVM view + boot test once turso tag is clean                                                                                                    | Medium   | S      |
| 20 | mesh-demo: system.New-backed variant (runtime coeffect gate demo)                                                                                                         | Medium   | M      |
| 21 | E019: non-literal DataProduct scanner support                                                                                                                             | Low      | M      |
| 22 | docserver: DataProduct badges/hidden flag rendering                                                                                                                       | Low      | S      |
| 23 | check-eventcatalog: lockfile-pin the linter install                                                                                                                       | Low      | S      |
| 24 | Sweep daemon formatting anomalies (spot-check vs `nix fmt --fail-on-change`)                                                                                              | Low      | S      |
| 25 | Fix cqrs-lint D007 repair-path "unsafe path" failures (report-only today)                                                                                                 | Medium   | M      |
| 26 | Address 3 high Dependabot vulnerabilities on master                                                                                                                       | High     | M      |
| 27 | Push decision: ~10 local commits in go-cqrs-lite + hub commits awaiting authorization                                                                                     | High     | S      |
| 28 | bank-sync + cqrs-htmx: adopt the new exporter options (owners, plain-refs lint command) once catalog tag lands                                                            | Medium   | S      |

## g) Questions I cannot answer myself

1. **The hub build workflow has NEVER published a dist branch** (discovered
   this session — no `dist` ref on origin, so no build-stamp baseline exists;
   `catalog.home.lan` is presumably serving stale or empty content). The
   workflow + setup script exist (`setup-forgejo.sh` documents a manual
   root-run provisioning step: PAT minting, mirror creation, Actions secret).
   Is the runner/secrets provisioning simply not done yet, or did runs fail?
   I cannot read the Forgejo runner's logs or secrets from here — if you want
   the hub actually live, the next move is either yours (run
   `scripts/setup-forgejo.sh` / check the Actions tab) or an explicit go-ahead
   for me to debug what is reachable from this machine.
2. **Release trigger (carried over, now load-bearing):** may I cut the
   catalog/v4.6+ tag wave? The new zip-content guard is in place, so the cut
   is safer than ever — but it publishes to the proxy and unblocks the
   mesh-demo replace-strip + hub pin-bump. The mesh-demo hub onboarding can
   ride `master` meanwhile (validated from a fresh clone), so nothing is
   hard-blocked; the replace directive just stays pre-release until you say
   "cut".
3. **Poisoned-tag surgery (carried over, unchanged):** for
   `metaengine/tursoengine/v4.2.0` (binary junk zip) and `storage/v4.10.0`
   (deleted after push): retract-and-re-cut, or re-create
   `storage/v4.10.0` at the v4.10.1 content commit? Re-creating a deleted tag
   re-poisons anyone who cached the absence; retraction leaves published
   `system/v4.9.0`'s graph broken until the next system tag. Consumers'
   pain vs tag aesthetics — your call, still untouched by me.

---

**Report discipline:** this file is a snapshot; §f belongs in TODO_LIST/ROADMAP
via docs-health HARVEST once execution resumes. Immediate next actions on
resume: finish the hub phase (lint step, sources.json, README/TODO), then f72,
then the docs-health harvest, then the final `#verify` gate.
