# nix/vm/mysql.nix — NixOS module for the MySQL integration test VM.
#
# Built via `nix build .#mysql-vm` or used by scripts/vm-mysql.sh.
# Also imported by the runNixOSTest checks in flake.nix.
{ pkgs, ... }:
{
  services.mysql = {
    enable = true;
    package = pkgs.mariadb;
    ensureDatabases = [ "cqrs_test" ];
    ensureUsers = [
      {
        name = "cqrs";
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

  system.stateVersion = "25.05";
}
