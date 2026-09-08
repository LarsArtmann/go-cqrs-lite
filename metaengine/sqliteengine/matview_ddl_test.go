package sqliteengine

import (
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestMatViewDDL_Golden pins the exact DDL matViewDDL derives for every
// aggregate shape × scalar/grouped (modulo the deterministic view name). The
// DDL is the operator-visible contract of ADR-0135: a change here changes
// what deployments create on disk and must be conscious (and CHANGELOG'd).
func TestMatViewDDL_Golden(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		spec     metaengine.MaterializedViewSpec
		wantBody string
	}{
		{
			name: "scalar count",
			spec: metaengine.MaterializedViewSpec{Collection: "order_views", Fn: metaengine.MatViewCount},
			wantBody: " AS SELECT COUNT(*) AS agg FROM meta_map WHERE collection = 'order_views'",
		},
		{
			name: "scalar sum",
			spec: metaengine.MaterializedViewSpec{Collection: "order_views", Fn: metaengine.MatViewSum, Column: "total"},
			wantBody: " AS SELECT SUM(json_extract(value, '$.total')) AS agg FROM meta_map WHERE collection = 'order_views'",
		},
		{
			name: "scalar min",
			spec: metaengine.MaterializedViewSpec{Collection: "sensors", Fn: metaengine.MatViewMin, Column: "temp"},
			wantBody: " AS SELECT MIN(json_extract(value, '$.temp')) AS agg FROM meta_map WHERE collection = 'sensors'",
		},
		{
			name: "scalar avg stores sum+count",
			spec: metaengine.MaterializedViewSpec{Collection: "sensors", Fn: metaengine.MatViewAvg, Column: "temp"},
			wantBody: " AS SELECT SUM(json_extract(value, '$.temp')) AS agg, COUNT(json_extract(value, '$.temp')) AS cnt " +
				"FROM meta_map WHERE collection = 'sensors'",
		},
		{
			name: "grouped sum",
			spec: metaengine.MaterializedViewSpec{Collection: "orders", Fn: metaengine.MatViewSum, Column: "total", GroupBy: "region"},
			wantBody: " AS SELECT json_extract(value, '$.region') AS grp, SUM(json_extract(value, '$.total')) AS agg " +
				"FROM meta_map WHERE collection = 'orders' GROUP BY grp",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := matViewDDL(tt.spec)

			if !strings.HasPrefix(got, "CREATE MATERIALIZED VIEW IF NOT EXISTS ") {
				t.Errorf("missing IF NOT EXISTS preamble: %s", got)
			}
			if wantIdent := metaengine.QuoteIdent(tt.spec.ViewName()); !strings.Contains(got, wantIdent+" ") {
				t.Errorf("view name %s not embedded: %s", wantIdent, got)
			}
			if !strings.HasSuffix(got, tt.wantBody) {
				t.Errorf("matViewDDL(%+v)\n got: %s\nwant suffix: %s", tt.spec, got, tt.wantBody)
			}
		})
	}
}

// TestMatViewDDL_EscapesCollection: a collection name containing a quote
// must not be able to break out of the string literal.
func TestMatViewDDL_EscapesCollection(t *testing.T) {
	t.Parallel()

	got := matViewDDL(metaengine.MaterializedViewSpec{
		Collection: "evil'--coll", Fn: metaengine.MatViewCount,
	})
	if !strings.Contains(got, "WHERE collection = 'evil''--coll'") {
		t.Errorf("collection quote not escaped:\n got: %s", got)
	}
}
