# Session Status: Maintenance Sweep (9 TODO_LIST items)

**Date:** 2026-08-16 19:52
**Scope:** The 10-item maintenance batch pasted from the session TODO (docker
prune, flake examplePaths, CI timeout headroom, shfmt drift check, quic
flake-watch, reset_db helper, backuptest wiring, quickstart demos, soak suite,
repo-root junk).
**Overall:** 8 of 9 actionable items fully done and verified; 1 skipped by
user (repo-root junk). Soak suite re-run is GREEN across every engine,
including external PG + Dgraph. 2 pre-existing repo defects found and worked
around (broken `storage/backuptest/v4.0.0` tag; standalone-build replace rot
in bbolt/pebble/projectionadapter). 1 environment casualty (`/mnt/buildcache`
died mid-session — see §d).
**Verification:** per-module GOWORK=off tests + `-race` GREEN for everything
touched; live functional tests for reset-db.sh against throwaway PG and MySQL;
soak suites green on 11 engine packages; doc-check 898/898 references valid.
Full `nix run .#verify` NOT yet run post-batch (see §e).

---

## a) FULLY DONE

### 1. Docker image prune (TODO item 8)

- Verified zero repo references first: `mysql:8` and `mariadb:11` are absent
  from scripts/, .github/, flake.nix, nix/, and Go testcontainer code.
- **Kept**: `postgres:16-alpine` (testcontainers), `mysql:8.0`
  (`stack/mysql/testcontainer_test.go:44` uses it), `mysql:8.4` (backing a
  running container — see §f Q1).
- **Removed**: `mysql:8` (813MB) + `mariadb:11` (335MB) ≈ **1.1GB reclaimed**.
- Incidental: `mariadb:11` appears in a manual-setup comment in
  `stack/bench/benchkit_suite_mysql_test.go` (documentation only — docker run
  instructions for the bench suite, harmless to keep as text).

### 2. `example/metaengine-quickstart` in flake build surface (TODO item 1)

- Added `"./example/metaengine-quickstart/..."` to `examplePaths` in
  flake.nix — `#verify`/`#build`/CI now compile it (drift previously
  invisible).
- **Fixed the standalone build it revealed**: the example replaced local
  `event/` but not `metadata/`, so GOWORK=off builds failed on unpublished
  `metadata.BrandedString`/`ActorString` (documented replace-cascade gotcha).
  Added `metadata/v4 => ../../metadata`.
- Verified: `GOWORK=off go build` OK, `nix eval` of the build app OK,
  `go run .` produces correct output.
- Committed (folded by the auto-commit daemon into `9a0d38843`).

### 3. CI timeout headroom (TODO item 9)

- `metaengine/irohengine/convergence_suite.go`: `pollTimeout` 15s → **30s**
  with rationale comment. Passing runs still poll-exit early; only genuinely
  slow convergence under `-race` load pays the extra budget.
- flake.nix `#verify`: per-package Test `-timeout` 8m → **10m**, Race 8m →
  **12m**; `#verify-fast` Race(short) 8m → **10m**.
- Verified: irohengine vet + full convergence suite green after the change
  (loopback suite); flake evals clean.

### 4. CI `shfmt -d` drift check (TODO item 3)

- New `shfmt-drift` job in ci.yml: `nix shell nixpkgs#shfmt --command shfmt -d
  scripts/` on every push, 5-minute budget, placed right after the `check:`
  job.
- Rationale documented in the job comment: the pre-commit hook formats staged
  files only, so whole-tree drift was invisible (root cause of the 4x
  `LAYER[storage/memory]` map-key mangling).
- Verified: ci.yml parses as YAML; local `shfmt -d scripts/` clean.

### 5. QUIC convergence flake watch in CI (TODO item 6)

- New `quic-flake-watch` job in ci.yml: `TestQuicConvergenceSuite` under
  `-race -count=3 -timeout=10m` on every push (models the `cgo` job's nix
  develop + CGO + GOWORK=off pattern).
- **Locally verified the exact CI command**: 3 consecutive race runs green
  (1.4s total) — the Log order-tolerance fix from 2026-08-15 holds.

### 6. `reset_db` helper (TODO item 4)

- **New `scripts/reset-db.sh`**: `--pg` / `--mysql` / `--dry-run`. For each
  target: drops leftover `test_%` databases (crashed-run residue) and
  recreates the DSN's default database. Handles PG URL and keyword/value DSN
  formats (maintenance connection via appended `dbname=postgres`, libpq
  last-key-wins) and Go-style MySQL DSNs (`user:pass@tcp(host:port)/db`).
  DANGER banner: only point at throwaway test servers.
- **Wired into `scripts/test-integration.sh`**: external-DSN paths now call
  `reset_shared_db pg|mysql` first. Gated by `RESET_DB` (default **on** for
  external DSNs; `RESET_DB=0` opts out). Missing SQL client or failed reset
  → warning + continue, never blocks a test run.
