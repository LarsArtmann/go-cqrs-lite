// Package dgraphengine provides a Dgraph-backed metaengine Engine.
//
// Dgraph is a distributed graph database with native DQL (GraphQL+-) query
// support. This engine implements MapBackend, CounterBackend, ScanBackend,
// GraphBackend, SetBackend, and SearchBackend.
//
// GraphBackend is Dgraph's native strength — O(degree^depth) traversal with
// zero degradation. SearchBackend leverages Dgraph's built-in term index
// (@index(term)) for efficient full-text search. MapBackend uses Dgraph's
// exact index (@index(exact)) for O(logN) point lookups.
//
// Pure Go (no CGo): uses the dgo v240 gRPC client.
//
// Calibrated cost model: gRPC round-trip + RAFT consensus write.
// Point-lookup benchmarks estimate ~10K ns/op (write) and ~8K ns/op (read)
// for a same-datacenter deployment. Dgraph uses single-leader replication
// per group (RAFT), so all writes go through the group leader.
package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"
	"sync"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// DG_NsPerOp models production Dgraph (RAFT consensus + gRPC round-trip).
const DG_NsPerOp = 10000.0

// DG_NsPerRead models production Dgraph (index lookup + gRPC response).
const DG_NsPerRead = 8000.0

// dgraphEngine implements metaengine.Engine with Dgraph as the backend.
type dgraphEngine struct {
	client      *dgo.Dgraph
	mu          sync.Mutex
	done        bool
	schemaMu    sync.Mutex
	appliedSchemas map[string]bool
	cal         metaengine.Calibration
}

// New creates a Dgraph-backed metaengine Engine from a gRPC address.
// The address should be in "host:port" format (e.g., "localhost:9080").
// The connection uses insecure transport (no TLS).
func New(addr string) (metaengine.Engine, error) {
	client, err := dgo.NewClient(addr,
		dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, fmt.Errorf("dgraphengine.New: connect: %w", err)
	}

	eng := &dgraphEngine{client: client}

	if err := eng.init(); err != nil {
		client.Close()
		return nil, err
	}

	return eng, nil
}

// NewFromClient wraps an existing dgo.Dgraph client.
func NewFromClient(client *dgo.Dgraph) (metaengine.Engine, error) {
	eng := &dgraphEngine{client: client}

	if err := eng.init(); err != nil {
		return nil, err
	}

	return eng, nil
}

func (e *dgraphEngine) init() error {
	schema := `
		cqrs.map_collection: string @index(exact) @upsert .
		cqrs.map_key: string @index(exact) @upsert .
		cqrs.map_value: string .
		cqrs.counter_collection: string @index(exact) @upsert .
		cqrs.counter_key: string @index(exact) @upsert .
		cqrs.counter_value: int .
		cqrs.set_collection: string @index(exact) @upsert .
		cqrs.set_key: string @index(exact) @upsert .
		cqrs.node_collection: string @index(exact) @upsert .
		cqrs.node_id: string @index(exact) @upsert .
		cqrs.search_collection: string @index(exact) .
		cqrs.search_id: string @index(exact) @upsert .
		cqrs.search_content: string @index(term) .
	`

	return e.client.Alter(context.Background(), &api.Operation{Schema: schema})
}

// Profile returns the cost profile for this Dgraph engine.
func (e *dgraphEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        "dgraph",
		NsPerOp:     DG_NsPerOp,
		NsPerRead:   DG_NsPerRead,
		Persistence: metaengine.PersistencePersistent,
		Replication: metaengine.ReplicationSingleLeader,
		ReadCosts: metaengine.ReadCosts{
			NsPerPointLookup:  8_000,
			NsPerFilteredScan: 1_000,
			NsPerAggregate:    500,
			NsPerScan:         2_000,
		},
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap:       metaengine.ComplexityOLogN,
			metaengine.ADTCounter:   metaengine.ComplexityO1,
			metaengine.ADTGraph:     metaengine.ComplexityODegree,
			metaengine.ADTSet:       metaengine.ComplexityOLogN,
			metaengine.ADTSortedMap: metaengine.ComplexityON,
			metaengine.ADTSearch:    metaengine.ComplexityOLogN,
		},
		DegradedADTs: map[metaengine.ADT]bool{
			metaengine.ADTSortedMap: true,
		},
		Layouts: map[metaengine.ADT]metaengine.StorageLayout{
			metaengine.ADTMap:       metaengine.LayoutKV,
			metaengine.ADTCounter:   metaengine.LayoutKV,
			metaengine.ADTGraph:     metaengine.LayoutKV,
			metaengine.ADTSet:       metaengine.LayoutKV,
			metaengine.ADTSortedMap: metaengine.LayoutKV,
			metaengine.ADTSearch:    metaengine.LayoutKV,
		},
	}
	e.cal.ApplyCalibration(&p)

	return p
}

// SetCalibration implements metaengine.Calibratable.
func (e *dgraphEngine) SetCalibration(costs metaengine.CalibrationCosts) {
	e.cal.SetCalibration(costs)
}

// Close closes the underlying Dgraph client. Safe to call multiple times.
func (e *dgraphEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.done {
		return nil
	}

	e.done = true
	e.client.Close()

	return nil
}

