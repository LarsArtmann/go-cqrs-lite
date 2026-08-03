# nix/vm/postgres.nix — NixOS module for the Postgres integration test VM.
#
# Built via `nix build .#pg-vm` or used by scripts/vm-pg.sh.
# Also imported by the runNixOSTest checks in flake.nix.
{ pkgs, ... }:
{
  services.postgresql = {
    enable = true;
    package = pkgs.postgresql_16;
    enableTCPIP = true;

    # Create both the user and matching database (ensureDBOwnership requires this).
    ensureDatabases = [ "cqrs" "cqrs_test" ];
    ensureUsers = [
      {
        name = "cqrs";
        ensureDBOwnership = true;
      }
    ];
  };

  # Lean VM — no docs, no X11
  documentation.enable = false;
  services.xserver.enable = false;

  system.stateVersion = "25.05";
}
