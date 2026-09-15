package main

// Recipe catalog, part B: recipes.md §2.10–§2.22 (blocks 26–58; the §2.23+
// tail lives in recipes_catalog_meta2.go).

const taskViewPreamble = "type TaskView struct{ ID, Title string }\n"

var recipeCatalogB = map[string]recipeSpec{
	"### 2.10 Cost-Based Storage Planning (metaengine) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
		},
		preamble: "type TaskID string\n" +
			"type FindTaskInput struct{ ID TaskID }\n" +
			"type FindTaskResult struct {\n\tID    TaskID\n\tTitle string\n}\n" +
			"type TaskCreated struct {\n\tID    TaskID\n\tTitle string\n}\n" +
			"type TaskDeleted struct{ ID TaskID }\n",
		trailers: "_ = result",
	},
	"### 2.11 Live Latency Measurement — Dynamic RTT + Auto-Replan (metaengine) #1": {
		imports: []string{
			`"context"`,
			`"fmt"`,
			`"time"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"`,
		},
		preamble: "var query any\n",
	},
	"### 2.12 Capability Diagnostics — declared-vs-implemented audit (metaengine) #1": {
		imports: []string{
			`"context"`,
			`"errors"`,
			`"fmt"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
		},
		preamble: "ctx := context.Background()\nvar pg metaengine.Engine\n" +
			"var store *metaengine.Store\n",
		errFunc: true,
	},
	"### 2.12 Capability Diagnostics — declared-vs-implemented audit (metaengine) #2": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`},
		preamble: "var engines []metaengine.Engine\nvar myQuery any\n",
		trailers: "_ = store\n_ = err",
	},
	"### 2.13 SQL-Backed Idempotency (idempotency/sqlstore) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/idempotency/sqlstore/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"database/sql"`,
			`"context"`,
			`"time"`,
		},
		preamble: "ctx := context.Background()\nvar db *sql.DB\ncmds := command.NewDispatcher()\n",
	},
	"### 2.13b Retry with Backoff (retry) #1": {
		imports: []string{
			`"github.com/larsartmann/go-retry"`,
			`"context"`,
			`"errors"`,
			`"time"`,
		},
		preamble: "ctx := context.Background()\n" +
			"flakyOperation := func(ctx context.Context) error { return nil }\n",
	},
	"### 2.15 CBOR→JSON for Browser SSE Clients (codec + transport/http) #1": {
		imports: []string{
			`cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"`,
			`cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"`,
		},
		preamble: "bus := cqrswatermill.NewEventBus()\n",
		trailers: "_ = broker\n_ = err",
	},
	"### 2.16 Metaengine SSE Streaming with Reconnection (metaengine) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"net/http"`,
			`"time"`,
		},
		preamble: taskViewPreamble + "var store *metaengine.Store\n",
		trailers: "_ = replay",
	},
	"### 2.17 Metaengine Cursor Pagination with PrefetchCache (metaengine) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"context"`,
		},
		preamble: taskViewPreamble + "ctx := context.Background()\nvar store *metaengine.Store\n",
		trailers: "_ = page1\n_ = cursor2\n_ = page2",
	},
	"### 2.18 Flight Recorder — Capture Trace on Slow/Error (flightrecorder + middleware) #1": {
		imports: []string{
			`"github.com/larsartmann/go-flightrecorder"`,
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/query/v4"`,
			`"time"`,
		},
		preamble: "bus := cqrswatermill.NewEventBus()\ncmdDisp := command.NewDispatcher()\n" +
			"qryDisp := query.NewDispatcher()\n",
	},
	"### 2.18 Flight Recorder — Capture Trace on Slow/Error (flightrecorder + middleware) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/stack/v4"`,
			`"github.com/larsartmann/go-flightrecorder"`,
		},
		preamble: "var dsn string\nvar recorder *flightrecorder.Recorder\n",
	},
	"### 2.19 Command Lifecycle Tracking (ADR-0117) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
		},
		preamble: "var eventStore event.Store\nvar config middleware.RetryConfig\n",
		trailers: "_ = config\n_ = cfg",
	},
	"### 2.19 Command Lifecycle Tracking (ADR-0117) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"log/slog"`,
		},
		preamble: "var eventStore event.Store\nlogger := slog.Default()\n" +
			"dispatcher := command.NewDispatcher()\nvar config middleware.RetryConfig\n",
	},
	"### 2.19 Command Lifecycle Tracking (ADR-0117) #3": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/commandlifecycle/projections/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\nvar engines []metaengine.Engine\n",
		trailers: "_ = result\n_ = counts\n_ = pt\n_ = byActor",
	},
	"### 2.19b Schema Evolution for Command Lifecycle Streams (schema + commandlifecycle) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/schema/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"time"`,
		},
		preamble: "var raw event.Store\n",
		trailers: "_ = upcasted\n_ = recorder",
	},
	"### 2.20 Engine Roles, Shadow Replication & Promote Cutover (metaengine) #1": {
		skip: "ellipsis statement bodies (`{ ... }`) — illustrative control flow, not compilable Go",
	},
	"### 2.20 Engine Roles, Shadow Replication & Promote Cutover (metaengine) #2": {
		imports: []string{
			`"bytes"`,
			`"context"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
		},
		preamble: "ctx := context.Background()\nvar store *metaengine.Store\n" +
			"var freshStore *metaengine.Store\nvar ids []string\n" +
			"type TaskCreated struct{ ID string }\ntype FindTask struct{ ID string }\n",
		trailers: "_ = summary",
	},
	"### 2.20 Engine Roles, Shadow Replication & Promote Cutover (metaengine) #3": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`},
		preamble: "var engines []metaengine.Engine\nvar queries []any\n",
		trailers: "_ = store",
	},
	"### 2.21 Actor Propagation — \"Who Did It\" Audit Trail (id + command + middleware + event) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/id/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/middleware/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/decider/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
		},
		preamble: "type State struct{}\nvar store event.Store\nvar bus event.Publisher\n" +
			"var d decider.Decider[State]\nvar streamID id.StreamID\n",
		trailers: "_ = basic\n_ = repo",
	},
	"### 2.21 Actor Propagation — \"Who Did It\" Audit Trail (id + command + middleware + event) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/id/v4"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\n",
		trailers: "_ = actor\n_ = ok",
	},
	"### 2.21 Actor Propagation — \"Who Did It\" Audit Trail (id + command + middleware + event) #3": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/scheduling/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/id/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"context"`,
			`"time"`,
		},
		preamble: "type CancelOrderPayload struct {\n\tStreamID id.StreamID\n\tOrderID  string\n}\n" +
			"type CancelOrderCmd struct {\n\t*command.BasicCommand\n\n\tOrderID string\n}\n" +
			"var timerStore scheduling.TimerStore[CancelOrderPayload]\nvar due time.Time\n" +
			"var streamID id.StreamID\n" +
			"actor := id.NewServiceActor(\"order-api\")\n" +
			"ctx := context.Background()\ncmds := command.NewDispatcher()\n",
		trailers: "_ = scheduler",
	},
	"### 2.21 Actor Propagation — \"Who Did It\" Audit Trail (id + command + middleware + event) #4": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/id/v4"`},
		preamble: "actor := id.NewUserActor(id.NewUserID())\n",
		trailers: "_ = parsed\n_ = err",
	},
	"### 2.21b Metaengine + Stack Bundle Integration (v4 bundle path) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/stack/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"`,
			`"context"`,
		},
		preamble: "type TaskCreated struct {\n\tID     string\n\tStatus string\n}\n" +
			"type TaskCompleted struct{ ID string }\n" +
			"var dsn string\nctx := context.Background()\n" +
			"var payloadDecoder projectionadapter.PayloadDecoder\nvar host *projectionhost.Host\n",
		trailers: "_ = counts\n_ = adapter",
	},
	"#### Filtered Scan with Metaengine (Map + FilterOnField + SQLite Pushdown) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
			`"context"`,
		},
		preamble: "type createdEvt struct {\n\tID       string\n\tPriority int\n}\n" +
			"type deletedEvt struct{ ID string }\n" +
			"var sqliteEng metaengine.Engine\nctx := context.Background()\n",
		trailers: "_ = store\n_ = reader\n_ = active\n_ = item\n_ = found",
	},
	"#### Bridging Stream IDs to Map Keys (TypeDecoder + EventWithID) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"`,
		},
		preamble: "type CreatedPayload struct{}\ntype UpdatedPayload struct{}\n" +
			"var store *metaengine.Store\n",
		trailers: "_ = decoder\n_ = adapter",
	},
	"#### Encoded Applies: projection.Projection → ApplyEncodedRecord (metaengine) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/projection/v4"`,
			`"context"`,
		},
		preamble: "var store *metaengine.Store\n",
		trailers: "_ = proj",
	},
	"#### Multi-Engine Distribution (Counter to Memory, Map to SQLite) #1": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`},
		preamble: "var counterQuery any\nvar mapQuery any\nvar sqliteEng metaengine.Engine\n",
		trailers: "_ = store",
	},
	"#### Vector ADT — Semantic Search (k-NN) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\n",
		trailers: "_ = results",
	},
	"#### Search ADT — Full-Text Search #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\n",
		trailers: "_ = results",
	},
	"#### Spatial ADT — Geo Proximity Search #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\n",
		trailers: "_ = results",
	},
	"#### Temporal Queries — Point-in-Time Reads #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"context"`,
			`"time"`,
		},
		preamble: "ctx := context.Background()\nvar someTimestamp time.Time\n",
		trailers: "_ = val",
	},
	"#### DuckDB Engine — Columnar Analytics #1": {
		skip: "DuckDB builds need CGo; excluded from this pure-Go compile gate (the duckdbengine module's own CI build covers it)",
	},
	"#### Postgres Engine — Production Durability #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
		},
		preamble: "type GetInput struct{ ID string }\n" +
			"type TodoCreated struct {\n\tID    string\n\tTitle string\n}\n" +
			"type TodoView struct {\n\tID    string\n\tTitle string\n}\n",
		trailers: "_ = store",
	},
	"#### Operator-Driven Layout Planning (metaengine — ADR-0124) #1": {
		imports:  []string{`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`},
		preamble: "var engines []metaengine.Engine\nvar query any\n",
		trailers: "_ = store",
	},
	"#### Operator-Driven Layout Planning (metaengine — ADR-0124) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"context"`,
			`"fmt"`,
		},
		preamble: "ctx := context.Background()\nvar store *metaengine.Store\n",
	},
	"#### Operator-Driven Layout Planning (metaengine — ADR-0124) #3": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"fmt"`,
		},
		preamble: "var store *metaengine.Store\n",
	},
	"#### Operator-Driven Layout Planning (metaengine — ADR-0124) #4": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"fmt"`,
			`"context"`,
		},
		preamble: "ctx := context.Background()\nvar store *metaengine.Store\n",
	},
}
