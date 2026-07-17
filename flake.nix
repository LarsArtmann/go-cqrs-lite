{
  description = "go-cqrs-lite — Lightweight CQRS library for Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    systems.url = "github:nix-systems/default";

    # Build infrastructure for distributable CLI packages (cmd/cqrs-lint, etc.)
    # These are `flake = false` tarballs — fetched via SSH at eval time, used as
    # local replace targets by mkPreparedSource so the Nix sandbox (no SSH keys)
    # can build private deps without network access.
    go-nix-helpers = {
      url = "git+ssh://git@github.com/LarsArtmann/go-nix-helpers?ref=master";
      flake = false;
    };
    go-finding = {
      url = "git+ssh://git@github.com/LarsArtmann/go-finding?ref=master";
      flake = false;
    };
    cmdguard = {
      url = "git+ssh://git@github.com/LarsArtmann/cmdguard?ref=master";
      flake = false;
    };
    go-output = {
      url = "git+ssh://git@github.com/LarsArtmann/go-output?ref=master";
      flake = false;
    };
    gogenfilter = {
      url = "git+ssh://git@github.com/LarsArtmann/gogenfilter?ref=master";
      flake = false;
    };
    go-branded-id = {
      url = "git+ssh://git@github.com/LarsArtmann/go-branded-id?ref=master";
      flake = false;
    };
    samber-do-auditlog = {
      url = "git+ssh://git@github.com/LarsArtmann/samber-do-auditlog?ref=master";
      flake = false;
    };
  };

  outputs =
    inputs@{
      self,
      nixpkgs,
      flake-parts,
      treefmt-nix,
      systems,
      go-nix-helpers,
      go-finding,
      cmdguard,
      go-output,
      gogenfilter,
      go-branded-id,
      samber-do-auditlog,
    }:
    let
      # Prepared source for building cmd/cqrs-lint as a distributable binary.
      # Only go-finding is private (GOPRIVATE); all other LarsArtmann deps are
      # public and served by proxy.golang.org during the vendor phase.
      version = self.rev or self.dirtyRev or "dev";

      mkCqrsLintSource =
        pkgs:
        let
          inherit (pkgs) lib;
          mkPreparedSourceFn = import (go-nix-helpers + "/mkPreparedSource.nix") {
            inherit pkgs lib;
            goPkg = pkgs.go_1_26;
          };
        in
        mkPreparedSourceFn {
          name = "cqrs-lint";
          inherit version;
          src = builtins.path {
            path = ./cmd/cqrs-lint;
            name = "source";
            filter =
              path: type:
              !(builtins.elem (baseNameOf path) [
                "flake.lock"
                ".envrc"
                "CONTRIBUTING.md"
                "README.md"
              ]);
          };
          deps = {
            "github.com/larsartmann/go-finding" = go-finding;
            "github.com/larsartmann/cmdguard/v3" = cmdguard;
            "github.com/larsartmann/go-output" = go-output;
            "github.com/LarsArtmann/gogenfilter/v3" = gogenfilter;
            "github.com/larsartmann/go-branded-id" = go-branded-id;
            "github.com/larsartmann/samber-do-auditlog" = samber-do-auditlog;
          };
          subModules = {
            "github.com/larsartmann/go-finding" = [ "pipeline" ];
            "github.com/larsartmann/go-output" = [
              "d2"
              "daghtml"
              "delimited"
              "escape"
              "graph"
              "markdown"
              "markup"
              "plantuml"
              "serialization"
              "table"
              "tree"
            ];
          };
        };
    in
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import systems;

      imports = [
        treefmt-nix.flakeModule
      ];

      perSystem =
        {
          config,
          pkgs,
          system,
          ...
        }:
        let
          inherit (pkgs) lib;
          goPkg = pkgs.go_1_26;

          goTags = [
            "goexperiment.jsonv2"
          ];
          tagFlags = builtins.concatStringsSep " " (map (t: "-tags=${t}") goTags);

          testModules = [
            "event"
            "command"
            "query"
            "decider"
            "id"
            "dispatcher"
            "schema"
            "snapshot"
            "codec"
            "dedup"
            "deriver"
            "graph"
            "metadata"
            "projection"
            "projectionhost"
            "scenario"
            "scheduling"
            "storage/memory"
            "storage/pebble"
            "storage/turso"
            "catalog"
            "middleware"
            "integration"
            "transport/http"
            "transport/grpc"
            "prometheus"
            "signing"
            "storage"
            "watermill"
            "encryption"
            "kv"
            "idempotency"
            "listing"
            "otel"
            "testutil"
            "stack"
            "stack/memory"
            "stack/sqlite"
            "stack/pebble"
            "stack/postgres"
            "stack/turso"
            "stack/bench"
            "cmd/cqrs-gen"
            "cmd/cqrs-lint"
          ];
          modulePaths = builtins.concatStringsSep " " (map (m: "./${m}/...") testModules);

          examplePaths = builtins.concatStringsSep " " [
            "./example/getting-started/..."
            "./example/taskmanager/..."
          ];

          allPaths = "${modulePaths} ${examplePaths}";

          mkApp = name: runtimeInputs: text: {
            type = "app";
            program = "${pkgs.writeShellApplication { inherit name runtimeInputs text; }}/bin/${name}";
          };

          goModules = [ goPkg ];

          benchstat = pkgs.buildGoModule {
            pname = "benchstat";
            version = "unstable-2026-06-14";
            src = pkgs.fetchFromGitHub {
              owner = "golang";
              repo = "perf";
              rev = "master";
              hash = "sha256-NA6V4sHZlvHCfdV2758IoMrDFAszmfrTjszZ+HB+PbM=";
            };
            subPackages = [ "cmd/benchstat" ];
            vendorHash = "sha256-qGQpf0T1qBcu+25VF2xnbvImj+Fs81Ru9tho/0RJwzo=";
          };
        in
        {
          treefmt = {
            projectRootFile = "go.work";
            settings.excludes = [
              "**/testdata/golden/**"
            ];
            programs = {
              gofumpt.enable = true;
              goimports.enable = true;
              golines.enable = true;
              nixfmt.enable = true;
            };
          };

          devShells.default = pkgs.mkShellNoCC {
            packages = [
              goPkg
              pkgs.golangci-lint
              pkgs.gotools
              pkgs.trash-cli
              pkgs.gosec
              pkgs.go-arch-lint
              pkgs.govulncheck
              pkgs.gitleaks
            ];

            GOWORK = "off";
            GOFLAGS = tagFlags;

            shellHook = ''
              echo "go-cqrs-lite dev shell — $(go version)"
              # Make the Crush skill globally available so AI assistants trigger it
              # from any consumer project, not just inside this repo. Idempotent & non-destructive.
              if [ -d "''${HOME:-}/.config/crush/skills" ]; then
                ln -sfn "${self}/.agents/skills/go-cqrs-lite" "$HOME/.config/crush/skills/go-cqrs-lite"
              fi
            '';
          };

          checks = {
            build = config.packages.default;
            format = config.treefmt.build.check self;
          };

          # No-op default package so `nix build .` (BuildFlow's full mode) succeeds.
          # This is a Go library with no single binary; real artifacts live in apps.*
          # and are invoked via `nix run .#<app>`. Pattern mirrors cqrs-htmx/flake.nix.
          packages.default = pkgs.stdenvNoCC.mkDerivation {
            pname = "go-cqrs-lite";
            version = self.rev or self.dirtyRev or "dev";

            dontUnpack = true;
            dontConfigure = true;
            dontBuild = true;

            installPhase = ''
              mkdir -p $out
            '';

            meta = with lib; {
              description = "Lightweight CQRS/Event-Sourcing library for Go";
              homepage = "https://github.com/larsartmann/go-cqrs-lite";
              license = licenses.mit;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
              platforms = platforms.unix;
            };
          };

          # Domain-aware linter for go-cqrs-lite consumers.
          # Built from cmd/cqrs-lint/ which has its own go.mod (standalone module).
          # Only go-finding is replaced via mkPreparedSource (private repo);
          # all other LarsArtmann deps are public and served by proxy.golang.org.
          packages.cqrs-lint = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) {
            pname = "cqrs-lint";
            inherit version;

            src = mkCqrsLintSource pkgs;

            vendorHash = "sha256-iYsgtIvIluo0ZSr5trFHWfG2RZ+DYdlxG/IFxHycw0Y=";
            proxyVendor = true;

            subPackages = [ "." ];

            ldflags = [
              "-s"
              "-w"
            ];

            env = {
              CGO_ENABLED = "0";
              GOWORK = "off";
            };

            # buildGoModule silently drops GOEXPERIMENT from env (not in its
            # whitelist), so export it in preBuild. The "goexperiment.jsonv2"
            # build tag is set internally by the toolchain from GOEXPERIMENT.
            preBuild = ''
              export GOEXPERIMENT=jsonv2
            '';

            doCheck = false;

            meta = with lib; {
              description = "Domain-aware linter for go-cqrs-lite consumers";
              license = licenses.mit;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
              mainProgram = "cqrs-lint";
              platforms = platforms.unix;
            };
          };

          apps = {
            test = mkApp "test" goModules ''
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -count=1 "$@"
            '';

            test-race = mkApp "test-race" goModules ''
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -race -count=1 "$@"
            '';

            build = mkApp "build" goModules ''
              ${goPkg}/bin/go build ${tagFlags} ${allPaths} "$@"
            '';

            vet = mkApp "vet" goModules ''
              ${goPkg}/bin/go vet ${tagFlags} ${modulePaths} "$@"
            '';

            lint = mkApp "lint" [ goPkg pkgs.golangci-lint ] ''
              configFile="$PWD/.golangci.yml"
              failed=0
              for mod in ${builtins.concatStringsSep " " testModules}; do
                echo "==> Linting $mod"
                (cd "$mod" && ${pkgs.golangci-lint}/bin/golangci-lint run --config "$configFile" ./...) || failed=1
              done
              exit "$failed"
            '';

            coverage = mkApp "coverage" goModules ''
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -coverprofile=coverage.out -covermode=atomic "$@"
              ${goPkg}/bin/go tool cover -func=coverage.out
            '';

            bench = mkApp "bench" goModules ''
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -bench=. -benchmem -count=1 -timeout=30m "$@"
            '';

            check-layers = mkApp "check-layers" [ goPkg pkgs.bash ] ''
              ${pkgs.bash}/bin/bash "$PWD/scripts/check-module-layers.sh"
            '';

            check-isolation = mkApp "check-isolation" [ goPkg pkgs.bash ] ''
              ${pkgs.bash}/bin/bash "$PWD/scripts/check-module-isolation.sh"
            '';

            check-doc-stubs = mkApp "check-doc-stubs" [ pkgs.findutils pkgs.gnugrep ] ''
              ${pkgs.bash}/bin/bash "$PWD/scripts/check-doc-stubs.sh"
            '';

            check-arch = mkApp "check-arch" [ goPkg pkgs.bash ] ''
              ${pkgs.bash}/bin/bash "$PWD/scripts/check-arch.sh"
            '';

            check-file-size = mkApp "check-file-size" [ pkgs.findutils ] ''
              failed=false
              while IFS= read -r f; do
                lines=$(wc -l < "$f")
                if [ "$lines" -gt 350 ]; then
                  echo "ERROR: $f has $lines lines (max 350)"
                  failed=true
                fi
              done < <(find . -name "*.go" -not -name "*_test.go" \
                -not -name "*.pb.go" \
                -not -name "*.gen.go" \
                -not -path "*/example/*" \
                -not -path "*/testdata/*" \
                -not -path "*/internal/cattest/*" \
                -not -path "*/.git/*")
              if [ "$failed" = true ]; then
                echo "One or more production files exceed 350 lines"
                exit 1
              fi
              echo "All production files within 350-line limit"
            '';

            test-grpc = mkApp "test-grpc" goModules ''
              echo "==> Testing transport/grpc (GOWORK=off)"
              (cd transport/grpc && GOWORK=off ${goPkg}/bin/go test ./... -count=1 "$@")
            '';

            check-wasm = mkApp "check-wasm" goModules ''
              wasmMods="id codec dispatcher event command query decider"
              failed=0
              for mod in $wasmMods; do
                echo "==> WASM build: $mod"
                (cd "$mod" && GOWORK=off GOOS=js GOARCH=wasm ${goPkg}/bin/go build ./...) || failed=1
              done
              exit "$failed"
            '';

            check-api-stability = mkApp "check-api-stability" goModules ''
              echo "==> API surface check"
              (cd cmd/api-stability && GOWORK=off ${goPkg}/bin/go run main.go)
            '';

            ci = mkApp "ci" [ goPkg pkgs.golangci-lint pkgs.bash pkgs.findutils ] ''
              ${pkgs.bash}/bin/bash -c '
                set -e
                echo "=== Build ===" && ${goPkg}/bin/go build ${tagFlags} ${allPaths}
                echo "=== Vet ===" && ${goPkg}/bin/go vet ${tagFlags} ${modulePaths}
                echo "=== Test ===" && ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -count=1
                echo "=== Check Layers ===" && bash "$PWD/scripts/check-module-layers.sh"
                echo "=== API Stability ===" && (cd cmd/api-stability && GOWORK=off ${goPkg}/bin/go run main.go)
                echo "=== transport/grpc ===" && (cd transport/grpc && GOWORK=off ${goPkg}/bin/go test ./... -count=1)
                echo "✅ All CI checks passed"
              '
            '';

            clean = mkApp "clean" [ goPkg pkgs.trash-cli ] ''
              ${pkgs.trash-cli}/bin/trash-put coverage.out 2>/dev/null || true
              ${goPkg}/bin/go clean -testcache
            '';

            benchstat = mkApp "benchstat" [ benchstat ] ''
              ${benchstat}/bin/benchstat "$@"
            '';

            vulncheck = mkApp "vulncheck" [ goPkg ] ''
              for mod in ${builtins.concatStringsSep " " testModules}; do
                echo "==> Vulnerability scan: $mod"
                (cd "$mod" && GOWORK=off ${goPkg}/bin/go list -json ./... | ${pkgs.govulncheck}/bin/govulncheck -mode=source)
              done
            '';

            secrets-scan = mkApp "secrets-scan" [ pkgs.gitleaks ] ''
              ${pkgs.gitleaks}/bin/gitleaks detect --source . --no-banner --no-git
              echo "==> Secret scan complete"
            '';

            verify = mkApp "verify" [ goPkg pkgs.golangci-lint pkgs.bash ] ''
              ${pkgs.bash}/bin/bash scripts/verify-docs.sh && \
              echo "=== Build ===" && ${goPkg}/bin/go build ${tagFlags} ${allPaths} && \
              echo "=== Vet ===" && ${goPkg}/bin/go vet ${tagFlags} ${modulePaths} && \
              echo "=== Test ===" && ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -count=1 && \
              echo "=== Race ===" && ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -race -count=1 && \
              echo "=== Lint ===" && nix run .#lint && \
              echo "=== Doc Check ===" && (cd cmd/doc-check && GOWORK=off ${goPkg}/bin/go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md ../../README.md ../../TODO_LIST.md ../../ROADMAP.md ../../FEATURES.md ../../CONTRIBUTING.md) && \
              echo "✅ All verification checks passed"
            '';
          };
        };

      flake = {
        overlays.cqrs-lint = final: _prev: {
          cqrs-lint = self.packages.${final.stdenv.system}.cqrs-lint;
        };
      };
    };
}
