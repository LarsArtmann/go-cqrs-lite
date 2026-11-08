# nix/vm/cqrs.nix — NixOS module for the ADR-0165 module test VM.
#
# Boots flake.nixosModules.cqrs with a standard-topology deployment
# (postgres WAL + sqlite projections) exactly as an operator would declare
# it. The runNixOSTest check in flake.nix then asserts the rendered config
# and the peer-auth provisioning on real hardware. The service BINARY stays
# app-owned by design; running the example binary inside the VM lands with
# the system/v4.18.0 tag (the example's pre-release replace dies there and a
# plain buildGoModule package becomes possible — TODO_LIST §ADR-0165).
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
