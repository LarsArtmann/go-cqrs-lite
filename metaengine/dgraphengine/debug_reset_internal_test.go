package dgraphengine

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/dgraph-io/dgo/v240/protos/api"
)

// dgraphAddrEnv mirrors the external test helper's DGRAPH_ADDR resolution.
func dgraphAddrEnv() string {
	if addr := os.Getenv("DGRAPH_ADDR"); addr != "" {
		return addr
	}

	return "localhost:9080"
}

// TestResetUpsertVariants diagnoses which upsert shape actually deletes
// engine nodes: (a) one query block with multiple var roots + DelNquads,
// (b) the same with DelJson, (c) raw upsert-string form. Temporary — remove
// once the working shape is folded into ResetEngine. Live test.
func TestResetUpsertVariants(t *testing.T) {
	engIface, err := New(dgraphAddrEnv())
	if err != nil {
		t.Skipf("Dgraph not available: %v", err)
	}

	eng := engIface.(*dgraphEngine)

	t.Cleanup(func() { _ = eng.Close() })

	ctx := context.Background()
	col := fmt.Sprintf("dbg_%p", eng)

	mb := eng.(interface {
		MapSet(ctx context.Context, col string, key, value any) error
		MapGet(ctx context.Context, col string, key any) (any, bool, error)
	})

	seed := func() {
		if err := mb.MapSet(ctx, col, "t1", "v1"); err != nil {
			t.Fatalf("MapSet: %v", err)
		}
	}

	gone := func() bool {
		_, ok, err := mb.MapGet(ctx, col, "t1")
		if err != nil {
			t.Fatalf("MapGet: %v", err)
		}

		return !ok
	}

	// Variant A: query + DelNquads (current ResetEngine shape).
	seed()

	reqA := &api.Request{
		Query: resetTypeQuery(resetNodeTypes),
		Mutations: []*api.Mutation{{
			DelNquads: []byte(resetDeleteNQuads(resetNodeTypes)),
		}},
	}

	respA, err := eng.doWrite(ctx, reqA)
	if err != nil {
		t.Logf("variant A error: %v", err)
	} else {
		t.Logf("variant A resp json=%s uids=%v metrics=%v",
			string(respA.Json), respA.Uids, respA.Metrics)
	}

	t.Logf("variant A deleted: %v", gone())

	// Variant B: raw upsert string with delete block embedded.
	seed()

	q := "upsert { query { v as var(func: type(MetaMapEntry)) } "
	q += "mutation { delete { uid(v) * * . } } }"

	respB, err := eng.client.NewTxn().Do(ctx, &api.Request{Query: q, CommitNow: true})
	if err != nil {
		t.Logf("variant B error: %v", err)
	} else {
		t.Logf("variant B resp json=%s", string(respB.Json))
	}

	t.Logf("variant B deleted: %v", gone())

	// Variant C: query only — does the multi-root var block bind at all?
	seed()

	respC, err := eng.client.NewTxn().Query(ctx, resetTypeQuery(resetNodeTypes))
	if err != nil {
		t.Logf("variant C error: %v", err)
	} else {
		t.Logf("variant C resp json=%s", string(respC.Json))
	}

	t.Logf("variant C still present (expected true): %v", !gone())
}
