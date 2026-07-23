# SKILL.md Restructure — Brutal Self-Review & Status

> **2026-07-23 20:09** — Session focused on compressing SKILL.md to ≤1000 chars and fixing dropped content.

---

## Executive Summary

The primary task (≤1000 char SKILL.md body) **succeeded** — body is now **804 chars**. But the self-review surfaced **1 shipped bug, 3 missed issues, and several improvements** that were not caught before declaring done. The changes were **swept into an unrelated refactor commit** by a parallel process rather than getting their own focused commit.

---

## a) FULLY DONE ✅

| Item                             | Detail                                                                          |
| -------------------------------- | ------------------------------------------------------------------------------- |
| SKILL.md body ≤1000 chars        | **804 chars** — compact bulleted list, no table (formatter-safe)                |
| Stale `/v3` in evals.json        | Fixed → `/v4` (matches actual `go.mod` module paths)                            |
| Stale "28 modules" in AGENTS.md  | Removed — now says "per-module table" / "module READMEs"                        |
| Provenance note added to core.md | "About This Skill" section at bottom of core.md                                 |
| `nix run .#verify`               | Full gate passes: build + vet + test + race + lint + doc-check (923 refs valid) |
| Root SKILL.md symlink            | Confirmed working: `SKILL.md → .agents/skills/go-cqrs-lite/SKILL.md`            |

---

## b) PARTIALLY DONE 🟡

| Item                        | What's done                                                               | What's missing                                                                   |
| --------------------------- | ------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Provenance note in core.md  | Section added with doc-check command + progressive disclosure description | **Command path is WRONG** (see section d)                                        |
| evals.json version fix      | `/v3` → `/v4` fixed                                                       | Other stale version refs in core.md and faq.md not touched                       |
| AGENTS.md skill description | Updated to describe new structure + removed "28 modules"                  | Description still says "~1000-char index" — could be more precise ("≤1000-char") |

---

## c) NOT STARTED ⬜

1. **modules.md completeness audit** — 41 entries vs 55 actual `go.mod` files. Missing modules: `benchkit`, `dedup`, `metadata`, `idempotency/kvstore`, `example/readme-quickstart`, `cmd/cqrs-bench` (at least 6 modules missing from the reference table).
2. **Stale version refs cleanup in skill files** — `core.md` line 377 says "v3.6+", `faq.md` mentions "v3.5.0". User said "only the LATEST version is interesting."
3. **SKILL.md description field audit** — The `description:` in YAML frontmatter is **1051 characters**. This is the trigger text loaded into context on EVERY invocation. It's enormous and could be trimmed significantly without losing trigger accuracy.
4. **Symlink verification from `~/.config/crush/skills/`** — Never checked if the flake.nix shellHook symlink actually resolves correctly.
5. **Commit hygiene** — Changes were swept into `bd2bb7f7` ("refactor(core, benchkit, storage)") by a parallel process. No focused commit with a clear "docs(skill): compress SKILL.md to ≤1000 chars" message.

---

## d) TOTALLY FUCKED UP 💥

### BUG 1: Wrong doc-check command path in provenance note

**File:** `.agents/skills/go-cqrs-lite/references/core.md` (bottom, "About This Skill" section)

**What I wrote:**

```bash
cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../AGENTS.md ../../references/*.md
```

**What it SHOULD be:**

```bash
cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md ../../.agents/skills/go-cqrs-lite/references/*.md ../../AGENTS.md
```

The path `../../references/*.md` resolves to `/home/lars/projects/go-cqrs-lite/references/*.md` — **a directory that doesn't exist**. The correct path is `../../.agents/skills/go-cqrs-lite/references/*.md`. Anyone following the provenance note would get a file-not-found error.

**Root cause:** I wrote the command from memory without verifying against the actual AGENTS.md contributing section, which has the correct path.

### BUG 2: Premature "all done" declaration (process failure)

I marked all 7 todos complete and declared success **without**:

