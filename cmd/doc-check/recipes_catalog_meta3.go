package main

// Recipe catalog, part B3: recipes.md §2.36–end (tail entries of part B2, split
// out to stay under the 350-line file cap).

var recipeCatalogB3 = map[string]recipeSpec{
	"### 2.36 Watch Dgraph Contention Retries (dgraphengine observer) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"`,
			`"sync/atomic"`,
		},
		preamble: "var addr string\nvar contentionRetries atomic.Int64\n",
		trailers: "_ = eng",
		errFunc:  true,
	},
	"### 2.37 Point-in-Time Reads: Versioned Cells & AsOf Routing (ADR-0141) #1": {
		imports: []string{`"time"`},
	},
	"### 2.37 Point-in-Time Reads: Versioned Cells & AsOf Routing (ADR-0141) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"`,
			`"time"`,
		},
		trailers: "_ = mem\n_ = sqlEng\n_ = err",
	},
	"### 2.37 Point-in-Time Reads: Versioned Cells & AsOf Routing (ADR-0141) #3": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"context"`,
			`"time"`,
		},
		preamble: "ctx := context.Background()\nvar mem metaengine.Engine\n" +
			"var lastTuesday, from, to time.Time\n",
		trailers: "_ = past\n_ = hist\n_ = err",
	},
	"### 2.38 Engine-Backed Timers, Queue Claims & Dedup — the ONE Substrate (ADR-0142) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/scheduling/v4"`,
			`"context"`,
			`"log"`,
			`"time"`,
		},
		preamble: "type DelayCommand struct {\n\tOrderID string\n\tReason  string\n}\n\n" +
			"ctx := context.Background()\ndispatch := func(DelayCommand) {}\n",
		trailers: "_ = eng",
	},
	"### 2.38 Engine-Backed Timers, Queue Claims & Dedup — the ONE Substrate (ADR-0142) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/queue/mysql/v4"`,
			`"context"`,
			`"log"`,
			`"time"`,
		},
		preamble: "ctx := context.Background()\nvar cmdID string\n",
		trailers: "_ = eng",
		errFunc:  true,
	},
	"### 2.38 Engine-Backed Timers, Queue Claims & Dedup — the ONE Substrate (ADR-0142) #3": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/scheduling/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`"context"`,
			`"log"`,
		},
		preamble: "type DelayCommand struct {\n\tOrderID string\n\tReason  string\n}\n\n" +
			"var sys *system.System\n" +
			"dispatch := func(ctx context.Context, p DelayCommand) error { return nil }\n",
	},
	"### 2.39 The Goal in 5 Minutes — declare types, swap engines by config (goal-shaped app) #1": {
		imports: []string{`"github.com/larsartmann/go-cqrs-lite/system/v4"`},
		preamble: "type TaskCreated struct {\n\tID       string\n\tTitle    string\n\tPriority int\n}\n\n" +
			"type TaskUpdated struct {\n\tID       string\n\tTitle    string\n\tStatus   string\n\tPriority int\n}\n\n" +
			"type TaskDeleted struct {\n\tID string\n}\n\n" +
			"type TaskView struct {\n\tID       string\n\tTitle    string\n\tStatus   string\n\tPriority int\n}\n\n",
		trailers: "_ = tasks\n_ = lookup\n_ = openTasks",
	},
	"### 2.40 Statistical Rigor: repeats, CoV, benchstat (benchkit) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/benchkit/v4"`,
			`"context"`,
			`"log"`,
			`"os"`,
		},
		preamble: "ctx := context.Background()\nvar factory benchkit.Factory\n",
	},
	"### 2.40 Statistical Rigor: repeats, CoV, benchstat (benchkit) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/benchkit/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/stack/v4"`,
			`"testing"`,
		},
	},
	"### 2.41 Declare Data Products + Contracts End-to-End (catalog, data mesh) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"`,
			`"log"`,
		},
		preamble: "type OrderPlaced struct{ OrderID string }\n",
	},
	"### 2.43 System-Level BDD Scenarios — Given/When/Then over a real system.New (systemscenario, ADR-0153) #1": {
		imports: []string{
			`"context"`,
			`"testing"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/id/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"`,
		},
		preamble: "type TaskCreated struct{ Title string }\n" +
			"type TaskView struct{ Title, Status string }\n" +
			"var _ = system.DomainConfig{}\n" +
			"func taskDomain() system.DomainConfig { return system.DomainConfig{} }\n" +
			"func memoryDeployment() system.DeploymentConfig { return system.DeploymentConfig{} }\n" +
			"func completeCmd(_ id.StreamRef) command.Command { return nil }\n" +
			"func viewQuery(_ *systemscenario.Scenario, _ context.Context, _ id.StreamRef) func() (any, error) {\n" +
			"	return func() (any, error) { return TaskView{}, nil }\n" +
			"}\n",
		trailers: "",
	},
	"### 2.43 System-Level BDD Scenarios — Given/When/Then over a real system.New (systemscenario, ADR-0153) #2": {
		imports: []string{
			`errorfamily "github.com/larsartmann/go-error-family"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"`,
		},
		preamble: "var registerCmd, renameCmd command.Command\n" +
			"var sc *systemscenario.Scenario\n",
		trailers: "",
	},
	"### 2.43 System-Level BDD Scenarios — Given/When/Then over a real system.New (systemscenario, ADR-0153) #3": {
		imports: []string{
			`"context"`,
			`"testing"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"`,
		},
		preamble: "func taskDomain() system.DomainConfig { return system.DomainConfig{} }\n" +
			"var t testing.TB\n" +
			"var ctx context.Context\n",
		trailers: "_ = sc\n_ = scSQLite\n",
	},
	"### 2.42 Request Correlation — RequestScope Enricher (event + decider) #1": {
		imports: []string{
			`"context"`,
			`"net/http"`,
			`"github.com/larsartmann/go-cqrs-lite/decider/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/id/v4"`,
		},
		preamble: "type State struct{}\nvar store event.Store\nvar bus event.Publisher\n" +
			"var d decider.Decider[State]\n",
		trailers: "_ = repo\n_ = err",
	},
	"### 2.44 Graph-Native Read Models — Edge folds, traversal, retraction (system + metaengine) #1": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
		},
		trailers: "_ = query\n_ = decoder",
	},
	"### 2.44 Graph-Native Read Models — Edge folds, traversal, retraction (system + metaengine) #2": {
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"`,
			`_ "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
		},
		preamble: "var query any\n" +
			"var decoder *projectionadapter.TypeDecoder\n" +
			"var registerFollowDeciderAndCommands func(*system.System)\n",
		trailers: "_ = domain\n_ = deploy",
	},
	"### 2.44 Graph-Native Read Models — Edge folds, traversal, retraction (system + metaengine) #3": {
		imports: []string{
			`"context"`,
			`"slices"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/decider/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
		},
		preamble: "type Followed struct{ Follower, Followee string }\n" +
			"type Unfollowed struct{ Follower, Followee string }\n" +
			"func follow(follower any, followee string) decider.DecideFunc[FollowState] { return nil }\n",
	},
	"### 2.44 Graph-Native Read Models — Edge folds, traversal, retraction (system + metaengine) #4": {
		imports: []string{
			`"context"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
		},
		preamble: "type Reachability struct {\n" +
			"\tNode  string\n" +
			"\tDepth int\n" +
			"}\n" +
			"ctx := context.Background()\nvar sys *system.System\n",
		trailers: "_ = reach\n_ = err",
	},
	"### 2.44 Graph-Native Read Models — Edge folds, traversal, retraction (system + metaengine) #5": {
		imports: []string{
			`"context"`,
			`"testing"`,
			`errorfamily "github.com/larsartmann/go-error-family"`,
			`"github.com/larsartmann/go-cqrs-lite/command/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"`,
		},
		preamble: "type Reachability struct {\n" +
			"\tNode  string\n" +
			"\tDepth int\n" +
			"}\n" +
			"var t testing.TB\n" +
			"var ctx context.Context\n" +
			"func graphDomain() system.DomainConfig { return system.DomainConfig{} }\n" +
			"func followCmd(t testing.TB, follower, followee string) command.Command { return nil }\n",
		trailers: "_ = sc",
	},
	"### 2.45 Chaos Journal & SSE Stream Assertions (systemscenario) #1": {
		imports: []string{
			`"testing"`,
			`"time"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"`,
		},
		preamble: "var t testing.TB\n",
		trailers: "_ = deploy",
	},
	"### 2.45 Chaos Journal & SSE Stream Assertions (systemscenario) #2": {
		imports: []string{
			`"context"`,
			`"testing"`,
			`"time"`,
			`"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"`,
		},
		preamble: "type TaskView struct{ Title string }\n" +
			"var t testing.TB\n" +
			"var ctx context.Context\n" +
			"var sc *systemscenario.Scenario\n",
		trailers: "_ = sub",
	},
	"### 2.46 Schema Evolution — Named Upcast Ops, Declared Once (schema + system) #1": {
		errFunc: true,
		imports: []string{
			`"context"`,
			`"github.com/larsartmann/go-cqrs-lite/schema/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
		},
		preamble: "type UserCreated struct{ Name string }\n" +
			"type UserDeleted struct{}\n" +
			"var ctx context.Context\n" +
			"var deployment system.DeploymentConfig\n",
		trailers: "_ = sys",
	},
	"### 2.46 Schema Evolution — Named Upcast Ops, Declared Once (schema + system) #2": {
		errFunc: true,
		imports: []string{
			`"github.com/larsartmann/go-cqrs-lite/event/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/schema/v4"`,
		},
		preamble: "var store event.Journal\n" +
			"var events = struct{ AttachmentBackedUp event.Type }{AttachmentBackedUp: \"discord.attachment.backed_up\"}\n",
		trailers: "_ = journal",
	},
	"### 2.47 Bind a system Query to an HTTP endpoint — ETag, Server-Timing, safe errors (system + go-etag + httputil) #1": {
		requires: []string{
			`github.com/larsartmann/go-etag/server`,
			`github.com/larsartmann/httputil`,
			`github.com/larsartmann/httputil/server_timing`,
		},
		imports: []string{
			`"context"`,
			`"crypto/sha256"`,
			`"encoding/hex"`,
			`"net/http"`,
			`"github.com/larsartmann/go-codec"`,
			`errorfamily "github.com/larsartmann/go-error-family"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/query/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`etag "github.com/larsartmann/go-etag/server"`,
			`"github.com/larsartmann/httputil"`,
			`"github.com/larsartmann/httputil/server_timing"`,
		},
		preamble: queryEndpointHelperPreamble +
			"type TaskView struct {\n\tID    string\n\tTitle string\n}\n\n" +
			"var sys *system.System\n" +
			"var mux *http.ServeMux\n" +
			"var taskReader *metaengine.TypedReader[TaskView]\n",
	},
	"### 2.47 Bind a system Query to an HTTP endpoint — ETag, Server-Timing, safe errors (system + go-etag + httputil) #2": {
		wholeProgram: true,
		requires: []string{
			`github.com/larsartmann/go-etag/server`,
			`github.com/larsartmann/httputil`,
			`github.com/larsartmann/httputil/server_timing`,
		},
	},
	"### 2.47 Bind a system Query to an HTTP endpoint — ETag, Server-Timing, safe errors (system + go-etag + httputil) #3": {
		requires: []string{
			`github.com/larsartmann/go-etag/server`,
			`github.com/larsartmann/httputil`,
			`github.com/larsartmann/httputil/server_timing`,
		},
		imports: []string{
			`"context"`,
			`"crypto/sha256"`,
			`"encoding/hex"`,
			`"net/http"`,
			`"github.com/larsartmann/go-codec"`,
			`errorfamily "github.com/larsartmann/go-error-family"`,
			`"github.com/larsartmann/go-cqrs-lite/query/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
			`etag "github.com/larsartmann/go-etag/server"`,
			`"github.com/larsartmann/httputil/server_timing"`,
		},
		preamble: queryEndpointHelperPreamble +
			"type TaskView struct {\n\tID    string\n\tTitle string\n}\n\n" +
			"type ListTasksQuery struct {\n\tFilter string\n\tLimit  int\n}\n\n" +
			"func (ListTasksQuery) Type() query.Type { return \"list_tasks\" }\n\n" +
			"func decodeListTasks(r *http.Request) (ListTasksQuery, error) {\n" +
			"\treturn ListTasksQuery{Filter: r.URL.Query().Get(\"filter\"), Limit: 50}, nil\n}\n\n" +
			"var sys *system.System\n" +
			"var mux *http.ServeMux\n",
	},
	"### 2.47 Bind a system Query to an HTTP endpoint — ETag, Server-Timing, safe errors (system + go-etag + httputil) #4": {
		requires: []string{
			`github.com/larsartmann/httputil`,
			`github.com/larsartmann/httputil/server_timing`,
		},
		imports: []string{
			`"log/slog"`,
			`"net/http"`,
			`"time"`,
			`"github.com/larsartmann/httputil"`,
			`"github.com/larsartmann/httputil/server_timing"`,
		},
		preamble: "var mux *http.ServeMux\n",
		trailers: "_ = srv",
	},
}

// queryEndpointHelperPreamble scaffolds the §2.47 httpx helper (fence #2) for
// the fences that consume it (#1/#3): the compiled usage blocks call the same
// Query/dispatchEncode pair the doc teaches, not a stub.
const queryEndpointHelperPreamble = `func Query[R any, Q query.Query](
	sys *system.System,
	decode func(r *http.Request) (Q, error),
	c codec.Codec,
) errorfamily.HandlerFunc {
	mediaType := "application/json"
	if c.Encoding() == codec.EncodingCBOR {
		mediaType = "application/cbor"
	}
	return func(w http.ResponseWriter, r *http.Request) error {
		q, err := decode(r)
		if err != nil {
			return err
		}
		stop := servertiming.MeasureServerTiming(r.Context(), "query")
		data, tag, err := dispatchEncode[Q, R](r.Context(), sys, q, c)
		stop()
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", mediaType)
		w.Header().Set("ETag", tag.String())
		if etag.MatchesIfNoneMatch(tag, r.Header.Get("If-None-Match")) {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
		_, err = w.Write(data)
		return err
	}
}

func dispatchEncode[Q query.Query, R any](ctx context.Context, sys *system.System, q Q, c codec.Codec) ([]byte, etag.ETag, error) {
	result, err := system.DispatchQuery[Q, R](ctx, sys, q)
	if err != nil {
		return nil, etag.ETag{}, err
	}
	data, err := c.Encode(result)
	if err != nil {
		return nil, etag.ETag{}, err
	}
	sum := sha256.Sum256(data)
	return data, etag.NewETag(hex.EncodeToString(sum[:]), etag.Strong), nil
}
`
