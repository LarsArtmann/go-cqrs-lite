# CI-Heal Round 3 — Lint Campaign Closeout, Green-CI Chain, and the CI Flake Harvest (2026-11-21 23:59)

> Executive session executing `docs/planning/2026-11-21_19-50_SUPERB-lint-campaign-closeout-green-ci-chain.md` (M01–M12, F01–F52).
> Predecessors: [13-13 first sweep](2026-11-21_13-13_ci-heal-and-lint-campaign-first-sweep.md) · [17-02 mass execution](2026-11-21_17-02_ci-heal-round2-lint-campaign-mass-execution.md) · [19-21 landing delta](2026-11-21_19-21_report-landing-row-insert-battle-delta.md).

## §a What was done (all twelve medium tasks)

| ID  | Outcome                                                                                                                                                                                                                            | Evidence                          |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------- |
| M01 | Wave landed + pushed (first-ever push grant honored); CI run 44941784962                                                                                                                                                           | 9abcfc200 · d23ac9087 · 8e02a4b48 |
| M02 | duckdbengine → lint 0 (placeholders, sentinels, extractions)                                                                                                                                                                       | 3178ee607                         |
| M03 | storage/turso → lint 0; the "5 REAL bugs" were sqlclosecheck×DeferClose false positives — rows VERIFIED closed; the dead `turso/indexing/` row revived with corrected path, trimmed                                                | 46750c502                         |
| M04 | pgengine → lint 0 (priors, strides, sentinels, writeAggregateSelect)                                                                                                                                                               | 9c4be2a19                         |
| M05 | irohengine → lint 0 (checked assertions, suite fixtures, demo phases)                                                                                                                                                              | e331b6e4f                         |
| M06 | irohengine/quic → lint 0 (error-return demo, peer renames, consts)                                                                                                                                                                 | 4abf8405c                         |
| M07 | **Full #lint matrix 89/89 at ZERO**; config taxonomy header; the M01-push CI fallout healed (shfmt 3.14.1 heredoc, api golden, 4 orphaned nolints outside the red set)                                                             | cde46eb97                         |
| M08 | Local `nix flake check` PASS; two closeout pushes; green-candidate runs chained                                                                                                                                                    | 2f633621d · 0945ea6bc             |
| M09 | Coverage gate GREEN with REPAIRS not re-baselines (graphadapter +8.0, stack/pebble +10.7 via new tests); erraudit 104/104 clean; package-surface ✓; test-runtime ✓; untested-exports ✓ via 8 pin tests                             | 77040d720                         |
| M10 | Live backends: PG ✓ (ephemeral), Dgraph ✓ (120s full), KV full suites ✓ (pebble 42s, badger, bbolt matrix + 790s soak); MySQL-VM local runs hit server-teardown races (passing tests pass; CI MySQL legs green on the candidate)   | below                             |
| M11 | 44859030708 root cause = ambient-Go toolchain (job installs only nix; go.mod floor 1.27.2) + stderr swallowed; fixed via setup-go + stderr capture; Gates Rollup = red-CI aggregator (self-heals); SSE flake root-caused and fixed | 0945ea6bc                         |
| M12 | This report + add-status-row.sh helper (self-tested) + harvest                                                                                                                                                                     | below                             |

## §b The green-CI chain (three candidates, each teaching something)

1. **44941784962** (8e02a4b48): 5 red legs — shfmt heredoc form, orphaned queue nolints, api golden (new consts), flake-check lint leg, and the first SSE flake.
2. **44955692966** (cde46eb97): 3 red legs, all one root cause — quic transport.go 352 lines (my const names pushed it over the 350 ratchet) reddening File Size, Verify-fast, and flake-check at once. Fixed by splitting limits.go (2f633621d).
3. **44957878531** (2f633621d): 2 red legs — the SSE flake AGAIN (now a reproducible class) and the runtime gate's two environment-mode rows (doc-check cold-proxy, pgengine Docker-present). All fixed in 0945ea6bc; the next run is the receipt.

## §c The two honest corrections of inherited beliefs

- **The turso "5 REAL sqlclosecheck bugs" were not bugs**: every flagged site closes via `record.DeferClose` (ADR-0144 idiom, `_ = c.Close()` verified). sqlclosecheck cannot see through the helper — the same false-positive class already config-excluded for mysql/duckdb/pg engines and storage/v4. The exclusion row existed but its path `turso/indexing/` never matched any file (real paths: `storage/turso/v4/indexing/`); revived with the correct path, TRIMMED to sqlclosecheck only (the dead row's other 12 linters were never suppressing anything — reviving them would have newly silenced 12 green linters).
- **Exclusion rows match FULL repo-relative paths even though golangci runs from inside each module dir** — proven empirically (mysqlengine's row suppresses its DeferClose sites from a cwd-inside invocation). The dead-row lesson: verify a row matches by finding a path substring, not by trusting the author's intent.

## §d The SSE flake root cause (a test measuring the wrong stream)

`TestServeSSE_FastConsumerNeverDrops` asserted `SSEDroppedEvents() - before == 0` — but the counter is PACKAGE-GLOBAL and the parallel unit test `TestForwardWithDropOld_EvictionsCounted` deliberately evicts exactly two events. On a loaded runner those land inside the fast-consumer window: "fast consumer dropped 2 event(s)" was always a false attribution (the stream under test writes one event into a 64-slot ring and cannot drop). The assertion now tests only prompt streaming; the counter contract stays pinned deterministically. Two CI flakes, one test edit, zero behavior change.

## §e Verification ledger (the trust tier)

- Full `#lint` re-measure: **89/89 modules, 0 issues** (twice — the second run after the straggler sweep).
- `nix flake check`: **all checks passed** (includes the NixOS VM test).
- Coverage gate: **all 89 module + 170 package floors hold** with zero re-baselining — both drifts repaired by new tests (graphadapter `NewWithDriver`/`Driver` round-trip; stack/pebble Checkpoint-error + options/lifecycle).
- erraudit 104/104 clean; package-surface, test-runtime, untested-exports all green.
- Calibration priors: 8 engines' named consts frozen by `TestCalibrationPriorsPinned` (silent prior changes now test-visible).
- Live backends: PG, Dgraph, pebble, bbolt (incl. 790s soak), badger — green. Local MySQL-VM: teardown races at suite tail (infra class; CI legs authoritative and green).

## §f Residue / next items

1. **Green receipt**: the 0945ea6bc run must go green — then M21's TODO_LIST row closes with the URL (the T01.5 rule: scheduled/Dependabot runs don't count).
2. **Benchmarks re-dispatch after 0945ea6bc**: the setup-go fix is untested on CI (the fix is mechanically sound; first green Calibration Drift run proves it).
3. **Local shfmt is 3.13.1 vs CI 3.14.1**: probe-proxy-tags.sh is CI-form (local `shfmt -d` disagrees). Align the local tool (nix-provided shfmt 3.14) or accept the skew knowingly.
4. **mysql-vm local teardown race**: the QEMU server shuts down while late tests still connect ("invalid connection" class). Consider a drain wait in the VM harness or accept CI legs as authoritative.
5. **pgengine runtime row is CI-calibrated** (24s): a Docker-capable LOCAL host running the full suite would take ~70s vs a 68s local limit — documented, low risk.

## §g Questions for the owner

None blocking — the push grant covered every push made (never `--force`, fast-forward only).
