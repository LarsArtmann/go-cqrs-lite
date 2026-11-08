# nix/vm/cqrs.nix — NixOS module for the ADR-0165 module test VM.
#
# Boots flake.nixosModules.cqrs with a standard-topology deployment
# (postgres WAL + sqlite projections) exactly as an operator would declare
# it. The runNixOSTest check in flake.nix asserts the rendered config, the
# peer-auth provisioning on real hardware, and runs the example binary
# (packages.standard-topology-example) against the rendered config — the
# service ExecStart itself stays app-owned by design; the test composes the
# state dir the way systemd StateDirectory would.
{ ... }:
{
  imports = [ ../nixos/cqrs.nix ];

  cqrs.deployments.demo.engines = {
    primary.driver = "postgres";
    proj.driver = "sqlite";
  };

  documentation.enable = false;
  services.xserver.enable = false;

  system.stateVersion = "26.11";
}
