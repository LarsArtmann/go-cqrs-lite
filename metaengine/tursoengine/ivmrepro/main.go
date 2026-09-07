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
