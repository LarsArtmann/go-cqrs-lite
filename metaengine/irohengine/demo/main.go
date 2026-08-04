package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

type demoNode struct {
	name   string
	engine metaengine.Engine
}

func main() {
	ctx := context.Background()

	printBanner("Iroh Level 2 CRDT Replication Demo")
	fmt.Println("Simulating a 3-node leaderless cluster with in-process P2P transport.")
	fmt.Println()

	// --- Setup: 3 nodes on a shared network ---
	net := irohengine.NewNetwork()

	nodes := make([]demoNode, 3)
	for i, id := range []string{"alpha", "beta", "gamma"} {
		nodes[i] = demoNode{
			name: id,
			engine: irohengine.Replicated(
				metaengine.NewMemoryEngine(),
				irohengine.WithAuthor(id),
				irohengine.WithTransport(net.Join(id)),
				irohengine.WithReplicationLag(100*time.Millisecond),
				irohengine.WithNetworkRTT(50*time.Millisecond),
			),
		}
	}
	defer func() {
		for _, n := range nodes {
			n.engine.Close()
		}
	}()

	// --- Engine Profile ---
	section("Engine Profile (leaderless replication declared)")
	profile := nodes[0].engine.Profile()
	fmt.Printf("  Name           : %s\n", profile.Name)
	fmt.Printf("  Replication    : %s\n", profile.Replication)
	fmt.Printf("  ReplicationLag : %v\n", profile.ReplicationLag)
	fmt.Printf("  NetworkRTT     : %v\n", profile.NetworkRTT)
	fmt.Println()

	mb := func(e metaengine.Engine) metaengine.MapBackend { return e.(metaengine.MapBackend) }
	cb := func(e metaengine.Engine) metaengine.CounterBackend { return e.(metaengine.CounterBackend) }
	sb := func(e metaengine.Engine) metaengine.SetBackend { return e.(metaengine.SetBackend) }
	lb := func(e metaengine.Engine) metaengine.LogBackend { return e.(metaengine.LogBackend) }
	mmb := func(e metaengine.Engine) metaengine.MultimapBackend { return e.(metaengine.MultimapBackend) }

	// --- 1. Map (LWW-Map) convergence ---
	section("1. Map (LWW-Map) — write on alpha, read on all nodes")
	must(
		mb(
			nodes[0].engine,
		).MapSet(ctx, "users", "u1", map[string]any{"name": "Alice", "role": "admin"}),
	)
	printAllNodes(nodes, func(n demoNode) string {
		val, ok, _ := mb(n.engine).MapGet(ctx, "users", "u1")
		if !ok {
			return "(missing)"
		}
		return fmt.Sprintf("%v", val)
	})

	// --- 2. LWW conflict resolution ---
	section("2. LWW Conflict Resolution — concurrent writes, latest timestamp wins")
	fmt.Println("  alpha writes u1=name:Alice   (t0)")
	must(mb(nodes[0].engine).MapSet(ctx, "users", "u2", "from-alpha"))
	time.Sleep(15 * time.Millisecond) // ensure beta's timestamp is later
	fmt.Println("  beta  writes u1=name:Bob     (t1 > t0) — should win")
	must(mb(nodes[1].engine).MapSet(ctx, "users", "u2", "from-beta"))
	printAllNodes(nodes, func(n demoNode) string {
		val, _, _ := mb(n.engine).MapGet(ctx, "users", "u2")
		return fmt.Sprintf("u2=%v", val)
	})

	// --- 3. PN-Counter convergence ---
	section("3. PN-Counter — increments from multiple nodes converge to sum")
	must(cb(nodes[0].engine).CounterIncrement(ctx, "page-views", metaengine.Delta{"total": 100}))
	must(cb(nodes[1].engine).CounterIncrement(ctx, "page-views", metaengine.Delta{"total": 50}))
	must(cb(nodes[2].engine).CounterIncrement(ctx, "page-views", metaengine.Delta{"total": 25}))
	printAllNodes(nodes, func(n demoNode) string {
		counts, _ := cb(n.engine).CounterGet(ctx, "page-views")
		return fmt.Sprintf("total=%d", counts["total"])
	})

	// --- 4. OR-Set convergence ---
	section("4. OR-Set (add-only) — union of tags across nodes")
	must(sb(nodes[0].engine).SetAdd(ctx, "tags", "go"))
	must(sb(nodes[0].engine).SetAdd(ctx, "tags", "cqrs"))
	must(sb(nodes[1].engine).SetAdd(ctx, "tags", "crdt"))
	must(sb(nodes[2].engine).SetAdd(ctx, "tags", "event-sourcing"))
	printAllNodes(nodes, func(n demoNode) string {
		var tags []string
		for _, t := range []string{"go", "cqrs", "crdt", "event-sourcing"} {
			ok, _ := sb(n.engine).SetContains(ctx, "tags", t)
			if ok {
				tags = append(tags, t)
			}
		}
		return "[" + strings.Join(tags, ", ") + "]"
	})

	// --- 5. Log (append-only) convergence ---
	section("5. Log (per-author append) — audit trail converges")
	must(lb(nodes[0].engine).LogAppend(ctx, "audit", "alpha:user-login"))
	must(lb(nodes[1].engine).LogAppend(ctx, "audit", "beta:config-change"))
	must(lb(nodes[2].engine).LogAppend(ctx, "audit", "gamma:file-upload"))
	printAllNodes(nodes, func(n demoNode) string {
		entries, _ := lb(n.engine).LogTail(ctx, "audit", 10)
		var strs []string
		for _, e := range entries {
			strs = append(strs, fmt.Sprintf("%v", e))
		}
		return "[" + strings.Join(strs, ", ") + "]"
	})

	// --- 6. Multimap (OR-Set per key) convergence ---
	section("6. Multimap (OR-Set per key) — team members union")
	must(mmb(nodes[0].engine).MultiAdd(ctx, "teams", "platform", "alice"))
	must(mmb(nodes[1].engine).MultiAdd(ctx, "teams", "platform", "bob"))
	must(mmb(nodes[2].engine).MultiAdd(ctx, "teams", "platform", "carol"))
	printAllNodes(nodes, func(n demoNode) string {
		vals, _ := mmb(n.engine).MultiGet(ctx, "teams", "platform")
		var strs []string
		for _, v := range vals {
			strs = append(strs, fmt.Sprintf("%v", v))
		}
		return "[" + strings.Join(strs, ", ") + "]"
	})

	// --- 7. MapDelete (LWW tombstone) ---
	section("7. MapDelete (LWW tombstone) — delete propagates")
	must(mb(nodes[1].engine).MapDelete(ctx, "users", "u1"))
	printAllNodes(nodes, func(n demoNode) string {
		_, ok, _ := mb(n.engine).MapGet(ctx, "users", "u1")
		if ok {
			return "STILL EXISTS (not yet converged)"
		}
		return "deleted (converged)"
	})

	// --- 8. MapUpdate does NOT replicate (CALM theorem boundary) ---
	section("8. MapUpdate — NON-CRDT, stays local (CALM theorem)")
	mu := nodes[0].engine.(metaengine.MapUpdater)
	must(mu.MapUpdate(ctx, "local-cache", "counter", func(prev any) any {
		if n, ok := prev.(int64); ok {
			return n + 1
		}
		return int64(1)
	}))
	fmt.Println("  alpha did MapUpdate (read-modify-write) on local-cache/counter")
	printAllNodes(nodes, func(n demoNode) string {
		val, ok, _ := mb(n.engine).MapGet(ctx, "local-cache", "counter")
		if !ok {
			return "(missing)"
		}
		return fmt.Sprintf("counter=%v", val)
	})
	fmt.Println("  ^ Only alpha has it. MapUpdate requires coordination — NOT replicated.")

	fmt.Println()
	printBanner("Demo complete — all CRDT types converged, non-CRDT stayed local")
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
}

func printBanner(msg string) {
	line := strings.Repeat("=", len(msg)+4)
	fmt.Printf("\n%s\n  %s\n%s\n", line, msg, line)
}

func section(msg string) {
	fmt.Printf("\n--- %s ---\n", msg)
}

func printAllNodes(nodes []demoNode, read func(n demoNode) string) {
	for _, n := range nodes {
		fmt.Printf("  %-8s: %s\n", n.name, read(n))
	}
}
