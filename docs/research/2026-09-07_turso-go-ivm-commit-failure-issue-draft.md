# ISSUE DRAFT (for consideration — not yet filed)

> **Status**: DRAFT. File at https://github.com/tursodatabase/turso/issues
> after working through the "Pre-filing checklist" at the bottom.
> Everything below the `---` line is the proposed issue body.

---

## Title

```
turso-go v0.7.2: COMMIT of transactions that update materialized views
intermittently fails with "cannot commit - no transaction is active"
```

## Summary

When writing to tables that underlie **materialized views** (incremental
view maintenance) through `turso.tech/database/tursogo` v0.7.2, `COMMIT`
intermittently fails with:

```
turso: error: Transaction error: cannot commit - no transaction is active
```

The failure is **probabilistic and dependent on cumulative IVM work within
the process**: it does not reproduce with small datasets, grows with the
number of maintained views, the number of rows written through views, and
the number of distinct group keys the views touch. Once a transaction hits
it, the process keeps failing at similar volume unless the database file
(and sometimes the process) is fresh.

Committed data is never corrupted — the failing transaction is simply
aborted. Engines/tables **without** materialized views never exhibit the
failure at any size we tested.

## Environment

| Item | Value |
| --- | --- |
| Driver | `turso.tech/database/tursogo` **v0.7.2** (embedded libSQL, `database/sql` driver, purego) |
| Mode | Embedded local file databases (`<path>?experimental=views`) |
| Go | 1.26.x (`GOWORK=off` per-module builds; also reproduces in workspace mode) |
| OS / Arch | Linux x86_64 (NixOS), AMD Ryzen AI MAX+ 395 |
| Concurrency | Single writer, `db.SetMaxOpenConns(1)` |

## Repro

Self-contained program (no dependencies beyond the driver). It creates a
fresh database file per round, defines 7 materialized views over one table,
inserts 10k rows in 1,000-statement transactions, and reports COMMIT
failures per round.

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
	rows      = 10_000
	customers = 99
	chunk     = 1_000
	rounds    = 12
)

var views = []string{
	`CREATE MATERIALIZED VIEW IF NOT EXISTS mv_sum AS
	 SELECT SUM(json_extract(value, '$.amount')) AS agg
	 FROM orders WHERE collection = 'o'`,
	`CREATE MATERIALIZED VIEW IF NOT EXISTS mv_count AS
	 SELECT COUNT(*) AS agg FROM orders WHERE collection = 'o'`,
	`CREATE MATERIALIZED VIEW IF NOT EXISTS mv_avg AS
	 SELECT SUM(json_extract(value, '$.amount')) AS agg,
	        COUNT(json_extract(value, '$.amount')) AS cnt
	 FROM orders WHERE collection = 'o'`,
	`CREATE MATERIALIZED VIEW IF NOT EXISTS mv_min AS
	 SELECT MIN(json_extract(value, '$.amount')) AS agg
	 FROM orders WHERE collection = 'o'`,
	`CREATE MATERIALIZED VIEW IF NOT EXISTS mv_max AS
	 SELECT MAX(json_extract(value, '$.amount')) AS agg
	 FROM orders WHERE collection = 'o'`,
	`CREATE MATERIALIZED VIEW IF NOT EXISTS mv_gsum AS
	 SELECT json_extract(value, '$.customer') AS grp,
	        SUM(json_extract(value, '$.amount')) AS agg
	 FROM orders WHERE collection = 'o' GROUP BY grp`,
	`CREATE MATERIALIZED VIEW IF NOT EXISTS mv_gavg AS
	 SELECT json_extract(value, '$.customer') AS grp,
	        SUM(json_extract(value, '$.amount')) AS agg,
	        COUNT(json_extract(value, '$.amount')) AS cnt
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

### Observed output shape (our machine, multiple sessions)

```
round  0: ok
round  1: ok
round  2: FAILED: chunk 2000 commit: turso: error: Transaction error: cannot commit - no transaction is active
...
4/12 rounds failed to COMMIT
```

The failing round index varies between processes (sometimes round 2, sometimes
no round fails in 12); the failure point is consistently around 2k–3k
view-maintained rows for this shape once conditions are unfavorable.

## Characterization (measured on v0.7.2)

All shapes seed 10,000 rows in 1,000-statement transactions unless noted.

| Shape | Outcome |
| --- | --- |
| 1,000 rows × 7 views, 31 groups | stable (no failure in any run) |
| 10,000 rows × **1 scalar view** | passes in some processes (4/4 rounds), fails 3/3 attempts in others |
| 10,000 rows × 5 views, 99 groups | failed 2/3 rounds at the 3rd chunk |
| 10,000 rows × 7 views, 99 groups | failed at the 3rd chunk in every bench-scale attempt |
| 10,000 rows × 7 views, **1 group** | stable (20/20 chunks, 2 independent runs) |
| 100,000 rows × 1 grouped view, 316 groups | fails at ~25k–31k cumulative rows (2/2 runs) |
| 100,000 rows × **no views** (plain table) | never fails (any tx size) |

Additional observations:

- Failure probability grows with **prior scan activity** in the same process
  (heavy `SELECT`/`json_extract` scanning before the writes makes seeding
  fail earlier).
- **Grouped views fail earlier than scalar views** (per-group delta churn).
- Re-attempting on a **fresh database file** sometimes succeeds — the state
  that fails appears tied to the accumulated IVM structures, not the file.
- The failure is a **clean abort**: committed data is always intact and
  correct; no corruption or partial commits observed.
- `INSERT OR REPLACE` (upsert), `UPDATE`, and `DELETE` all maintain views
  correctly when the transaction commits.

## Expected behavior

`COMMIT` should either succeed (maintaining the views in the same
transaction, per documented IVM semantics) or fail with a precise,
actionable error. It must not report "no transaction is active" for a
transaction that was successfully begun and whose statements all succeeded.

## Impact

Any turso-go user bulk-loading rows into tables under materialized views
hits this at moderate scale (~10k+ rows, a handful of views): an abort with
a misleading error, requiring chunk-size guesswork to work around. The
probabilistic nature makes it look like flaky infrastructure rather than a
driver bug.

## Workarounds we ship in the meantime

- Chunk bulk writes to ≤ ~1,000 statements per transaction (necessary, but
  NOT sufficient above ~30k cumulative view-maintained writes per process).
- On failure, retry the chunk set against a fresh database file.
- Keep view counts per table low for write-heavy tables.

## Pre-filing checklist (for the author of this draft)

- [ ] Search tursodatabase/turso issues for
      `"cannot commit - no transaction is active"` and
      `"materialized view commit"` duplicates.
- [ ] Confirm v0.7.2 is the latest released turso-go; re-run the repro on
      the newest release (and on main if easily built).
- [ ] Re-run the repro 3× on the filing machine and paste actual output.
- [ ] Optionally add `-race` and a non-Windows confirmation.
- [ ] Strip anything repo-confidential (none — repro is self-contained).
