package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
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

	printBanner("Iroh CRDT Replication — Live Measurement Demo")
	fmt.Println("3-node leaderless cluster. Simulated 0–20ms link latency.")
	fmt.Println("ALL latency numbers are MEASURED from real traffic — no hardcoded values.")
	fmt.Println()

	net := irohengine.NewNetwork(
		irohengine.WithNetworkDelay(20 * time.Millisecond),
	)

	nodes := []demoNode{}
	for _, id := range []string{"alpha", "beta", "gamma"} {
		nodes = append(nodes, demoNode{
			name: id,
			engine: irohengine.Replicated(
				metaengine.NewMemoryEngine(),
				irohengine.WithAuthor(id),
				irohengine.WithTransport(net.Join(id)),
			),
		})
	}
	defer func() {
		for _, n := range nodes {
			n.engine.Close()
		}
	}()

	mb := func(e metaengine.Engine) metaengine.MapBackend { return e.(metaengine.MapBackend) }
	cb := func(e metaengine.Engine) metaengine.CounterBackend { return e.(metaengine.CounterBackend) }
	sb := func(e metaengine.Engine) metaengine.SetBackend { return e.(metaengine.SetBackend) }
	lb := func(e metaengine.Engine) metaengine.LogBackend { return e.(metaengine.LogBackend) }
	mmb := func(e metaengine.Engine) metaengine.MultimapBackend { return e.(metaengine.MultimapBackend) }

	// --- Phase 1: Warmup ---
	section("Phase 1: Warmup — 20 Map operations to build measurement history")
	for i := range 20 {
		n := nodes[rand.Intn(len(nodes))]
		must(mb(n.engine).MapSet(ctx, "warmup", fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i)))
		progressBar("warmup", i+1, 20)
	}
	printStats(net)

	// --- Phase 2: Concurrent write storm ---
	section("Phase 2: Concurrent storm — each node writes 50 entries simultaneously")
	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go func(n demoNode) {
			defer wg.Done()
			for i := range 50 {
				key := fmt.Sprintf("%s-%d", n.name, i)
				_ = mb(n.engine).MapSet(ctx, "storm", key, fmt.Sprintf("val-%d", i))
			}
		}(n)
	}
	wg.Wait()
	fmt.Println("  150 concurrent writes completed.")
	printStats(net)

	// --- Phase 3: Correctness ---
	section("Phase 3: Correctness — verify all nodes converged")
	allOK := true
	for i := range 20 {
		key := fmt.Sprintf("k%d", i)
		for _, n := range nodes {
			_, ok, _ := mb(n.engine).MapGet(ctx, "warmup", key)
			if !ok {
				fmt.Printf("  FAIL: node %s missing warmup/%s\n", n.name, key)
				allOK = false
			}
		}
	}
	for _, name := range []string{"alpha", "beta", "gamma"} {
		for i := range 50 {
			key := fmt.Sprintf("%s-%d", name, i)
			for _, n := range nodes {
				_, ok, _ := mb(n.engine).MapGet(ctx, "storm", key)
				if !ok {
					fmt.Printf("  FAIL: node %s missing storm/%s\n", n.name, key)
					allOK = false
				}
			}
		}
	}
	if allOK {
		fmt.Println("  OK: all 3 nodes see all 170 keys (20 warmup + 150 storm)")
	}

	// --- Phase 4: Counter ---
	section("Phase 4: PN-Counter — 30 increments across 3 nodes")
	for range 10 {
		for _, n := range nodes {
			must(cb(n.engine).CounterIncrement(ctx, "hits", metaengine.Delta{"total": 1}))
		}
	}
	fmt.Println("  Verifying PN-Counter sum (expect total=30):")
	for _, n := range nodes {
		counts, _ := cb(n.engine).CounterGet(ctx, "hits")
		fmt.Printf("  %-8s: hits=%d\n", n.name, counts["total"])
	}

	// --- Phase 5: Set + Log + Multimap ---
	section("Phase 5: OR-Set + Log + Multimap")
	must(sb(nodes[0].engine).SetAdd(ctx, "tags", "go"))
	must(sb(nodes[1].engine).SetAdd(ctx, "tags", "crdt"))
	must(sb(nodes[2].engine).SetAdd(ctx, "tags", "event-sourcing"))
	must(lb(nodes[0].engine).LogAppend(ctx, "audit", "alpha:deploy"))
	must(lb(nodes[1].engine).LogAppend(ctx, "audit", "beta:rollback"))
	must(mmb(nodes[0].engine).MultiAdd(ctx, "teams", "platform", "alice"))
	must(mmb(nodes[2].engine).MultiAdd(ctx, "teams", "platform", "carol"))

	fmt.Println("  OR-Set tags:")
	for _, n := range nodes {
		var found []string
		for _, t := range []string{"go", "crdt", "event-sourcing"} {
			ok, _ := sb(n.engine).SetContains(ctx, "tags", t)
			if ok {
				found = append(found, t)
			}
		}
		fmt.Printf("  %-8s: [%s]\n", n.name, strings.Join(found, ", "))
	}
	fmt.Println("  Log audit:")
	for _, n := range nodes {
		entries, _ := lb(n.engine).LogTail(ctx, "audit", 10)
		var strs []string
		for _, e := range entries {
			strs = append(strs, fmt.Sprintf("%v", e))
		}
		fmt.Printf("  %-8s: [%s]\n", n.name, strings.Join(strs, ", "))
	}
	fmt.Println("  Multimap teams/platform:")
	for _, n := range nodes {
		vals, _ := mmb(n.engine).MultiGet(ctx, "teams", "platform")
		var strs []string
		for _, v := range vals {
			strs = append(strs, fmt.Sprintf("%v", v))
		}
		fmt.Printf("  %-8s: [%s]\n", n.name, strings.Join(strs, ", "))
	}

	// --- Phase 6: EngineProfile ---
	section("Phase 6: EngineProfile — computed from REAL measurements")
	for _, n := range nodes {
		p := n.engine.Profile()
		fmt.Printf("  %-8s: replication=%s  lag=%s  rtt=%s\n",
			n.name, p.Replication, p.ReplicationLag, p.NetworkRTT)
	}
	fmt.Println("  ^ MEASURED, not hardcoded.")
	fmt.Println("    ReplicationLag = P99 convergence (last peer to apply).")
	fmt.Println("    NetworkRTT     = 2 × P50 delivery latency.")

	// --- Phase 7: CALM boundary ---
	section("Phase 7: MapUpdate — NON-CRDT (CALM theorem boundary)")
	mu := nodes[0].engine.(metaengine.MapUpdater)
	must(mu.MapUpdate(ctx, "local", "counter", func(prev any) any {
		if n, ok := prev.(int64); ok {
			return n + 1
		}
		return int64(1)
	}))
	fmt.Println("  alpha did MapUpdate — stays local:")
	for _, n := range nodes {
		val, ok, _ := mb(n.engine).MapGet(ctx, "local", "counter")
		if !ok {
			fmt.Printf("  %-8s: (missing)\n", n.name)
		} else {
			fmt.Printf("  %-8s: counter=%v\n", n.name, val)
		}
	}
	fmt.Println("  ^ Only alpha. Non-monotonic ops require coordination.")

	printStats(net)
	fmt.Println()
	printBanner("Demo complete — all stats measured from real replication traffic")
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nFATAL: %v\n", err)
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

func progressBar(label string, done, total int) {
	width := 30
	pct := done * width / total
	bar := strings.Repeat("█", pct) + strings.Repeat("░", width-pct)
	fmt.Printf("\r  %s [%s] %d/%d", label, bar, done, total)
	if done == total {
		fmt.Println()
	}
}

func printStats(net *irohengine.InProcessNetwork) {
	c := net.Collector()
	d := c.DeliveryStats()
	conv := c.ConvergenceStats()
	fmt.Println()
	fmt.Println("  Delivery Latency (one-way, measured per message):")
	fmt.Printf("    samples=%d  mean=%s  P50=%s  P95=%s  P99=%s  max=%s\n",
		d.Samples, d.Mean, d.P50, d.P95, d.P99, d.Max)
	fmt.Println("  Convergence Time (publish → last peer applied):")
	fmt.Printf("    samples=%d  mean=%s  P50=%s  P95=%s  P99=%s  max=%s\n",
		conv.Samples, conv.Mean, conv.P50, conv.P95, conv.P99, conv.Max)
	fmt.Println()
}
