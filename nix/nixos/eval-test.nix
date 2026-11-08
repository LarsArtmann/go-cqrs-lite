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
  # lockstep — compare via fromJSON so key order never matters.
  expected = {
    engines = {
      primary = {
        driver = "postgres";
        dsn = "postgres:///cqrs_demo_primary?host=/run/postgresql&user=cqrs_demo_primary";
      };
      proj = {
        driver = "sqlite";
        dsn = "/var/lib/cqrs/demo/proj.db";
      };
    };
    instances = [
      {
        engine = "primary";
        role = "source-of-truth";
      }
      {
        engine = "primary";
        role = "queries";
      }
      {
        engines = [
          "primary"
          "proj"
        ];
        role = "projections";
      }
    ];
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
          check =
            cond: msg:
            if cond then null else throw "cqrs-nixos-eval[${system}]: ${msg}";
        in
        [
          (check (builtins.fromJSON rendered == expected) "rendered deployment JSON drifted")
          (check pg.enable "postgres not enabled for auto-provisioned engines")
          (check (pg.ensureDatabases == [ "cqrs_demo_primary" ]) "ensureDatabases mismatch")
          (check (
            map (u: {
              name = u.name;
              owns = u.ensureDBOwnership;
            }) pg.ensureUsers
            == [
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
          (check (svc.environment.CQRS_DEPLOYMENT_CONFIG == "/etc/cqrs/demo/deployment.json") "hardenedService config env mismatch")
          (check (svc.serviceConfig.ProtectSystem == "strict") "hardenedService sandboxing mismatch")
        ]
      )
      [
        "x86_64-linux"
        "aarch64-linux"
      ]
  );

  rendered = (evalFor "x86_64-linux").environment.etc."cqrs/demo/deployment.json".text;

  ok = all (c: c == null) assertions;

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