- **Tooling**: added `postgresql` + `mariadb.client` to the devShell and to
  the `#test-integration` app runtimeInputs (note: nixpkgs renamed
  `mariadb-client` → `mariadb.client`).
- **Verified live**: throwaway PG 17 (URL + kv DSNs; dry-run leaves state
  intact; real run drops `test_1/2` and recreates `cqrs_test` empty) and a
  throwaway MySQL 8.0 container (after granting the repo-standard
  `ALL ON *.*` to the `cqrs` user — real setups grant this per
  vm-mysql.sh / testcontainer_test.go). shellcheck + shfmt clean.

### 7. `storage/backuptest` wired into bbolt + pebble (TODO item 2)

- **Decision: WIRE, not delete** — tag `storage/backuptest/v4.0.0` is
  published API; deletion would break the tag.
- Recovered both thin adapters verbatim from git history (`a6613ef0d^`):
  `storage/bbolt/backup_lifecycle_test.go` (bbolt `tx.WriteTo` file copy) and
  `storage/pebble/backup_lifecycle_test.go` (`Flush()` + `Checkpoint(dir)`).
- **Defect 1 found**: the published tag points at `d49311e12`, one commit
  BEFORE the module's go.mod was added (`934f3a852`) — the version is
  unresolvable from proxy/VCS. Worked around with `=> ../backuptest`
  replaces in both go.mod files (standard unpublished-sibling pattern).
  Proper fix = re-cut the tag (see §f Q2).
- **Defect 2 found**: bbolt+pebble required `event/v4 v4.6.0` but their
  source uses post-v4.7.0 `ReconstructEventWithAdoptedPayload` — their
  GOWORK=off standalone builds were silently broken. Bumped requires to
  v4.7.0 + added `=> ../../event` and `=> ../../metadata` replaces.
- Verified: `TestBackupRestore_FullLifecycle` + `IncrementalCheckpoints`
  pass in both modules; **full module suites pass with `-race` standalone**.

### 8. metaengine-quickstart graph + vector demos (TODO item 7)

- Split into `graph_demo.go` (follow network → `metaengine.OnRecordTyped` →
  `metaengine.Edge` folds; depth-1/2 BFS traversal via `ExecuteCtx`) and
  `vector_demo.go` (doc embeddings → `OnRecord` → `metaengine.Embedding`
  folds; euclidean k-NN via `VectorExecuteTyped`).
- `main.go` now runs three titled sections (Map/Graph/Vector ADT) through
  one pipeline shape: declare folds + queries → `Plan` over the Memory
  engine → `Apply` → typed execute.
- Verified: `go run .` output correct — graph depth-1 `{bob carol}`, depth-2
  `{bob carol dave}`; vector k-NN `go-basics d=0.000`, `go-cqrs d=1.000`.

### 9. Soak suite re-run after the graph/vector engine wave (TODO item 5)

All `TestSoak*` green, `SOAK_SKIP_*` unset, per-module GOWORK=off:

| Engine | Result | Time |
| --- | --- | --- |
| metaengine root (`MemoryBounded_10M`, `AutoCRUDByConvention`, `RecordAwarePipeline`) | PASS | 12.5s |
| sqliteengine | PASS | 0.8s |
| badgerengine | PASS | 0.4s |
| pebbleengine | PASS | 0.3s |
| bboltengine | PASS | 0.4s |
| projectionadapter | PASS | 0.2s (after replace fix, below) |
| duckdbengine (CGo) | PASS | 67.3s |
| tursoengine (embedded libSQL) | PASS | 2.6s |
| pgengine (ephemeral nixpkgs PG 17) | PASS | 3.9s |
| dgraphengine (`nix run .#ephemeral-dgraph`) | PASS | 82.9s |

- mysqlengine has no soak file (out of scope by construction).
- **Fixed en route**: `metaengine/projectionadapter` had the same standalone
  replace rot (missing `=> ../../event` + `=> ../../metadata`); added, tidy,
  soak green.
- Doc-check re-run after all edits: **898/898 references valid**.

### 10. Repo-root junk deletion — SKIPPED by user

Deferred with explicit "Skip: Deleting repo-root junk and orphaned stash for
now". Research preserved for when it resumes: `t/tasks.buf` (tracked binary),
`result` symlink (nix store), `reports/coverage.out` + `jscpd-report.json`,
`stash@{0}` (1-line flake.nix WIP on ancestor commit — safe to drop).

---

## b) PARTIALLY DONE

Nothing in-between: every started item is either fully done or user-skipped.

## c) NOT STARTED (out of batch scope)

- Repo-root junk deletion (user skip, above).
- Full `nix run .#verify` / `#verify-fast` gate for the batch — deferred to
  the next quiet window on purpose (see §d/§e: integration+verify must not
  run concurrently, and the soak batch just consumed the machine for ~20m).

## d) WHAT WENT WRONG / DEFECTS & SURPRISES

