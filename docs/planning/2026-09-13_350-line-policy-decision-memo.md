# Decision Memo — The 350-Line/File Policy: Ratchet, Split Waves, or Harness Exemptions?

**Date:** 2026-09-13 · **Status:** FOR OWNER DECISION · **Feeds:** Pareto plan §5 "Split-waves program (XL)" unblock

## The question

`nix run .#check-file-size` enforces max 350 lines/file, 30 lines/function,
with 58 historical offenders baselined (ratchet: shrink allowed, grow
forbidden). The policy's next phase is undecided. Three options are on the
table; they are not mutually exclusive.

## Evidence

| Fact                                   | Number                                                           | Source                           |
| -------------------------------------- | ---------------------------------------------------------------- | -------------------------------- |
| Baselined offenders                    | 58 files                                                         | `scripts/file-size-baseline.txt` |
| Largest offenders                      | ~950 lines (store.go, others)                                    | baseline file                    |
| Baseline violations since ratchet      | 0 (gate holds; 1 growth attempt caught and paid down 2026-09-13) | gate log                         |
| Harness/test monsters cited in reviews | `adttest` 953 / `enginetest` 935 lines                           | 2026-09-11 audit                 |
| Consumers of the policy                | every module; CI leg                                             | ci.yml                           |

## Options

### A. Ratchet-only (status quo, formalized)

Keep 58-file baseline permanent. No new offenders, existing ones shrink
opportunistically (store.go did: 953 → 940 this week via an extraction).

- **For:** zero churn; the gate demonstrably prevents accretion.
- **Against:** god-files persist indefinitely; "opportunistic" has no owner.

### B. Split-wave program (the deferred XL work)

Scheduled waves extracting the largest offenders module-by-module
(metaengine/store.go first: types out, apply-path out, health out).

- **For:** real readability/merge-conflict wins on the hottest file.
- **Against:** large refactor risk on the strategic module; needs dedicated
  quiet windows (same scarce resource as calibration re-runs); touches
  dozens of files → rebase pain for parallel sessions (the 2026-09-13
  queue/ extraction was fight-free only because store.go was untouched).

### C. Harness exemptions

`adttest`/`enginetest`-style table-driven test harnesses get a documented
exemption (e.g. baseline annotation `# harnessexempt`), shrinking the
baseline to production-code offenders only.

- **For:** honest signal — the baseline currently mixes "needs refactor"
  with "table-driven harness, size is intrinsic".
- **Against:** an exemption knob invites abuse; needs a tight definition.

## Recommendation

**A + C now, B decoupled from the policy.** Keep the ratchet (A). Add the
harness exemption for the two test-harness monsters (C) — this shrinks the
baseline list to ~56 and makes every remaining row a genuine refactor
candidate. Treat store.go's decomposition (B) as its own initiative with
its own risk budget, NOT as policy enforcement — the ratchet already
delivers the incremental shrink for free (proven 2026-09-13).

## Decision requested

1. Ratify A (ratchet as permanent policy).
2. Approve C (exemption annotation syntax + the two harness files).
3. Schedule or drop B (store.go split wave) as an independent effort.
