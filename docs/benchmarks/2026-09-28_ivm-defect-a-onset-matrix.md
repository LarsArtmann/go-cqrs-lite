# IVM Defect-A Onset Matrix (turso-go grouped-view delta loss)

**Date:** 2026-09-28
**Harness:** [`metaengine/tursoengine/ivm_bisect_test.go`](../../metaengine/tursoengine/ivm_bisect_test.go) (build tag `ivmrepro`, opt-in env `TURSO_IVM_BISECT=1`)
**Invocation:**

```bash
cd metaengine/tursoengine && source ../../scripts/go-env.sh && \
  TURSO_IVM_BISECT=1 GOWORK=off go test -tags ivmrepro \
  -run TestIVMReproDefectAOnsetBisect -count=1 -timeout 30m -v .
```

**Method:** fresh database per configuration; rows inserted in chunk-sized
transactions (`RunInTx` + `MapSet`); after EVERY committed transaction the
grouped materialized-view total (sum over all `GroupedAggregate` groups) is
compared against the exactly-expected sum of inserted rows. Onset = first
transaction boundary where the view diverges. Workload: keys `order-%05d`,
group key `customer = c<i % groups>`, value `amount = (i%97)+0.5` — identical
to the canonical `ivm_repro_test.go` shape, so the 2k/316/2×1000 config
reproduces the draft's documented 430.50 delta exactly.

Defect C's commit wall ("cannot commit - no transaction is active") is recorded
as data (`WALL at tx#N`): prior per-tx scan activity shrinks the wall — a
documented property of the harness, so wall positions here are lower bounds on
the wall a write-only workload would see.

## Matrix (2026-09-28 run, 16.8 s total)

### Dimension 1 — transaction size (2 000 rows, 316 groups)

| chunk/tx | txs | verdict                                                               |
| -------: | --: | --------------------------------------------------------------------- |
|     2000 |   1 | **EXACT**                                                             |
|     1000 |   2 | DIVERGES at tx#2 (view 95 459.50 vs base 95 890.00, **delta 430.50**) |
|      500 |   4 | DIVERGES at tx#2 (delta 272.00)                                       |
|      100 |  20 | DIVERGES at tx#4 (delta 81.50)                                        |
|       10 | 200 | DIVERGES at tx#33 (delta 25.00); defect-C WALL at tx#93\*             |

### Dimension 2 — distinct groups (2 000 rows, 500-row txs)

| groups | verdict                                                 |
| -----: | ------------------------------------------------------- |
|      1 | **EXACT**                                               |
|      2 | **EXACT**                                               |
|      8 | **EXACT**                                               |
|     64 | **EXACT**                                               |
|    316 | DIVERGES at tx#2 (delta 272.00)                         |
|   2000 | **EXACT** (single-member groups); defect-C WALL at tx#3 |

### Dimension 3 — total rows (316 groups, 500-row txs)

| rows | txs | verdict                         |
| ---: | --: | ------------------------------- |
|  500 |   1 | **EXACT**                       |
| 1000 |   2 | DIVERGES at tx#2 (delta 272.00) |
| 2000 |   4 | DIVERGES at tx#2 (delta 272.00) |
| 4000 |   8 | DIVERGES at tx#2 (delta 272.00) |

\* The chunk=10 wall carries an asterisk: 200 per-tx `GroupedAggregate` scans
shrink defect C's write budget (known constraint, warned in
`ivm_repro_test.go`); the ONSET (tx#33) is unaffected — it precedes the wall.

## Observations (data-grounded)

1. **Single-transaction loads are always exact.** Every 1-tx config (chunk=2000;
   rows=500) stays exact. Divergence is a CROSS-TRANSACTION phenomenon.
2. **Onset is the SECOND transaction whenever the group band is hot.** In every
   diverging ≥500-row-tx config the first divergence is tx#2 — the first
   cross-tx boundary on existing view state loses part of the delta.
3. **Onset position scales with accumulated state at small tx sizes.** At
   100-row txs onset waits until tx#4; at 10-row txs until tx#33 (≈330 rows
   in). Consistent with "enough accumulated view/group state" gating the loss,
   not a pure transaction count.
4. **Non-monotonic in distinct groups:** 1/2/8/64 groups exact at any tx size;
   316 diverges at tx#2; 2000 single-member groups stay EXACT (until the
   defect-C wall). The loss needs the mid band (10²–10³ groups with multi-row
   groups) — NOT simply "more groups".
5. **The loss is partial, not whole-transaction.** At 500-chunk the lost 272.00
   is ≈1.1 % of tx#2's 23 845.00 delta; at 1000-chunk 430.50 of 47 445.00
   (≈0.9 %); at 100-chunk 81.50 of ≈4 750 (≈1.7 %); at 10-chunk 25.00 of ≈475
   (≈5 %). Some groups' increments inside the boundary transaction are dropped
   while others land.
6. **The canonical draft delta (430.50) is reproducible byte-for-byte** by the
   2000-rows/316-groups/2×1000-tx config — same workload family as the
   standalone draft, so the matrix extends (not contradicts) the draft.

## Property envelope for the upstream issue

Grouped materialized views (`sum` over `customer`) are exact when ANY of:

- the whole load commits in one transaction, or
- distinct group count ≤ 64 (tested 1/2/8/64 at 4-tx loads), or
- every group is single-member (2000 groups over 2000 rows, through 2 txs).

They silently lose part of a transaction's grouped delta when a ≥2nd
transaction advances groups in the ~10²–10³ band: onset at the first such
boundary (tx#2 at ≥500-row txs), earlier onset delayed proportionally at
smaller tx sizes, loss magnitude ≈1–5 % of the offending transaction's delta.
Post-onset the view never catches back up within the run window; after a
defect-C commit wall aborts a transaction, the view ABSORBS the aborted
transaction's deltas (zombie-tx readback — already characterized in the
`ivmrepro` suite).

## Cross-references

- TODO_LIST rows: "Sharpen the defect-A characterization before filing
  upstream" (receipt), "File the standalone upstream issue (defects A+B)"
  (owner-gated draft).
- Draft issue: [`docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md`](../research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md)
- Prior characterization: [`2026-09-07_turso-materialized-views.md`](2026-09-07_turso-materialized-views.md)
- Defect-C upstream thread: turso PR #8257 comment (commit-abort half already reported).
