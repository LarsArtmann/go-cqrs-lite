# Status Report: SKILL.md Restructure — 2026-07-23 17:07

## Task Summary

**Goal:** Make the base `SKILL.md` body ~1000 chars (an index/routing file) with all content moved into proper reference sub-files.

---

## a) FULLY DONE

1. **Created `references/core.md`** (26KB) — extracted all decision-making content from the old monolithic SKILL.md: mental model, 30-second quickstart, module decision matrix (45 rows), projection tier table, 8 critical conventions with code samples, 9 anti-patterns, testing patterns, dependency layering graph, examples table, "where to find more" table, and the full API cheat sheet.
2. **Created `references/faq.md`** (7.9KB) — extracted all 13 common pitfalls/FAQ entries with code samples.
3. **Rewrote base `SKILL.md`** to a thin index: one-line mental model + routing table to 6 reference guides.
4. **Converted root `SKILL.md` to a symlink** → `.agents/skills/go-cqrs-lite/SKILL.md` (was a divergent regular file; now a proper symlink as AGENTS.md always claimed it was).
5. **Updated `AGENTS.md`** skill structure description to reflect the 6 reference files.
6. **Ran `cmd/doc-check`** — all 928 Go references valid across 35 packages. Zero broken imports or symbols.

## b) PARTIALLY DONE

1. **Base SKILL.md body size** — Target was ≤1000 chars. Initial write was 984 chars, but `nix fmt` (or an auto-formatter) reformatted the markdown table with uniform column padding, inflating it to **1422 chars**. Needs compression (see section e).
2. **Content preservation** — Most content moved successfully, but two pieces were lost in the extraction (see section d).

## c) NOT STARTED

1. **Lint/format pass** — `nix run .#lint` not run after changes.
2. **Full `nix run .#verify`** — only doc-check was run, not the full verification gate (build + vet + test + race + lint).
3. **Evals update** — `evals/evals.json` and `evals/trigger-eval-set.json` not checked for stale references to old SKILL.md section numbers.
4. **Commit** — Changes are uncommitted.

## d) TOTALLY FUCKED UP / LOST

1. **Versioning note DROPPED** — The original SKILL.md had a critical note: _"recipes and imports target go-cqrs-lite v3.x... All import paths use the /v4 suffix, e.g. github.com/larsartmann/go-cqrs-lite/event/v4"_. This is NOT in `core.md` or anywhere else. **This is important context for AI consumers** — without it, they may use wrong import paths.
2. **"About This Skill" section DROPPED** — The provenance/verification metadata (doc-check verification count, restructure history) was in the original §13 and is now nowhere. Less critical but still a loss.
3. **Body exceeds 1000 chars** — The user's explicit constraint ("ONLY 1000 chars") is currently violated at 1422 chars due to formatter table padding. This is a direct failure of the primary requirement.

## e) WHAT WE SHOULD IMPROVE

1. **Fix body size** — Replace the padded markdown table with a compact bulleted list to get under 1000 chars and resist formatter inflation.
2. **Restore the versioning note** — Add the v3.x/v4 import path note to `core.md`.
3. **Restore "About" metadata** — Add a brief provenance/verification note to `core.md` or the base SKILL.md.
4. **Check evals** — Verify `evals/evals.json` prompts don't reference removed sections.
5. **Run full verification** — `nix run .#verify` to confirm nothing else broke.

## f) Up to 50 Things to Get Done Next

| #   | Task                                                                                                                          | Priority     |
| --- | ----------------------------------------------------------------------------------------------------------------------------- | ------------ |
| 1   | Compress base SKILL.md body to ≤1000 chars (use list, not table)                                                              | **CRITICAL** |
| 2   | Restore versioning note (v3.x / `/v4` suffix) into `core.md`                                                                  | **CRITICAL** |
| 3   | Restore "About This Skill" provenance section into `core.md`                                                                  | Medium       |
| 4   | Run `nix fmt` to confirm body stays under 1000 after formatting                                                               | **CRITICAL** |
| 5   | Run `nix run .#verify` (full gate: build+vet+test+race+lint+doc-check)                                                        | High         |
| 6   | Check `evals/evals.json` for stale section references                                                                         | Medium       |
| 7   | Check `evals/trigger-eval-set.json` for stale references                                                                      | Medium       |
| 8   | Verify global symlink `~/.config/crush/skills/go-cqrs-lite` still resolves                                                    | Medium       |
| 9   | Verify git tracks the symlink correctly (was regular file before)                                                             | High         |
| 10  | Consider whether `faq.md` content overlaps with `core.md` §4 anti-patterns                                                    | Low          |
| 11  | Consider adding a "start here" breadcrumb at the top of each reference file                                                   | Low          |
| 12  | Consider adding cross-links between reference files (e.g. core.md → recipes.md)                                               | Low          |
| 13  | Run the skill-creator eval workflow to test the restructured skill                                                            | Low          |
| 14  | Commit the changes once verified                                                                                              | High         |
| 15  | Note: `benchkit/generator.go` and `cmd/cqrs-bench/cqrs-bench` have unrelated modifications in git status — do NOT touch these | Info         |

## g) Questions I Cannot Answer Myself

1. **None.** The remaining issues (body size, lost content, verification) are all actionable without user input. The formatter inflation is solvable by switching to a list format. The lost content is identifiable from the diff. No business decisions or ambiguities remain.

---

## Files Changed This Session

| File                                             | Change                                 | Status                                   |
| ------------------------------------------------ | -------------------------------------- | ---------------------------------------- |
| `.agents/skills/go-cqrs-lite/SKILL.md`           | Rewritten from 34KB → ~2.2KB index     | Done (but body is 1422 chars, not ≤1000) |
| `.agents/skills/go-cqrs-lite/references/core.md` | **NEW** — 26KB decision-making content | Done (missing versioning note)           |
| `.agents/skills/go-cqrs-lite/references/faq.md`  | **NEW** — 7.9KB pitfalls               | Done                                     |
| `SKILL.md` (root)                                | Converted from regular file → symlink  | Done                                     |
| `AGENTS.md`                                      | Updated skill structure description    | Done                                     |

## Verification Results

```
doc-check: ✓ All 928 references valid across 35 package(s).
body size: ✗ 1422 chars (target: ≤1000)
versioning note: ✗ MISSING from core.md
provenance section: ✗ MISSING
```
