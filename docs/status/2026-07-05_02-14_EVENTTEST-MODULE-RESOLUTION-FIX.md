# Status Report: eventtest Module Resolution Fix — COMPLETE

**Date:** 2026-07-05 02:14
**Session scope:** Fix the `event/v3/eventtest` nested-module "unknown revision" error
**Verdict:** Root cause identified, fixed, pushed, and verified end-to-end.

---

## Executive Summary

The `eventtest` module was **permanently unresolvable from VCS** due to a directory/module-path mismatch. This was NOT a Go nested-module limitation (as the repo docs previously claimed) — it was a misconfigured directory location combined with wrong-version tags and placeholder pseudo-versions. All three defects are now fixed. External consumers can `go get` eventtest@v0.1.0 successfully.

---

## a) FULLY DONE ✅

| #   | Work item                                             | Evidence                                                                                                                                                                                                                                                       |
| --- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Root cause diagnosis**                              | Confirmed via [Go module spec](https://go.dev/ref/mod): directory must match module path. The `/v3/` in `.../event/v3/eventtest` is NOT a major-version suffix (not the last path element), so Go looks for `event/v3/eventtest/go.mod` — which never existed. |
| 2   | **Directory move**                                    | `git mv event/eventtest → event/v3/eventtest` (preserves all import paths, only physical location changes)                                                                                                                                                     |
| 3   | **go.work updated**                                   | `./event/eventtest` → `./event/v3/eventtest`                                                                                                                                                                                                                   |
| 4   | **All 20 consumer go.mod replace directives updated** | `=> ../event/eventtest` → `=> ../event/v3/eventtest` across decider, middleware, signing, integration, stack/_, storage/turso, transport/_, watermill, example/\*                                                                                              |
| 5   | **eventtest go.mod: real published versions**         | Changed `v3.0.0-00010101000000-000000000000` placeholders → `v3.5.0` for event/id/snapshot deps (external consumers can't use replace directives)                                                                                                              |
| 6   | **sync-replaces.sh script created**                   | `scripts/sync-replaces.sh` — ensures every go.mod has replace directives for all transitive sibling deps. Uses `go mod edit -replace` (official, reliable).                                                                                                    |
| 7   | **Convergence tidy**                                  | 3 iterations of `go mod tidy -e` across all 53 modules until dependency graph stabilized. Fixed 28 pre-existing GOWORK=off build failures.                                                                                                                     |
| 8   | **Tag fix**                                           | Deleted wrong tags `event/v3/eventtest/v3.3.0` and `v3.5.0` (v3 on a v0-path module). Created correct `event/v3/eventtest/v0.1.0`.                                                                                                                             |
| 9   | **Remote operations**                                 | Pushed 6 commits to master. Pushed v0.1.0 tag. Deleted wrong v3 tags from remote.                                                                                                                                                                              |
| 10  | **Documentation corrected**                           | Fixed false "Go doesn't support nested modules" claim in AGENTS.md and `docs/planning/2026-06-08_06-47_PACKAGING_HYGIENE_AND_ADOPTION_UNLOCK.md`. Added ADR-0045. Updated event/README.md, cmd/api-stability/README.md.                                        |
| 11  | **End-to-end external resolution verified**           | Clean consumer project: `go get ...@v0.1.0` + `go build` with import → EXIT 0                                                                                                                                                                                  |
| 12  | **All 53 modules build GOWORK=off**                   | Verified post-convergence                                                                                                                                                                                                                                      |
| 13  | **Broad test suite passes**                           | event, command, decider, id, dispatcher, codec, middleware, signing, schema, snapshot, query, kv, graph, scenario, projection — all green                                                                                                                      |
| 14  | **BuildFlow pre-commit checks pass**                  | 30/30 across all 54 modules (auto-fixed 37 lint issues)                                                                                                                                                                                                        |

---

## b) PARTIALLY DONE 🟡

| #   | Item                           | Status                                                               | What remains                                                                                                                                                                                                             |
| --- | ------------------------------ | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **GOPRIVATE documentation**    | eventtest resolves via git with `GOPRIVATE=github.com/larsartmann/*` | The repo is **private** — consumers MUST set GOPRIVATE. This is documented in ADR-0045 but not yet in README.md "Getting Started" or consumer-facing SKILL.md.                                                           |
| 2   | **Binary artifacts in git**    | BuildFlow recompilation produced modified binaries                   | 4 example binaries (`example/*/deployer-first*`, `deriver`, `graph-demo`, `projectionhost`) are tracked in git and got modified by BuildFlow. These should be `.gitignore`d, not committed. Uncommitted in working tree. |
| 3   | **SEC consumer feedback file** | `docs/feedback/sec-consumer-feedback.md` exists (untracked)          | Appears to be pre-existing consumer feedback. Not reviewed this session.                                                                                                                                                 |

---

## c) NOT STARTED ⬜

| #   | Item                                                                                                                                                                                                                                                                                        |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Making the repo public** — would enable Go proxy caching (faster resolution, checksum DB). Currently private → consumers must use GOPRIVATE + git auth. Decision deferred to user.                                                                                                        |
| 2   | **`nix flake check` in the nix store** — the original trigger. The eventtest fix unblocks this, but actual nix-store sandbox build was not tested (requires `nix build` with FOD + vendored deps). The structural blockers (directory mismatch, unresolvable versions) are eliminated.      |
| 3   | **Flake buildGoModule for the library itself** — the flake has no `buildGoModule` for the main library (only `benchstat`). The `packages.default` is a no-op `stdenvNoCC.mkDerivation`. For true nix-store reproducibility, a vendored `buildGoModule` or `go-modules` FOD would be needed. |
| 4   | **CI verification** — `.github/workflows/ci.yml` was not run. The GOWORK=off per-module build pattern matches CI, but the actual CI pipeline wasn't triggered.                                                                                                                              |

---

## d) TOTALLY FUCKED UP 💥

| #   | What                                                       | Impact                                                                                                                                                                                                                                                                      | Status                                                                                            |
| --- | ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| 1   | **The original "Go doesn't support nested modules" claim** | Sent the project down the wrong path for weeks. Multiple status reports, planning docs, and brutal-self-reviews all treated this as an unfixable Go limitation. It was a simple directory misconfiguration. The real fix took ~30 minutes once the spec was read correctly. | **Fixed.** False claims corrected in AGENTS.md, planning docs. ADR-0045 documents the truth.      |
| 2   | **The `sync-replaces.sh` first draft**                     | Used fragile `sed` for replace-block manipulation. Threw `unterminated s command` errors.                                                                                                                                                                                   | **Replaced** with `go mod edit -replace` approach (official, reliable). Script now works cleanly. |
| 3   | **Convergence tidy cascade**                               | Tidying individual modules one-at-a-time caused version drift — fixing one module's go.sum broke its dependents, cascading to 20 failures. Took 3 iterations to stabilize.                                                                                                  | **Fixed.** Convergence loop reached stable state. All 53 modules pass.                            |

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Process

1. **Read the spec first.** The Go module spec is clear: directory must match path suffix (minus major-version). An hour reading `go.dev/ref/mod` would have saved weeks of misdiagnosis.
2. **Don't trust status reports as ground truth.** Multiple "comprehensive" status reports repeated the false nested-module claim. Always verify claims against primary sources.
3. **Monorepo tidy needs convergence.** Tidying modules individually in a multi-module workspace creates oscillation. A convergence loop (tidy-all until stable) is the correct pattern. This should be automated.
4. **Binary artifacts should not be in git.** BuildFlow recompiles examples, producing modified tracked binaries. These should be `.gitignore`d.

### Technical

5. **The `go mod tidy -e` requirement is still there.** Modules that transitively depend on eventtest through event's test files still need `-e`. This is a minor inconvenience, not a correctness issue, but a clean `go mod tidy` (without `-e`) would be better.
6. **Pseudo-version placeholders (`v3.0.0-00010101000000-000000000000`) are a code smell.** They only work with `replace` directives (local-only). Any module intended for external consumption should reference real published versions. The other 52 modules still have placeholders in their require blocks (masked by replaces). If any of those modules need to be consumed externally without replaces, they'll hit the same issue.

---

## f) Top 25 Things to Do Next

| #   | Task                                                                                                                                                                                                                                                                                   | Priority | Effort   |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | -------- |
| 1   | **`.gitignore` the example binaries** (`example/*/deployer-first*`, `deriver`, `graph-demo`, `projectionhost`) — they're tracked and cause noise                                                                                                                                       | High     | 5m       |
| 2   | **Test `nix flake check` actually passes** now that eventtest is fixed                                                                                                                                                                                                                 | High     | 10m      |
| 3   | **Test `nix run .#build` and `nix run .#test`** — the actual CI commands                                                                                                                                                                                                               | High     | 10m      |
| 4   | **Add `GOPRIVATE=github.com/larsartmann/*` to consumer-facing docs** (README.md Getting Started, SKILL.md)                                                                                                                                                                             | High     | 15m      |
| 5   | **Decide: make repo public or keep private.** Public enables Go proxy. Private requires GOPRIVATE for every consumer.                                                                                                                                                                  | High     | Decision |
| 6   | **Audit ALL other modules for the same pseudo-version problem** — eventtest was fixed, but 52 other modules still use placeholders masked by replaces                                                                                                                                  | Med      | 30m      |
| 7   | **Add `scripts/sync-replaces.sh` to CI** — run after go.mod changes to catch missing replaces early                                                                                                                                                                                    | Med      | 15m      |
| 8   | **Add convergence-tidy script** (`scripts/tidy-all.sh`) — automate the 3-iteration tidy loop                                                                                                                                                                                           | Med      | 20m      |
| 9   | **Review `docs/feedback/sec-consumer-feedback.md`** — untracked consumer feedback file exists, not yet addressed                                                                                                                                                                       | Med      | 20m      |
| 10  | **Run `.github/workflows/ci.yml` locally or trigger via push** — verify CI pipeline passes with the fix                                                                                                                                                                                | Med      | 15m      |
| 11  | **Consider a `buildGoModule` FOD in flake.nix** for true nix-store sandbox reproducibility                                                                                                                                                                                             | Med      | 1-2h     |
| 12  | **Eliminate the `go mod tidy -e` requirement** — the `-e` flag is needed because event's test files import eventtest, creating a graph edge that tidy follows. Could be fixed by moving event's test files that use eventtest into eventtest itself, or by splitting event's test deps | Low      | 1-2h     |
| 13  | **Tag other modules at current HEAD** — if external consumers need stable versions of modules other than eventtest, they need tags too                                                                                                                                                 | Low      | 30m      |
| 14  | **Review the `example/deployer-first-heterogeneous/deployer-first-heterogeneous` binary** (32MB tracked in git!) — definitely should not be committed                                                                                                                                  | Low      | 5m       |
| 15  | **Update `docs/sessions/SESSION_MILESTONES.md`** with the eventtest fix                                                                                                                                                                                                                | Low      | 10m      |
| 16  | **Add `scripts/sync-replaces.sh --check` mode** for CI (exit 1 if missing replaces detected)                                                                                                                                                                                           | Low      | 15m      |
| 17  | **Consider `GOFLAGS=-mod=mod` in devShell** to avoid `-e` flag need                                                                                                                                                                                                                    | Low      | 10m      |
| 18  | **Audit `replace` directives for correctness** — some may point to wrong relative paths after future directory moves                                                                                                                                                                   | Low      | 20m      |
| 19  | **Document the module versioning strategy** — which modules are v0-path (eventtest) vs v3-path (all others) and why                                                                                                                                                                    | Low      | 20m      |
| 20  | **Add a `make tags` / `nix run .#tags` command** to list all module versions for release management                                                                                                                                                                                    | Low      | 15m      |
| 21  | **Review whether eventtest should be v0 or v3** — currently v0 because path ends in `/eventtest`. Could rename to `.../eventtest/v3` to make it a v3 module, but that changes the import path for ~92 files                                                                            | Low      | Decision |
| 22  | **Clean up historical docs** — many planning/status docs still reference `event/eventtest/` directory path (cosmetic, they're historical)                                                                                                                                              | Low      | 30m      |
| 23  | **Add integration test for external resolution** — a CI step that simulates `go get` from a clean consumer                                                                                                                                                                             | Low      | 30m      |
| 24  | **Review the `go.work` replace for genproto** — still needed? Check if cockroachdb/errors dropped the monolithic genproto dep                                                                                                                                                          | Low      | 10m      |
| 25  | **Consider `go work sync`** to align module versions across the workspace automatically                                                                                                                                                                                                | Low      | 10m      |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the repo be made public?**

The repo is currently **private** (`gh repo view` confirms `isPrivate:true`). This has major implications:

- **If public:** The Go module proxy (`proxy.golang.org`) will cache all modules. Consumers get fast resolution, checksum verification, and don't need git auth. The "unknown revision" class of problem disappears entirely (proxy serves from cache).
- **If private:** Every consumer MUST set `GOPRIVATE=github.com/larsartmann/*` and have SSH/HTTPS auth to the repo. This works (verified this session) but adds friction for every consumer.

The AGENTS.md says this is a **library consumers import** — which implies public. But it's private. I cannot determine whether this is intentional (early development, not ready for public consumption) or an oversight. This is a business/product decision, not a technical one.

**What I'd do with the answer:** If public → verify proxy caches all modules, update README with standard `go get` instructions. If private → add a prominent GOPRIVATE setup section to README + SKILL.md + every module README.

---

## Session Metrics

| Metric                  | Value                                                   |
| ----------------------- | ------------------------------------------------------- |
| Commits                 | 6                                                       |
| Files changed           | ~145 (across all commits)                               |
| Modules fixed           | 53 (all build GOWORK=off)                               |
| Tags created            | 1 (`event/v3/eventtest/v0.1.0`)                         |
| Tags deleted            | 2 (`v3.3.0`, `v3.5.0` — wrong major version)            |
| Pre-existing bugs fixed | 28 (GOWORK=off build failures from missing replaces)    |
| Documentation corrected | 4 files (AGENTS.md, 2× README, planning doc) + ADR-0045 |
| Scripts created         | 1 (`scripts/sync-replaces.sh`)                          |
| Tests run               | 18 packages, all pass                                   |
| External resolution     | Verified end-to-end from clean consumer                 |
