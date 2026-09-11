# cqrs-lint F091 Tier 1 — Typed Qualifier Resolution Wall Time

**Date:** 2026-09-07 · **Binary:** `cmd/cqrs-lint` @ master (post v4.9.0)
**Question:** what does F091 Tier 1 (typed qualifier resolution via
`packages.TypesInfo` + the F090 dot-import walk) cost at wall-clock level?
**Verdict:** **no visible wall-time regression** — adoption gate passed.

## Context the design doc lacked

`docs/planning/archived/2026-09-06_cqrs-lint-t23-design-passes.md` gated Tier 1 on
"measure first: `NeedTypes` roughly doubles load cost". Empirical finding:
the loader has shipped `NeedTypes|NeedTypesInfo` **since day one**
(`457b039a0`, 2026-07-16 — verified via `git log -S NeedTypes`). The load-cost
concern was already paid before this change; F044's wall-time baseline was
measured with the mode on. Tier 1 therefore adds only per-selector work: one
`TypesInfo.Uses` map lookup, plus one import-spec walk per file for the F090
dot-import check.

## Method

Two binaries from the same tree, differing ONLY in the Tier-1 wiring:
`cqrs-lint-OLD` has the typed call short-circuited to the import-table string
scan (pre-F091 behavior); `cqrs-lint-NEW` resolves qualifiers through
`ResolveQualifierTyped` with string-scan fallback. Interleaved pairs
(old-then-new), 6 pairs, first pair discarded as page-cache warmup.
Corpus: repo root (`cqrs-lint . --quiet`) — 82 go.work modules, self-lint
mode (V007 auto-skipped; this corpus measures the detector-loop and
dot-import-walk overhead, not selector resolution, which is consumer-only).

**Correctness differential** (the shadow fixture, `schema/v4` consumer with a
local value shadowing the package qualifier + a same-named field access):
OLD fires V007 twice (string scan false-matches the shadow), NEW fires once
(`TypesInfo.Uses` resolves the shadow to a `*types.Var` — authoritative
not-a-package, no fallback). This is the Tier-1 false-positive class kill,
pinned end-to-end.

## Results

Ambient load 28→65 during the run (shared host, parallel session active —
absolute numbers are NOT comparable across sessions; only the within-set
ratio is meaningful, and even that is degraded at load 65).

| Pair | OLD (ms) | NEW (ms) |
| ---- | -------- | -------- |
| 1†   | 28547    | 37767    |
| 2    | 21140    | 16211    |
| 3    | 10687    | 10032    |
| 4    | 9562     | 11391    |
| 5    | 15929    | 12418    |
| 6    | 14696    | 20040    |

† cold, discarded.

Median OLD = 14696 ms, NEW = 12418 ms. On-median the typed build is faster;
run-to-run jitter (±50% at load 65) dwarfs any marginal cost. Consistent with
F044: V007's marginal cost is below measurement noise.

## Decision

Tier 1 adopted (no flag — load-mode change only, per the design's sequencing).
Tiers 2–3 (C008/C035 payload-flow confirmation) remain future work behind
`--typed-info=auto` as designed.
