# ISSUE DRAFT (for consideration — not yet filed)

> **Status**: DRAFT, repro VERIFIED on our machine (24/24 rounds fail at the
> same statement across 2 fresh processes). File at
> https://github.com/tursodatabase/turso/issues after the "Pre-filing
> checklist" at the bottom. Everything below the `---` line is the proposed
> issue body.

---

## Title

```
turso-go v0.7.2: deterministic COMMIT failure ("no transaction is active")
after ~27k rows written through a materialized view in one process
```

## Summary

With incremental view maintenance enabled (`experimental=views`), writing
rows through a materialized view **deterministically stops working after
~27,000 view-maintained rows within one process**: the 27th
1,000-statement transaction fails at `COMMIT` with

```
turso: error: Transaction error: cannot commit - no transaction is active
```

every time, from a freshly opened database file. `BEGIN` and all 1,000
statements succeed; only the `COMMIT` aborts. New database files fail at
the same cumulative point. Committed data is never corrupted — the failing
transaction is simply rolled back — but **no write path through a
materialized view can exceed the ~27k-row ceiling in a process**, which
makes bulk loads and large incremental backfills impossible once IVM is
involved.

Tables **without** materialized views never exhibit the failure at any size
tested (100k+ rows, single or chunked transactions).

## Environment

| Item | Value |
| --- | --- |
| Driver | `turso.tech/database/tursogo` **v0.7.2** (embedded libSQL, purego, `database/sql`) |
| Mode | Embedded local file databases (`<path>?experimental=views`) |
| Go | 1.26.x |
| OS / Arch | Linux x86_64 (NixOS), AMD Ryzen AI MAX+ 395 |
| Concurrency | Single writer, `db.SetMaxOpenConns(1)` |

## Repro (verified: 12/12 rounds fail at chunk 27000, twice from scratch)

Self-contained; only dependency is the driver. Each round uses a fresh
database file: one table, ONE materialized view (grouped SUM), then 50,000
rows inserted in 1,000-statement transactions.

```go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "turso.tech/database/tursogo"
)

const (
	rows      = 50_000
	customers = 316
	chunk     = 1_000
	rounds    = 12
)

var views = []string{
	`CREATE MATERIALIZED VIEW IF NOT EXISTS mv_gsum AS
	 SELECT json_extract(value, '$.customer') AS grp,
	        SUM(json_extract(value, '$.amount')) AS agg
	 FROM orders WHERE collection = 'o' GROUP BY grp`,
}

func main() {
	dir, err := os.MkdirTemp("", "ivm-repro")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	failed := 0

	for r := 0; r < rounds; r++ {
		db, err := sql.Open("turso", filepath.Join(dir, fmt.Sprintf("r%d.db", r))+"?experimental=views")
		if err != nil {
			log.Fatal(err)
		}
		db.SetMaxOpenConns(1)

		if _, err := db.Exec(`CREATE TABLE orders (
			collection TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (collection, key))`); err != nil {
			log.Fatal(err)
		}
		for _, ddl := range views {
			if _, err := db.Exec(ddl); err != nil {
				log.Fatalf("round %d: create view: %v", r, err)
			}
		}

		err = seed(db)

		if cerr := db.Close(); cerr != nil && err == nil {
			err = cerr
		}

		if err != nil {
			failed++
			fmt.Printf("round %2d: FAILED: %v\n", r, err)
			continue
		}
		fmt.Printf("round %2d: ok\n", r)
	}

	fmt.Printf("\n%d/%d rounds failed to COMMIT\n", failed, rounds)
}

func seed(db *sql.DB) error {
	for start := 0; start < rows; start += chunk {
		end := min(start+chunk, rows)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("chunk %d begin: %w", start, err)
		}

		for i := start; i < end; i++ {
			v := fmt.Sprintf(`{"customer":"c%d","amount":%f}`, i%customers, float64(i%97)+0.5)
			if _, err := tx.Exec(
				`INSERT OR REPLACE INTO orders VALUES ('o', ?, ?)`,
				fmt.Sprintf("order-%04d", i), v,
			); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("chunk %d insert: %w", start, err)
			}
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("chunk %d commit: %w", start, err)
		}
	}

	return nil
}
```

### Actual output (2 fresh processes, identical result)

```
round  0: FAILED: chunk 27000 commit: turso: error: Transaction error: cannot commit - no transaction is active
round  1: FAILED: chunk 27000 commit: turso: error: Transaction error: cannot commit - no transaction is active
...
round 11: FAILED: chunk 27000 commit: turso: error: Transaction error: cannot commit - no transaction is active

12/12 rounds failed to COMMIT
```

The failure point is **exactly the same every time**: 26 chunks (26,000
rows) commit; the 27th chunk aborts. Fresh database files fail identically,
so the ceiling is process/runtime-scoped, not file-scoped.

## Characterization

| Shape (rows × views × distinct group keys) | Outcome |
| --- | --- |
| 50,000 × 1 grouped SUM view × 316 groups | **fails deterministically at 27,000 cumulative rows** (24/24 rounds, 2 processes) |
| 10,000 × 1 scalar SUM view × 99 groups | passes in some processes, fails in others (near the boundary) |
| 10,000 × 5–7 views × 99 groups | fails at ~2k–3k cumulative rows once conditions are unfavorable |
| 10,000 × 7 views × **1 group** | stable (20/20 chunks, 2 independent runs) |
| 100,000+ × **no views** (plain table) | never fails, any transaction size |

Additional observations:

- The ceiling tracks **cumulative view-maintained writes per process**, not
  per database file (fresh files keep counting from the process total...
  each fresh file restarts at 0 and still fails at ~27k, so the budget is
  per-runtime-per-file, but the threshold value itself is stable).
- **Scalar views tolerate more writes than grouped views** before the
  boundary (scalar 10k is borderline; grouped 10k×99 groups already
  borderline; grouped ≥27k is deterministic failure).
- Prior heavy `SELECT`/`json_extract` scanning in the same process makes
  smaller shapes fail earlier (boundary appears to shrink under memory/timing
  pressure).
- Smaller shapes (≤1k rows) have never failed in any run.
- `INSERT OR REPLACE` (upsert), `UPDATE`, and `DELETE` all maintain views
  correctly for every transaction that commits.

## Expected behavior

`COMMIT` of a transaction whose statements all succeeded should either
commit (maintaining the views per documented IVM semantics) or return a
precise error naming the actual problem. It must not report "no transaction
is active" for an active transaction, and IVM writes should not carry a
hard ~27k-row per-process ceiling.

## Impact

Bulk loads and large incremental writes through any table under a
materialized view hit a hard, misleading wall at ~27k rows per process.
The error text suggests an application-side transaction bug, so it reads as
flaky infrastructure rather than an IVM limit. Workarounds (below) cap
throughput and are easy to get wrong because the boundary is undocumented.

## Workarounds

- Keep cumulative view-maintained writes below ~27k per process (reopen the
  process / rotate files beyond that) — awkward for long-running services.
- Chunking alone does NOT help past the boundary (each 1k-statement chunk
  counts toward the same total).

## Pre-filing checklist (author of this draft)

- [ ] Search tursodatabase/turso issues for
      `"cannot commit - no transaction is active"` and IVM-related dups.
- [ ] Confirm v0.7.2 is the latest turso-go release; re-run the repro on
      the newest release/main if practical.
- [ ] Optionally: test whether the boundary moves with `-race`, other OSes,
      or a plain (non-json_extract) view body to help maintainers localize.
- [ ] Fill in the filing account + paste the verified output (already
      captured above).
