package main

// Recipe catalog, part A: recipes.md §2.0–§2.9 (blocks 1–25).
//
// Every fenced Go block in recipes.md must have exactly one entry, keyed by
// "heading #ordinal". Compiled entries declare the scaffold (imports +
// package-level preamble for free variables + trailers blanking unused
// locals) a generated package needs to type-check; skipped entries carry the
// reason the block is illustrative rather than compilable.

// todoPreamble scaffolds the TodoView/TodoID pair used by §2.0 snippets.
const todoPreamble = `type TodoView struct {
	ID    string
	Title string
}

type TodoID string

func (id TodoID) String() string { return string(id) }
`

const skipSeparateScopes = "multiple `b, _ :=` redeclarations in one scope: the fence is a set of separate copy-paste snippets, not one program"

var recipeCatalogA = map[string]recipeSpec{
	"### 2.0 Bundle Presets — one-call infrastructure wiring #1": {
		imports: []string{
			`cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/kv/v4"`,
			`"fmt"`,
		},
		preamble: todoPreamble,
		trailers: "_ = err\n_ = store",
	},
	"### 2.0 Bundle Presets — one-call infrastructure wiring #2": {
		imports: []string{
			`cqrspebble "github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"`,
			`"github.com/cockroachdb/pebble"`,
		},
		trailers: "_ = b\n_ = err",
	},
	"#### Production options (SQLite / Turso) #1": {skip: skipSeparateScopes},
	"#### Postgres preset #1": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"`},
		trailers: "_ = b",
	},
	"#### Postgres preset #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"`,
		},
		trailers: "_ = b",
	},
	"#### Postgres preset #3": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/kv/v4"`,
			`"time"`,
		},
		preamble: todoPreamble + "\nvar store *kv.TypedStore[TodoView, TodoID]\n",
		trailers: "_ = cached",
	},
	"#### Shared database (events + reads in one \\*sql.DB) #1": {
		imports: []string{
			`"database/sql"`,
			`"github.com/larsartmann/go-cqrs-lite/storage/v4"`,
		},
		preamble: todoPreamble + "\nvar mapper storage.ViewMapper[TodoView]\n",
		trailers: "_ = eventStore\n_ = viewStore",
	},
	"#### Bundle.Debug() — verify your wiring #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"`,
			`"fmt"`,
		},
	},
	"### 2.0b Framework-style lifecycle — go-appkit `cqrs` EventService (external module) #1": {
		skip: "external module github.com/larsartmann/go-appkit/cqrs is not in the workspace module graph",
	},
	"### 2.1 Minimal Event Sourcing (event + command + decider + id + memory) #1": {
		wholeProgram: true,
	},
	"### 2.1b Command-Aware Decisions — automatic causation stamping (decider) #1": {
		wholeProgram: true,
	},
	"### 2.2 Production Persistence (storage or pebble) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/storage/v4"`,
			`"database/sql"`,
		},
		preamble: "var db *sql.DB\n",
		trailers: "_ = eventStore\n_ = cmdStore\n_ = qStore\n_ = snapStore\n_ = cpStore",
	},
	"### 2.2 Production Persistence (storage or pebble) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"`,
			`"log/slog"`,
		},
		preamble: "var dir string\nvar logger *slog.Logger\n",
		trailers: "_ = eventStore\n_ = snapStore\n_ = cpStore",
	},
	"### 2.4 Snapshots for Performance (snapshot) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/snapshot/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/decider/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
		},
		preamble: "type UserState struct{ Name string }\n" +
			"var store event.Store\nvar bus event.Publisher\n" +
			"var d decider.Decider[UserState]\nvar snapStore snapshot.SnapshotStore\n",
		trailers: "_ = repo",
	},
	"### 2.5 Schema Evolution (schema) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/schema/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-codec"`,
		},
		preamble: "type UserCreatedV1 struct{ Name string }\n" +
			"type UserCreatedV2 struct {\n\tName  string\n\tEmail string\n}\n" +
			"var eventStore event.Store\n",
		trailers: "_ = versioned",
	},
	"### 2.6 Tamper-Proof Event Streams (signing) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/signing/v4"`,
			`cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"`,
		},
		preamble: "var secret []byte\nvar bus *cqrswatermill.EventBus\n",
	},
	"### 2.7 Encrypted Payloads (encryption) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/encryption/v4"`,
			`"github.com/larsartmann/go-codec"`,
			`cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"`,
		},
		preamble: "var oldDecrypter encryption.Decrypter\n" +
			"var newDecrypter encryption.Decrypter\nvar bus *cqrswatermill.EventBus\n",
		trailers: "_ = encryptedCodec\n_ = resolver\n_ = b64\n_ = key2\n_ = key3\n_ = bad",
	},
	"### 2.7b Decorating Stores — Encryption/Upcasting at the Store Layer (event) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/encryption/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/schema/v4"`,
		},
		preamble: "var eventStore event.Store\nvar encrypter encryption.Encrypter\n" +
			"var decrypter encryption.Decrypter\nvar upcaster schema.Upcaster\n" +
			"var upcaster1 schema.Upcaster\nvar upcaster2 schema.Upcaster\n",
		trailers: "_ = encryptedStore\n_ = versioned\n_ = stacked",
	},
	"### 2.8 Observability & Middleware (otel + middleware) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/otel/v4"`,
			`cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"time"`,
		},
		preamble: "var bus *cqrswatermill.EventBus\nvar cmdDispatcher *command.Dispatcher\n",
	},
	"#### Tracing + Prometheus metrics (otel.Setup + prometheus.Setup) #1": {
		imports: []string{
			`cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"`,
			`cqrsprometheus "github.com/larsartmann/go-cqrs-lite/prometheus/v4"`,
			`"go.opentelemetry.io/otel"`,
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/query/v4"`,
			`"context"`,
		},
		preamble: "var ctx context.Context\nvar bus *cqrswatermill.EventBus\n" +
			"var cmdDispatcher *command.Dispatcher\nvar qDispatcher *query.Dispatcher\n",
	},
	"#### One-call OTLP export (otel/otlp.SetupOTLP) #1": {
		imports: []string{
			`cqrsotlp "github.com/larsartmann/go-cqrs-lite/otel/otlp/v4"`,
			`"context"`,
		},
		preamble: "var ctx context.Context\n",
	},
	"#### Command Idempotency (dedup on retry) #1": {
		imports: []string{
			`"github.com/larsartmann/go-idempotency"`,
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"time"`,
		},
		preamble: "var cmdDispatcher *command.Dispatcher\n",
	},
	"#### Query Middleware (symmetric with command middleware) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/query/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/otel/v4"`,
			`"log/slog"`,
			`"time"`,
		},
		preamble: "var qDisp *query.Dispatcher\n",
	},
	"### 2.9 Auto-Documentation (catalog) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4/d2"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4/openapi"`,
		},
		preamble: "type CreateUser struct{ Name string }\ntype UserCreated struct{ Name string }\n",
		trailers: "_ = cat\n_ = asyncYAML\n_ = openAPIYAML\n_ = d2Text",
	},
	"### 2.9 Auto-Documentation (catalog) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4/docserver"`,
			`"net/http"`,
		},
		preamble: "var reg *catalog.Registry\n",
		trailers: "_ = ds\n_ = mux",
	},
}
