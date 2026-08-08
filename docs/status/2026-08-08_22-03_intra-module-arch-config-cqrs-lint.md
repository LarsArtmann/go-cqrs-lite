# Intra-Module Architecture Config for cmd/cqrs-lint — Honest Status

**Date:** 2026-08-08 22:03
**Session scope:** Add `.go-arch-lint.yml` for `cmd/cqrs-lint` (18 production packages), verify enforcement, discover CI gap.

---

## a) FULLY DONE

1. **Mapped all 18 production packages** in `cmd/cqrs-lint` and their intra-module import dependencies via grep analysis (non-test `.go` files only). Confirmed: no circular dependencies, no cross-imports between rule category sub-packages.

2. **Designed 5-layer model** matching the actual import graph:
   - **L0 (leaves):** `analyzer`, `fix`, `ruletest`, `suppression` — no internal deps
   - **L1:** `lintutil` → analyzer
   - **L2:** 11 rule categories (adoption, api, architecture, boilerplate, consistency, correctness, performance, resilience, security, testrules, version) → analyzer, lintutil
   - **L3:** `rules` root → analyzer + all 11 rule categories
   - **L4:** `core` (main package) → analyzer, fix, rules, suppression

3. **Created `cmd/cqrs-lint/.go-arch-lint.yml`** — 132-line config with version 3 schema, matching the style of existing configs (`storage/`, `catalog/`, `event/`, `command/`, `kv/`, `middleware/`).

4. **Verified component mapping** via `go-arch-lint mapping` — all 18 packages resolve to distinct components with correct paths.

5. **Verified enforcement passes** via `go-arch-lint check` — "OK - No warnings found".

6. **Negative-tested enforcement** — removed `fix` from core's deps, confirmed go-arch-lint caught `run.go:15` importing `pkg/fix`. Restored config. This proves the config is not vacuous.

7. **Ran full `scripts/check-arch.sh`** — both layers pass (Layer 1 cross-module + Layer 2 per-module, now 7 modules including the new `cqrs-lint`).

8. **Updated `TODO_LIST.md`** — marked the item as `[x]` with summary.

9. **Auto-commit daemon committed** as `ba4bc862a` with a good message.

---

## b) PARTIALLY DONE

### ⚠️ Layer 2 (go-arch-lint) is NOT wired into CI or the verify gate

This is the single biggest finding. While the config works locally, **it will not catch violations in CI**.

| Gate | What it runs | Layer 2? |
|------|-------------|----------|
| `.github/workflows/ci.yml` job `module-layers` | `bash scripts/check-module-layers.sh` | **No** — Layer 1 only |
| `.github/workflows/ci.yml` job (main quality) | `nix run .#check-layers` | **No** — Layer 1 only |
| `nix run .#verify` | `nix run .#check-layers` | **No** — Layer 1 only |
| `nix run .#verify-fast` | `nix run .#check-layers` | **No** — Layer 1 only |
| `nix run .#check-arch` | `scripts/check-arch.sh` (both layers) | **Yes** — but nobody calls it |

The `nix run .#check-arch` app exists in `flake.nix:759` but is **orphaned** — not referenced by CI, verify, or verify-fast. This means all 7 per-module go-arch-lint configs (`event`, `command`, `kv`, `middleware`, `storage`, `catalog`, and now `cmd/cqrs-lint`) are **local-only enforcement**.

The fix is simple: replace `nix run .#check-layers` with `nix run .#check-arch` in the verify gate (and optionally add it to CI). `check-arch.sh` is a superset — it runs Layer 1 (`check-module-layers.sh`) then Layer 2 (per-module go-arch-lint).

**Impact:** I created a config that is correct and tested but effectively dead from a CI perspective. The config will only help developers who know to run `nix run .#check-arch` manually.

---

## c) NOT STARTED

- **Wire `check-arch` into the verify gate and CI** — the orphaned `#check-arch` app needs to replace or supplement `#check-layers` in `flake.nix` verify/verify-fast and `.github/workflows/ci.yml`. This is the critical follow-up.
- **Add configs for `metaengine/`** — 16+ production files in the root package, no intra-module enforcement. Identified as candidate in the 08:23 status doc.
- **Add configs for `stack/` presets** — 11 production files, no intra-module enforcement.
- **Meta-test: every `.go-arch-lint.yml` is valid** — no test asserts that the configs parse or that all declared components match real packages.
- **Meta-test: go-arch-lint is installed** — CI doesn't install go-arch-lint; the `#check-arch` app doesn't include it as a nix dependency either (it only lists `goPkg` and `pkgs.bash`). Running `nix run .#check-arch` in CI would fail with "go-arch-lint: command not found".

---

## d) TOTALLY FUCKED UP

Nothing in this session. The config is correct, verified, and negative-tested. The CI gap is a pre-existing issue I discovered, not one I created.

---

## e) WHAT WE SHOULD IMPROVE

