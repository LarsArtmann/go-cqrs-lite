package main

// Recipe catalog, part B2: recipes.md §2.23–§2.35 (tail entries of part B, split
// out to stay under the 350-line file cap).

var recipeCatalogB2 = map[string]recipeSpec{
	"### 2.23 Hand-Rolled Catch-Up: Subscribe BEFORE You Drain (projectionhost TOCTOU) #1": {
		skip: "anti-pattern demo: intentionally wrong order and `sub` is redeclared — not compilable by design",
	},
	"### 2.24 Atomic Read-Modify-Write: Engine RunInTx (metaengine.Transactional) #1": {
		skip: "the `found` local is unused inside the tx closure — illustrative, not compilable without rewriting",
	},
	"### 2.25 Vector Size Introspection: VectorCounter (metaengine) #1": {
		skip: "the `n`/`cols` locals are unused inside the if-scope — illustrative, not compilable without rewriting",
	},
	"### 2.26 Multi-Instance Timers: ClaimingTimerStore (scheduling/sqlstore) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/scheduling/v4"`,
			`"database/sql"`,
			`"context"`,
		},
		preamble: "type MyPayload struct{ OrderID string }\nvar dsn string\n" +
			"ctx := context.Background()\n" +
			"var dispatch scheduling.DispatchFunc[MyPayload]\n",
		trailers: "_ = scheduler\n_ = err",
	},
	"### 2.27 Planned Tables: LayoutPlanApplier (pgengine/mysqlengine/sqliteengine/duckdbengine) #1": {
		skip: "ellipsis statement body (`{ ... }`) — illustrative error handling, not compilable Go",
	},
	"### 2.28 Planned Tables: Pushdown, Evolution, Backfill, and the EXPLAIN Proof #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\nvar eng metaengine.Engine\n",
		trailers: "_ = n\n_ = err",
	},
	"### 2.28 Planned Tables: Pushdown, Evolution, Backfill, and the EXPLAIN Proof #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\nvar eng metaengine.Engine\n" +
			"var grownPlan metaengine.LayoutPlan\n",
		trailers: "_ = applied\n_ = err",
	},
	"### 2.28 Planned Tables: Pushdown, Evolution, Backfill, and the EXPLAIN Proof #3": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"database/sql"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\nvar eng metaengine.Engine\nvar db *sql.DB\n",
		trailers: "_ = rows",
	},
	"### 2.29 Materialized Views: Operator-Declared Aggregate Acceleration (tursoengine, ADR-0135) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/tursoengine/v4"`,
		},
		preamble: "var dsn string\n",
		trailers: "_ = eng\n_ = err",
	},
	"### 2.30 Operator Priority Routing — global / perEngine / perQuery (system, verified v4.6.0) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
		},
		trailers: "_ = deployment",
	},
	"### 2.31 Evolutions — declare folds for a result type (system, verified v4.6.0) #1": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/system/v4"`},
		trailers: "_ = domain",
	},
	"### 2.32 Pre-v5 Snapshot Bytes Still Decode (snapshot wire fallback) #1": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/snapshot/v4"`},
		preamble: "var oldBytes []byte\n",
		trailers: "_ = err",
	},
	"### 2.33 Encrypted Payloads: Envelope v2 + Key Rotation (encryption) #1": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/encryption/v4"`},
		preamble: "var oldKey []byte\n",
		trailers: "_ = resolver\n_ = err",
	},
	"### 2.35 Survive a Dead Engine (health-driven deactivation, ADR-0137) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"context"`,
			`"log/slog"`,
			`"time"`,
		},
		preamble: "ctx := context.Background()\nvar store *metaengine.Store\n",
	},
	"### 2.35 Survive a Dead Engine (health-driven deactivation, ADR-0137) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/otelobserver/v4"`,
			`cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"`,
		},
		preamble: "var store *metaengine.Store\nvar meter cqrsotel.Meter\n",
		trailers: "_ = obs",
	},
}
