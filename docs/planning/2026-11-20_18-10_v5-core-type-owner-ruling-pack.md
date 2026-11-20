# v5 Core-Type Finalization — Owner Ruling Pack (M01)

> One page, three decisions. Companion to
> [the plan](2026-11-19_20-52_SUPERB-v5-core-type-finalization-pareto-plan.md) (§1 records
> the evidence base; M22 pinned the window as gate-ordered). Each ruling names the exact
> consequence of the opposite answer, so a veto is one sentence.

**Execution status (2026-11-20 18:10):** the session runs under a blanket execute order
("keep going until everything works") for the commissioned plan of record. All three
rulings are **EXECUTING PER RECOMMENDATION** — every change lands in `core/v5` (zero
fleet consumers, verified by `grep -rl 'go-cqrs-lite/core/v5' ~/projects/*/go.mod`),
leaves the v4 train byte-identical (M23 proves it), and is git-revertible as single
wave commits. Veto any ruling retroactively by naming it; the corresponding commits
revert cleanly. Recorded as TODO_LIST D-sheet D32.

---

## Ruling A — T26 reconciliation: deprecated twins stay; new surface finalizes now

**Question.** The 2026-11-19 review (R1) originally prescribed deleting core/v5's 61
`Deprecated:` markers now. Research found the recorded counter-decision. Ratify the
reconciliation?

**Evidence.** `cmd/cqrs-lint/v4/pkg/rules/version/v007_tables_symbols_v5.go` documents the
deprecated twins as *"removed together with the v4 train in T26"* — the strategy that keeps
`cqrs-upgrade --to-v5` import rewriting mechanical (a v4 consumer of a deprecated API lands
on the same-name core/v5 API; the deprecated surface dies once, with the v4 train).
art-dupl accepts in core/v5 say the same.

**Options.**
- **Stay (recommended):** twins ride until T26; the pre-first-tag window is spent on the
  NEW surface (brands, ADR-0166 alias, Payload, Cause, stamps).
- **Cut now (review's original R1):** strands mid-migration consumers on names the codemod
  can no longer map mechanically; contradicts the recorded T26 strategy.

**If you veto:** Wave 1 targets change from additive retyping to deletion — plan re-run
required; nothing else in W0–W5 survives as ordered.

## Ruling B — ADR-0166 ratification timing: flip Proposed→Accepted when M04 lands

**Question.** ADR-0166 says "ratifies at the v5 cut". M04 implements it in core/v5 before
the cut. Flip the status at implementation, or keep Proposed until the full cut?

**Evidence.** The ADR's own Decision text is what M04 implements, verbatim (alias, validating
ctors, StreamKey disk truth, slash form dies). The "at the v5 cut" line was written
2026-11-08, before the pre-first-tag window existed as a concept; core/v5 pre-first-tag IS
the v5 surface being born. Leaving Proposed after enactment repeats the
docs-lag-behind-code drift class this repo gates against.

**Options.**
- **Flip at M04 (recommended):** Accepted + dated implementation note citing the landing
  commit; `#check-adr-numbering` green in the same edit window.
- **Keep Proposed until the full cut:** an enacted-but-unratified ADR for the entire
  runway; every reader must rediscover why the code and the ADR disagree.

**If you veto:** M04·f is skipped (one line); the wave is unaffected otherwise.

## Ruling C — brand-now over string-shape-compat

**Question.** Retyping `core/v5/record` fields to branded types (`InstanceID`,
`StreamType`, `Version uint64`, `CorrelationID`) changes shapes that M43's 23 copy-forwards
will inherit. Brand now (codemod absorbs the mechanical rewrites), or keep
string-compatible shapes to ease the copy-forward?

**Evidence.** core/v5 has zero consumers (fleet grep, 2026-11-19); `cqrs-upgrade
--to-v5` already exists with its v5 rewrite skeleton (`rewrite_v5.go`); the
`event.Type = record.Type` alias + lockstep-test pattern (ADR-0111 contract 21a) is the
proven in-repo precedent for exactly this kind of typed unification; ADR-0166's
Alternatives section already rejected string-typed canonical for StreamRef.

**Options.**
- **Brand now (recommended):** final typed surface before the first tag; every later task
  lands on the final shape; M24 compat review catches metaengine/system friction before
  the shape hardens.
- **String-compat now, brand later:** pays the retyping cost TWICE (once to keep strings
  comfy, once after the tag — where it is a v6-scale break) or never (the window closes
  with the debt).

**If you veto:** M03 shrinks to `Version uint64` only; M04/M07/M09 descope their typed
fields; the plan's 51% tier collapses to the alias alone.

---

**Where verdicts land:** TODO_LIST OWNER DECISION SHEET row D32 (per-decision verdicts
fold into the row as they arrive; the row links back here).