1. **The verify gate should use `#check-arch` not `#check-layers`** — this is a one-line change in `flake.nix` (2 occurrences: verify + verify-fast). `check-arch.sh` already calls `check-module-layers.sh` as Layer 1, so it's a strict superset.
2. **CI should add a `check-arch` step** — the `module-layers` CI job should run `nix run .#check-arch` instead of just `check-module-layers.sh`.
3. **`#check-arch` needs go-arch-lint as a nix dependency** — the app definition at `flake.nix:759` lists `[goPkg pkgs.bash]` but not `pkgs.go-arch-lint` (or wherever it comes from). The tool is currently in the user's system PATH (`/run/current-system/sw/bin/go-arch-lint`), which won't exist in CI.
4. **Test files are excluded from all configs** — consistent with existing configs, but means test-only architecture violations (e.g., test files importing across rule categories) are not caught. This is a known tradeoff across all 7 modules.
5. **`pkg/ruletest` is included as a production component** but is purely test infrastructure (only imported by `*_test.go` files). It's correctly modeled as a Layer 0 leaf, but its inclusion is debatable.

---

## f) Up to 50 things to do next

### Critical — make Layer 2 actually enforced

1. **Add go-arch-lint to `#check-arch` nix deps** — `flake.nix:759` needs the tool as a dependency, not just PATH luck.
2. **Replace `#check-layers` with `#check-arch` in verify gate** — `flake.nix` verify + verify-fast (2 lines).
3. **Replace `check-module-layers.sh` with `nix run .#check-arch` in CI** — `.github/workflows/ci.yml` `module-layers` job.
4. **Verify `nix run .#check-arch` works in a clean shell** — confirm go-arch-lint resolves from nix, not system PATH.

### Architecture configs for other modules

5. **Add `.go-arch-lint.yml` for `metaengine/`** — 16+ production files, planner.go + engine.go + dsl.go, complex internal structure.
6. **Add `.go-arch-lint.yml` for `stack/`** — 11 production files, composition layer with clear internal deps.
7. **Add `.go-arch-lint.yml` for `decider/`** — repository.go, cache.go, execute.go. Moderate complexity.
8. **Add `.go-arch-lint.yml` for `projectionhost/`** — worker.go, host.go, deadletter.go. Multi-package internal structure.
9. **Audit which of the 78 modules would benefit from intra-module configs** — most single-package modules don't need one.

### Meta-tests and guards

10. **Add a meta-test: every `.go-arch-lint.yml` is parseable** — iterate over all configs, feed to a YAML parser, assert no errors.
11. **Add a meta-test: every module with 3+ production packages has a `.go-arch-lint.yml`** — prevents the gap from recurring.
12. **Add a meta-test: every declared component in a config matches a real Go package** — catches stale configs after package renames/deletes.
13. **Add go-arch-lint version pin to flake.nix** — currently uses whatever is in PATH; should be reproducible.

### Documentation

14. **Document which modules have go-arch-lint configs** — add a section to AGENTS.md or CONTRIBUTING.md listing all 7 (soon 8+) configured modules.
15. **Document the two-layer enforcement model in CONTRIBUTING.md** — the `check-arch.sh` script header is the only documentation; a contributor-facing explanation would help.
16. **Update the 08:23 status doc's remaining items** — items 11-14 from that doc (go-arch-lint for metaengine, stack, audit EXCEPTIONS, verify EXCEPTIONS).

### Pre-existing issues noticed (not caused by this session)

17. **Metaengine tier split-brain** — `check-module-layers.sh` says `LAYER[metaengine]=0`, ADR-0046 amendment says Tier 3. Still unresolved from the 08:23 session.
18. **"44 of 78" is wrong in SEVEN-TIER-MODEL.md** — should be "48 of 78". Unresolved from the 08:23 session.
19. **ADR-0046 stale module counts** — still says "68 modules" in 3 places. Unresolved.
20. **CI `module-layers` job doesn't use Nix for go-arch-lint** — even if we add it, the CI runner needs the tool installed.

---

## g) Questions I CANNOT figure out myself

### Q1: Should the verify gate replace `#check-layers` with `#check-arch`, or should they coexist?

`check-arch.sh` is a strict superset of `check-module-layers.sh` (it calls Layer 1 internally). Replacing `#check-layers` with `#check-arch` in the verify gate is cleaner but means a go-arch-lint failure blocks the entire verify gate. Is that the desired behavior, or should Layer 2 be a separate opt-in gate?

### Q2: Where does go-arch-lint come from in CI?

The tool is at `/run/current-system/sw/bin/go-arch-lint` on this machine (NixOS system package). In CI (Ubuntu), it would need to be installed. Is it available as a nixpkgs package (`pkgs.go-arch-lint`), or does it need a `go install github.com/fe3dback/go-arch-lint/...@latest` step? The `#check-arch` app definition doesn't list it as a dependency, so it would fail in any shell that doesn't have it in PATH.

### Q3: Should test files be included in go-arch-lint enforcement?

All 7 existing configs exclude `*_test.go`. This means a test in `pkg/rules/adoption` could import `pkg/rules/api` without detection. Is this intentional (test files get more freedom) or an oversight that should be corrected across all configs?
