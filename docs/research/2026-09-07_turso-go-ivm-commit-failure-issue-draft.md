# ISSUE DRAFT (for consideration — not yet filed)

> **Status**: DRAFT, all repros VERIFIED on our machine. Two-part filing plan:
> (1) comment on PR #8257 for the COMMIT abort (defect C);
> (2) standalone issue for the silent wrong-results bugs (defects A+B).
> Everything below the first `---` is the proposed standalone-issue body.

---

## Title

```
Grouped materialized views return silently wrong SUMs once a group is
updated by a second transaction; view state collapses at ~27k rows
(experimental=views, v0.7.2 and v0.8.0-pre.8)
```

## Summary

With incremental view maintenance enabled (`experimental=views`), a GROUPED
`SUM` materialized view stops matching its base table **as soon as a group
receives updates from a second transaction** — silently: reads succeed,
no error, wrong numbers. The error grows with volume until the view state
collapses outright (~62% of the true total missing at 27k rows), after
which further view-maintaining COMMITs also begin failing with
`cannot commit - no transaction is active` (tracked separately in PR
#8257).

Scalar (ungrouped) SUM views stayed exact in every test we ran.

## Environment

| Item | Value |
| --- | --- |
| Driver | `turso.tech/database/tursogo` **v0.7.2** and **v0.8.0-pre.8** (official Go SDK for the embedded Turso database — SQLite-compatible ground-up rewrite; purego, no CGo; `database/sql`) |
| Mode | Embedded local file databases (`<path>?experimental=views`), `SetMaxOpenConns(1)`, single writer |
| Go | 1.26.x |
| OS / Arch | Linux x86_64 (NixOS), AMD Ryzen AI MAX+ 395 |

## Defect A — grouped SUM deltas lost across transactions (silent wrong results)

One table, one grouped view, rows inserted in 1,000-statement transactions:

```sql
CREATE TABLE orders (
  collection TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL,
  PRIMARY KEY (collection, key));
CREATE MATERIALIZED VIEW IF NOT EXISTS mv AS
  SELECT json_extract(value, '$.customer') AS grp,
         SUM(json_extract(value, '$.amount')) AS agg
  FROM orders WHERE collection = 'o' GROUP BY grp;
```

Rows: `key = order-%04d`, `value = {"customer":"c<i%316>","amount":<i%97>+0.5}`.

| Rows (all committed, no failures) | Base SUM | View SUM(agg) | Delta |
| --- | --- | --- | --- |
| 1,000 (one transaction) | 47,495.00 | 47,495.00 | exact |
| 1,100 (two transactions) | 71,580.00 | 71,454.50 | 125.50 |
| 2,000 | 95,890.00 | 95,459.50 | 430.50 |
| 26,000 | 1,260,814.00 | 1,260,383.50 | 430.50 (constant) |

Per-group diff at 2,000 rows — exactly the groups whose rows span BOTH
transactions are wrong, each missing roughly half its sum:

```
group c81   base=  319.50 view=  194.00 delta= 125.50
group c184  base=  318.00 view=  174.50 delta= 143.50
group c287  base=  354.00 view=  192.50 delta= 161.50
```

Groups touched by only one transaction are exact. This reads like a
cross-transaction delta-propagation loss for groups that span transactions
— possibly the same family as #8531 (emptied group lost permanently) and
#6771 (aborted INSERT leaves stale deltas), but it reproduces with fully
successful transactions.

## Defect B — view state collapse at ~27k rows (independent of any failure)

Continuing the same workload to 27,000 rows, with every transaction
committing successfully, the view's total collapses:

```
27,000 rows, all committed: base SUM=1,308,429.00  view SUM=496,034.00
```

Reproduced twice with identical numbers; base table count and per-row data
are correct. At the same cumulative point, the first COMMIT failures also
appear (Defect C).

## Defect C — COMMIT aborts (covered by PR #8257)

From 27,000 cumulative view-maintained rows on, transactions driving the
view fail at COMMIT with `turso: error: Transaction error: cannot commit -
no transaction is active` — deterministically (24/24 rounds across 2 fresh
processes on v0.7.2; 12/12 on v0.8.0-pre.8; always at the same chunk), from
fresh database files too. All statements of the failing transaction report
success, and the base table shows no partial persistence of the aborted
chunk. PR #8257 ("keep the transaction alive when COMMIT's view merge
yields I/O") describes this mechanism; we are commenting there with our
repro. After a first failed COMMIT, the file rejects even single-row
view-maintaining transactions.

## Scalar views: exact

Scalar `SUM(json_extract(...))` views matched the base table exactly at
1,000 and 27,000 rows (and in all our smaller tests), including while
grouped views on the same shape diverged.

## Expected behavior

A materialized view must stay consistent with its base table across
transactions, or fail loudly. Silent divergence is the worst failure mode
possible for a precomputed aggregate: consumers have no signal.

## Pre-filing checklist (author of this draft)

- [x] Searched tursodatabase/turso issues/PRs — no exact duplicate found
      for silent cross-transaction group divergence (related: #8531,
      #6771, #8639, #8640 — different shapes).
- [x] Reproduced on v0.8.0-pre.8 (latest release) — NOT fixed.
- [ ] Confirm repro on a second OS/arch if maintainers ask.
- [ ] Paste verified outputs (captured above, machine-generated).