- Doing a diff-based content coverage check
- Verifying the doc-check command I wrote actually works
- Checking for stale version references in the other skill files
- Auditing modules.md completeness

This is the same failure mode described in the task brief as a "key mistake." I repeated it.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Immediate fixes (bugs I introduced)

1. **Fix the doc-check path in core.md provenance note** — `../../references/*.md` → `../../.agents/skills/go-cqrs-lite/references/*.md`
2. **Remove stale version refs** in core.md ("v3.6+") and faq.md ("v3.5.0") — user said only latest version matters

### Quality improvements

3. **modules.md is incomplete** — 41 entries vs 55 actual modules. Missing: `benchkit`, `dedup`, `metadata`, `idempotency/kvstore`, `example/readme-quickstart`, `cmd/cqrs-bench`. A consumer looking up these modules finds nothing.
4. **SKILL.md description is 1051 chars** — the trigger description loaded on every invocation. Could trim ~40% by removing the exhaustive module name list (the routing table handles discovery).
5. **Add a verification step to my workflow** — after writing ANY command/path in docs, copy-paste verify it works. After editing, grep for stale patterns across ALL skill files, not just the one I touched.
6. **Commit isolation** — changes should go in focused commits, not be swept into unrelated refactors. Need to either commit before parallel processes run, or explicitly stage only relevant files.

### Process improvements

7. **Content coverage diff** — Before declaring an extraction/restructure "done," diff old content against new to verify nothing was lost. The task brief literally listed this as a past mistake; I still didn't do it.
8. **Verify-then-declare** — Run the actual commands written in docs before marking complete. The doc-check path bug would have been caught instantly.

---

## f) Up to 50 Things We Should Get Done Next

### Critical (bugs introduced this session)

1. Fix doc-check path in core.md provenance note (`../../references/` → `../../.agents/skills/go-cqrs-lite/references/`)
2. Remove "v3.6+" from core.md line 377
3. Remove "v3.5.0" / "v3.5+" from faq.md lines 171, 174

### Skill content completeness

4. Add missing modules to modules.md: `benchkit`, `dedup`, `metadata`, `idempotency/kvstore`
5. Add missing examples to modules.md: `example/readme-quickstart`, `cmd/cqrs-bench`
6. Audit all 55 go.mod paths against modules.md entries — close the gap
7. Verify every module in the decision matrix (core.md §1) has a corresponding entry in modules.md
8. Check if `id/idtest`, `query/querytest`, `event/v4/eventtest` sub-packages need their own entries

### Skill quality

