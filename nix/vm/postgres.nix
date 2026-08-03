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

    # Allow TCP connections from the host (for QEMU port forwarding).
    # The runNixOSTest checks connect via Unix socket (trust by default),
    # but the standalone VM image (nix build .#pg-vm) needs explicit TCP auth.
    authentication = ''
      local all all trust
      host all all 0.0.0.0/0 trust
      host all all ::/0 trust
    '';

    # initialScript runs as the postgres superuser on first init.
    # This is more reliable than ensureDatabases + ensureUsers for test VMs.
    initialScript = pkgs.writeText "pg-init.sql" ''
      CREATE USER cqrs WITH SUPERUSER;
      CREATE DATABASE cqrs_test OWNER cqrs;
      CREATE DATABASE cqrs OWNER cqrs;
      GRANT ALL PRIVILEGES ON DATABASE cqrs_test TO cqrs;
      GRANT ALL PRIVILEGES ON DATABASE cqrs TO cqrs;
    '';
  };

  # Open firewall for TCP connections from the host (QEMU port forwarding).
  # The runNixOSTest checks connect via Unix socket/localhost (unaffected by
  # firewall), but the standalone VM scripts need external TCP access.
  networking.firewall.allowedTCPPorts = [ 5432 ];

  # Lean VM — no docs, no X11
  documentation.enable = false;
  services.xserver.enable = false;

  system.stateVersion = "25.05";
}
