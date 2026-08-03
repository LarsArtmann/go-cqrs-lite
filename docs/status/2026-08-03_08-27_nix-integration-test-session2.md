# Status Report: Nix Integration Test Infrastructure — Session 2

**Date:** 2026-08-03 08:27
**Session scope:** Executing the Pareto plan from `2026-08-03_04-24_nix-integration-test-execution-plan.md`

---

## A) FULLY DONE

1. **M01: verify-fast** — `nix run .#verify-fast` completed. All 64+ modules build, vet, test GREEN. Only failure was `api-stability` golden drift (stale from prior session, not our changes) — re-tested standalone, passes.
2. **M03: stack/postgres build fix** — `storage.SQLiteSetSynchronous` was undefined because `stack/postgres/go.mod` pinned `storage/v4@v4.5.0` (published tag predates the function). Bumped to pseudo-version `v4.5.1-0.20260731213813-4aa575d166cf` (commit `4aa575d1`). Build and test pass standalone (`GOWORK=off`).
3. **M04: vm-pg.sh E2E verified** — The script now uses the `runNixOSTest` driver (not the broken `eval-config.nix` standalone VM path). Root cause chain: (1) standalone VM boots but PG never accepts TCP → missing NixOS firewall rule. (2) Added `networking.firewall.allowedTCPPorts = [ 5432 ]` to `postgres.nix`. (3) Added `authentication = "host all all 0.0.0.0/0 trust"` for TCP auth. (4) Migrated script from `eval-config.nix` standalone VM to `runNixOSTest` driver with `--test-script` override (boots VM, waits for PG service, sleeps forever keeping VM alive). **Result: VM boots in ~20s, PG accepts TCP connections on port 55432, Go tests run and pass.**
4. **M06: go build workspace** — `go build -tags "goexperiment.jsonv2" ./...` passes clean.
5. **M07: nix flake check** — Both `postgres-vm` (20s) and `mysql-vm` (181s, +50s from new mysql-post-init service) checks pass. Verified AFTER all NixOS module changes.
6. **M08: flake.lock** — No changes to `flake.lock`. Nixpkgs input stable.

---

## B) PARTIALLY DONE

