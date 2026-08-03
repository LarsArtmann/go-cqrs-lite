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
        # Allow passwordless TCP connections for test VMs
        skip-name-resolve = true;
      };
    };
  };

  # Set up TCP-accessible user after MySQL starts.
  # The ensureUsers mechanism creates unix-socket-only users; for TCP access
  # from the host (via QEMU port forwarding), we need a password-authenticated user.
  systemd.services.mysql-post-init = {
    description = "Set up TCP user for MySQL";
    after = [ "mysql.service" ];
    requires = [ "mysql.service" ];
    wantedBy = [ "multi-user.target" ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
    };
    path = [ pkgs.mariadb ];
    script = ''
      mysql -u root <<'SQL'
      CREATE USER IF NOT EXISTS 'cqrs'@'%' IDENTIFIED BY 'cqrs';
      GRANT ALL PRIVILEGES ON *.* TO 'cqrs'@'%';
      FLUSH PRIVILEGES;
      SQL
    '';
  };

  # Open firewall for TCP connections from the host (QEMU port forwarding).
  networking.firewall.allowedTCPPorts = [ 3306 ];

  # Lean VM
  documentation.enable = false;
  services.xserver.enable = false;

  system.stateVersion = "25.05";
}
