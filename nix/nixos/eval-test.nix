# Evaluation-level regression tests for flake.nixosModules.cqrs (ADR-0165).
#
# Pure nixosSystem evaluation — no VM, no build — asserting:
#   * the rendered DeploymentConfig JSON (engines, peer-socket DSNs, standard
#     three-instance topology),
#   * postgres auto-provisioning including the peer-auth pg_hba/pg_ident
#     wiring (nixpkgs defaults to md5 on TCP; the module maps service user
#     cqrs-<name> onto role cqrs_<name>_<engine> instead of using passwords),
#   * the cqrs.lib.hardenedService fragment.
#
# Evaluated for BOTH x86_64-linux and aarch64-linux on every host, so the
# module's eval surface is cross-architecturally pinned. The boot-level
# counterpart (real postgres, real service binary) is the vm-test check.
#
# Returns { script, rendered } — script is a runCommand body that forces every
# assertion and, on success, writes the rendered JSON to $out for debuggability.
{
  pkgs,
  module,
  # Optional: the flake.nixosModules.default import — when non-null its
  # render must be byte-identical to `module`'s (F39 alias pin).
  defaultModule ? null,
  # Shared instance-triple fixture (system/testdata/standard-instance-triple.json):
  # the SAME file the Go suite (system.Standard) and the CUE gate assert
  # against — the three preset twins cannot drift apart (ADR-0165 pin).
  tripleFixture,
}:
let
  inherit (pkgs.lib) flatten all hasInfix;

  evalFor =
    system:
    (import (pkgs.path + "/nixos/lib/eval-config.nix") {
      system = null;
      modules = [
        { nixpkgs.hostPlatform = system; }
        { system.stateVersion = "26.11"; }
        module
        {
          cqrs.deployments.demo.engines = {
            primary.driver = "postgres";
            proj.driver = "sqlite";
          };
        }
      ];
    }).config;

  # The exact deployment the round-trip fixture (system/testdata/
  # standard-topology.nixos-rendered.json) is generated from: keep the two in
  # lockstep — compare via fromJSON so key order never matters. Instances come
  # from the shared triple fixture, not inline, so the Nix twin is pinned to
  # the same bytes as the Go and CUE twins.
  expected = {
    engines = {
      primary = {
        driver = "postgres";
        dsn = "postgres:///cqrs_demo_primary?host=/run/postgresql&user=cqrs_demo_primary";
      };
      proj = {
        driver = "sqlite";
        dsn = "file:/var/lib/cqrs/demo/proj.db";
      };
    };
    instances = builtins.fromJSON (builtins.readFile tripleFixture);
  };

  assertions = flatten (
    map
      (
        system:
        let
          cfg = evalFor system;
          rendered = cfg.environment.etc."cqrs/demo/deployment.json".text;
          pg = cfg.services.postgresql;
          svc = cfg.cqrs.lib.hardenedService "demo";
          check = cond: msg: if cond then null else throw "cqrs-nixos-eval[${system}]: ${msg}";
        in
        [
          (check (builtins.fromJSON rendered == expected) "rendered deployment JSON drifted")
          (check pg.enable "postgres not enabled for auto-provisioned engines")
          (check (pg.ensureDatabases == [ "cqrs_demo_primary" ]) "ensureDatabases mismatch")
          (check (
            map (u: {
              inherit (u) name;
              owns = u.ensureDBOwnership;
            }) pg.ensureUsers == [
              {
                name = "cqrs_demo_primary";
                owns = true;
              }
            ]
          ) "ensureUsers / ensureDBOwnership mismatch")
          (check (hasInfix "local cqrs_demo_primary cqrs_demo_primary peer map=cqrs_peer" pg.authentication) "scoped peer pg_hba line missing")
          (check (hasInfix "cqrs_peer cqrs-demo cqrs_demo_primary" pg.identMap) "pg_ident map entry missing")
          (check (svc.serviceConfig.User == "cqrs-demo") "hardenedService user mismatch")
          (check (svc.serviceConfig.Group == "cqrs-demo") "hardenedService group mismatch")
          (check (svc.serviceConfig.StateDirectory == "cqrs-demo") "hardenedService StateDirectory mismatch")
          (check (
            svc.environment.CQRS_DEPLOYMENT_CONFIG == "/etc/cqrs/demo/deployment.json"
          ) "hardenedService config env mismatch")
          (check (svc.serviceConfig.ProtectSystem == "strict") "hardenedService sandboxing mismatch")
        ]
      )
      [
        "x86_64-linux"
        "aarch64-linux"
      ]
  );

  rendered = (evalFor "x86_64-linux").environment.etc."cqrs/demo/deployment.json".text;

  # F35: standardTopology = false with no explicit instances renders an EMPTY
  # instance list — the filler is opt-out per deployment.
  evalOptOut =
    (import (pkgs.path + "/nixos/lib/eval-config.nix") {
      system = null;
      modules = [
        { nixpkgs.hostPlatform = "x86_64-linux"; }
        { system.stateVersion = "26.11"; }
        module
        {
          cqrs.deployments.off = {
            standardTopology = false;
            engines.primary.driver = "postgres";
          };
        }
      ];
    }).config;

  # F36: an explicit dsn wins and provisions NOTHING (postgres stays off).
  evalExplicitDSN =
    (import (pkgs.path + "/nixos/lib/eval-config.nix") {
      system = null;
      modules = [
        { nixpkgs.hostPlatform = "x86_64-linux"; }
        { system.stateVersion = "26.11"; }
        module
        {
          cqrs.deployments.demo.engines.primary = {
            driver = "postgres";
            dsn = "postgres://external.example.com/app";
          };
        }
      ];
    }).config;

  # F39: the flake.nixosModules.default alias renders byte-identically to
  # nixosModules.cqrs (the alias site lives in flake.nix; the module passed
  # here is the default-alias import).
  evalDefaultAlias =
    if defaultModule == null then
      null
    else
      (import (pkgs.path + "/nixos/lib/eval-config.nix") {
        system = null;
        modules = [
          { nixpkgs.hostPlatform = "x86_64-linux"; }
          { system.stateVersion = "26.11"; }
          defaultModule
          {
            cqrs.deployments.demo.engines = {
              primary.driver = "postgres";
              proj.driver = "sqlite";
            };
          }
        ];
      }).config.environment.etc."cqrs/demo/deployment.json".text;

  extraAssertions = [
    (
      (builtins.fromJSON evalOptOut.environment.etc."cqrs/off/deployment.json".text).instances == [ ]
      || throw "cqrs-nixos-eval[standardTopology=false]: instances must be empty"
    )
    (
      evalOptOut.services.postgresql.enable
      || throw "cqrs-nixos-eval[standardTopology=false]: engine provisioning must stay on (it is independent of the instance filler)"
    )
    (
      !evalExplicitDSN.services.postgresql.enable
      || throw "cqrs-nixos-eval[explicit-dsn]: postgres must stay off when dsn is explicit"
    )
    (
      (builtins.fromJSON evalExplicitDSN.environment.etc."cqrs/demo/deployment.json".text)
      .engines.primary.dsn == "postgres://external.example.com/app"
      || throw "cqrs-nixos-eval[explicit-dsn]: explicit dsn lost"
    )
    (
      evalDefaultAlias == null
      || evalDefaultAlias == rendered
      || throw "cqrs-nixos-eval[default-alias]: renders differently from nixosModules.cqrs"
    )
  ];

  ok = all (c: c == null) assertions && all (v: v == true) extraAssertions;

  script =
    if ok then
      ''
        cat >$out <<'CQRS_JSON'
        ${rendered}
        CQRS_JSON
      ''
    else
      throw "cqrs-nixos-eval: assertions failed";
in
{
  inherit script rendered;
}
