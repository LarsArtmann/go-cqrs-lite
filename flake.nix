{
  description = "go-cqrs-lite — Lightweight CQRS library for Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
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
          goPkg = pkgs.go_1_26;

          goTags = [
            "goexperiment.arenas"
            "goexperiment.goroutineleakprofile"
            "goexperiment.runtimesecret"
            "goexperiment.simd"
          ];
          tagFlags = builtins.concatStringsSep " " (map (t: "-tags=${t}") goTags);

          testModules = [
            "core"
            "memory"
            "catalog"
            "middleware"
            "integration"
            "projection"
            "storage"
            "testhelpers"
            "saga"
            "watermill"
            "cmd/cqrs-gen"
          ];
          modulePaths = builtins.concatStringsSep " " (map (m: "./${m}/...") testModules);

          examplePaths = builtins.concatStringsSep " " [
            "./example/todo/..."
            "./example/user/..."
          ];

          mkApp = name: script: {
            type = "app";
            program = "${pkgs.writeShellScriptBin name script}/bin/${name}";
          };
        in
        {
          treefmt = {
            projectRootFile = "go.work";
            programs = {
              gofumpt.enable = true;
              goimports.enable = true;
              golines.enable = true;
              nixfmt.enable = true;
            };
          };

          devShells.default = pkgs.mkShell {
            packages = [
              goPkg
              pkgs.golangci-lint
              pkgs.gofumpt
              pkgs.golines
              pkgs.gotools
              pkgs.trash-cli
            ];

            shellHook = ''
              export GOWORK=off
              export GOFLAGS="${tagFlags}"
              echo "go-cqrs-lite dev shell — $(go version)"
            '';
          };

          checks = {
            build = pkgs.runCommand "go-cqrs-lite-build" { nativeBuildInputs = [ goPkg ]; } ''
              export GOFLAGS="${tagFlags}"
              export GOWORK=off
              cp -r ${./.} src && chmod -R u+w src && cd src
              ${goPkg}/bin/go build ${modulePaths}
              touch $out
            '';
          };

          apps = {
            test = mkApp "test" ''
              set -euo pipefail
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -count=1 "$@"
            '';

            test-race = mkApp "test-race" ''
              set -euo pipefail
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -race -count=1 "$@"
            '';

            build = mkApp "build" ''
              set -euo pipefail
              ${goPkg}/bin/go build ${modulePaths} "$@"
              ${goPkg}/bin/go build ${examplePaths}
            '';

            vet = mkApp "vet" ''
              set -euo pipefail
              ${goPkg}/bin/go vet ${tagFlags} ${modulePaths} "$@"
            '';

            lint = mkApp "lint" ''
              set -euo pipefail
              for mod in ${builtins.concatStringsSep " " testModules}; do
                echo "==> Linting $mod"
                (cd "$mod" && ${pkgs.golangci-lint}/bin/golangci-lint run --config ../.golangci.yml ./...)
              done
            '';

            coverage = mkApp "coverage" ''
              set -euo pipefail
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -coverprofile=coverage.out -covermode=atomic "$@"
              ${goPkg}/bin/go tool cover -func=coverage.out
            '';

            clean = mkApp "clean" ''
              set -euo pipefail
              ${pkgs.trash-cli}/bin/trash-put coverage.out 2>/dev/null || true
              ${goPkg}/bin/go clean -testcache
            '';
          };
        };
    };
}