9. Trim SKILL.md description from 1051 chars — remove exhaustive module list, keep trigger phrases
10. Consider whether the description's module list ("event, command, query, decider, id, codec, storage, stack, kv, listing, projection, projectionhost, schema, signing, encryption, middleware, otel, catalog, watermill, scheduling, deriver, graph, prometheus, scenario") is needed or if trigger phrases suffice
11. Verify symlink works from `~/.config/crush/skills/go-cqrs-lite` after `nix develop`
12. Check if `nix fmt` touches any markdown via editor format-on-save (confirmed treefmt doesn't, but editors might)

### Evals

13. Add eval cases for newer modules (scheduling, deriver, graph, projectionhost)
14. Add negative eval case for "build a REST API in Go" (should NOT trigger — might confuse with transport/http)
15. Add eval case for "soft-delete" → should route to advanced.md §6.1 (tombstone)
16. Review trigger-eval-set.json — 10 true / 10 false, seems balanced but could add edge cases

### Cross-reference integrity

17. Verify all §-references in core.md decision matrix point to sections that actually exist
18. Add a cross-reference checker to doc-check (currently only checks Go symbols, not §-refs)
19. core.md §1 references "readmodels §2.3" — verify this section exists (it does in readmodels.md)

### Documentation alignment

20. AGENTS.md "AI Skill" section says "~1000-char index" — make it "≤1000-char" for precision
21. Check if CHANGELOG.md needs an entry for the SKILL.md restructure
22. Verify FEATURES.md doesn't reference the old monolithic SKILL.md structure
23. Check README.md for skill references that might be stale

### Broader skill ecosystem

24. Run the skill-creator eval workflow to test trigger accuracy post-restructure
25. Test that loading core.md + a reference file stays within context budget
26. Consider splitting core.md further if it grows (currently 26KB — approaching large)
27. Add a "last verified" date stamp to core.md provenance note
28. Consider adding mermaid/d2 diagrams to core.md for the core loop visualization
29. Check if the skill description triggers on "cqrs-htmx" questions (sibling project)

### Commit hygiene

30. Create a focused commit for the remaining fixes (bugs #1-3 above)
31. Add pre-commit hook that verifies doc-check commands in markdown actually resolve
32. Document the skill editing workflow in CONTRIBUTING.md

### Version/staleness

33. Audit ALL skill files for any "v2.", "v3." references that should be updated or removed
34. Check if `event/v3_compat_aliases.go` is referenced anywhere in skill docs
35. Verify the `event/v4/eventtest` import path in all skill docs matches actual module structure
36. Check if any doc references the old `eventtest` standalone module (pre-merge into event/)

### modules.md deep audit

37. Verify each module's one-liner in modules.md matches its current doc.go description
38. Check if import paths in modules.md use `/v4` suffix consistently
39. Add dependency information to modules.md (which modules depend on which)
40. Consider grouping modules.md by tier (matching the four-tier model from ADR-0046)

### Testing

41. Run `nix run .#test` to ensure no test references the old SKILL.md structure
42. Check if any golden test files reference skill content
43. Verify `cmd/doc-check` test (if any) passes with the new file structure

### Operational

44. Update the `docs/status/2026-07-23_17-07_SKILL-RESTRUCTURE-STATUS.md` to mark it as superseded
45. Clean up any temporary files from the restructure session
46. Verify `nix run .#verify` still passes after the remaining fixes
47. Consider adding a CI check that validates SKILL.md body ≤1000 chars
48. Consider adding a CI check that modules.md count matches go.mod count
49. Review if the provenance note's doc-check command should use the full verify set (with README.md, TODO_LIST.md, etc.) or just the skill files
50. Celebrate that the core structure is sound — progressive disclosure with thin index + reference files is the right pattern

---

## g) Questions I CANNOT Figure Out Myself

### Q1: Should the SKILL.md `description` field be trimmed?

The frontmatter `description:` is **1051 characters** and loaded into context on every trigger. It contains an exhaustive list of all module names AND trigger phrases. Trimming it risks reducing trigger accuracy (the skill might not fire when a user asks about `scheduling` or `deriver`). But 1051 chars is a lot of always-in-context tokens.

**I cannot determine the right tradeoff without knowing:** How sensitive is the trigger system to the description length vs. keyword coverage? Does removing module names from the description reduce trigger accuracy in practice?

### Q2: Should I create a new focused commit for the remaining fixes, or amend?

The changes from this session were swept into commit `bd2bb7f7` by a parallel process. The 3 remaining bug fixes (doc-check path, stale version refs) need their own commit. But the working tree currently has 282+ unrelated changes from the core refactoring session.

**I cannot determine:** Should I stage ONLY the 3 skill files and commit them in isolation, or wait for the parallel refactor to settle first? The AGENTS.md global rules say "NEVER revert changes you didn't author" — but these ARE my changes that need fixing.

### Q3: Is `example/readme-quickstart` and `cmd/cqrs-bench` supposed to be in the consumer-facing module reference?

These are internal tooling/example modules, not library modules consumers import. `modules.md` is described as a consumer reference. Adding internal-only modules might confuse consumers; omitting them means the count is inaccurate.

**I cannot determine:** What is the inclusion criteria for modules.md — "everything with a go.mod" or "everything a consumer would import"?
