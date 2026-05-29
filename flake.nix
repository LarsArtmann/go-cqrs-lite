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
            "signing"
            "storage"
            "testhelpers"
            "watermill"
            "cmd/cqrs-gen"
          ];
          modulePaths = builtins.concatStringsSep " " (map (m: "./${m}/...") testModules);

          examplePaths = builtins.concatStringsSep " " [
            "./example/projection/..."
            "./example/saga-pattern/..."
            "./example/storage/..."
            "./example/todo/..."
            "./example/user/..."
          ];

          allPaths = "${modulePaths} ${examplePaths}";

          mkApp = name: runtimeInputs: text: {
            type = "app";
            program = "${pkgs.writeShellApplication { inherit name runtimeInputs text; }}/bin/${name}";
          };

          goModules = [ goPkg ];
        in
        {
          treefmt = {
            projectRootFile = "go.work";
            settings.excludes = [
              "catalog/testdata/golden/**"
            ];
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
              pkgs.gotools
              pkgs.trash-cli
            ];

            GOWORK = "off";
            GOFLAGS = tagFlags;

            shellHook = ''
              echo "go-cqrs-lite dev shell — $(go version)"
            '';
          };

          checks = { };

          apps = {
            test = mkApp "test" goModules ''
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -count=1 "$@"
            '';

            test-race = mkApp "test-race" goModules ''
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -race -count=1 "$@"
            '';

            build = mkApp "build" goModules ''
              ${goPkg}/bin/go build ${allPaths} "$@"
            '';

            vet = mkApp "vet" goModules ''
              ${goPkg}/bin/go vet ${tagFlags} ${modulePaths} "$@"
            '';

            lint = mkApp "lint" [ goPkg pkgs.golangci-lint ] ''
              configFile="$PWD/.golangci.yml"
              for mod in ${builtins.concatStringsSep " " testModules}; do
                echo "==> Linting $mod"
                (cd "$mod" && ${pkgs.golangci-lint}/bin/golangci-lint run --config "$configFile" ./...)
              done
            '';

            coverage = mkApp "coverage" goModules ''
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -coverprofile=coverage.out -covermode=atomic "$@"
              ${goPkg}/bin/go tool cover -func=coverage.out
            '';

            clean = mkApp "clean" [ goPkg pkgs.trash-cli ] ''
              ${pkgs.trash-cli}/bin/trash-put coverage.out 2>/dev/null || true
              ${goPkg}/bin/go clean -testcache
            '';
          };
        };
    };
}
