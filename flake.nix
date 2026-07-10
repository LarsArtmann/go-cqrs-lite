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
  };

  outputs =
    inputs@{
      self,
      nixpkgs,
      flake-parts,
      treefmt-nix,
      systems,
    }:
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
            "goexperiment.arenas"
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
                ln -sfn "${inputs.self}/.agents/skills/go-cqrs-lite" "$HOME/.config/crush/skills/go-cqrs-lite"
              fi
            '';
          };

          checks = {
            build = config.packages.default;
            format = config.treefmt.build.check inputs.self;
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
              maintainers = [ maintainers.larsartmann ];
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
          };
        };
    };
}
