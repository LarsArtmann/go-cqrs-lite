package dgraphengine_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	dgo "github.com/dgraph-io/dgo/v240"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	dgraphengine "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type dbgGraph interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

func TestDebugRecurseShape(t *testing.T) {
	addr := os.Getenv("DGRAPH_ADDR")
	if addr == "" {
		addr = "localhost:9080"
	}

	client, err := dgo.NewClient(addr,
		dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		t.Skipf("dgraph unavailable: %v", err)
	}

	eng, err := dgraphengine.NewFromClient(client)
	if err != nil {
		t.Skipf("engine unavailable: %v", err)
	}

	defer func() { _ = eng.Close() }()

	gb := eng.(dbgGraph)
	ctx := context.Background()
	col := uniqueCollection(t, "dbg_chain")

	edges := []metaengine.Edge{
		{From: "A", To: "B"},
		{From: "B", To: "C"},
		{From: "C", To: "D"},
		{From: "D", To: "E"},
	}
	for _, e := range edges {
		if err := gb.GraphAddEdge(ctx, col, e); err != nil {
			t.Fatalf("add edge: %v", err)
		}
	}

	got, err := gb.GraphNeighbors(ctx, col, "A", 2)
	if err != nil {
		t.Fatalf("neighbors: %v", err)
	}
	fmt.Printf(">>> GraphNeighbors depth2 = %v\n", got)

	got3, err := gb.GraphNeighbors(ctx, col, "A", 3)
	if err != nil {
		t.Fatalf("neighbors3: %v", err)
	}
	fmt.Printf(">>> GraphNeighbors depth3 = %v\n", got3)

	// Raw DQL dump for the same fixture.
	pred := fmt.Sprintf("cqrs.node_edge_%s", col)
	for _, d := range []int{2, 3} {
		q := fmt.Sprintf(`query root($col: string, $node: string) {
			root(func: eq(cqrs.node_collection, $col)) @filter(eq(cqrs.node_id, $node)) @recurse(depth: %d, loop: false) {
				cqrs.node_id
				%s
			}
		}`, d, pred)

		resp, qerr := client.NewReadOnlyTxn().QueryWithVars(ctx, q,
			map[string]string{"$col": col, "$node": "A"})
		if qerr != nil {
			t.Fatalf("raw query d=%d: %v", d, qerr)
		}
		fmt.Printf(">>> RAW depth=%d json=%s\n", d, string(resp.GetJson()))
	}
}
