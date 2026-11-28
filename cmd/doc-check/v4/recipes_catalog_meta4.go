package main

// Recipe catalog, part B4: recipes.md §2.47–§2.52 (the tail of section 2,
// split out of part B3 to keep files under the 350-line file cap).

var recipeCatalogB4 = map[string]recipeSpec{
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
	"### 2.47 Bind a system Query to an HTTP endpoint — ETag, Server-Timing, safe errors (system + go-etag + httputil) #5": {
		requires: []string{
			`github.com/larsartmann/go-health`,
		},
		imports: []string{
			`"net/http"`,
			`"github.com/larsartmann/go-health"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
		},
		preamble: "var sys *system.System\n" +
			"var mux *http.ServeMux\n",
	},
	"### 2.48 Publish a Self-Contained Architecture Reference (catalog/architecture) #1": {
		imports: []string{
			`"log"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"`,
		},
		preamble: "type OrderPlaced struct{ OrderID string }\n",
	},
	"### 2.49 Pin Your Engine's Stored Bytes (sqliteengine / pebbleengine / bboltengine / badgerengine) #1": {
		imports: []string{
			`"context"`,
			`"database/sql"`,
			`"strings"`,
			`"testing"`,
			`_ "modernc.org/sqlite"`,
			`sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"`,
			`metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
		},
	},
	"### 2.50 Per-Group Caps — keep the newest N rows PER owner (`WithCapPerGroup`, ADR-0164) #1": {
		imports: []string{
			`"context"`,
			`"github.com/larsartmann/go-cqrs-lite/metaengine/v4"`,
			`"github.com/larsartmann/go-cqrs-lite/record/v4"`,
		},
		preamble: "type TranscriptRecorded struct {\n" +
			"\tID, Owner string\n" +
			"\tStartedAt int64\n" +
			"}\n" +
			"type TranscriptView struct {\n" +
			"\tID, Owner string\n" +
			"\tStartedAt int64\n" +
			"}\n" +
			"var eng metaengine.Engine\n" +
			"var ctx context.Context\n",
		errFunc:  true,
		trailers: "_ = removed\n_ = err",
	},
	"### 2.51 The Deployment Ladder — same preset from Go, CUE, or NixOS (`system.Standard`, ADR-0165) #1": {
		imports: []string{
			`"context"`,
			`"fmt"`,
			`"github.com/larsartmann/go-cqrs-lite/system/v4"`,
		},
		preamble: "var ctx = context.Background()\n" +
			"var domain system.DomainConfig\n",
		trailers: "_ = sys\n_ = err",
	},
	"### 2.52 Search: fields declared once, one call (`search/v4`) #1": {
		imports: []string{
			`"context"`,
			`"fmt"`,
			`"net/http"`,
			`"github.com/larsartmann/go-cqrs-lite/search/v4"`,
		},
		preamble: "var ctx = context.Background()\n" +
			"var r *http.Request\n" +
			"var allNotes []Note\n",
		errFunc: true,
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
