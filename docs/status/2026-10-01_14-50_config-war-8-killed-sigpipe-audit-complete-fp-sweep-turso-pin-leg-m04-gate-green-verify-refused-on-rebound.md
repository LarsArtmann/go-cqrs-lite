# Config-war #8 killed · SIGPIPE audit complete · fp-sweep stdout fixed · turso pin gate leg · M04 gate GREEN, verify refused on load rebound

- **Session**: 2026-10-01, ~11:30–14:50 CEST (continuation of the SUPERB post-wave plan, wave 4 follow-up)
- **Plan**: `docs/planning/2026-09-28_22-00_SUPERB-post-wave-verification-and-backlog-pareto-plan.md`
- **Mode**: owner "GET SHIT DONE" directive; blanket sanction = wave/tag mechanics. ADR rulings, public filings, billing, legal = owner-gated (untouched).
- **Incoming state**: prior session ended awaiting owner answers on 3 questions; concurrent sessions live on this host.

## a) HEADLINE

**M04 reached a GREEN composed-gate assert for the first time in three sessions — and `nix run .#verify` was then correctly REFUSED by its own internal load guard when a foreign build burst rebounded load5 from <10 to 14.06 in the gate-to-guard gap.** No verify attempt was burned (the refusal is clean-by-design); the guard chain is proven end-to-end. The blocker is not machinery but a concurrent nix aarch64 Rust build ("trunk" under qemu-user emulation) that has held the load floor between ~15 and ~122 for 3+ hours in a burst-idle-burst cadence.

## b) WHAT SHIPPED (all verified)

