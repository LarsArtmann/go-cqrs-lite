# ADR-0095: Nix-based Integration Testing

**Date:** 2026-08-03
**Status:** Accepted

## Context

The go-cqrs-lite project needs integration tests that verify database-dependent
code paths (PostgreSQL event stores, MySQL projections, DuckDB analytics, etc.)
against real database servers. Historically, these tests used `testcontainers-go`
which requires Docker.

Problems with the testcontainers approach:

- **Docker dependency**: Developers and CI must run Docker, which is heavy,
  unreliable in CI, and unavailable on some platforms (notably NixOS without
  Docker installed).
- **Slow startup**: Each test container starts a fresh database instance
  (~5-10s for PostgreSQL, ~10-15s for MySQL).
- **Non-hermetic**: Container images may drift between runs; the exact database
  version depends on the image tag.
- **Resource intensive**: Docker containers consume significant CPU and memory.

## Decision

Adopt **Nix-based integration testing** using three complementary approaches,
all powered by nixpkgs and pinned by `flake.lock`:

### 1. Ephemeral Process (PostgreSQL only)

`scripts/ephemeral-pg.sh` starts a PostgreSQL server from nixpkgs in a temp
directory using `initdb` + `pg_ctl`. No VM, no Docker. Auto-selects a free port,
overrides `unix_socket_directories` (NixOS requires root for `/run/postgresql/`),
runs tests with `GOWORK=off`, and cleans up on exit.

**Speed**: ~3s startup. **Best for**: Fast developer iteration on PG tests.

```bash
nix run .#integration-pg              # all PG integration tests
nix run .#integration-pg -- -run TestPostgresEventStore_CRUD
```

**Limitation**: Ephemeral MySQL without a VM is impossible on NixOS because
MariaDB's `mariadb-install-db` fails (the Nix store plugin directory is
read-only). MySQL 8.0 was removed from nixpkgs (EOL April 2026).

### 2. NixOS QEMU VM Tests (PostgreSQL + MySQL)

`nix/vm/postgres.nix` and `nix/vm/mysql.nix` define NixOS modules with the
database service. `pkgs.testers.runNixOSTest` boots a QEMU VM, waits for the
service to be healthy, and runs SQL assertions (JSONB, LISTEN/NOTIFY, etc.).

**Speed**: PG ~17s, MySQL ~131s (cached). **Best for**: CI, hermetic verification.

```bash
nix build .#checks.x86_64-linux.postgres-vm -L
nix build .#checks.x86_64-linux.mysql-vm -L
```

### 3. VM Launcher Scripts (interactive developer use)

`scripts/vm-pg.sh` and `scripts/vm-mysql.sh` use the `runNixOSTest` driver
to boot a VM, wait for the database service to be ready, then keep the VM
alive while Go tests run on the host against the port-forwarded database.

**Speed**: PG ~25s to first connection. **Best for**: Running specific
integration tests interactively against a real database.

```bash
nix run .#integration-pg-vm                          # run all PG integration tests
nix run .#integration-pg-vm -- ./storage/...         # specific package
nix run .#integration-mysql-vm                       # MySQL tests
```

## Alternatives Considered

| Approach                         | Docker? | Hermetic?                  | Speed          | Platforms             |
| -------------------------------- | ------- | -------------------------- | -------------- | --------------------- |
| **testcontainers-go** (previous) | Yes     | No (image drift)           | ~10s/container | Docker-capable only   |
| **Ephemeral process** (nixpkgs)  | No      | Yes (pinned by flake.lock) | ~3s            | Linux, macOS          |
| **NixOS QEMU VM** (runNixOSTest) | No      | Yes                        | ~17-131s       | Linux only            |
| **systemd-nspawn** (future)      | No      | Yes                        | ~5s est.       | Linux (kernel shared) |

## Key Design Decisions

1. **runNixOSTest driver, not eval-config.nix standalone VMs**: Standalone VM
   images from `eval-config.nix` boot but don't reliably start services (the
   service lifecycle is unmanaged). The `runNixOSTest` driver uses
   `machine.wait_for_unit()` which guarantees service readiness.

2. **Firewall rules required for TCP**: NixOS enables the firewall by default.
   The VM modules open the database ports (`networking.firewall.allowedTCPPorts
= [ 5432 ]` for PG, `[ 3306 ]` for MySQL) so QEMU port forwarding works.

3. **PostgreSQL TCP authentication**: The `authentication` option in
   `postgres.nix` sets `pg_hba.conf` to `trust` for all TCP connections, since
   these are test VMs with no sensitive data.

4. **Distributed-bus multi-VM test removed**: The 2-VM LISTEN/NOTIFY test was
   unverified, slow (3-5 min for 2 QEMU boots), and the single-VM checks already
   validate the same protocol semantics.

## Tradeoffs

- **NixOS-only for VM tests**: The VM tests require Linux (QEMU + KVM). macOS
  users use the ephemeral PG path (which works on Darwin) or testcontainers.
- **MySQL VM is slow**: 131s+ per test run due to QEMU boot. systemd-nspawn
  containers (future work) could reduce this to ~15s.
- **No ephemeral MySQL**: MariaDB's init is broken on NixOS. VM is the only
  Nix-only path for MySQL integration tests.

## References

- [Status report: Nix integration test infrastructure](../status/2026-08-03_04-19_nix-integration-test-infrastructure.md)
- [Execution plan](../planning/2026-08-03_04-24_nix-integration-test-execution-plan.md)
- [Session 2 status](../status/2026-08-03_08-27_nix-integration-test-session2.md)
- NixOS test driver: `nix build .#checks.x86_64-linux.postgres-vm.driver`
