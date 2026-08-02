# nix/vm/mysql.nix — NixOS module for the MySQL integration test VM.
#
# Built via `nix build .#mysql-vm` or used by scripts/vm-mysql.sh.
# Run manually: nix build .#mysql-vm && QEMU_NET_OPTS="hostfwd=tcp::33070-:3306" result/bin/run-nixos-vm
{ pkgs, ... }:
{
  services.mysql = {
    enable = true;
    package = pkgs.mariadb;
    ensureDatabases = [ "cqrs_test" ];
    ensureUsers = [
      {
        name = "cqrs";
        password = "cqrs";
        ensurePermissions = {
          "*.*" = "ALL PRIVILEGES";
        };
      }
    ];
    settings = {
      mysqld = {
        bind-address = "*";
        port = 3306;
      };
    };
  };

  # Lean VM
  documentation.enable = false;
  services.xserver.enable = false;

  # Port forwarding is set via QEMU_NET_OPTS in scripts/vm-mysql.sh.

  system.stateVersion = "25.05";
}