1. **`/mnt/buildcache` died mid-session** (I/O errors: `mkdir
   /mnt/buildcache/go-build: no such device`). All subsequent Go invocations
   used `GOCACHE=/tmp/gocache GOMODCACHE=$HOME/go/pkg/mod`. Earlier
   BuildFlow-hook govulncheck output shows the same failure, so the device
   was already unhealthy before this session. AGENTS.md still recommends
   /mnt/buildcache — needs a decision (§f Q3).
2. **Broken published tag `storage/backuptest/v4.0.0`** — points one commit
   before the module existed. Found while wiring; replaced-around locally.
   Anyone requiring that version from the proxy gets "missing go.mod".
3. **Standalone-build replace rot is systemic**: bbolt, pebble, AND
   projectionadapter all silently lost GOWORK=off buildability when local
   `event/`+`metadata/` gained unpublished symbols (workspace mode masked
   it). Each fixed with the documented require-published + replace-local
   pattern. `nix run .#vulncheck` should have caught bbolt/pebble earlier —
   worth checking why it didn't (likely the broken buildcache).
4. **Auto-commit daemon raced my explicit commit** (task 2): BuildFlow hook
   takes ~3.5min; the daemon committed mid-hook → `cannot lock ref HEAD`,
   exit 128. No work lost (daemon folded the changes into its own commit
   `9a0d38843`). Later tasks: let the daemon sweep, verified content landed.
5. **First manual-commit attempt failed in the hook** on the known
   host-toolchain issue (`GOTOOLCHAIN=local` + go 1.26.5 vs go.work ≥1.26.6);
   retry with `GOTOOLCHAIN=auto` passed the hook. Same class as documented
   AGENTS.md noise.
6. **Test-fixture stumble (reset-db MySQL)**: junk-creation `docker exec`
   failed silently because the plain `MYSQL_USER` lacks global CREATE (repo
   setups grant `ALL ON *.*` to `cqrs`). Re-tested after granting — revealed
   a real visibility constraint (information_schema hides schemas the DSN
   user cannot drop), which the DANGER docs now encode.

## e) VERIFICATION STATE & OUTSTANDING

- GREEN per-module: irohengine (vet + convergence), bbolt, pebble,
  projectionadapter, pgengine/dgraphengine/tursoengine/duckdbengine soaks,
  metaengine root soaks, example build+run, doc-check.
- **NOT yet run for this batch**: `nix run .#verify` (build+vet+test+race+
  lint+doc-check+api-stability+duplication+arch+coverage gates).
  Specifically unverified: `#check-arch` (backuptest is a new direct dep in
  bbolt/pebble — test-only usage, budget should be fine but unproven),
  `#check-duplication` (the two recovered adapters are intentionally similar
  test twins), api-stability golden (no exported symbols changed — test files
  and go.mod only — so no regen expected).
- Run the full gate exclusively (no concurrent heavy jobs) in the next
  session window.

## f) NEXT UP (prioritized)

1. Run `nix run .#verify` (+ `#verify-fast`) exclusively; fix anything it
   flags (arch budget / duplication baseline are the likely candidates).
2. Re-cut `storage/backuptest/v4.0.0` → tag `v4.0.1` at a commit containing
   the module (e.g. `934f3a852` or master), then drop the `../backuptest`
   replaces from bbolt/pebble. Requires tag + push approval (§ Q2).
3. Cut `event/v4.8.0` (contains `ReconstructEventWithAdoptedPayload` +
   branded metadata strings), then drop the `../../event` + `../../metadata`
   replaces from bbolt/pebble/projectionadapter and the example.
4. Investigate why `#vulncheck` didn't flag the broken standalone builds
   (probably the dead buildcache); re-run after cache decision.
5. Record the buildcache incident + `GOCACHE=/tmp/gocache` fallback in
   AGENTS.md once Q3 is answered.
6. mysql:8.4 + running `cqrs-conf-mysql` container: unreferenced by the repo
   — confirm ownership, then prune (~813MB).
7. Consider a weekly CI workflow running the engine soak matrix (the exact
   table above) so "re-run soaks after engine waves" stops being manual.
8. Optional polish: 4th quickstart section for the Search ADT
   (`IndexedText`), and route graph/vector demos through projectionadapter
   for full event-sourced symmetry.

## OPEN QUESTIONS (need user decisions)

1. **`cqrs-conf-mysql` container** (mysql:8.4, 127.0.0.1:33307, up 27h,
   zero repo references): keep (another project's?) or remove container +
   image?
2. **Broken tag repair**: re-cut `storage/backuptest/v4.0.1` and push
   (deleting/moving the broken `v4.0.0` remote tag), or leave replaces
   in place until the next scheduled release?
3. **`/mnt/buildcache`**: is the device coming back (re-mount/reboot), or
   should docs + workflows switch to `/tmp`/HOME caches permanently?

---

*Report generated after batch completion; soak table current as of 19:52.*
