package mysqlengine

import (
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestDetectDialectFromClassification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		version string
		want    string
	}{
		{"8.4.11", dialectMySQL},
		{"8.0.36", dialectMySQL},
		{"11.8.8-MariaDB-ubu2404", dialectMariaDB},
		{"10.11.8-MariaDB", dialectMariaDB},
		{"5.5.68-MariaDB", dialectMariaDB},
		{"8.0.35-0ubuntu0.22.04.1", dialectMySQL},
	}

	for _, tc := range cases {
		if got := classifyVersion(tc.version); got != tc.want {
			t.Errorf("classifyVersion(%q) = %q, want %q", tc.version, got, tc.want)
		}
	}
}

// TestMariaDBSQLContainsNoMySQL8OnlySyntax verifies the MariaDB dialect never
// emits the -> operator or CAST(... AS JSON), which MariaDB rejects with
// Error 1064 (the nspawn integration-env failure this dialect exists to fix).
func TestMariaDBSQLContainsNoMySQL8OnlySyntax(t *testing.T) {
	t.Parallel()

	e := &mysqlEngine{dialect: dialectMariaDB}

	if expr := e.jsonFieldExpr("status"); expr != `JSON_EXTRACT(value, '$.status')` {
		t.Errorf("jsonFieldExpr = %q", expr)
	}

	want := `JSON_UNQUOTE(JSON_EXTRACT(value, '$.status'))`
	if expr := e.jsonCompareExpr("status"); expr != want {
		t.Errorf("jsonCompareExpr = %q, want %q", expr, want)
	}

	if ph := e.jsonParamPlaceholder(); ph != "?" {
		t.Errorf("jsonParamPlaceholder = %q, want ?", ph)
	}

	sql, args := e.ExplainScanQuery(t.Context(), "c", metaengine.ExplainOptions{
		Filters: []metaengine.FilterSpec{
			{Column: "status", Op: metaengine.FilterEq, Value: "open"},
		},
		Sort:   &metaengine.SortSpec{Column: "priority", Desc: true},
		Cursor: float64(2),
	})

	for _, banned := range []string{"->", "CAST(? AS JSON)"} {
		if strings.Contains(sql, banned) {
			t.Errorf("MariaDB SQL %q contains banned syntax %q", sql, banned)
		}
	}

	if len(args) != 3 { // collection + filter value + cursor value
		t.Errorf("args = %v, want 3 entries", args)
	}
}

// TestMySQLDialectKeepsJSONOperators verifies the MySQL dialect is unchanged:
// JSON-typed -> access plus CAST(? AS JSON) parameter casts.
func TestMySQLDialectKeepsJSONOperators(t *testing.T) {
	t.Parallel()

	e := &mysqlEngine{dialect: dialectMySQL}

	if expr := e.jsonFieldExpr("status"); expr != `value->'$.status'` {
		t.Errorf("jsonFieldExpr = %q", expr)
	}

	if ph := e.jsonParamPlaceholder(); ph != "CAST(? AS JSON)" {
		t.Errorf("jsonParamPlaceholder = %q", ph)
	}

	if p := e.jsonFilterParam("open"); p != `"open"` {
		t.Errorf("jsonFilterParam(string) = %v, want quoted JSON text", p)
	}

	if p := e.jsonFilterParam(float64(3)); p != "3" {
		t.Errorf("jsonFilterParam(number) = %v, want 3", p)
	}
}

// TestMariaDBFilterParamScalarBinding verifies MariaDB binds scalars natively
// so text comparison matches the JSON_UNQUOTE output form.
func TestMariaDBFilterParamScalarBinding(t *testing.T) {
	t.Parallel()

	e := &mysqlEngine{dialect: dialectMariaDB}

	if p := e.jsonFilterParam("open"); p != "open" {
		t.Errorf("string param = %v, want open", p)
	}

	if p := e.jsonFilterParam(true); p != "true" {
		t.Errorf("bool param = %v, want true", p)
	}

	if p := e.jsonFilterParam(false); p != "false" {
		t.Errorf("bool param = %v, want false", p)
	}

	if p := e.jsonFilterParam(float64(10)); p != float64(10) {
		t.Errorf("number param = %v, want 10", p)
	}

	if p := e.jsonFilterParam(map[string]any{"a": 1.0}); p != `{"a":1}` {
		t.Errorf("object param = %v, want JSON text", p)
	}
}
