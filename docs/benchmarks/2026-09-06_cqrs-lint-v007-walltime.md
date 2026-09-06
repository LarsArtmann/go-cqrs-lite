# cqrs-lint V007 Self-Lint Wall Time (F044)

**Date:** 2026-09-06 · **Binary:** `cmd/cqrs-lint` @ master (post v4.9.0, 204 rules)
**Question:** what does the V007 (`v5-removed-api-usage`) detector cost at wall-clock level?
**Verdict:** **below measurement noise** — enabling V007 is free in practice.

## Method

`cqrs-lint --quiet` over two fixed corpora, 5 runs per variant, first run of
each set discarded as page-cache warmup, medians compared. Variants:
default (V007 on) vs `--exclude-rules V007` (single-rule delta only — no
other work changes between variants).

Timing pairs interleaved (all runs of one variant back-to-back per set);
ambient load recorded before/after each set. Absolute numbers are NOT
comparable across sessions or machines (shared host, 30–40 load average);
only the within-set ratio is meaningful.

## Results

### Corpus A: `cmd/` + `example/` (196 files)

| Variant      | Runs (ms)                     | Median |
| ------------ | ----------------------------- | ------ |
| V007 on      | 1777† 478 472 479 500         | 479 ms |
| V007 off     | 479 478 503 490 574           | 490 ms |

† cold run, discarded as warmup.

### Corpus B: `storage/` (263 files)

| Variant      | Runs (ms)                     | Median |
| ------------ | ----------------------------- | ------ |
| V007 on      | 858 828 769 757 775           | 775 ms |
| V007 off     | 840 923 889 753 718           | 840 ms |

## Interpretation

- Both corpora: on-median ≤ off-median, i.e. V007's marginal cost is
  smaller than run-to-run jitter (±10% under load 28–41, 37 users).
- V007 is a pure AST pass over selector/ident names with no I/O; its
  theoretical cost is one extra comparison set per selector — consistent
  with the sub-noise measurement.
- No fast-path exemption for V007 needed; presets may enable it by default
  without a perf budget.

## Context

```
load avg: 28.15 → 41.49 (set A), 32.96 (set B) · 37 users · up 13:51
shared host: absolute values inflated vs idle machine
```
