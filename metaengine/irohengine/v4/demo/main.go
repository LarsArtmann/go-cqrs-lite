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

// backendOf asserts a backend capability, panicking with the engine's type
// (the demo wires engines that implement every backend — a miss is a bug).
func backendOf[T any](e metaengine.Engine) T {
	backend, ok := e.(T)
	if !ok {
		panic(fmt.Sprintf("engine %T does not implement %T", e, backend))
	}

	return backend
}

func newDemoNodes(net *irohengine.InProcessNetwork) []demoNode {
	nodes := []demoNode{}
	for _, nodeName := range []string{"alpha", "beta", "gamma"} {
		nodes = append(nodes, demoNode{
			name: nodeName,
			engine: irohengine.Replicated(
				metaengine.NewMemoryEngine(),
				irohengine.WithAuthor(nodeName),
				irohengine.WithTransport(net.Join(nodeName)),
			),
		})
	}

	return nodes
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

	nodes := newDemoNodes(net)

	defer func() {
		for _, n := range nodes {
			_ = n.engine.Close()
		}
	}()

	warmupPhase(ctx, nodes)
	printStats(net)

	stormPhase(ctx, nodes)
	printStats(net)

	verifyPhase(ctx, nodes)
	counterPhase(ctx, nodes)
	structuresPhase(ctx, nodes)
	profilePhase(nodes)
	calmPhase(ctx, nodes)

	printStats(net)
	fmt.Println()
	printBanner("Demo complete — all stats measured from real replication traffic")
}

func warmupPhase(ctx context.Context, nodes []demoNode) {
	mapBackend := backendOf[metaengine.MapBackend]

	section("Phase 1: Warmup — 20 Map operations to build measurement history")

	for i := range 20 {
		// #nosec G404 -- demo picks a node uniformly at random; not a security decision
		n := nodes[rand.Intn(len(nodes))]
		must(mapBackend(n.engine).MapSet(ctx, "warmup", fmt.Sprintf("k%d", i), fmt.Sprintf("v%d", i)))
		progressBar("warmup", i+1, 20)
	}
}

func stormPhase(ctx context.Context, nodes []demoNode) {
	mapBackend := backendOf[metaengine.MapBackend]

	section("Phase 2: Concurrent storm — each node writes 50 entries simultaneously")

	var wg sync.WaitGroup
	for _, n := range nodes {
		wg.Add(1)
		go func(n demoNode) {
			defer wg.Done()

			for i := range 50 {
				key := fmt.Sprintf("%s-%d", n.name, i)
				_ = mapBackend(n.engine).MapSet(ctx, "storm", key, fmt.Sprintf("val-%d", i))
			}
		}(n)
	}

	wg.Wait()
	fmt.Println("  150 concurrent writes completed.")
}

func verifyPhase(ctx context.Context, nodes []demoNode) {
	mapBackend := backendOf[metaengine.MapBackend]

	section("Phase 3: Correctness — verify all nodes converged")

	allOK := true

	for i := range 20 {
		key := fmt.Sprintf("k%d", i)
		for _, n := range nodes {
			_, ok, _ := mapBackend(n.engine).MapGet(ctx, "warmup", key)
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
				_, ok, _ := mapBackend(n.engine).MapGet(ctx, "storm", key)
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
}

func counterPhase(ctx context.Context, nodes []demoNode) {
	counterBackend := backendOf[metaengine.CounterBackend]

	section("Phase 4: PN-Counter — 30 increments across 3 nodes")

	for range 10 {
		for _, n := range nodes {
			must(counterBackend(n.engine).CounterIncrement(ctx, "hits", metaengine.Delta{"total": 1}))
		}
	}

	fmt.Println("  Verifying PN-Counter sum (expect total=30):")

	for _, n := range nodes {
		counts, _ := counterBackend(n.engine).CounterGet(ctx, "hits")
		fmt.Printf("  %-8s: hits=%d\n", n.name, counts["total"])
	}
}

func structuresPhase(ctx context.Context, nodes []demoNode) {
	setBackend := backendOf[metaengine.SetBackend]
	logBackend := backendOf[metaengine.LogBackend]
	multimapBackend := backendOf[metaengine.MultimapBackend]

	section("Phase 5: OR-Set + Log + Multimap")
	must(setBackend(nodes[0].engine).SetAdd(ctx, "tags", "go"))
	must(setBackend(nodes[1].engine).SetAdd(ctx, "tags", "crdt"))
	must(setBackend(nodes[2].engine).SetAdd(ctx, "tags", "event-sourcing"))
	must(logBackend(nodes[0].engine).LogAppend(ctx, "audit", "alpha:deploy"))
	must(logBackend(nodes[1].engine).LogAppend(ctx, "audit", "beta:rollback"))
	must(multimapBackend(nodes[0].engine).MultiAdd(ctx, "teams", "platform", "alice"))
	must(multimapBackend(nodes[2].engine).MultiAdd(ctx, "teams", "platform", "carol"))

	fmt.Println("  OR-Set tags:")

	for _, n := range nodes {
		var found []string

		for _, t := range []string{"go", "crdt", "event-sourcing"} {
			ok, _ := setBackend(n.engine).SetContains(ctx, "tags", t)
			if ok {
				found = append(found, t)
			}
		}

		fmt.Printf("  %-8s: [%s]\n", n.name, strings.Join(found, ", "))
	}

	fmt.Println("  Log audit:")

	for _, n := range nodes {
		entries, _ := logBackend(n.engine).LogTail(ctx, "audit", 10)

		var strs []string
		for _, e := range entries {
			strs = append(strs, fmt.Sprintf("%v", e))
		}

		fmt.Printf("  %-8s: [%s]\n", n.name, strings.Join(strs, ", "))
	}

	fmt.Println("  Multimap teams/platform:")

	for _, n := range nodes {
		vals, _ := multimapBackend(n.engine).MultiGet(ctx, "teams", "platform")

		var strs []string
		for _, v := range vals {
			strs = append(strs, fmt.Sprintf("%v", v))
		}

		fmt.Printf("  %-8s: [%s]\n", n.name, strings.Join(strs, ", "))
	}
}

func profilePhase(nodes []demoNode) {
	section("Phase 6: EngineProfile — computed from REAL measurements")

	for _, n := range nodes {
		p := n.engine.Profile()
		fmt.Printf("  %-8s: replication=%s  lag=%s  rtt=%s\n",
			n.name, p.Replication, p.ReplicationLag, p.NetworkRTT)
	}

	fmt.Println("  ^ MEASURED, not hardcoded.")
	fmt.Println("    ReplicationLag = P99 convergence (last peer to apply).")
	fmt.Println("    NetworkRTT     = 2 × P50 delivery latency.")
}

func calmPhase(ctx context.Context, nodes []demoNode) {
	mapBackend := backendOf[metaengine.MapBackend]

	section("Phase 7: MapUpdate — NON-CRDT (CALM theorem boundary)")

	mu := backendOf[metaengine.MapUpdater](nodes[0].engine)
	must(mu.MapUpdate(ctx, "local", "counter", func(prev any) any {
		if n, ok := prev.(int64); ok {
			return n + 1
		}

		return int64(1)
	}))
	fmt.Println("  alpha did MapUpdate — stays local:")

	for _, n := range nodes {
		val, ok, _ := mapBackend(n.engine).MapGet(ctx, "local", "counter")
		if !ok {
			fmt.Printf("  %-8s: (missing)\n", n.name)
		} else {
			fmt.Printf("  %-8s: counter=%v\n", n.name, val)
		}
	}

	fmt.Println("  ^ Only alpha. Non-monotonic ops require coordination.")
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
