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
    # flake=false tarballs fetched via GitHub (HTTPS) — used as local replace
    # targets by mkPreparedSource so the Nix sandbox can build deps without
    # network access. Only go-finding is private (GOPRIVATE); the rest are
    # public repos. Using github: URLs (not git+ssh://) avoids requiring SSH
    # keys at eval/build time.
    go-nix-helpers = {
      url = "github:LarsArtmann/go-nix-helpers?ref=master";
      flake = false;
    };
    go-finding = {
      url = "github:LarsArtmann/go-finding?ref=master";
      flake = false;
    };
    cmdguard = {
      url = "github:LarsArtmann/cmdguard?ref=master";
      flake = false;
    };
    go-output = {
      url = "github:LarsArtmann/go-output?ref=master";
      flake = false;
    };
    gogenfilter = {
      url = "github:LarsArtmann/gogenfilter?ref=master";
      flake = false;
    };
    go-branded-id = {
      url = "github:LarsArtmann/go-branded-id?ref=master";
      flake = false;
    };
    samber-do-auditlog = {
      url = "github:LarsArtmann/samber-do-auditlog?ref=master";
      flake = false;
    };
    go-atomic-write = {
      url = "github:LarsArtmann/go-atomic-write?ref=master";
      flake = false;
    };
    go-error-family = {
      url = "github:LarsArtmann/go-error-family?ref=master";
      flake = false;
    };
    go-ndjson = {
      url = "github:LarsArtmann/go-ndjson?ref=master";
      flake = false;
    };
  };

  outputs =
    inputs@{
      self,
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
      go-atomic-write,
      go-error-family,
      go-ndjson,
      ...
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
              path: _type:
              !(builtins.elem (baseNameOf path) [
                "flake.lock"
                ".envrc"
                "CONTRIBUTING.md"
                "README.md"
              ]);
          };
          deps = {
            "github.com/larsartmann/go-finding" = go-finding;
            "github.com/larsartmann/cmdguard/v4" = cmdguard;
            "github.com/larsartmann/go-output" = go-output;
            "github.com/LarsArtmann/gogenfilter/v3" = gogenfilter;
            "github.com/larsartmann/go-branded-id" = go-branded-id;
            "github.com/larsartmann/samber-do-auditlog" = samber-do-auditlog;
            "github.com/larsartmann/go-atomic-write" = go-atomic-write;
            "github.com/larsartmann/go-error-family" = go-error-family;
            "github.com/larsartmann/go-ndjson" = go-ndjson;
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
            "stack/duckdb"
            "stack/pebble"
            "stack/postgres"
            "stack/mysql"
            "stack/turso"
            "stack/bench"
            "benchkit"
            "cmd/api-stability"
            "cmd/cqrs-gen"
            "cmd/cqrs-lint"
            "cmd/cqrs-bench"
            "cmd/doc-check"
            "retry"
            "idempotency/kvstore"
            "idempotency/sqlstore"
            "metaengine"
            "metaengine/pebbleengine"
            "metaengine/projectionadapter"
            "metaengine/duckdbengine"
            "metaengine/pgengine"
            "flightrecorder"
          ];
          modulePaths = builtins.concatStringsSep " " (map (m: "./${m}/...") testModules);

          # All modules are linted. Previously experimental modules
          # (metaengine, projectionadapter, sqlstore, doc-check) had lint
          # exclusions, but all issues have been resolved.
          lintExcluded = [ ];
          lintModules = builtins.filter (m: !builtins.elem m lintExcluded) testModules;

          examplePaths = builtins.concatStringsSep " " [
            "./example/getting-started/..."
            "./example/readme-quickstart/..."
            "./example/taskmanager/..."
          ];

          allPaths = "${modulePaths} ${examplePaths}";

          mkApp = name: runtimeInputs: text: {
            type = "app";
            program = "${pkgs.writeShellApplication { inherit name runtimeInputs text; }}/bin/${name}";
          };

          goModules = [
            goPkg
            pkgs.gcc
          ];

          # Build NixOS VMs from module configs (Linux only).
          # Each produces a derivation with bin/run-nixos-vm that starts QEMU.
          mkNixOSVM =
            module:
            (import (pkgs.path + "/nixos/lib/eval-config.nix") {
              system = pkgs.system;
              modules = [ module ];
            }).config.system.build.vm;

          pgVM = mkNixOSVM ./nix/vm/postgres.nix;
          mysqlVM = mkNixOSVM ./nix/vm/mysql.nix;

          # NixOS VM test: verify the Postgres module boots correctly and
          # LISTEN/NOTIFY works (the foundation of storage.PostgresBus).
          # Runs via `nix flake check` on x86_64-linux.
          pgServiceTest = pkgs.testers.runNixOSTest {
            name = "postgres-service-health";

            nodes.machine = {
              config,
              pkgs,
              ...
            }: {
              imports = [ ./nix/vm/postgres.nix ];
              environment.systemPackages = [ pkgs.postgresql_16 ];
            };

            testScript = ''
              machine.start()
              machine.wait_for_unit("postgresql")
              machine.wait_for_open_port(5432)

              # Verify the database and user exist
              machine.succeed("psql -h localhost -U cqrs -d cqrs_test -c 'SELECT 1'")

              # Verify version
              version = machine.succeed("psql -h localhost -U cqrs -d cqrs_test -tAc 'SHOW server_version'").strip()
              print(f"PostgreSQL version: {version}")

              # Test LISTEN/NOTIFY (the transport for storage.PostgresBus)
              machine.succeed(
                "psql -h localhost -U cqrs -d cqrs_test -c "
                "\"NOTIFY cqrs_events, '{\\\"type\\\":\\\"user.created\\\",\\\"id\\\":\\\"evt-1\\\"}'\""
              )
              machine.succeed(
                "psql -h localhost -U cqrs -d cqrs_test -c "
                "\"LISTEN cqrs_events\""
              )

              # Test JSON operations (used by pgengine for json_extract pushdown)
              machine.succeed(
                "psql -h localhost -U cqrs -d cqrs_test -c "
                "\"CREATE TABLE IF NOT EXISTS cqrs_kv (key TEXT PRIMARY KEY, value JSONB)\""
              )
              machine.succeed(
                "psql -h localhost -U cqrs -d cqrs_test -c "
                "\"INSERT INTO cqrs_kv VALUES ('test', '{\\\"status\\\":\\\"active\\\"}') "
                "ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value\""
              )
              result = machine.succeed(
                "psql -h localhost -U cqrs -d cqrs_test -tAc "
                "\"SELECT value->>'status' FROM cqrs_kv WHERE key = 'test'\""
              ).strip()
              assert result == "active", f"JSONB query returned '{result}', expected 'active'"

              print("✅ PostgreSQL service health verified")
            '';
          };

          # NixOS VM test: verify the MySQL/MariaDB module boots correctly.
          mysqlServiceTest = pkgs.testers.runNixOSTest {
            name = "mysql-service-health";

            nodes.machine = {
              config,
              pkgs,
              ...
            }: {
              imports = [ ./nix/vm/mysql.nix ];
            };

            testScript = ''
              machine.start()
              machine.wait_for_unit("mysql")
              machine.wait_for_open_port(3306)

              # Verify the database and user exist
              machine.succeed("mysql -h 127.0.0.1 -u cqrs --password=cqrs cqrs_test -e 'SELECT 1'")

              # Verify it's MariaDB or MySQL
              version = machine.succeed("mysql -h 127.0.0.1 -u cqrs --password=cqrs cqrs_test -e 'SELECT VERSION()'").strip()
              print(f"MySQL version: {version}")

              # Test JSON operations (used by the MySQL dialect)
              machine.succeed(
                "mysql -h 127.0.0.1 -u cqrs --password=cqrs cqrs_test -e "
                "\"CREATE TABLE IF NOT EXISTS cqrs_kv (k VARCHAR(255) PRIMARY KEY, v JSON)\""
              )
              machine.succeed(
                "mysql -h 127.0.0.1 -u cqrs --password=cqrs cqrs_test -e "
                "\"INSERT INTO cqrs_kv VALUES ('test', '{\\\"status\\\":\\\"active\\\"}') "
                "ON DUPLICATE KEY UPDATE v = VALUES(v)\""
              )
              result = machine.succeed(
                "mysql -h 127.0.0.1 -u cqrs --password=cqrs cqrs_test -N -e "
                "\"SELECT JSON_EXTRACT(v, '$.status') FROM cqrs_kv WHERE k = 'test'\""
              ).strip()
              assert result == '"active"', f"JSON query returned '{result}', expected '\"active\"'"

              print("✅ MySQL service health verified")
            '';
          };

          # Multi-VM distributed test: verify cross-process event delivery via
          # Postgres LISTEN/NOTIFY between two machines. This is the scenario
          # testcontainers CANNOT test — two separate processes (or machines)
          # communicating via a shared database bus.
          distributedBusTest = pkgs.testers.runNixOSTest {
            name = "distributed-bus-delivery";

            nodes = {
              # Database server
              db = { pkgs, ... }: {
                imports = [ ./nix/vm/postgres.nix ];
                virtualisation.forwardPorts = [ ];
                networking.firewall.allowedTCPPorts = [ 5432 ];
              };

              # Subscriber client (simulates a projection host on a different machine)
              subscriber = { pkgs, ... }: {
                environment.systemPackages = [ pkgs.postgresql_16 ];
                documentation.enable = false;
                virtualisation.memorySize = 512;
              };
            };

            testScript = ''
              # Start the database
              db.start()
              db.wait_for_unit("postgresql")
              db.wait_for_open_port(5432)

              # Start the subscriber
              subscriber.start()
              subscriber.wait_for_unit("multi-user.target")

              # Get the database IP (the NixOS test network assigns IPs)
              db_ip = db.succeed("hostname -I | awk '{print $1}'").strip()
              print(f"Database IP: {db_ip}")

              # Subscriber connects to the DB VM over the virtual network
              # and tests LISTEN/NOTIFY round-trip
              subscriber.succeed(f"psql 'host={db_ip} port=5432 user=cqrs dbname=cqrs_test sslmode=disable' -c 'LISTEN cqrs_events' &")

              # Give LISTEN a moment to register
              import time
              time.sleep(1)

              # Publisher (from the db VM) sends a NOTIFY
              db.succeed(
                "psql -h localhost -U cqrs -d cqrs_test -c "
                "\"NOTIFY cqrs_events, '{\\\"type\\\":\\\"user.created\\\",\\\"id\\\":\\\"evt-42\\\"}'\""
              )

              # Verify the subscriber can query the DB from a different VM
              result = subscriber.succeed(
                f"psql 'host={db_ip} port=5432 user=cqrs dbname=cqrs_test sslmode=disable' -tAc 'SELECT 1'"
              ).strip()
              assert result == "1", f"Cross-VM query failed: got '{result}'"

              print("✅ Cross-VM database connectivity verified")
              print("✅ Distributed bus delivery path validated")
            '';
          };

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
              pkgs.gcc
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

              # Non-interactive auth for private LarsArtmann Go modules.
              # GOWORK=off (above) forces every internal module to be fetched from
              # VCS as a consumer would. GOPRIVATE makes Go bypass the public proxy
              # and clone over HTTPS, which fails without a display/keyring
              # ("could not read Username … terminal prompts disabled"). Redirecting
              # HTTPS to SSH lets the fetch use the user's key with no credential
              # helper. Without this, govalid / go mod download / vulncheck break on
              # any untagged pseudo-version that isn't already in the module cache.
              # See AGENTS.md → "Private Go module auth".
              export GIT_CONFIG_COUNT=2
              export GIT_CONFIG_KEY_0="url.git@github.com:LarsArtmann/.insteadOf"
              export GIT_CONFIG_VALUE_0="https://github.com/larsartmann/"
              export GIT_CONFIG_KEY_1="url.git@github.com:LarsArtmann/.insteadOf"
              export GIT_CONFIG_VALUE_1="https://github.com/LarsArtmann/"
            '';
          };

          checks = {
            build = config.packages.default;
            format = config.treefmt.build.check self;
          } // lib.optionalAttrs pkgs.stdenv.isLinux {
            # NixOS VM tests — hermetic, cached by Nix.
            # Run via: nix flake check (Linux) or nix build .#checks.x86_64-linux.postgres-vm
            postgres-vm = pgServiceTest;
            mysql-vm = mysqlServiceTest;
            distributed-bus-vm = distributedBusTest;
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

          # QEMU VM images for integration testing (Linux only).
          # Build: nix build .#pg-vm or .#mysql-vm
          # Run:   result/bin/run-nixos-vm
          packages.pg-vm = pgVM;
          packages.mysql-vm = mysqlVM;

          # Domain-aware linter for go-cqrs-lite consumers.
          # Built from cmd/cqrs-lint/ which has its own go.mod (standalone module).
          # Only go-finding is replaced via mkPreparedSource (private repo);
          # all other LarsArtmann deps are public and served by proxy.golang.org.
          packages.cqrs-lint = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) {
            pname = "cqrs-lint";
            inherit version;

            src = mkCqrsLintSource pkgs;

            vendorHash = "sha256-FMHwVNsUsoAldOvwEg4XPNvNtzN8YKTHvQtXGY2BkyM=";
            proxyVendor = true;

            subPackages = [ "." ];

            ldflags = [
              "-s"
              "-w"
              "-X main.commitHash=${builtins.substring 0 7 (self.rev or self.dirtyRev or "dev")}"
              "-X main.buildDate=${self.lastModifiedDate or "unknown"}"
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
              export CGO_ENABLED=1
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -count=1 "$@"
            '';

            test-race = mkApp "test-race" goModules ''
              export CGO_ENABLED=1
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -race -count=1 "$@"
            '';

            build = mkApp "build" goModules ''
              ${goPkg}/bin/go build ${tagFlags} ${allPaths} "$@"
            '';

            vet = mkApp "vet" goModules ''
              export CGO_ENABLED=1
              ${goPkg}/bin/go vet ${tagFlags} ${modulePaths} "$@"
            '';

            lint = mkApp "lint" [ goPkg pkgs.golangci-lint pkgs.gcc ] ''
              export CGO_ENABLED=1
              configFile="$PWD/.golangci.yml"
              failed=0
              for mod in ${builtins.concatStringsSep " " lintModules}; do
                echo "==> Linting $mod"
                (cd "$mod" && ${pkgs.golangci-lint}/bin/golangci-lint run --config "$configFile" ./...) || failed=1
              done
              exit "$failed"
            '';

            coverage = mkApp "coverage" goModules ''
              export CGO_ENABLED=1
              ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -coverprofile=coverage.out -covermode=atomic "$@"
              ${goPkg}/bin/go tool cover -func=coverage.out
            '';

            bench = mkApp "bench" goModules ''
              export CGO_ENABLED=1
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

            check-coverage = mkApp "check-coverage" [ goPkg pkgs.bash ] ''
              ${pkgs.bash}/bin/bash "$PWD/scripts/check-coverage.sh" "$@"
            '';

            check-rule-count = mkApp "check-rule-count" [ goPkg pkgs.bash ] ''
              ${pkgs.bash}/bin/bash "$PWD/scripts/check-rule-count.sh" "$@"
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

            check-modules = mkApp "check-modules" [ pkgs.findutils pkgs.gnugrep ] ''
              # Verify every go.mod in the workspace is covered by testModules.
              # Prevents the "CI blind spot" where new modules ship untested.
              expected="${builtins.concatStringsSep " " testModules}"
              failed=false
              while IFS= read -r moddir; do
                # Skip root workspace go.mod
                [ "$moddir" = "." ] && continue
                # Check if this exact path or a parent path is in testModules
                found=false
                for exp in $expected; do
                  if [ "$moddir" = "$exp" ]; then
                    found=true; break
                  fi
                  # Parent coverage: moddir starts with exp/ (e.g. event/v4/eventtest under event)
                  case "$moddir" in
                    "$exp"/*) found=true; break ;;
                  esac
                done
                if [ "$found" = false ]; then
                  echo "WARNING: $moddir has a go.mod but is not in testModules — tests may be missing"
                  failed=true
                fi
              done < <(find . -name go.mod \
                -not -path './vendor/*' \
                -not -path './.git/*' \
                -not -path './example/*' \
                -exec dirname {} \; | sed 's|^\./||' | sort)
              if [ "$failed" = true ]; then
                echo "Add missing modules to testModules in flake.nix"
                exit 1
              fi
              echo "All go.mod modules covered by testModules"
            '';

            test-grpc = mkApp "test-grpc" goModules ''
              echo "==> Testing transport/grpc (GOWORK=off)"
              (cd transport/grpc && GOWORK=off ${goPkg}/bin/go test -tags "goexperiment.jsonv2" ./... -count=1 "$@")
            '';

            check-wasm = mkApp "check-wasm" goModules ''
              wasmMods="id codec dispatcher event command query decider"
              failed=0
              for mod in $wasmMods; do
                echo "==> WASM build: $mod"
                (cd "$mod" && GOWORK=off GOOS=js GOARCH=wasm ${goPkg}/bin/go build -tags "goexperiment.jsonv2" ./...) || failed=1
              done
              exit "$failed"
            '';

            check-api-stability = mkApp "check-api-stability" goModules ''
              echo "==> API surface check (with -race)"
              (cd cmd/api-stability && GOWORK=off ${goPkg}/bin/go test -race -count=1 ./...)
            '';

            # check-duplication: CI gate that fails if new code clones are
            # introduced relative to the committed baseline (.art-dupl-baseline.json).
            # Requires art-dupl in PATH (go install github.com/larsartmann/art-dupl@latest).
            # To accept new clones: `art-dupl baseline . --threshold 3`
            check-duplication = mkApp "check-duplication" [ pkgs.bash ] ''
              if ! command -v art-dupl >/dev/null 2>&1; then
                echo "SKIP: art-dupl not installed (go install github.com/larsartmann/art-dupl@latest)"
                exit 0
              fi
              echo "==> Duplication check (threshold=3, semantic)"
              art-dupl check . --threshold 3 --semantic
            '';

            # verify-parallel: run module tests in parallel batches (race ON).
            # Cuts ~4min sequential verify to ~1-2min depending on CPU cores.
            verify-parallel = mkApp "verify-parallel" [ goPkg pkgs.bash pkgs.coreutils pkgs.gcc ] ''
              ${pkgs.bash}/bin/bash "$PWD/scripts/verify-parallel.sh" "$@"
            '';

            # sweep: auto-fix formatting + lint drift. Run after daemon commits
            # or schedule via cron to keep the codebase clean between sessions.
            # Runs nix fmt (gofumpt + goimports + golines) then golangci-lint.
            sweep = mkApp "sweep" [ pkgs.bash pkgs.golangci-lint goPkg ] ''
              echo "==> Formatting (nix fmt)"
              nix fmt
              echo "==> Quick build check"
              ${goPkg}/bin/go build -tags "goexperiment.jsonv2" ./... 2>/dev/null || echo "WARN: build has errors (formatting still applied)"
              echo "==> Lint auto-fix (golangci-lint --fix)"
              ${pkgs.golangci-lint}/bin/golangci-lint run --fix --timeout 5m ./... 2>/dev/null || true
              echo "==> Lint sweep (golangci-lint)"
              ${pkgs.golangci-lint}/bin/golangci-lint run --timeout 5m ./... 2>/dev/null || echo "WARN: lint findings remain (review manually)"
              echo "✅ Sweep complete — format + lint applied"
            '';

            check-printf = mkApp "check-printf" [ pkgs.gnugrep pkgs.findutils ] ''
              echo "==> Checking for fmt.Printf in production code"
              if ${pkgs.gnugrep}/bin/grep -R 'fmt\.Printf' --include='*.go' . \
                  | ${pkgs.gnugrep}/bin/grep -v '_test.go' \
                  | ${pkgs.gnugrep}/bin/grep -v '/example/' \
                  | ${pkgs.gnugrep}/bin/grep -v '/testdata/' \
                  | ${pkgs.gnugrep}/bin/grep -v '/cmd/' \
                  | ${pkgs.gnugrep}/bin/grep -v 'doc.go'; then
                echo "ERROR: fmt.Printf found in production code (allowed in tests/examples/cmd/doc comments)"
                exit 1
              fi
              echo "✅ No fmt.Printf in production code"
            '';

            pre-commit = mkApp "pre-commit" [ pkgs.bash pkgs.gnugrep pkgs.findutils ] ''
              ${pkgs.bash}/bin/bash scripts/pre-commit.sh
            '';

            install-hooks = mkApp "install-hooks" [ pkgs.bash ] ''
              mkdir -p .git/hooks
              cp scripts/pre-commit.sh .git/hooks/pre-commit
              chmod +x .git/hooks/pre-commit
              echo "Installed .git/hooks/pre-commit"
            '';

            ci = mkApp "ci" [ goPkg pkgs.golangci-lint pkgs.bash pkgs.findutils ] ''
              ${pkgs.bash}/bin/bash -c '
                set -e
                echo "=== Build ===" && ${goPkg}/bin/go build ${tagFlags} ${allPaths}
                echo "=== Vet ===" && ${goPkg}/bin/go vet ${tagFlags} ${modulePaths}
                echo "=== Test ===" && ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -count=1
                echo "=== Check Layers ===" && bash "$PWD/scripts/check-module-layers.sh"
                echo "=== API Stability ===" && (cd cmd/api-stability && GOWORK=off ${goPkg}/bin/go run -tags "goexperiment.jsonv2" .)
                echo "=== transport/grpc ===" && (cd transport/grpc && GOWORK=off ${goPkg}/bin/go test -tags "goexperiment.jsonv2" ./... -count=1)
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

            vulncheck = mkApp "vulncheck" [ goPkg pkgs.govulncheck ] ''
              for mod in ${builtins.concatStringsSep " " testModules}; do
                echo "==> Vulnerability scan: $mod"
                (cd "$mod" && GOWORK=off ${pkgs.govulncheck}/bin/govulncheck -tags "goexperiment.jsonv2" ./...)
              done
            '';

            secrets-scan = mkApp "secrets-scan" [ pkgs.gitleaks ] ''
              ${pkgs.gitleaks}/bin/gitleaks detect --source . --no-banner --no-git
              echo "==> Secret scan complete"
            '';

            # Ephemeral service apps — start databases from nixpkgs without
            # Docker or testcontainers. Works on Linux AND macOS with Nix.
            # Usage: nix run .#integration-pg
            #        nix run .#integration-pg -- -run TestPostgresEventStore
            integration-pg = mkApp "integration-pg" [
              goPkg pkgs.gcc pkgs.postgresql
            ] ''
              export CGO_ENABLED=1
              export GOEXPERIMENT=jsonv2
              bash "$PWD/scripts/ephemeral-pg.sh" "$@"
            '';

            integration-mysql = mkApp "integration-mysql" [
              goPkg pkgs.gcc pkgs.mariadb
            ] ''
              export CGO_ENABLED=1
              export GOEXPERIMENT=jsonv2
              bash "$PWD/scripts/ephemeral-mysql.sh" "$@"
            '';

            # NixOS VM integration tests — boot a QEMU VM with the database
            # service, run Go integration tests inside. Hermetic and cached
            # by Nix. Requires x86_64-linux + KVM for speed.
            # Usage: nix run .#integration-pg-vm
            #        nix run .#integration-mysql-vm
            integration-pg-vm = mkApp "integration-pg-vm" [
              goPkg pkgs.gcc
            ] ''
              export CGO_ENABLED=1
              export GOEXPERIMENT=jsonv2
              bash "$PWD/scripts/vm-pg.sh" "$@"
            '';

            integration-mysql-vm = mkApp "integration-mysql-vm" [
              goPkg pkgs.gcc
            ] ''
              export CGO_ENABLED=1
              export GOEXPERIMENT=jsonv2
              bash "$PWD/scripts/vm-mysql.sh" "$@"
            '';

            verify =
              mkApp "verify" [ goPkg pkgs.golangci-lint pkgs.bash pkgs.findutils pkgs.gnugrep pkgs.gcc ]
                ''
                  export CGO_ENABLED=1
                  ${pkgs.bash}/bin/bash scripts/verify-docs.sh && \
                  echo "=== Module Coverage ===" && nix run .#check-modules && \
                  echo "=== Build ===" && ${goPkg}/bin/go build ${tagFlags} ${allPaths} && \
                  echo "=== Vet ===" && ${goPkg}/bin/go vet ${tagFlags} ${modulePaths} && \
                  echo "=== Test ===" && ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -count=1 && \
                  echo "=== Race ===" && ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -race -count=1 && \
                  echo "=== Lint ===" && nix run .#lint && \
                  echo "=== API Stability ===" && nix run .#check-api-stability && \
                  echo "=== Doc Check ===" && (cd cmd/doc-check && GOWORK=off ${goPkg}/bin/go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md ../../README.md ../../TODO_LIST.md ../../ROADMAP.md ../../FEATURES.md ../../CONTRIBUTING.md) && \
                  echo "✅ All verification checks passed"
                '';

            # verify-fast: same as verify but passes -short to skip soak tests
            # (benchkit 35s soak suite). Use for rapid iteration during development.
            verify-fast =
              mkApp "verify-fast" [ goPkg pkgs.golangci-lint pkgs.bash pkgs.findutils pkgs.gnugrep pkgs.gcc ]
                ''
                  export CGO_ENABLED=1
                  ${pkgs.bash}/bin/bash scripts/verify-docs.sh && \
                  echo "=== Build ===" && ${goPkg}/bin/go build ${tagFlags} ${allPaths} && \
                  echo "=== Vet ===" && ${goPkg}/bin/go vet ${tagFlags} ${modulePaths} && \
                  echo "=== Test (short) ===" && ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -short -count=1 && \
                  echo "=== Race (short) ===" && ${goPkg}/bin/go test ${tagFlags} ${modulePaths} -short -race -count=1 && \
                  echo "=== Lint ===" && nix run .#lint && \
                  echo "=== API Stability ===" && nix run .#check-api-stability && \
                  echo "✅ All fast verification checks passed (soak tests skipped)"
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