// --- MapBackend ---

func (e *dgraphEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	keyStr := fmt.Sprint(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("dgraphengine.MapSet: marshal: %w", err)
	}

	valueStr := string(data)

	req := &api.Request{CommitNow: true}
	req.Query = fmt.Sprintf(`{
		entry as var(func: eq(cqrs.map_collection, %s)) @filter(eq(cqrs.map_key, %s))
	}`, dqlString(col), dqlString(keyStr))

	createJSON, _ := json.Marshal(map[string]any{
		"uid":                 "_:new",
		"cqrs.map_collection": col,
		"cqrs.map_key":        keyStr,
		"cqrs.map_value":      valueStr,
		"dgraph.type":         []string{"MetaMapEntry"},
	})

	updateJSON, _ := json.Marshal(map[string]any{
		"uid":            "uid(entry)",
		"cqrs.map_value": valueStr,
	})

	req.Mutations = []*api.Mutation{
		{SetJson: createJSON, Cond: "@if(eq(len(entry), 0))"},
		{SetJson: updateJSON, Cond: "@if(eq(len(entry), 1))"},
	}

	if _, err := e.client.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.MapSet: %w", err)
	}

	return nil
}

func (e *dgraphEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	keyStr := fmt.Sprint(key)

	q := fmt.Sprintf(`{
		entry(func: eq(cqrs.map_collection, %s)) @filter(eq(cqrs.map_key, %s)) {
			cqrs.map_value
		}
	}`, dqlString(col), dqlString(keyStr))

	resp, err := e.client.NewReadOnlyTxn().Query(ctx, q)
	if err != nil {
		return nil, false, fmt.Errorf("dgraphengine.MapGet: %w", err)
	}

	var result struct {
		Entry []struct {
			MapValue string `json:"cqrs.map_value"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, false, fmt.Errorf("dgraphengine.MapGet: unmarshal: %w", err)
	}

	if len(result.Entry) == 0 {
		return nil, false, nil
	}

	var val any

	if err := json.Unmarshal([]byte(result.Entry[0].MapValue), &val); err != nil {
		return nil, false, fmt.Errorf("dgraphengine.MapGet: decode value: %w", err)
	}

	return val, true, nil
}

func (e *dgraphEngine) MapDelete(ctx context.Context, col string, key any) error {
	keyStr := fmt.Sprint(key)

	req := &api.Request{CommitNow: true}
	req.Query = fmt.Sprintf(`{
		entry as var(func: eq(cqrs.map_collection, %s)) @filter(eq(cqrs.map_key, %s))
	}`, dqlString(col), dqlString(keyStr))

	deleteJSON, _ := json.Marshal(map[string]any{
		"uid": "uid(entry)",
	})

	req.Mutations = []*api.Mutation{
		{DeleteJson: deleteJSON},
	}

	if _, err := e.client.NewTxn().Do(ctx, req); err != nil {
		return fmt.Errorf("dgraphengine.MapDelete: %w", err)
	}

	return nil
}

// --- Helpers ---

// dqlString escapes a Go string into a DQL string literal (double-quoted).
func dqlString(s string) string {
	var b strings.Builder
	b.WriteByte('"')

	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}

	b.WriteByte('"')

	return b.String()
}

// sanitizePredicate builds a safe Dgraph predicate name from components.
func sanitizePredicate(parts ...string) string {
	var b strings.Builder

	for i, p := range parts {
		if i > 0 {
			b.WriteByte('.')
		}

		for _, r := range p {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '_' || r == '.' {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
	}

	return b.String()
}

// graphEdgePredicate returns the Dgraph predicate name for a graph edge collection.
func graphEdgePredicate(collection string) string {
	return sanitizePredicate("cqrs", "edge", collection)
}

// ensureEdgeSchema lazily creates the edge predicate schema with @reverse.
func (e *dgraphEngine) ensureEdgeSchema(ctx context.Context, collection string) error {
	e.schemaMu.Lock()
	defer e.schemaMu.Unlock()

	if e.appliedSchemas == nil {
		e.appliedSchemas = make(map[string]bool)
	}

	pred := graphEdgePredicate(collection)
	if e.appliedSchemas[pred] {
		return nil
	}

	schema := fmt.Sprintf("%s: [uid] @reverse .", pred)
	if err := e.client.Alter(ctx, &api.Operation{Schema: schema}); err != nil {
		return fmt.Errorf("dgraphengine.ensureEdgeSchema: %w", err)
	}

	e.appliedSchemas[pred] = true

	return nil
}

// Compile-time assertions.
var (
	_ metaengine.Engine         = (*dgraphEngine)(nil)
	_ metaengine.MapBackend     = (*dgraphEngine)(nil)
	_ metaengine.CounterBackend = (*dgraphEngine)(nil)
	_ metaengine.ScanBackend    = (*dgraphEngine)(nil)
	_ metaengine.GraphBackend   = (*dgraphEngine)(nil)
	_ metaengine.SetBackend     = (*dgraphEngine)(nil)
	_ metaengine.SearchBackend  = (*dgraphEngine)(nil)
	_ metaengine.Calibratable   = (*dgraphEngine)(nil)
)
