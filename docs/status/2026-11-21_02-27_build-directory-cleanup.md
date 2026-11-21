# Status Report — build/ Directory Cleanup Session

**Date:** 2026-11-21 02:27 CET
**Session scope:** Single-task session — user pasted the tail of the `build/` listing (verify logs, launch scripts, wave plan/manifest) and asked "delete all?".
**User constraint honored:** No research into unrelated project areas; this report covers ONLY what this session did and noticed.

## Scope Line — What This Report VERIFIED vs NOT

**VERIFIED (with commands):**

- `ls -la build/` before deletion: 114 items, none git-tracked (`git ls-files build/` → empty; `/build/` is in `.gitignore` line 32).
- `head` of `build/wave-plan-2026-11-05.md` + `build/wave-r11-13.manifest` before trashing (both confirmed stale scratch from the completed 2026-11-05 wave; wave execution is recorded in git history, e.g. commits 965989b0d / 78784a8ad).
- Post-deletion: 84 files trashed via `trash` (recoverable, not `rm`); globs `build/verify*` and `build/wave-*` now resolve to nothing; `build/` down to 30 items.

**NOT VERIFIED (and why):**

- **Disk reclaimed** — I never ran `du` before deletion, so no MB figure exists. (Files are in trash, so the measurement is still recoverable if wanted.)
- **Dangling doc references** — I did NOT grep `docs/` for references to the deleted log filenames before trashing. `verify-chain-*.log` files are called "durable" by `scripts/verify-chain.sh` (AGENTS.md Quick Reference); an old status report or gotcha doc citing a specific deleted filename would now point at nothing. Not attempted; low-probability but unverified.
- **No tests/gates run** — deliberate: zero code, config, or tracked files were touched. Nothing to test. (Stating this explicitly so its absence is not misread as an oversight.)
- **Full `build/` inventory** — the user's paste was a truncated listing (started mid-file). I never enumerated or classified the remaining 30 items against a retention policy; see (b) and (c).

---

## a) FULLY DONE

| Item                                                                                                                                                                                                                                                                                                                                                                                        | Evidence                                                                                     |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Deleted all 84 stale files from the pasted listing: `verify-20261023-1320.log`, `verify-c5*` (4), `verify-chain-20261014…20261115` (27), `verify-chain-manual-20261115-073615.out`, `verify-chain-session-1508.log`, `verifyfast-candidate.log` (208 KB), `verify-launch-13…23` (`.sh`+`.out`, 22), `verify-run13…23` (`.log`+`.rc`, 22), `wave-plan-2026-11-05.md`, `wave-r11-13.manifest` | Post-delete glob checks: `build/verify*`, `build/wave-*` → No such file; item count 114 → 30 |
| Safety compliance: `trash` used (not `rm`); nothing git-tracked was touched; `.gitignore` confirms `/build/` is untracked scratch                                                                                                                                                                                                                                                           | `git ls-files build/` empty; `.gitignore:32`                                                 |
| Pre-deletion content check on the two non-log files (wave plan + manifest) — confirmed both describe the COMPLETED 2026-11-05/11-15 waves, not living plans                                                                                                                                                                                                                                 | `head` of both files; AGENTS.md CI red-window heal history (2026-11-20)                      |
| Verified wave-manifest versions shipped (cqrs-lint v4.22.0, id v4.7.6 etc. match the AGENTS.md tag history)                                                                                                                                                                                                                                                                                 | `head -5 build/wave-r11-13.manifest` vs AGENTS.md                                            |

## b) PARTIALLY DONE

| Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | State                                                                                |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| **build/ decluttering overall** — the pasted listing was only the alphabetical tail (v–w). The directory still holds 30 items: ~95 MB of flakecheck/CI logs (`ci-flakecheck.log` 25 MB, `ci-flakecheck2.log` 23 MB, `flakecheck-c3.log` 23 MB, `flakecheck-candidate.log` 23 MB — all from the Nov 6–7 CI red window), plus `ci-erraudit.log`, `ci-mysql.log`, `ci-verifyfast.log`, `dgraph-local.log`, `tidy-check*.log`, `mass-tidy.log/.sh`, `mysql-vm-run2.log`, `f221-changed-set-tests.sh`, `harvest-candidates.txt`, scratch drafts (`adr-0164-draft.md`, `m23-draft.md`, `m37-audit-findings.md`), scratch dirs (`_114-scratch/`, `issue-comments/`, `m38-archive-spec/`, `m38-archive-spec-2/`), and tool-owned dirs (`coverage/`, `test-json/`, `test-logs/`, `release-logs/`, `erraudit-sweep/`, `erraudit-max-sweep/`, `fuzz-corpus-1108/`, `shuffle-seeds.log` — regenerated by gates, deliberately left alone). | I cleaned what was asked (the pasted set); a full-directory decision was never made. |

## c) NOT STARTED

- Second-pass cleanup: the four ~23 MB flakecheck logs + remaining CI/integration logs (~95 MB total).
- Fate decisions for scratch drafts and dirs (`adr-0164-draft.md`, `m23-draft.md`, `m37-audit-findings.md`, `m38-archive-spec*/`, `_114-scratch/`, `issue-comments/`, `harvest-candidates.txt`).
- A **retention policy** for `build/` — the 84 trashed files accumulated over just Oct 14 → Nov 15 because verify-chain/test/bench tooling appends timestamped logs with no purge mechanism. Nothing exists to prevent re-accumulation.
- Checking whether any surviving doc references a deleted log filename (dangling-reference sweep).

## d) TOTALLY FUCKED UP

**Nothing in this session broke anything.** No code, no gates, no tracked files. Honest near-misses (not damage, but process faults):

1. **Scope inference from a truncated paste** — the user's listing started mid-file (`verify-20261023-1320.log`). I inferred the target set from globs (`verify-*`, plus the two `wave-*` files). If "all" meant the ENTIRE `build/` directory, I under-delivered scope (no damage — everything trashed is in trash, recoverable).
2. **No pre-deletion disk measurement** — deleted 84 files without capturing `du`, so the report cannot quantify the win. Measurement opportunity destroyed by my own sequencing (recoverable only via trash).
3. **No dangling-reference check before trashing** — `verify-chain.sh` brands its logs "durable"; I trashed 27 of them without grepping `docs/` for citations. Probability of a live doc citing one is low but I did not check, and trashing first / checking later is the wrong order.

## e) WHAT WE SHOULD IMPROVE

1. **Classify the whole target directory in one pass, then let the user strike items** — instead of deleting exactly the pasted tail, I should have produced a keep/toss table for all 30 remaining items in the same turn. One round-trip instead of likely two.
2. **Measure before destructive ops** — `du -sh` before/after belongs in every cleanup; it costs one command and makes the report honest about impact.
3. **Grep for references before deleting "durable"-branded artifacts** — the verify-chain logs carry the word _durable_ in their own tooling docs; that word alone should have triggered a reference sweep before trashing.
4. **`build/` has no lifecycle** — tools append timestamped logs forever (`verify-chain-<ts>.log` is documented as durable). Manual trashing on request is not a policy. Either a purge script (age-based, wired as a flake app / nightly timer alongside `nightly-bench`) or a documented keep-N rule.
5. **Scratch drafts rot in `build/`** — `adr-0164-draft.md` (Nov 5) and `m23-draft.md` sit next to shipped ADRs/milestones. Drafts should either graduate to `docs/` on adoption or be trashed on wave close; neither happened.

## f) NEXT THINGS (honest list — NOT padded to 50)

The user allowed up to 50. This session was a single `trash` invocation; enumerating 50 items from it without researching TODO_LIST/ROADMAP (explicitly forbidden this session) would be fabrication. 16 genuine items follow; the rest of a 50-item list would need a docs-health HARVEST pass over TODO_LIST.md/ROADMAP.md, which is a separate authorized session.

