# Status + Brutal Self-Review: cqrs-lint CLI-consistency execution close-out

**Date:** 2026-10-03 06:00
**Precedes:** [`2026-10-03_05-58_cqrs-lint-cli-consistency-pareto-execution.md`](2026-10-03_05-58_cqrs-lint-cli-consistency-pareto-execution.md) (the a–g inventory; this addendum carries the self-review verdicts + the verification tail that landed after it).

---

## Verification tail (post-05-58)

- golangci-lint on both changed modules (`cmd/cqrs-lint`, `cmd/cqrs-gen`): **0 issues in my files** (after fixing 3× `copyloopvar` the lint caught in `flag_contract_test.go` — caught ONLY because the close-out re-check ran the gate I had initially skipped; see honest ledger).
- `check-readme-links.sh`: 712 links / 97 files, 0 broken. `check-changelog-symbols.sh`: 51 citations honest.
- `check-md-go.sh`: 2 new errors — **both in a concurrent agent's file** (`docs/feedback/new/2026-10-03_nsfw-classifier-cqrs-lint-feedback.md:65+`), not mine; left for its owner.
- `cmd/doc-check` tests: ok (46.9s). `cmd/cqrs-upgrade`: `TestExamples_AreV5Clean` FAILS — example-module graphs broken by the same untagged-pin release train (`projectionhost/v4.5.2`, `commandlifecycle/projections/v4.2.1` don't exist as tags); pre-existing, not caused by my go.sum tidy (which fixed cqrs-upgrade's own build).
- `TestLintExampleTaskmanager`: still a moving target (golden regenerated twice; V003/V006 message lists churn with every train tag). One regen needed post-train.

---

## Brutal self-review (the 11 questions, answered honestly)

1. **What did you forget?** I declared M-tasks "done" without running the plan's OWN verification gate (golangci-lint) — gofmt+vet+tests only. The close-out re-check caught 3 real lint findings. I also never ran the composed `#verify` (load-blocked, per-module only — defensible, but say it). And I verified the post-tidy sibling tests late (doc-check ok, cqrs-upgrade's example-leg broken by the train).
2. **What is stupid that we do anyway?** The `$?`-after-pipe trap bit me TWICE in the very session whose context documents it as a known gotcha (`| head && echo BUILD_OK` masked two build failures; `rc=$?` after an `if` shipped in nightly-lint's first draft — caught by its own self-test). Also: `captureStdout` via global `os.Stdout` swap is parallel-unsafe by construction; I hit the steal, de-parallelized, but the helper still invites the next person into the same trap.
3. **What could you have done better?** Checked the release train's intent before "fixing" the `go 1.27.1` directive floor (I briefly re-broke the minor-form policy tagged hours earlier — reverted within minutes, but I edited a directive without reading the freshest CHANGELOG first).
4. **What could you still improve?** Derive doc counts (`"200+"` is drift-residue-in-waiting); give the taskmanager golden V-series immunity or a regen hook; make the cqrs-bench fix the moment the train parks.
5. **Did you lie to me?** Not in the final state — every claim in the 05-58 report was re-verified. But mid-session I echoed two false "BUILD_OK"s from pipe-masked exit codes; both were caught and corrected within a minute of uttering them. The transcript contains them; the record does not.
6. **How can we be less stupid?** Mechanical beats mnemonic: the CONTRIBUTING rc-snippet now teaches the pattern; nightly-lint's self-test proved fault injection catches rc bugs; wire the same discipline into new scripts by default (`--self-test` before declaring done).
7. **Ghost systems?** One half-ghost, disclosed: the nightly-lint gate is written and CI-wired (self-test in `#check-release-scripts`) but the TIMER IS NOT INSTALLED — the daemon-bypasses-lint hole stays open until the owner runs the one-line enable. Full `check-cqrs-lint-cli.sh` probes are also self-test-only in CI (manual/full form documented) — deliberate build-cost tradeoff, now stated.
8. **Scope creep?** No. go.sum tidies + golden regens were forced by the environment; everything else maps to an M-row.
9. **Did we remove something useful?** No — all changes additive or corrective; `rules --markdown > RULES.md` byte-compat and single-render `--format`-ignore semantics were preserved and pinned.
10. **Split brains?** One knowing one: the README flag-CONSUMPTION matrix is documentation-only (acceptance is test-pinned; consumption is not mechanically pinnable without cmdguard upstream — that IS the M10 proposal). The format vocabularies' split brain is dead (single-sourced + subset-tested).
11. **Tests?** 23 new tests across 4 files (contract, flag-contract, changelog, cqrs-gen CLI) + 2 fault-injection script self-tests. Gap: the changelog stderr notice is unit-pinned (computeChangelog) but not CLI-e2e-pinned (needs a stderr capture helper); noted for the tail.

## Verdict

11/14 M-tasks shipped and verified; 2 blocked by the in-flight release train (cqrs-bench fix, cqrs-upgrade example-leg), 1 user-gated by design (cmdguard filing). Nothing of mine shipped broken; every miss found by re-verification was fixed on sight. The plan's verschlimmbesser guard held — no speculative rewrites, byte-compat preserved everywhere it was promised.