1. **M05: vm-mysql.sh E2E** — MySQL driver rebuilt with firewall fix (`networking.firewall.allowedTCPPorts = [ 3306 ]`). The MySQL NixOS module was enhanced with a `mysql-post-init` systemd service that creates a TCP-accessible `cqrs@%` user (the `ensureUsers` mechanism creates Unix-socket-only users). **Script written and driver built, but not yet tested end-to-end.** The mysql VM test takes 181s (vs PG's 20s) so testing is slow.

2. **VM script architecture rewritten** — Both `vm-pg.sh` and `vm-mysql.sh` were rewritten from `eval-config.nix` standalone VM images to `runNixOSTest` driver-based approach. The key insight: `eval-config.nix` standalone VMs boot but don't properly manage service lifecycle (PG never started reliably). The `runNixOSTest` driver uses `machine.wait_for_unit("postgresql.service")` which guarantees service readiness. Scripts use `--test-script` flag to override the built-in test with a custom script that boots the VM, waits for the service, then sleeps forever (keeping the VM alive for external tests).

---

## C) NOT STARTED

3. **M02: Push** — 13 unpushed commits on master (auto-commit daemon committed all session work).
4. **M09: ADR-0094** — Not started. Latest ADR is `0093`.
5. **M10: KVM detection** — Not started.
6. **M11: Ephemeral PG in CI** — Not started.
7. **M12: Matrix-parallelize CI** — Not started.
8. **M13: actionlint CI YAML** — Not started.
9. **M14: systemd-nspawn** — Not started.
10. **M15–M48** — All not started (docs, quality hardening, future coverage).

---

## D) TOTALLY FUCKED UP

1. **Wasted ~2 hours on standalone VM debugging** — The `eval-config.nix` standalone VM path (`nix build .#pg-vm`) boots NixOS to a login prompt but PostgreSQL never accepts TCP connections. Tried: `-display none` (GTK fix), `-serial file:` (log capture), `authentication` config (pg_hba.conf), increasing wait time to 300s. The serial console showed only 530 bytes (just the agetty prompt, no kernel messages). Root cause was misdiagnosed as a display issue when it was actually the firewall. **Should have checked the NixOS firewall FIRST** instead of going down 4 different debugging paths.

2. **Postgres.nix/mysql.nix modified without immediate verification** — Changed the shared NixOS modules (adding `authentication`, `mysql-post-init` service) that are imported by BOTH the standalone VM AND the `runNixOSTest` checks. Didn't verify the runNixOSTest checks still pass until much later. Fortunately they did, but the risk of breaking the CI checks was real.

3. **Script argument passing bug in vm-pg.sh** — When passing `-run TestPostgresEventStore_CRUD`, the script's `else` branch prepends its own `-run`, producing `-run "-run TestPostgresEventStore_CRUD"` which matches nothing. Not critical (works fine when passing test name directly or `./package/...`), but sloppy.

4. **Stale processes left around** — Multiple QEMU processes and disk images were left running/created in `/tmp/` during debugging. Cleaned up eventually but left a mess during the session.

---

## E) WHAT WE SHOULD IMPROVE

1. **Check NixOS firewall FIRST** — The #1 reason TCP connections to VM services fail. Default NixOS has `networking.firewall.enable = true` which blocks all non-standard ports. Any VM service accessed via QEMU port forwarding needs `networking.firewall.allowedTCPPorts`.

2. **Use runNixOSTest driver, not eval-config.nix** — The standalone VM images from `eval-config.nix` are unreliable for service testing. The runNixOSTest driver handles the full lifecycle (boot, service readiness, cleanup) and is the canonical NixOS way to test services.

3. **Verify shared module changes immediately** — When changing modules imported by runNixOSTest checks, run the checks immediately after, not hours later.

4. **Fix the vm-pg.sh argument handling** — The `-run` flag doubling bug needs fixing before documenting the scripts as the primary dev path.

---

## F) Next Tasks (Priority Order)

### Immediate (P1)
1. **M05** — Test vm-mysql.sh E2E (driver built, just needs running)
2. **M02** — Push all commits
3. **Fix vm-pg.sh argument handling** — Strip leading `-run` from args

### P2 — Production Hardening
4. **M09** — Write ADR-0094 (Nix-based integration testing)
5. **M10** — Add KVM detection to VM scripts
6. **M11** — Add ephemeral PG fast path to CI
7. **M12** — Matrix-parallelize nixos-vm-tests CI job
8. **M13** — Validate CI YAML with actionlint
9. **M14** — Investigate systemd-nspawn

### P3 — Documentation
10. **M15** — Update CONTRIBUTING.md
11. **M16** — Add docs/testing-guide.md
12. **M17** — Update FEATURES.md
13. **M18** — Update TODO_LIST.md
14. **M19** — Document MariaDB/NixOS limitation
15. **M20** — Add example outputs

### P3 — Quality Hardening
16. **M21** — Verify pipefail on scripts
17. **M22** — Shellcheck linting
18. **M23** — Error handling for nix build
19. **M24** — Timeout on ephemeral PG
20. **M25** — Cleanup verification
21. **M26** — --keep-alive flag
22. **M27** — Serial log capture
23. **M28** — Connection retry logic
24. **M29** — Health check verification
25. **M30** — integration-all app
26. **M31** — verify-integration gate
27. **M32** — Cache VM images in CI
28. **M33** — CI badge

### P4 — Future
29–48. All future coverage tasks (DuckDB VM, SQLite WAL VM, Turso VM, Go tests in VM, Pebble VM, projectionhost crash-restart, scheduling timers, PostgresBus in VM, contract suite, Redis/NATS, test-integration.sh)

---

## G) Questions

1. **Should the `pg-vm`/`mysql-vm` standalone VM packages (eval-config.nix) be kept or removed?** They're now superseded by the test-driver-based scripts. Keeping them means maintaining two VM build paths. Removing them simplifies the flake but breaks the `nix build .#pg-vm` → `result/bin/run-nixos-vm` workflow.

2. **Should the mysql-post-init systemd service be moved into the testScript instead?** Currently it runs as a systemd service in the VM (slower boot, +50s). Moving it to the testScript would be faster but less robust (the service runs once at boot; the testScript runs on every test).

3. **Should we accept the 181s MySQL VM test time, or invest in systemd-nspawn (M14) now?** The MySQL VM test went from 131s to 181s (the mysql-post-init service adds overhead). At 3 minutes, it's borderline for per-commit CI. systemd-nspawn could bring it back under 30s.