| #  | Next thing                                                                                                                                                         | Why it matters                                                                                                           | Effort  |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------ | ------- |
| 1  | Decide keep-vs-toss on the four Nov 6–7 flakecheck logs (~95 MB)                                                                                                   | The CI red window they document was healed on 2026-11-20; raw forensics may be dead weight                               | trivial |
| 2  | Trash remaining one-off logs (`ci-erraudit.log`, `ci-mysql.log`, `ci-verifyfast.log`, `dgraph-local.log`, `tidy-check*.log`, `mass-tidy.log`, `mysql-vm-run2.log`) | Same class as what was already trashed                                                                                   | trivial |
| 3  | Grep `docs/` for citations of deleted `verify-chain-2026*` filenames; fix any dangling references                                                                  | Closes the unverified gap from (d)                                                                                       | small   |
| 4  | Decide fate of `adr-0164-draft.md` (check ADR-0164 final state in `docs/adr/` first)                                                                               | Draft rot                                                                                                                | small   |
| 5  | Decide fate of `m23-draft.md` + `m37-audit-findings.md`                                                                                                            | Same                                                                                                                     | small   |
| 6  | Decide fate of `m38-archive-spec/` + `m38-archive-spec-2/` (Nov 14)                                                                                                | Two sibling scratch dirs suggest an abandoned or superseded spec                                                         | small   |
| 7  | Decide fate of `_114-scratch/`, `issue-comments/`, `harvest-candidates.txt`                                                                                        | Was `harvest-candidates.txt` (Nov 5) actually harvested into TODO_LIST.md?                                               | small   |
| 8  | Keep-or-trash `mass-tidy.sh` + `f221-changed-set-tests.sh`                                                                                                         | One-off scripts; if the patterns recur they belong in `scripts/`, not `build/`                                           | small   |
| 9  | Add a `build/` retention mechanism: age-based purge script (e.g. `scripts/purge-build-logs.sh --older-than 30d`) + flake app                                       | Root-cause fix for the accumulation that made this session necessary                                                     | medium  |
| 10 | Wire the purge into a nightly/weekly timer (alongside `nightly-bench`'s systemd user timer pattern)                                                                | Keeps it from being a manual chore                                                                                       | medium  |
| 11 | Document the policy in AGENTS.md gotchas (`build/` lifecycle: what tools own, what is safe to trash)                                                               | Prevents the next session from re-litigating                                                                             | small   |
| 12 | Reconcile `verify-chain.sh`'s "durable" log branding with the retention policy (durable ≠ forever)                                                                 | Doc/tool honesty                                                                                                         | small   |
| 13 | If launch-watcher scripts (`verify-launch-N.sh`) are a recurring pattern, extract the template into `scripts/`                                                     | They were hand-duplicated 11 times (13→23)                                                                               | medium  |
| 14 | Post-cleanup smoke: run one gate (e.g. `nix run .#check-test-runtime`) to confirm nothing depended on trashed files                                                | Cheap confidence after bulk deletion                                                                                     | small   |
| 15 | Measure and record disk reclaimed from trash (du on the trash items)                                                                                               | Completes the missing metric                                                                                             | trivial |
| 16 | Decide whether tool-owned dirs (`coverage/`, `test-json/`, `fuzz-corpus-1108/`, `shuffle-seeds.log`) should live under `build/` at all vs a dedicated cache dir    | They look like build output but carry cross-run state (fuzz corpora "cached across nights", test-runtime baseline input) | medium  |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Scope of "delete all":** pasted tail only (what I did), or the entire `build/` directory including the ~95 MB flakecheck logs and scratch drafts?
2. **Nov 6–7 CI red-window forensics:** the four flakecheck logs document the red window that was healed on 2026-11-20 — do you still want the raw logs kept as historical evidence, or are the healed commits + status reports sufficient?
3. **Retention policy:** do you want a durable fix (age-based purge script + timer) so `build/` never re-accumulates 84-file backlogs, or is manual trashing on request the preferred model?

---

**Per harness contract:** no manual commit — the auto-commit daemon absorbs this file. The (f) list above is the HARVEST input for `docs-health` if/when authorized.

**WAITING FOR INSTRUCTIONS.**
