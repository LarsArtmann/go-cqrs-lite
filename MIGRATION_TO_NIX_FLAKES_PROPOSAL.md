# Migration to Nix Flakes — Proposal

**Date:** 2026-04-09
**Status:** Draft
**Scope:** `go-cqrs-lite` — Go 1.26.2 CQRS library

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Current State Analysis](#2-current-state-analysis)
3. [Why Nix Flakes](#3-why-nix-flakes)
4. [Proposed Architecture](#4-proposed-architecture)
5. [File-by-File Implementation Plan](#5-file-by-file-implementation-plan)
6. [Migration Steps](#6-migration-steps)
7. [GitHub Actions CI Migration](#7-github-actions-ci-migration)
8. [Risks & Mitigations](#8-risks--mitigations)
9. [Decision Matrix](#9-decision-matrix)
10. [Timeline](#10-timeline)

---

## 1. Executive Summary

Migrate `go-cqrs-lite` from Makefile + ad-hoc tooling to **Nix Flakes** as the single source of truth for development environment, build, formatting, linting, and CI. This eliminates the "works on my machine" problem, pins every tool to a hash, and gives contributors a one-command onboarding (`nix develop`).

**Key outcome:** A single `flake.nix` + `flake.lock` that defines the entire toolchain — Go 1.26, golangci-lint, gofumpt, goimports, golines, buildflow, and more — with deterministic, reproducible outputs across macOS and Linux.

---

## 2. Current State Analysis

### 2.1 Toolchain Inventory

The following table catalogs every tool and configuration currently used, with its source and pinning status.

| Tool / Config    | Purpose                            | Source                       | Pinned?                               |
| ---------------- | ---------------------------------- | ---------------------------- | ------------------------------------- |
| Go 1.26.2        | Compiler & runtime                 | `go.mod`                     | Partial (go.mod declares `go 1.26.2`) |
| golangci-lint v2 | Linting (126 linters enabled)      | `.golangci.yml`              | No — CI uses `@latest`, local varies  |
| gofumpt          | Code formatting                    | `.golangci.yml` formatters   | No                                    |
| goimports        | Import ordering                    | `.golangci.yml` formatters   | No                                    |
| golines          | Line length formatting             | `.golangci.yml` formatters   | No                                    |
| gci              | Import grouping                    | `.golangci.yml` formatters   | No                                    |
| go vet           | Static analysis                    | Makefile                     | Via Go toolchain                      |
| buildflow        | Semantic code analysis             | `.buildflow.yml`             | No — custom tool                      |
| branching-flow   | Git workflow automation            | AGENTS.md reference          | No — custom tool                      |
| trash            | Safe file deletion (replaces `rm`) | Makefile `clean` target      | No                                    |
| Makefile         | Build orchestration                | `Makefile` (85 lines)        | N/A                                   |
| codecov          | Coverage upload                    | `.github/workflows/test.yml` | No — CI action                        |

### 2.2 Makefile Targets (Current)

The `Makefile` provides 15 targets. All must be represented in the Nix flake.

| Target          | Command                                    | Nix Equivalent                          |
| --------------- | ------------------------------------------ | --------------------------------------- |
| `all`           | `fmt lint test build`                      | `nix flake check` + `nix build`         |
| `build`         | `go build ./...`                           | `nix build`                             |
| `test`          | `go test ./... -v`                         | `nix develop --run "go test ./... -v"`  |
| `test-race`     | `go test ./... -race`                      | devShell alias                          |
| `test-short`    | `go test ./... -short`                     | devShell alias                          |
| `coverage`      | `go test ./... -coverprofile=coverage.out` | devShell alias                          |
| `coverage-html` | `coverage` + `go tool cover -html`         | devShell alias                          |
| `lint`          | `golangci-lint run`                        | `nix develop --run "golangci-lint run"` |
| `fmt`           | `gofmt -w .`                               | `nix fmt` (via treefmt-nix)             |
| `imports`       | `goimports -w .`                           | Part of `nix fmt`                       |
| `vet`           | `go vet ./...`                             | Part of `nix flake check`               |
| `mod-tidy`      | `go mod tidy`                              | devShell alias                          |
| `mod-verify`    | `go mod verify`                            | devShell alias                          |
| `clean`         | `trash coverage.*` + `go clean -testcache` | devShell alias                          |
| `check`         | `fmt vet lint test`                        | `nix flake check`                       |
| `ci`            | `build test-race lint`                     | `nix flake check`                       |

### 2.3 GitHub Actions Workflows

Two workflows exist:

**`.github/workflows/lint.yml`** — 3 jobs:

- `golangci`: golangci-lint via `golangci-lint-action@v6`
- `fmt`: `gofmt -l .` diff check
- `imports`: `goimports -l .` diff check

**`.github/workflows/test.yml`** — 2 jobs:

- `test`: test + race + coverage + codecov upload
- `build`: `go build ./...` + `go vet ./...`

Both use `actions/setup-go@v5` with Go 1.26 and `actions/cache@v4` for module caching.

### 2.4 Special Considerations

| Concern                        | Detail                                                                                                           | Impact on Nix                                  |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| **GOWORK=off**                 | `~/go.work` exists but excludes this project; all `go` commands need `GOWORK=off`                                | Must set in `shellHook`                        |
| **Go experiment flags**        | `.golangci.yml` uses build tags: `goexperiment.goroutineleakprofile`, `goexperiment.jsonv2`, `goexperiment.simd` | Must pass `GOFLAGS=-tags=...` or set in shell  |
| **Multi-module**               | `example/user/` has its own `go.mod` with `replace` directive back to root                                       | Must handle as separate Nix package            |
| **buildflow**                  | Custom tool (github.com/larsartmann/buildflow), not in nixpkgs                                                   | Need custom derivation or `pkgs.buildGoModule` |
| **branching-flow**             | Custom tool, likely same source as buildflow                                                                     | Same approach                                  |
| **trash**                      | Used in Makefile `clean` target (safe delete)                                                                    | Available in nixpkgs                           |
| **Zero-dependency philosophy** | Only stdlib + `google/uuid` + `cockroachdb/errors` + `go-json-experiment/json`                                   | Small `vendorHash`, fast builds                |

### 2.5 Dependencies (go.mod)

```
Direct:
  github.com/cockroachdb/errors     v1.12.0
  github.com/go-json-experiment/json v0.0.0-20260214004413-d219187c3433
  github.com/google/uuid             v1.6.0

Indirect (8 transitive dependencies)
```

Minimal dependency tree — ideal for Nix.

---

## 3. Why Nix Flakes

### 3.1 Problems Solved

| Problem                     | Current Impact                                                                     | Nix Solution                          |
| --------------------------- | ---------------------------------------------------------------------------------- | ------------------------------------- |
| **Tool version drift**      | CI uses `@latest`, developers use whatever's installed                             | `flake.lock` pins exact hashes        |
| **Onboarding friction**     | "Install Go 1.26, golangci-lint, gofumpt, goimports, golines, buildflow, trash..." | `nix develop` — one command           |
| **CI inconsistency**        | `setup-go@v5` + `golangci-lint-action@v6` diverge from local tools                 | Same `flake.nix` in CI and local      |
| **Format checker mismatch** | `gofmt` in CI vs `gofumpt` locally                                                 | `nix fmt` uses exact same versions    |
| **Reproducibility**         | No guarantee build works with different tool versions                              | Hash-locked, bit-for-bit reproducible |
| **macOS ↔ Linux parity**    | Developers on macOS, CI on Ubuntu                                                  | Multi-system `flake.nix`              |

### 3.2 What Nix Flakes Give This Project

1. **Deterministic dev shell** — Every contributor gets identical Go + toolchain versions
2. **`nix fmt`** — Single formatter command (via `treefmt-nix`) replacing 4 separate tools
3. **`nix flake check`** — One command for lint + test + vet + format check (replaces `make ci`)
4. **`nix build`** — Reproducible build artifact
5. **direnv + nix-direnv** — Automatic shell activation on `cd`
6. **CI via Nix** — Replace 2 workflows with 1, using same `flake.nix`

### 3.3 Trade-offs

| Pro                    | Con                                              |
| ---------------------- | ------------------------------------------------ |
| Reproducibility        | Learning curve for Nix                           |
| One-command onboarding | `flake.lock` binary (occasional updates)         |
| Single source of truth | Custom tools (buildflow) need custom derivations |
| Cross-platform parity  | Nix itself must be installed                     |
| Replaces Makefile      | Makefile is simpler for non-Nix users            |

---

## 4. Proposed Architecture

### 4.1 File Structure

```
go-cqrs-lite/
├── flake.nix                    # Main flake — inputs, outputs, packages, devShells
├── flake.lock                   # Auto-generated — pins all dependencies
├── treefmt.nix                  # Formatter config (gofumpt, goimports, golines, gci)
├── .envrc                       # direnv integration (optional, 1 line)
├── go.mod                       # Unchanged
├── go.sum                       # Unchanged
├── Makefile                     # RETAINED as fallback (see §6.4)
├── .golangci.yml                # Unchanged — linter config
├── .buildflow.yml               # Unchanged — buildflow config
├── example/
│   └── user/
│       ├── go.mod               # Unchanged
│       └── go.sum               # Unchanged
├── .github/
│   └── workflows/
│       └── ci.yml               # NEW — single Nix-based CI workflow
│       ├── lint.yml             # DEPRECATED (kept temporarily)
│       └── test.yml             # DEPRECATED (kept temporarily)
└── ...
```

### 4.2 Flake Inputs

| Input         | URL                                   | Purpose                 | `follows` |
| ------------- | ------------------------------------- | ----------------------- | --------- |
| `nixpkgs`     | `github:NixOS/nixpkgs/nixos-unstable` | Package set             | —         |
| `flake-parts` | `github:hercules-ci/flake-parts`      | Modular flake outputs   | `nixpkgs` |
| `treefmt-nix` | `github:numtide/treefmt-nix`          | Formatter orchestration | `nixpkgs` |
| `git-hooks`   | `github:cachix/git-hooks.nix`         | Pre-commit hooks        | `nixpkgs` |
| `systems`     | `github:nix-systems/default`          | System matrix           | —         |

**Not included (deliberate):**

- `gomod2nix` — Overkill for 3 direct dependencies. `buildGoModule` with `vendorHash` is simpler and sufficient.
- `flake-utils` — `flake-parts` subsumes its functionality.

### 4.3 Flake Outputs

```
flake.nix outputs:
├── packages.${system}
│   ├── default          # Library build (go build ./...)
│   └── example-user     # Example binary (go build ./example/user/)
├── devShells.${system}
│   └── default          # Go + golangci-lint + gofumpt + goimports + golines + buildflow + trash
├── formatter.${system}  # treefmt wrapper (nix fmt)
├── checks.${system}
│   ├── formatting       # treefmt check (CI)
│   ├── lint             # golangci-lint run
│   ├── build            # go build ./...
│   ├── vet              # go vet ./...
│   ├── test             # go test ./...
│   └── test-race        # go test ./... -race
└── apps.${system}
    └── buildflow        # nix run .#buildflow -- --semantic --fix --dupl-threshold 50
```

---

## 5. File-by-File Implementation Plan

### 5.1 `flake.nix`

```nix
{
  description = "go-cqrs-lite — Lightweight CQRS library for Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    git-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    systems.url = "github:nix-systems/default";
  };

  outputs =
    inputs@{ self, nixpkgs, flake-parts, treefmt-nix, git-hooks, systems }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import systems;

      imports = [
        treefmt-nix.flakeModule
        git-hooks.flakeModule
      ];

      perSystem =
        { config, pkgs, system, ... }:
        let
          goPkg = pkgs.go_1_26;

          buildflow = pkgs.buildGoModule rec {
            pname = "buildflow";
            version = "latest";
            src = pkgs.fetchFromGitHub {
              owner = "larsartmann";
              repo = "buildflow";
              rev = "main"; # Pin to specific commit/tag when available
              hash = "sha256-PLACEHOLDER"; # Update after first build
            };
            vendorHash = "sha256-PLACEHOLDER"; # nix build will provide
            subPackages = [ "." ];
          };

          # Go experiment build tags from .golangci.yml
          goExperimentFlags = [
            "goexperiment.goroutineleakprofile"
            "goexperiment.jsonv2"
            "goexperiment.simd"
          ];
        in
        {
          # --- Packages ---
          packages = {
            default = pkgs.buildGoModule {
              pname = "go-cqrs-lite";
              version = "0.0.0";
              src = ./.;
              vendorHash = "sha256-PLACEHOLDER"; # Update after first build
              go = goPkg;
              # Library only — no main, just verify compilation
              proxyVendor = true;
            };

            example-user = pkgs.buildGoModule {
              pname = "example-user";
              version = "0.0.0";
              src = ./.;
              vendorHash = "sha256-PLACEHOLDER"; # Update after first build
              go = goPkg;
              subPackages = [ "example/user" ];
              proxyVendor = true;
            };
          };

          # --- Formatter (nix fmt) ---
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              gofumpt = {
                enable = true;
                package = pkgs.gofumpt;
              };
              goimports = {
                enable = true;
                package = pkgs.gotools;
              };
              golines = {
                enable = true;
                package = pkgs.golines;
              };
              gci = {
                enable = true;
              };
            };
          };

          # --- Pre-commit Hooks ---
          pre-commit = {
            settings = {
              hooks = {
                gofmt.enable = true;
                goimports.enable = true;
                golangci-lint = {
                  enable = true;
                  entry = "${pkgs.golangci-lint}/bin/golangci-lint run --new-from-rev HEAD";
                  files = "\\.go$";
                  pass_filenames = false;
                };
                treefmt.enable = true;
              };
            };
          };

          # --- Checks (nix flake check) ---
          checks =
            let
              goTestFlags = builtins.concatStringsSep " "
                (map (tag: "-tags=${tag}") goExperimentFlags);
            in
            {
              build = config.packages.default;
              vet = pkgs.runCommandLocal "vet" { } ''
                cd ${./.}
                ${goPkg}/bin/go vet ${goTestFlags} ./...
                touch $out
              '';
              test = pkgs.runCommandLocal "test" { } ''
                cd ${./.}
                ${goPkg}/bin/go test ${goTestFlags} ./... -v
                touch $out
              '';
              test-race = pkgs.runCommandLocal "test-race" { } ''
                cd ${./.}
                ${goPkg}/bin/go test ${goTestFlags} ./... -race
                touch $out
              '';
              lint = pkgs.runCommandLocal "lint" { } ''
                cd ${./.}
                ${pkgs.golangci-lint}/bin/golangci-lint run --timeout=5m
                touch $out
              '';
            };

          # --- Dev Shell ---
          devShells.default = pkgs.mkShell {
            shellHook = ''
              export GOWORK=off
              export GOFLAGS="${builtins.concatStringsSep " "
                (map (tag: "-tags=${tag}") goExperimentFlags)}"
              echo "go-cqrs-lite dev shell — Go $(${goPkg}/bin/go version)"
            '' + config.pre-commit.settings.installationScript;

            packages = [
              goPkg
              pkgs.golangci-lint
              pkgs.gotools # goimports
              pkgs.gofumpt
              pkgs.golines
              pkgs.trash-cli
              buildflow
            ];

            env = {
              GOROOT = "${goPkg}/share/go";
            };
          };

          # --- Apps ---
          apps.buildflow = {
            type = "app";
            program = "${buildflow}/bin/buildflow";
          };
        };
    };
}
```

### 5.2 `treefmt.nix` (Standalone, Optional)

If the team prefers a separate file over inline config:

```nix
# treefmt.nix
{ pkgs, ... }:
{
  projectRootFile = "go.mod";
  programs = {
    gofumpt.enable = true;
    goimports.enable = true;
    golines.enable = true;
    gci.enable = true;
  };
}
```

### 5.3 `.envrc`

```bash
use flake
```

One line. With nix-direnv installed, the dev shell activates automatically on `cd`.

### 5.4 `.github/workflows/ci.yml`

Replaces both `lint.yml` and `test.yml`:

```yaml
name: CI

on:
  push:
    branches: [master, main]
  pull_request:
    branches: [master, main]

jobs:
  check:
    name: Nix Flake Check
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: DeterminateSystems/nix-action@v4
        with:
          cache: true

      - name: Check formatting
        run: nix fmt -- --check

      - name: Lint
        run: nix develop --run "golangci-lint run --timeout=5m"

      - name: Build
        run: nix build

      - name: Test
        run: nix develop --run "go test ./... -v"

      - name: Test with race detector
        run: nix develop --run "go test ./... -race"

      - name: Vet
        run: nix develop --run "go vet ./..."

      - name: Generate coverage
        run: nix develop --run "go test ./... -coverprofile=coverage.out -covermode=atomic"

      - uses: codecov/codecov-action@v4
        with:
          files: coverage.out
          fail_ci_if_error: false
```

**Alternative: Full `nix flake check` approach** (replaces individual steps):

```yaml
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: DeterminateSystems/nix-action@v4
        with:
          cache: true
      - run: nix flake check
```

### 5.5 Files to Add to `.gitignore`

```gitignore
# Nix
result
result-*
.direnv/
```

---

## 6. Migration Steps

### Phase 1: Foundation (Day 1)

| Step | Action                                             | Verification                                |
| ---- | -------------------------------------------------- | ------------------------------------------- |
| 1.1  | Create `flake.nix` (initial draft with `fakeHash`) | `nix flake check` shows hash mismatch error |
| 1.2  | Run `nix build` to get real `vendorHash` values    | Build succeeds                              |
| 1.3  | Update `vendorHash` in `flake.nix`                 | `nix build` succeeds                        |
| 1.4  | Verify `nix develop` works                         | `nix develop --run "go test ./..."` passes  |
| 1.5  | Verify `nix fmt` works                             | `nix fmt -- --check` reports correctly      |
| 1.6  | Create `.envrc` with `use flake`                   | `direnv allow` loads shell                  |

### Phase 2: Buildflow Integration (Day 1-2)

| Step | Action                                                       | Verification                     |
| ---- | ------------------------------------------------------------ | -------------------------------- |
| 2.1  | Create `buildflow` derivation in flake                       | `nix build .#buildflow` succeeds |
| 2.2  | Test `nix run .#buildflow -- --semantic --dupl-threshold 50` | Output matches manual buildflow  |
| 2.3  | Pin buildflow to specific commit hash                        | Reproducible builds              |

### Phase 3: Checks & Hooks (Day 2)

| Step | Action                                               | Verification                      |
| ---- | ---------------------------------------------------- | --------------------------------- |
| 3.1  | Add `treefmt-nix` formatter config                   | `nix fmt` formats all Go files    |
| 3.2  | Add `git-hooks` pre-commit config                    | `.git/hooks/pre-commit` installed |
| 3.3  | Add all `checks` (build, vet, test, test-race, lint) | `nix flake check` passes          |

### Phase 4: CI Migration (Day 2-3)

| Step | Action                                      | Verification            |
| ---- | ------------------------------------------- | ----------------------- |
| 4.1  | Create `.github/workflows/ci.yml`           | CI passes on branch     |
| 4.2  | Verify coverage upload still works          | Codecov receives report |
| 4.3  | Rename old workflows with `.yml.bak` suffix | Old workflows disabled  |

### Phase 5: Documentation & Cleanup (Day 3)

| Step | Action                                                 | Verification                       |
| ---- | ------------------------------------------------------ | ---------------------------------- |
| 5.1  | Update `AGENTS.md` with Nix commands                   | Commands reference correct targets |
| 5.2  | Update `CONTRIBUTING.md` with `nix develop` onboarding | New contributors can onboard       |
| 5.3  | Update `.gitignore` with Nix entries                   | `result` and `.direnv` ignored     |
| 5.4  | Add Nix section to `README.md`                         | Documented for visitors            |
| 5.5  | Decide fate of `Makefile` (see §6.4)                   | Decision recorded                  |

### 6.4 Makefile: Keep or Remove?

**Recommendation: Keep as thin wrapper (Phase 5)**

Retain the Makefile for contributors who don't have Nix. Make it delegate to Nix when available:

```makefile
# Makefile (Nix-aware wrapper)
NIX_AVAILABLE := $(shell nix --version 2>/dev/null)

ifdef NIX_AVAILABLE
build:
	nix build

test:
	nix develop --run "go test ./... -v"

lint:
	nix develop --run "golangci-lint run"

fmt:
	nix fmt
else
# ... original targets as fallback ...
endif
```

This gives a graceful degradation path.

---

## 7. GitHub Actions CI Migration

### 7.1 Current vs Proposed

| Aspect       | Current                                 | Proposed                                 |
| ------------ | --------------------------------------- | ---------------------------------------- |
| Workflows    | 2 files (lint.yml, test.yml)            | 1 file (ci.yml)                          |
| Jobs         | 5 (golangci, fmt, imports, test, build) | 1-2 (check, optionally split)            |
| Go setup     | `setup-go@v5` + version string          | Via `flake.nix` — single source of truth |
| Module cache | `actions/cache@v4` with `go.sum` hash   | Nix store (automatic)                    |
| Lint action  | `golangci-lint-action@v6`               | Direct binary from nixpkgs               |
| Format check | `gofmt -l .` + `goimports -l .`         | `nix fmt -- --check`                     |
| Coverage     | `codecov-action@v4`                     | Unchanged                                |

### 7.2 CI Caching Strategy

| Option                                                    | Pros                                      | Cons                              |
| --------------------------------------------------------- | ----------------------------------------- | --------------------------------- |
| **DeterminateSystems/nix-action** with `cache: true`      | Simplest setup, uses GitHub Actions cache | External dependency               |
| **Cachix** (cachix-action)                                | Shared cache across branches/repos        | Requires Cachix account + secrets |
| **GitHub Actions cache** (nix-community/cache-nix-action) | No external service                       | More config                       |

**Recommendation:** Start with `DeterminateSystems/nix-action` for simplicity. Migrate to Cachix if/when build times warrant it.

---

## 8. Risks & Mitigations

| Risk                                        | Likelihood | Impact | Mitigation                                                               |
| ------------------------------------------- | ---------- | ------ | ------------------------------------------------------------------------ |
| **buildflow not in nixpkgs**                | High       | Medium | Package as custom `buildGoModule` derivation; pin to commit hash         |
| **Go 1.26 not yet in nixpkgs-unstable**     | Low        | High   | Use `go_1_26` or fall back to `master` nixpkgs; Go versions land quickly |
| **golangci-lint v2 config format**          | Low        | Low    | `.golangci.yml` is tool-config, not Nix concern                          |
| **Team unfamiliar with Nix**                | Medium     | Medium | Keep Makefile as fallback; document common workflows                     |
| **Nix build slower than bare `go test`**    | Low        | Low    | First build downloads; subsequent builds use Nix store cache             |
| **Go experiment flags not passing through** | Medium     | Medium | Test in `shellHook` and `checks` with explicit `-tags=` flags            |
| **flake.lock churn**                        | Low        | Low    | Monthly update cadence; `nix flake update` is explicit                   |
| **Multi-module complexity**                 | Low        | Low    | `example/user` is a separate package; main module is primary             |

---

## 9. Decision Matrix

Key decisions the team needs to make:

### 9.1 Go Dependency Management

| Option                                | Description                              | Recommendation                                               |
| ------------------------------------- | ---------------------------------------- | ------------------------------------------------------------ |
| **A. `buildGoModule` + `vendorHash`** | Nix fetches deps, hashes vendor dir      | ✅ **Recommended** — 3 direct deps, simple, no extra tooling |
| B. `gomod2nix`                        | Generates `gomod2nix.toml` from `go.sum` | Overkill for this project's small dependency tree            |
| C. Vendored (`go mod vendor`)         | Commit `vendor/` to git                  | Against project philosophy ("zero external dependencies")    |

### 9.2 Flake Framework

| Option                    | Description                    | Recommendation                                 |
| ------------------------- | ------------------------------ | ---------------------------------------------- |
| **A. `flake-parts`**      | Modular flake with imports     | ✅ **Recommended** — Cleaner, supports modules |
| B. Raw `outputs` function | Manual `forEachSystem` mapping | More boilerplate, but simpler to understand    |
| C. `flake-utils`          | Lightweight system mapper      | Superseded by `flake-parts` for this use case  |

### 9.3 Formatter Strategy

| Option                                     | Description                                    | Recommendation                                           |
| ------------------------------------------ | ---------------------------------------------- | -------------------------------------------------------- |
| **A. `treefmt-nix`**                       | Orchestrates multiple formatters via `nix fmt` | ✅ **Recommended** — Single command, Nix-pinned versions |
| B. `golangci-lint` formatters only         | Already configured in `.golangci.yml`          | Fragmented — doesn't cover `nix fmt`                     |
| C. Manual (gofumpt + goimports separately) | Current approach                               | No version pinning, multiple commands                    |

### 9.4 Pre-commit Hooks

| Option                             | Description                | Recommendation                                |
| ---------------------------------- | -------------------------- | --------------------------------------------- |
| **A. `git-hooks.nix` (cachix)**    | Nix-native hook management | ✅ **Recommended** — Integrated with devShell |
| B. `pre-commit` framework (Python) | Industry standard          | Adds Python dependency; defeats purpose       |
| C. None                            | Skip pre-commit            | Acceptable if CI catches everything           |

### 9.5 CI Migration Strategy

| Option                             | Description                                   | Recommendation                       |
| ---------------------------------- | --------------------------------------------- | ------------------------------------ |
| **A. Full Nix CI**                 | Single `nix flake check` or `ci.yml` with Nix | ✅ **Recommended** — Ultimate parity |
| B. Hybrid (Nix install + make)     | Use Nix to install tools, then run Makefile   | Transitional approach                |
| C. Keep current CI + add Nix check | Add a third workflow                          | Redundant, confusing                 |

---

## 10. Timeline

```
Day 1  ── Phase 1: Foundation
            Create flake.nix, get vendorHash, verify nix develop/fmt/build
         ── Phase 2: Buildflow
            Package buildflow derivation, verify nix run .#buildflow

Day 2  ── Phase 3: Checks & Hooks
            treefmt-nix, git-hooks, all checks
         ── Phase 4: CI Migration
            Create ci.yml, verify on branch

Day 3  ── Phase 5: Documentation & Cleanup
            AGENTS.md, CONTRIBUTING.md, README.md, .gitignore
            Decide Makefile fate
            Final review and merge
```

**Total effort estimate:** 2-3 days for an experienced Nix user, 4-5 days including learning curve.

---

## Appendix A: Command Reference

| Old Command                                      | New Command                                                    |
| ------------------------------------------------ | -------------------------------------------------------------- |
| `make all`                                       | `nix flake check && nix fmt`                                   |
| `make build`                                     | `nix build`                                                    |
| `make test`                                      | `nix develop --run "go test ./... -v"`                         |
| `make test-race`                                 | `nix develop --run "go test ./... -race"`                      |
| `make lint`                                      | `nix develop --run "golangci-lint run"`                        |
| `make fmt`                                       | `nix fmt`                                                      |
| `make imports`                                   | (included in `nix fmt`)                                        |
| `make vet`                                       | `nix develop --run "go vet ./..."`                             |
| `make coverage`                                  | `nix develop --run "go test ./... -coverprofile=coverage.out"` |
| `make check`                                     | `nix flake check`                                              |
| `make ci`                                        | `nix flake check`                                              |
| `make clean`                                     | `nix develop --run "trash coverage.* && go clean -testcache"`  |
| `buildflow --semantic --fix --dupl-threshold 50` | `nix run .#buildflow -- --semantic --fix --dupl-threshold 50`  |
| (new contributor setup)                          | `nix develop`                                                  |

## Appendix B: First-Time Setup Commands

```bash
# Install Nix (if not already installed)
curl --proto '=https' --tlsv1.2 -sSf -L \
  https://install.determinate.systems/nix | sh -s -- install

# Enter dev shell
nix develop

# Or with direnv
echo "use flake" > .envrc
direnv allow

# Build
nix build

# Run all checks
nix flake check

# Format
nix fmt
```

## Appendix C: Updating Dependencies

```bash
# Update Go dependencies
nix develop --run "go get ./..."
nix develop --run "go mod tidy"

# Update vendorHash (if using buildGoModule)
# Option 1: Replace vendorHash with fakeHash, then nix build will tell you
# Option 2: Use nix-prefetch
nix build 2>&1 | grep "got:"   # Copy the got: hash

# Update flake inputs (monthly recommended)
nix flake update
```
