# Fleet idempotency baseline — before any repair (C13)

> **Date:** 2026-11-20 (captured before C15–C17 swaps; the plan drafted this as `2026-11-18`, honest dating wins)
> **Tool:** `cmd/cqrs-lint` built at working tree following commit `11c5b23ec` (F030 scans test files; registry gated by C6b)
> **Command shape (the load-bearing part):** run from EACH REPO'S OWN cwd, `GOWORK=off`, `scorecard --path . --format json` for adoption rows and the bare lint command for F030 counts.

## 1. Measurement footguns discovered while capturing this baseline

1. **The `.cqrs-lint.json` preset binds from the process CWD, not `--path`.** ~~Running cqrs-lint against a
   consumer from `~/projects/go-cqrs-lite` silently applies the LIBRARY's own `preset: library`, which
   disables F030/V007 for the analyzed consumer — every finding vanished.~~ **FIXED 2026-11-20**
   (`8aaa19d9c` — cqrs-lint re-binds the project config from `--path` when the CWD is outside the target);
   verified fleet-wide by the 2026-11-21 receipts (§6b): all nine repos show per-repo evidence, none
   MISSING, F030 counts 0–3 by repo. All numbers below were captured from each consumer's own directory.
   (Tooling note: the cwd-vs-path config binding was a real footgun; fixed rather than warned.)
2. **Stale `go.work` files block `GOWORK=off`-less analysis and hide load errors under `GOWORK=off` too
   when module go.mods are stale.** `github-local-sync/go.work` still lists 23 pre-reorg `use` paths
   (`../go-cqrs-lite/catalog` etc.) that died in the /v4 module reorganization — every ambient `go`
   command in that repo fails. Other repos carry similar go.work files (43 cqrs-lite refs in Swetty,
   29 in precheck); under `GOWORK=off` they are bypassed, but then each module must be self-consistent:
3. **Two repos have modules that fail standalone loads**, which silences lint-level F030 there while
   the scorecard (AST-only, by design) still sees the imports:
   - `browser-history-precheck/api`: `behaviors_templ.go` calls `templruntime.NewAttributeExpression`,
     undefined in the vendored templ runtime — generated code newer than the dependency.
   - `games/SwettySwipperWeb/services/api`: `go mod tidy` required (`updates to go.mod needed`).

## 2. Baseline matrix (2026-11-20, per-repo cwd, GOWORK=off)

| # | Repo                     | HEAD        | Grade / cov | Modernity | Idempotency row                   | Deprecated API/transport | F030 |
| - | ------------------------ | ----------- | ----------- | --------- | --------------------------------- | ------------------------ | ---- |
| 1 | InboxClean               | `fb8c2385`  | Sparse 36%  | Legacy    | used \| shim `idempotency/v4`     | 0/1                      | 1    |
| 2 | browser-history          | `10402b3`   | Sparse 34%  | Legacy    | used \| `idempotency/sqlstore/v4` | 1/2                      | 2    |
| 3 | browser-history-precheck | `3bca1d4`   | Minimal 18% | Partial   | used \| shim `idempotency/v4`     | 0/0                      | 0 †  |
| 4 | games/KeyCountdown       | `d77118392` | Sparse 22%  | Legacy    | used \| `idempotency/sqlstore/v4` | 1/1                      | 1    |
| 5 | DiscordSync              | `935e8a10`  | Fair 44%    | Legacy    | used \| `idempotency/sqlstore/v4` | 174/0                    | 0 ‡  |
| 6 | games/SEC                | `ac388725`  | Sparse 25%  | Legacy    | used \| shim `idempotency/v4`     | 2/1                      | 1    |
| 7 | github-local-sync        | `277e3a6`   | Sparse 30%  | Legacy    | used \| shim `idempotency/v4`     | 3/1                      | 1    |
| 8 | Zlota44                  | `90ffcb8`   | Sparse 21%  | Legacy    | used \| shim `idempotency/v4`     | 0/3                      | 3    |
| 9 | games/SwettySwipperWeb   | `81e6d35ee` | Sparse 29%  | Partial   | used \| shim `idempotency/v4`     | 0/0                      | 0 †  |

† lint-level F030 invisible: a module of the repo fails standalone package loads (§1.3); the
scorecard's used-row (AST-level) still proves the shim import.

‡ DiscordSync imports `go-idempotency` DIRECTLY (survey row 5); its zero F030 is correct — no shim
import. The 174 removed-API uses are its own deprecated-surface debt, out of scope here.

