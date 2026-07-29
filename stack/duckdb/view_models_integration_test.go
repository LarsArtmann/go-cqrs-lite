//go:build cgo

package duckdb_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

type duckProductKey string

func (k duckProductKey) String() string { return string(k) }

type productView struct {
	Category string  `view:"category"`
	Price    float64 `view:"price"`
	Units    int     `view:"units"`
}

// TestDuckDB_SQLViewModel_AnalyticalQueries proves SQLViewModel creates a real
// columnar table in DuckDB and that analytical queries (GROUP BY aggregation,
// server-side WHERE/ORDER BY) run natively — DuckDB's reason for existing in
// this stack. This mirrors stack/sqlite's integration test but exercises the
// OLAP path that SQLite cannot do efficiently.
func TestDuckDB_SQLViewModel_AnalyticalQueries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "analytics.db")

	b, err := duckdb.New(dsn)
	if err != nil {
		t.Fatalf("duckdb.New: %v", err)
	}

	defer func() { _ = b.Close() }()

	mapper := storage.ViewMapper[productView]{
		Table: "products_view",
		Columns: []storage.ViewColumn[productView]{
			{Name: "category", Type: "VARCHAR", Extract: func(v *productView) any { return v.Category }},
			{Name: "price", Type: "DOUBLE", Extract: func(v *productView) any { return v.Price }},
			{Name: "units", Type: "INTEGER", Extract: func(v *productView) any { return v.Units }},
		},
		ScanRow: func(scan func(dest ...any) error) (*productView, error) {
			var p productView
			if err := scan(&p.Category, &p.Price, &p.Units); err != nil {
				return nil, err
			}

			return &p, nil
		},
	}

	store, err := duckdb.SQLViewModel[productView, duckProductKey](b, mapper)
	if err != nil {
		t.Fatalf("SQLViewModel: %v", err)
	}

	ctx := context.Background()

	seed := []struct {
		key      duckProductKey
		category string
		price    float64
		units    int
	}{
		{"p1", "books", 12.50, 3},
		{"p2", "books", 7.25, 10},
		{"p3", "toys", 29.99, 2},
		{"p4", "toys", 5.00, 8},
		{"p5", "books", 15.00, 1},
	}

	for _, s := range seed {
		if err := store.Set(ctx, s.key, &productView{Category: s.category, Price: s.price, Units: s.units}); err != nil {
			t.Fatalf("Set %s: %v", s.key, err)
		}
	}

	// Server-side filter: all books, ordered by price.
	books, err := store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "category", Op: kv.OpEq, Value: "books"}},
		OrderBy:    "price",
	})
	if err != nil {
		t.Fatalf("Query books: %v", err)
	}

	if len(books) != 3 || books[0].Price != 7.25 {
		t.Fatalf("Query books: got %d rows, first price=%.2f; want 3 rows, first 7.25",
			len(books), firstPrice(books))
	}

	// Analytical aggregation: revenue (sum(price*units)) and avg price per
	// category. This is DuckDB's sweet spot — columnar GROUP BY scan.
	raw, err := sql.Open("duckdb", dsn)
	if err != nil {
		t.Fatalf("sql.Open duckdb: %v", err)
	}

	defer func() { _ = raw.Close() }()

	rows, err := raw.QueryContext(ctx, `
		SELECT category,
		       SUM(price * units) AS revenue,
		       AVG(price)         AS avg_price
		FROM products_view
		GROUP BY category
		ORDER BY revenue DESC`)
	if err != nil {
		t.Fatalf("analytical GROUP BY query: %v", err)
	}

	defer func() { _ = rows.Close() }()

	type agg struct {
		category string
		revenue  float64
		avgPrice float64
	}

	var aggs []agg

	for rows.Next() {
		var a agg
		if err := rows.Scan(&a.category, &a.revenue, &a.avgPrice); err != nil {
			t.Fatalf("scan agg row: %v", err)
		}

		aggs = append(aggs, a)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	if len(aggs) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(aggs))
	}

	// toys revenue = 29.99*2 + 5.00*8 = 99.98; books revenue = 12.50*3 + 7.25*10 + 15.00*1 = 132.25.
	if aggs[0].category != "books" {
		t.Fatalf("expected books as top-revenue category, got %s (%.2f)",
			aggs[0].category, aggs[0].revenue)
	}

	const wantBooksRevenue = 12.50*3 + 7.25*10 + 15.00*1
	if !approxEqual(aggs[0].revenue, wantBooksRevenue) {
		t.Fatalf("books revenue: got %.2f, want %.2f", aggs[0].revenue, wantBooksRevenue)
	}

	const wantToysRevenue = 29.99*2 + 5.00*8
	if !approxEqual(aggs[1].revenue, wantToysRevenue) {
		t.Fatalf("toys revenue: got %.2f, want %.2f", aggs[1].revenue, wantToysRevenue)
	}

	// Prove the dialect placeholder style is DuckDB-compatible ($1).
	var count int
	if err := raw.QueryRowContext(ctx, "SELECT COUNT(*) FROM products_view WHERE category = $1", "toys").
		Scan(&count); err != nil {
		t.Fatalf("$1 placeholder query: %v", err)
	}

	if count != 2 {
		t.Fatalf("toys count via $1 placeholder: got %d, want 2", count)
	}

	_ = sqlpkg.DuckDBDialect{} // keep the dialect import meaningful
}

func firstPrice(rows []*productView) float64 {
	if len(rows) == 0 {
		return 0
	}

	return rows[0].Price
}

func approxEqual(a, b float64) bool {
	const eps = 0.01
	d := a - b
	if d < 0 {
		d = -d
	}

	return d < eps
}