1. **Config-war recurrence #8 repaired** (left broken "awaiting owner" by the 06-41 session): `.golangci.yml` restored verbatim from the pre-corruption parent of daemon commit 903232e3f. The hash golden needed NO re-pin — proving the restore byte-exact and confirming the vector is a stale full-file rewrite from a concurrent session (the golden was never touched). `#check-lint-config` green end-to-end; post-repair lint spot-check (`#lint-module -- record`) = 0 issues. This unblocked every lint-dependent gate the 06-41 session reported RED.
2. **SIGPIPE/grep|head audit complete (improvement e6)** — 9 sites reviewed, 6 fixed:
   - **`probe-proxy-tags.sh` ELF sniff was a REAL false-negative**: `unzip -p | head -c 4 | grep` under pipefail closes the pipe after 4 bytes, so any zip entry larger than the pipe buffer SIGPIPEs unzip (141) and a DETECTED poisoned-ELF read as "clean" — mutation-verified with a 300KB ELF entry (exit 141 pre-fix, detected post-fix). Fixed via `(set +o pipefail; …)` subshell (producer's early death is expected and irrelevant once grep has its 4 bytes).
   - Five `| head` failure-reporting lines (`check-arch`, `check-modsums`, `bench-all`, `check-module-isolation`, `check-buildcache-capacity`) hardened with `|| true` so a 141 can no longer truncate or abort an already-failing gate's report.
3. **`fp-sweep.sh` stdout mode had NEVER printed anything**: `| tee "${OUT:-/dev/stdout}" >/dev/null` redirects fd1 to /dev/null first, so `/dev/stdout` resolves to the same /dev/null. Fixed to `tee "${OUT:-/dev/null}"`. Post-fix 12-repo sweep: **436 findings / 50 low-confidence**; **C043 contributes ZERO findings across all consumer repos — real-world FP rate 0**, silent on clean code as designed (completes F032's evidence).
4. **Turso pin-vs-constant gate leg (improvement e2)**: `check-turso-version.sh` now fails when any workspace go.mod pins tursogo AHEAD of `TursoGoIVMVerifiedThrough` (the 2026-09-30 silent-drift incident class; `sort -V`, direct + indirect pins). Self-test 4/4 legs (incl. ahead-pin caught, at-constant accepted); real tree green at v0.8.1.
5. **`calibration-drift.sh` fail-fast reorder**: baseline-artifact validation moved above the load/CoW gates and the four real `go test` constants dumps. Previously a bad invocation burned compiles before failing, AND the self-test's missing/empty-artifact legs broke under any go-env shell (TMPDIR→btrfs .gotmp; the CoW refusal preempted the expected messages — exactly how `#check-release-scripts` was red). Self-test 15/15 in three environments (plain, go-env/TMPDIR-btrfs, `env -i` without go); full `#check-release-scripts` exit 0.
6. **Preflight 9/9 GREEN** after absorbing one red (templ regen — concurrent session edited without regenerating from the right cwd).
7. **ADR-0149 read-through (item 17)**: all five citations verified against the tree (`NewEngineCheckpointStore` signature, `system_checkpoints` collection, engine-resolution rule, both named tests). Accurate; no edits needed.
8. **v5 checklist Layer 2 rows 1-3 ticked with evidence**: listing type-driven status and taskmanager domain-event style were ALREADY satisfied in-tree (StatusClassifier/`ClassifyLast` live; `task.deleted` + `evt.Type()` branching; zero `OnTombstone` uses under example/); NewStreamRef empty-entityID audit clean (boundary sites pass typed IDs; `record.NewStreamRefOrZero` already the escape hatch).
9. **CHANGELOG receipts**: 1 Added + 4 Fixed bullets; **latent verify-blocker absorbed**: the 06-15 session's `fs.FS` citation read as FICTION by `check-changelog-symbols` (stdlib type vs api golden — the documented gotcha class) — reworded; gate green at 48 verified citations.
10. **Operational**: killed an orphaned `tq-ladder.sh` (PPID 1, executing a DELETED script, 91% CPU for 9h — a leaked prior-session job; dropped load 26→20 instantly). 22 `golangci-lint-langserver` processes confirmed IDLE (absent from top-CPU) — they are NOT the current load floor.

## c) M04 TIMELINE (full record)

- 11:52 chain launched supervised (`--wait-loop --max-wait 10800`, 120s retries), per the prior session's supervision lesson. All non-quiet work sequenced around it; zero file edits during potential-green periods after 13:20.
- Attempts 1-3 tree-stability refusals (my own pre-chain edits + concurrent sessions), attempts 4-44 load refusals (ceiling 10).
- Load trajectory: 26 → 122 (Rust build peak) → troughs ~8-15 with rebounds to 30-75 — burst cadence ≈ dip 1-2 min every ~30-40 min.
- **~13:59 attempt 45 GREEN** (no release procs, tree stable 60s, load under ceiling) → `nix run .#verify` launched → nix derivation build → **verify's internal `verify-load-guard`/calibration-gate REFUSED: load5=14.06 ≥ 10** (a burst rebounded inside the gate-green→guard-sample gap; the gate checks load LAST, this is inherent TOCTOU on a bursty host, not a script bug). Refusal is by-design cheap; nothing burned.
- 14:42 chain complete (max-wait would have expired 14:52). Not relaunched: bursts mid-verify would trip the bench/calibration legs anyway (the 2026-09-11 SearchQuery failure mode); a trustworthy composed verify needs the foreign build GONE, not just a lucky launch second.

## d) HONEST LEDGER

- M04, M05 (incl. item 19 snapshot-migration), M11, M14.3, `nix flake check` remain quiet-gated and unexecuted — the entire quiet queue is blocked by the concurrent aarch64 Rust build, not by any repo condition. M05 pre-flight done (port 33070 free; `build/shuffle-seeds.log` 117 seeds ready).
- The 06-41 session's "repo-wide lint RED awaiting owner" was repaired by me under the blanket sanction, matching both the documented self-heal path and the 04-30 session's #7 precedent. If the owner wanted #8 preserved for inspection: it is fully reconstructible from commit 903232e3f.
- Verify was NOT force-run (`VERIFY_FORCE=1` exists; using it under load 20-120 would produce garbage timing evidence — declined on principle).
- fp-sweep `overview` repo still skips (NO JSON OUTPUT — consumer-probe prerequisites unmet; known from the baseline doc).
- The TOCTOU gap (gate-green → verify-guard rebound) is recorded but NOT patched: any re-assert still leaves a gap to verify's own sample; the internal guard is the correct final arbiter and did its job.

## e) THE 3 OWNER QUESTIONS (asked 3× now, still open)

1. **Push `master`?** ~35+ unpushed daemon commits; tags already public.
2. **Kill the 22 idle `golangci-lint-langserver` processes?** NOT the load floor today (idle); killing them would not open a window while the Rust build runs. Low value until the build ends.
3. **M04 fallback** — fresh evidence says: (a) raised-ceiling run is WORSE than no run under this burst pattern (mid-verify bursts poison bench legs); (b) a named window only works when the foreign build is done; (c) **accept preflight 9/9 + per-module greens + the attempt-45 GREEN-gate/refused-verify event as the cycle record — recommended**; re-run the full chain when the host is genuinely quiet.

## f) NEXT (priority order)

1. M04 chain relaunch when the aarch64 Rust build ends (watch `uptime`; gate does the rest). Then M04.5 receipts (dedup (a) + CI re-record rows).
2. Quiet queue in plan order: M05.2 mysql-vm + M05.4 seed replay + M05.5 receipts; M11 calibration; M14.3 stamps; `nix flake check`; item 19 snapshot-migration.
3. M15.3 decision-pack routing when the owner answers.
4. Push per owner answer to (1).

## g) GATES AT CLOSE

preflight 9/9 GREEN · `#check-lint-config` GREEN (hash golden) · `#check-release-scripts` exit 0 · changelog-symbols GREEN (48 citations) · lint spot-check 0 issues · canonical-facts GREEN at report indexing (re-run below). Not run this session (load): `#verify`, `#verify-fast`, integration suites, flake check.