## 3. What "done" looks like after C15–C17 (this file is the BEFORE)

Per the plan's Definition of Done #6, each swapped repo must afterwards show:

- `Idempotency: used` with evidence `github.com/larsartmann/go-idempotency` (direct), no shim line
  in any `go.mod`, F030 count drops to 0 for the shim rows (sqlstore rows persist until V007's v5),
- scorecard receipt re-captured from the SAME cwd shape as this baseline, appended to this file.

## 4. Reproduction

```bash
source /home/lars/projects/go-cqrs-lite/scripts/go-env.sh   # go on PATH, cache chain
cd <consumer-repo>                                          # PRESET BINDS FROM CWD — see §1.1
GOWORK=off cqrs-lint scorecard --path . --format json | jq '.summary, .used[] | select(.key=="idempotency")?'
GOWORK=off cqrs-lint --path . --format text | grep -c "deleted module"
```

## 5. AFTER receipts — C15–C17 pure swaps (2026-11-20, same command shape as §2)

| Repo                   | Swap commit | go.mod                                       | Scorecard idempotency                           | F030 | Build/tests                                                                                                                                                                                                                 |
| ---------------------- | ----------- | -------------------------------------------- | ----------------------------------------------- | ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| InboxClean             | `71bb6dd`   | direct `go-idempotency v0.3.2`, no shim line | used \| `github.com/larsartmann/go-idempotency` | 1→0  | build ok; emailcqrs suite green EXCEPT two labels tests that fail identically pre-swap (isolated-worktree verified)                                                                                                         |
| games/SEC              | `9f12298a`  | direct `go-idempotency v0.3.2`, no shim line | used \| `github.com/larsartmann/go-idempotency` | 1→0  | build ok; 8 app-suite failures byte-identical pre-swap (dep-bump fallout, isolated-worktree verified)                                                                                                                       |
| github-local-sync      | `dc1009f`   | direct `go-idempotency v0.6.0`, no shim line | used \| `github.com/larsartmann/go-idempotency` | 1→0  | build ok, internal/branch green; local (gitignored) go.work repaired to /v4 paths                                                                                                                                           |
| games/SwettySwipperWeb | `84a7352e7` | direct `go-idempotency v0.6.0`, no shim line | used \| `github.com/larsartmann/go-idempotency` | 0→0* | build ok, ALL services/api suites green standalone (first time since the reorg — replace block repaired to /v4, dangling eventtest replace deleted; *F030 was previously hidden by the broken module loads, now honestly 0) |

Notes:

- The SEC and github-local-sync commits initially landed as daemon `chore:` blends/races; both were
  split/recommitted with real messages while still unpushed (SEC's foreign `ci.yml` change was left
  unstaged for its author). v0.6.0 (gls, Swetty) vs v0.3.2 (InboxClean, SEC): dropping the shim's
  v0.3.2 floor let tidy pick latest; both minors build and test green in their repos.
- DiscordSync (already direct) and the sqlstore consumers (browser-history, KeyCountdown — V007's
  v5 territory) are correctly untouched by this tranche. Zlota44's shim usage is C21 (D28/D29-gated).

## 6. Post-swap fleet-commit ledger (C22 truth fixes + C23-partial Close hygiene + C24-ungated sweep; 2026-11-20)

Receipts for the fleet commits that followed the pure swaps, so no hash lives only in a session log.
Push state verified live 2026-11-21 — **all unpushed** (push authorization is owner question 3 of the
2026-11-21 01-28 status report).

| Repo                     | Commit      | What                                                                                                                                                                                                                               | State (2026-11-21)                                                                                                       |
| ------------------------ | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| InboxClean               | `5c208788`  | C23.3: `Setup.Close` closes the dedup `MemoryStore` (sweep goroutine + ticker no longer leak); `TestSetup_Close_Idempotent` added                                                                                                  | ahead 2 of origin (`71bb6dda` swap below it); `go.sum` is gitignored there — swap receipt could not stage it             |
| games/SEC                | `9f12298a`  | C15 swap (split out of daemon blend `a46c25c2`)                                                                                                                                                                                    | ahead 2; a later daemon chore `d084837c` sits on top (foreign, left alone)                                               |
| github-local-sync        | `138520e`   | C23.4: `Service.Close` stops the store's sweep ticker (`idemStore` field)                                                                                                                                                          | ahead 2 (`dc1009f` swap below it); local gitignored `go.work` repaired to /v4 paths (disposition pending owner taste)    |
| games/SwettySwipperWeb   | `0c82adeff` | C24-ungated: engine dedup collection swept on a 24h cadence (`idempotency_sweeper.go`, clock-injected tests, stops first in `buildStopList`)                                                                                       | ahead 5 (`84a7352e7` swap + replace repair below it)                                                                     |
| browser-history          | `66b66ea`   | C22: `/debug/idempotency` reports `SweepEnabled = TTL>0 && !UseSQLiteReadModel`, real `DedupLayers = 2` (+ sqlite-path test); tests committed unexercised — api module pre-existing unbuildable (templ runtime matrix, §1.3 class) | ahead 2, on master                                                                                                       |
| browser-history-precheck | `d195bf3`   | C22 mirror on the linked worktree's detached HEAD (parked detached by design)                                                                                                                                                      | **referenced only by that worktree's HEAD — orphanable** if the worktree is re-checked-out; the master twin is `66b66ea` |

### 6a. 2026-11-21 close-out additions (wave bump, SEC repair, labels fix)

| Repo                     | Commit               | What                                                                                                                                                                                                                                            | State (2026-11-21)                                            |
| ------------------------ | -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| browser-history-precheck | `4f361eb`→`15ffa0e`  | daemon chores absorbing the precheck regen + T9 scaffolding; full api suite FIRST-TIME green at `15ffa0e` (141s)                                                                                                                                | unpushed, detached-HEAD lineage; disposition = owner D36        |
| browser-history          | `dfe9f9f`            | daemon chore: the v4.16 wave bump (loginpage+adminui v4.16.0, usermgmt v4.18.0, templ v0.3.1118 regen ×14); master suite FIRST-TIME green (193s) + all 4 sibling modules                                                                           | ahead on master; retroactive wave blessing = owner D35         |
| games/SEC                | `059ef997`→`65685033` | cbid v0.7.0 prefix-leak repairs (projection keying, island runId JSON, cli create, SSE broker broadcast, grouping keys); SSE live-event routing bug + broken flake outputs pattern fixed; suite FIRST-TIME green (16 pkgs)                        | unpushed; dep-bump policy = owner D39                           |
| InboxClean               | `c1f96596`           | labels tests fixed at root cause: raw-ctor events default-declare JSON, CBOR payload bytes need `event.WithEncoding(DefaultCodec.Encoding())`                                                                                                   | ahead 3 (`5c208788`+`71bb6dda` below); go.sum policy = owner D37 |

### 6b. 2026-11-21 fleet receipts (re-bind fix proof, 9/9)

`scripts/fleet-dedup-receipt.sh` (same run shape as §2; script fixed first — `set -e` + cqrs-lint's
findings exit code had killed the sweep at the first repo with error findings). Artifacts:
`build/fleet-receipts/20261121-080009/`.

| Repo                     | HEAD        | idempotency evidence                                     | F030 |
| ------------------------ | ----------- | -------------------------------------------------------- | ---- |
| InboxClean               | `c1f96596`  | `github.com/larsartmann/go-idempotency` (direct)         | 0    |
| browser-history          | `dfe9f9f`   | `idempotency/sqlstore/v4` (deprecated, v5-removed)       | 2    |
| browser-history-precheck | `15ffa0e`   | `idempotency/sqlstore/v4` (deprecated)                   | 2    |
| games/KeyCountdown       | `d77118392` | `idempotency/sqlstore/v4` (deprecated)                   | 1    |
| DiscordSync              | `30d2d553`  | `idempotency/sqlstore/v4` (deprecated; no flagged sites) | 0    |
| games/SEC                | `65685033`  | `github.com/larsartmann/go-idempotency` (direct)         | 0    |
| github-local-sync        | `138520e`   | `github.com/larsartmann/go-idempotency` (direct)         | 0    |
| Zlota44                  | `90ffcb8`   | `idempotency/v4` (DELETED shim, proxy-served)            | 3    |
| games/SwettySwipperWeb   | `0c82adeff` | `github.com/larsartmann/go-idempotency` (direct)         | 0    |

Per-repo evidence diversity + zero MISSING = the re-bind fix works fleet-wide. The four sqlstore/
parent-shim repos are the D28/D3 migration tails (TODO_LIST 2026-11-21 harvest).
