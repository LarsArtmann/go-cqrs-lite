# Postmortem — the cqrs-lint truth session (2026-11-17 → 2026-11-18)

> **Type:** postmortem, not a status report. It covers three failure classes from the 2026-11-17
> session and the verification pass that followed. Each ends in a habit, not an apology.
> **Inputs:** [`2026-11-17_22-38_…`](2026-11-17_22-38_idempotency-debt-review-cqrs-lint-truth-fixes-and-harvest.md)
> §d, [`docs/evidence/2026-11-17_idempotency-fleet-wiring-survey.md`](../evidence/2026-11-17_idempotency-fleet-wiring-survey.md) §7,
> and the plan [`2026-11-18_06-41_SUPERB-…`](../planning/2026-11-18_06-41_SUPERB-idempotency-fleet-repair-and-linter-follow-through.md).

## 1. The framing miss: "the tag resolves" was read as "the module exists"

**What happened.** The session opened on the question "do we still have `idempotency/kvstore`? If so,
nobody uses it." The answer was assembled from proxy/tag evidence: tags `v4.0.0`–`v4.3.3` exist and
resolve, consumers pin them, therefore the module is present and in use. A delete plan was nearly opened
for a module that had already been deleted **two months earlier** (2026-10-17, delete-at-zero).

**Why it was wrong.** Go module proxies serve frozen tags forever. `git log --diff-filter=D --
idempotency/kvstore/go.mod` answers existence; `go list -m` does not. The same mistake, one level up, hid
the actual finding for a full turn: the **parent** `idempotency/v4` module's source was deleted on
2026-08-15 (ADR-0128), and seven fleet modules still require its frozen tag. "Tag exists ≠ source exists"
was the whole story, and the first answer had it backwards.

**Cost.** One wasted turn, one near-miss delete proposal, one plan written against the wrong premise.

**Habit.** Module existence is a `go.mod` at HEAD (root or `/vN`), never a tag. Encode it in the tooling
so the mistake cannot recur: F030 now reports deleted-module imports, and its registry drift test asserts
every entry's module is really absent at HEAD. Both directions are gated.

## 2. Commit hygiene: authored work shipped as daemon chores, one commit blended two sessions

**What happened.** Six consecutive commits for authored, reviewed work landed as
`chore: auto-commit N changed file(s) (heuristic)`. The worst one, `2bb70ab84`, mixed this session's three
doc-truth files (`references/core.md`, `references/modules.md`, `docs/DOMAIN_LANGUAGE.md`) with five files
from a **different session** (`v002_v006_test.go`, `v003.go`, `v003_latest.go`, `v003_latest_test.go`,
`v006.go`). An authored commit attempt later printed "nothing to commit" because the daemon had already
absorbed it.

**Why it matters.** The record cannot be audited: `git log` cannot answer "what did this session change",
and a future `git bisect` over the v003 work lands on a commit whose message describes documentation.
AGENTS.md's owner ruling (2026-11-03, Pareto T03.2) already authorizes hand-commits at **phase
boundaries** — the ruling existed and was not used.

**Habit.** Commit at every phase boundary with a real message, immediately, before the daemon's next pass.
Verify with `git rev-list --count origin/master..HEAD` and `git status --porcelain` before assuming a
commit landed. When the tree shows files you did not author, leave them alone and say so — the daemon will
fold them into its own commit; do not "tidy" them into yours.

## 3. Two self-inflicted red iterations, and one overclaim

**3a. A drift test that asserted the wrong thing twice.** The registry drift test first asserted the
module _directory_ was gone — but `idempotency/` exists because `sqlstore/` lives under it, so the test
was red against correct code. The second attempt asserted `go.mod` at the module path — but the module is
at `/v4`, so it was red again. The fix was a `moduleGoMods` helper globbing both `rel/go.mod` and
`rel/v[0-9]*/go.mod`.

**3b. A test that contradicted the analyzer's own precedent.** `TestFragmentMatchesImport` initially
asserted that an import of `example.com/…/go-cqrs-lite/idempotency` must NOT match the fragment. That
contradicts `module_detect_test.go:223`, which documents the opposite for exactly this shape. The correct
move was to fix the **test**, not the code — and that is what shipped.

**3c. The overclaim.** The session reported "all suites green" while `nix run .#lint-scope` and
`nix run .#verify` had never run. That was honestly qualified in the status report's §d, which is the right
instinct, but the qualification should have been the trigger to run them, not a substitute.

**Cost.** Two red-test cycles (~15 minutes) and a 24-hour delay in discovering that the file-size ratchet
was red at HEAD for reasons unrelated to this session (`loader.go` 380 lines, `project_config.go` 357 —
both grown by parallel auto-committed work).

**Habit.** When a new test fails, first ask which side is wrong: the code or the assertion. Look for an
existing test or doc that already settles the question. And when a gate is named in a completion claim,
run it before making the claim.

## 4. What the verification pass then found

Re-verifying the survey rather than trusting it was worth the time: six of its claims were wrong (§7 of
the evidence file), including one that would have sent a repair to the wrong app (KeyCountdown's middleware
order was already correct; github-local-sync's was not) and one that inverted the durability reasoning
(games/SEC's journal is persistent, so its in-memory dedup store is the _only_ thing that forgets).

It also surfaced a library defect nobody was looking for: `command/v4/doc.go` and its v5 mirror documented
the middleware chain backwards. That one is now fixed, pinned by
`TestMemoryBus_MiddlewareOrder_FirstAddedIsOutermost`, and cross-checked against `docs/middleware-ordering.md`.

## 5. The habits, in one list

1. Existence is a `go.mod` at HEAD, not a resolvable tag.
2. Commit at phase boundaries with real messages; never assume the daemon preserved your authorship.
3. Files you did not author are not yours to stage, revert, or reformat.
4. A failing new test is a question, not an instruction: decide which side is wrong before editing.
5. A named gate in a completion claim must have been run in that session.
6. Re-verifying your own evidence is cheaper than repairing a fleet on a wrong claim.
