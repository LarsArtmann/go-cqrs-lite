# nix/vm/postgres.nix — NixOS module for the Postgres integration test VM.
#
# Built via `nix build .#pg-vm` or used by scripts/vm-pg.sh.
# Run manually: nix build .#pg-vm && QEMU_NET_OPTS="hostfwd=tcp::55432-:5432" result/bin/run-nixos-vm
{ pkgs, ... }:
{
  services.postgresql = {
    enable = true;
    package = pkgs.postgresql_16;
    enableTCPIP = true;
    ensureDatabases = [ "cqrs_test" ];
    ensureUsers = [
      {
        name = "cqrs";
        ensureDBOwnership = true;
      }
    ];
    # Trust all local connections — test VM only, never production.
    authentication = pkgs.lib.mkOverride 10 ''
      # TYPE  DATABASE  USER  ADDRESS       METHOD
      local   all       all                 trust
      host    all       all   127.0.0.1/32  trust
      host    all       all   ::1/128       trust
      host    all       all   10.0.2.0/24   trust
    '';
    settings = {
      listen_addresses = "*";
    };
  };

  # Lean VM — no docs, no X11, no audio
  documentation.enable = false;
  services.xserver.enable = false;

  # Port forwarding is set via QEMU_NET_OPTS in scripts/vm-pg.sh, not here.
  # The qemu-vm module (loaded by system.build.vm) provides:
  #   virtualisation.memorySize, virtualisation.diskSize
  virtualisation = {
    memorySize = 1024;
    diskSize = 4096;
  };

  system.stateVersion = "25.05";
}
