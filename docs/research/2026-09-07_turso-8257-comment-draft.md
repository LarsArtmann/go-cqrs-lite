# PR #8257 COMMENT (POSTED 2026-09-07)

> **Status**: POSTED at
> https://github.com/tursodatabase/turso/pull/8257#issuecomment-5576078646
> (posted under @LarsArtmann with the AI disclosure line up front).
> Note: the permalink inside points at commit `1c9f3bf` because the
> divergence-containing draft revision was local-only at posting time —
> after the next push, the comment link can be edited to the newer SHA
> (`18b2c495c`) with the full three-defect characterization.

---

**Disclosure:** report produced by an AI agent (GLM-5.3 via the
[Crush](https://github.com/charmbracelet/crush) CLI), operated and reviewed by
@LarsArtmann. Everything below is machine-verified and runnable.

---

Independent reproduction of this failure — plus a scope question about the
fix.

**What we see** (embedded Turso via `turso.tech/database/tursogo`, file DBs,
`?experimental=views`, single writer, `database/sql`):

- 1 grouped `SUM(json_extract(...))` matview, 316 groups, 50k rows written
  in 1k-statement transactions → **COMMIT fails deterministically at
  exactly 27,000 cumulative view-maintained rows** (chunks 1–26 commit,
  chunk 27 aborts; **24/24 rounds across 2 fresh processes on v0.7.2**,
  12/12 on v0.8.0-pre.8, always at the same chunk). Error: `turso: error:
  Transaction error: cannot commit - no transaction is active`.
- Plain tables (no views) never fail at any size; ≤1k view-maintained rows
  never fails; grouped views fail earlier than scalar; prior scan-heavy
  activity in the process shrinks the budget.
- All 1,000 statements of the failing transaction report success; the base
  table matches the committed transactions afterward (no partial
  persistence of the aborted chunk).
- **Separately, and independent of any commit failure: grouped SUM views
  return silently wrong results once a group is updated by a second
  transaction** (first transaction: exact; from the second on, groups
  spanning multiple transactions lose part of their delta — e.g. at 2k
  rows, 3 of 316 groups each miss ~half their sum; by 27k rows the view
  reports 496,034 vs a true 1,308,429 while reads succeed; scalar SUM
  views stayed exact in all our tests).

**Scope question:** your description says a _creating-connection_ merge
"completes without I/O" and is unaffected. Our repro **creates the views in
the same connection that writes** and still fails — but only once the delta
state reaches ~27k rows, i.e. it looks like the creating-connection merge
**does** go disk-backed at volume. Does the `commit_in_flight()` widening
cover that case? Our repro is deterministic (12/12 rounds at the same
chunk) and should work as a regression test either way.

<details>
<summary>Self-contained repro (verified: 3/3 fresh processes fail at chunk 27000 on v0.8.0-pre.8; 12-round variant: 24/24 on v0.7.2)</summary>

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

func main() {
	dir, _ := os.MkdirTemp("", "ivm")
	defer os.RemoveAll(dir)

	for r := 0; r < 3; r++ {
		db, err := sql.Open("turso", filepath.Join(dir, fmt.Sprintf("r%d.db", r))+"?experimental=views")
		if err != nil {
			log.Fatal(err)
		}
		db.SetMaxOpenConns(1)

		if _, err := db.Exec(`CREATE TABLE orders (
			collection TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL,
			PRIMARY KEY (collection, key))`); err != nil {
			log.Fatal(err)
		}
		if _, err := db.Exec(`CREATE MATERIALIZED VIEW IF NOT EXISTS mv AS
			SELECT json_extract(value, '$.customer') AS grp,
			       SUM(json_extract(value, '$.amount')) AS agg
			FROM orders WHERE collection = 'o' GROUP BY grp`); err != nil {
			log.Fatal(err)
		}

		err = func() error {
			for start := 0; start < 50_000; start += 1_000 {
				tx, err := db.Begin()
				if err != nil {
					return err
				}

				for i := start; i < start+1_000; i++ {
					if _, err := tx.Exec(`INSERT OR REPLACE INTO orders VALUES ('o', ?, ?)`,
						fmt.Sprintf("order-%04d", i),
						fmt.Sprintf(`{"customer":"c%d","amount":%f}`, i%316, float64(i%97)+0.5)); err != nil {
						_ = tx.Rollback()

						return err
					}
				}

				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()

					return fmt.Errorf("chunk %d: %w", start, err)
				}
			}

			return nil
		}()

		_ = db.Close()

		if err != nil {
			fmt.Printf("round %d: FAILED: %v\n", r, err)
			continue
		}

		fmt.Printf("round %d: ok\n", r)
	}
}
```

Full commit-abort characterization (shape/view-count/scan-pressure
sensitivity table):
[go-cqrs-lite research draft](https://github.com/LarsArtmann/go-cqrs-lite/blob/1c9f3bf33c002060fd563c3853edb7fb1a4923e5/docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md)
(SHA-pinned permalink).

</details>
