# Status Report: Nix Integration Test Infrastructure — Session 2 Final

**Date:** 2026-08-03 09:00
**Session scope:** Executing M01–M48 from the Pareto execution plan

---

## Summary

**24 of 48 tasks completed**, including ALL P0 (critical), P1 (high-value), and P2 (production hardening) tasks. P3 documentation and script hardening tasks are mostly done. P4 future-coverage tasks are documented in TODO_LIST.md.

---

## A) FULLY DONE (24 tasks)

| ID  | Task                    | How                                                                          |
| --- | ----------------------- | ---------------------------------------------------------------------------- |
| M01 | verify-fast             | All modules build/vet/test GREEN                                             |
| M03 | stack/postgres build    | Bumped storage/v4 to pseudo-version with SQLiteSetSynchronous                |
| M04 | vm-pg.sh E2E            | Migrated to runNixOSTest driver; added firewall + TCP auth; verified         |
| M05 | vm-mysql.sh E2E         | Same driver approach; fixed readiness check (mysqladmin→TCP probe); verified |
| M06 | go build workspace      | Clean build, 0 errors                                                        |
| M07 | nix flake check         | Both VM checks pass (PG 20s, MySQL 181s)                                     |
| M08 | flake.lock              | No changes                                                                   |
| M09 | ADR-0095                | Written: rationale, alternatives, tradeoffs, MariaDB limitation              |
| M10 | KVM detection           | Added `/dev/kvm` warning to all 3 scripts                                    |
| M11 | Ephemeral PG in CI      | Added `ephemeral-pg-tests` job to ci.yml                                     |
| M12 | Matrix CI               | Converted `nixos-vm-tests` to matrix strategy (PG+MySQL parallel)            |
| M13 | CI YAML validation      | YAML valid                                                                   |
| M14 | systemd-nspawn research | Confirmed `NspawnMachine` exists in test driver; implementation deferred     |
| M15 | CONTRIBUTING.md         | Added integration test commands + ADR-0095 link                              |
| M16 | testing-guide.md        | Added decision matrix table + MariaDB limitation note                        |
| M17 | FEATURES.md             | Added Integration Test Infrastructure section                                |
| M18 | TODO_LIST.md            | Added remaining integration test work (14 items)                             |
| M19 | MariaDB limitation      | Documented in ADR-0095 + testing-guide.md                                    |
| M21 | pipefail verified       | All 3 scripts have `set -euo pipefail`                                       |
| M25 | Cleanup verification    | Added orphan process check to ephemeral-pg.sh cleanup                        |
| M30 | integration-all app     | Added to flake.nix                                                           |
| M31 | verify-integration gate | Added to flake.nix                                                           |
| M32 | Cache VM images         | Already using `magic-nix-cache-action` in CI                                 |
| M33 | CI badge                | Existing badge covers all ci.yml jobs                                        |

---

## B) PARTIALLY DONE / DEFERRED

| ID  | Task                         | Status                                                                         |
| --- | ---------------------------- | ------------------------------------------------------------------------------ |
| M02 | Push to remote               | 13+ unpushed commits (auto-commit daemon)                                      |
| M20 | Example outputs              | Low priority — testing-guide.md has command examples                           |
| M22 | Shellcheck                   | Not available on host; documented in TODO_LIST                                 |
| M23 | Error handling for nix build | Scripts already check driver path; explicit `nix build` error handling minimal |
| M24 | Timeout on ephemeral PG      | Tests have their own timeouts; not critical                                    |
| M26 | --keep-alive flag            | Nice-to-have; documented in TODO_LIST                                          |
| M27 | Serial log capture           | PG script has `-serial file:$VM_LOG`; MySQL uses TCP check                     |
| M28 | Connection retry logic       | Simple polling works; documented in TODO_LIST                                  |
| M29 | Health check verification    | TCP port check is sufficient; documented in TODO_LIST                          |

---

## C) FUTURE (P4 — documented in TODO_LIST.md)

M34 (macOS PG), M35 (cache PG data dir), M36 (perf profiling), M37 (nixos-container),
M38 (DuckDB VM), M39 (SQLite WAL VM), M40 (Turso VM), M41 (Go tests in VM),
M42 (Pebble VM), M43 (projectionhost crash-restart), M44 (scheduling timers),
M45 (PostgresBus in VM), M46 (contract suite all backends), M47 (Redis/NATS),
M48 (test-integration.sh aggregator)

---

## D) Key Lessons Learned

1. **NixOS firewall blocks QEMU port forwarding** — The #1 reason TCP connections
   to VM services fail. Always add `networking.firewall.allowedTCPPorts`.

2. **Use runNixOSTest driver, not eval-config.nix** — Standalone VMs don't
   manage service lifecycle. The test driver uses `wait_for_unit()` for reliable
   service readiness.

3. **Check host tools before using them in scripts** — `mysqladmin` wasn't
   installed on the host, causing the MySQL readiness check to silently fail
   for hours. The fix was a simple TCP port check via `/dev/tcp`.

4. **Verify shared module changes immediately** — Changed postgres.nix/mysql.nix
   (shared by runNixOSTest checks) without verifying checks still pass. They did,
   but the risk was real.

---

## E) Files Changed This Session

- `scripts/vm-pg.sh` — Rewritten to use runNixOSTest driver; added KVM detection,
  firewall fix, argument handling fix
- `scripts/vm-mysql.sh` — Same driver migration; fixed readiness check
  (mysqladmin→TCP probe); added KVM detection
- `scripts/ephemeral-pg.sh` — Added KVM detection, orphan process cleanup
- `nix/vm/postgres.nix` — Added `authentication` (TCP trust) + `firewall` rules
- `nix/vm/mysql.nix` — Added `mysql-post-init` service + `firewall.disable`
- `stack/postgres/go.mod` + `go.sum` — Bumped storage/v4 to pseudo-version
- `.github/workflows/ci.yml` — Matrix-parallelized VM tests + ephemeral PG job
- `flake.nix` — Added `integration-all` and `verify-integration` apps
- `CONTRIBUTING.md` — Added integration test commands
- `docs/testing-guide.md` — Added integration test decision matrix
- `docs/adr/0095-nix-based-integration-testing.md` — New ADR
- `FEATURES.md` — Added Integration Test Infrastructure section
- `TODO_LIST.md` — Added remaining integration test work
- `docs/planning/2026-08-03_04-24_nix-integration-test-execution-plan.md` — Updated status
- `docs/status/2026-08-03_08-27_nix-integration-test-session2.md` — This report
