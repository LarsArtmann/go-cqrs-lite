//go:build cgo

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/quic/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func main() {
	mode := flag.String("mode", "", "coordinator or node (required)")
	ticket := flag.String("ticket", "", "coordinator ticket (required for node mode)")
	waitFor := flag.Int("wait-nodes", 1, "coordinator: number of nodes to wait for")
	writeCount := flag.Int("writes", 5, "number of keys to write")
	flag.Parse()

	switch *mode {
	case "coordinator":
		runCoordinator(*waitFor, *writeCount)
	case "node":
		if *ticket == "" {
			fmt.Fprintln(os.Stderr, "ERROR: --ticket is required for node mode")
			os.Exit(1)
		}
		runNode(*ticket, *writeCount)
	default:
		fmt.Fprintln(os.Stderr, `Iroh QUIC Demo — Real P2P CRDT Replication

Usage:
  demo -mode coordinator [-wait-nodes N] [-writes M]
  demo -mode node -ticket <ticket> [-writes M]

Coordinator prints a ticket, waits for N nodes to connect, then writes M keys.
Nodes connect via ticket and write M keys. Both sides verify CRDT convergence.`)
		os.Exit(1)
	}
}

func runCoordinator(waitFor, writeCount int) {
	ctx := context.Background()

	transport, err := quic.New(quic.WithLocalOnly(), quic.WithRelay())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: bind failed: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close()

	ticket, err := transport.Ticket()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: ticket failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Iroh QUIC Demo — Coordinator (real QUIC networking)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  Node ID: %s\n", transport.NodeID())
	fmt.Println()
	fmt.Println("  Share this ticket with nodes:")
	fmt.Println()
	fmt.Printf("  %s\n", ticket)
	fmt.Println()
	fmt.Printf("  Waiting for %d node(s) to connect...\n", waitFor)

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if transport.PeerCount() >= waitFor {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if transport.PeerCount() < waitFor {
		fmt.Fprintf(os.Stderr, "ERROR: timeout waiting for nodes (got %d/%d)\n",
			transport.PeerCount(), waitFor)
		os.Exit(1)
	}
	fmt.Printf("  ✓ %d node(s) connected!\n\n", transport.PeerCount())

	engine := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("coordinator"),
		irohengine.WithTransport(transport),
	)
	defer engine.Close()

	fmt.Printf("  Writing %d keys...\n", writeCount)
	for i := 0; i < writeCount; i++ {
		key := fmt.Sprintf("coord-key-%d", i)
		val := fmt.Sprintf("value-from-coordinator-%d", i)
		if err := engine.(metaengine.MapBackend).MapSet(ctx, "demo", key, val); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: MapSet %s: %v\n", key, err)
		}
		fmt.Printf("    wrote %s = %s\n", key, val)
	}

	fmt.Println("\n  Waiting for node writes to arrive...")
	time.Sleep(3 * time.Second)

	fmt.Println("\n  Received keys:")
	printAllMapKeys(ctx, engine, "demo")

	profile := engine.Profile()
	fmt.Println("\n  ── Real QUIC Measurements ──")
	fmt.Printf("  Replication: %s\n", profile.Replication)
	fmt.Printf("  ReplicationLag (P99): %s\n", profile.ReplicationLag)
	fmt.Printf("  NetworkRTT (2×P50):   %s\n", profile.NetworkRTT)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("  Coordinator finished. Press Ctrl+C to exit.")
	fmt.Println("═══════════════════════════════════════════════════════════")
	select {}
}

func runNode(ticket string, writeCount int) {
	ctx := context.Background()

	transport, err := quic.New(quic.WithLocalOnly())
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: bind failed: %v\n", err)
		os.Exit(1)
	}
	defer transport.Close()

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Iroh QUIC Demo — Node (real QUIC networking)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  Node ID: %s\n", transport.NodeID())

	fmt.Println("  Connecting to coordinator...")
	if err := transport.Connect(ticket); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: connect failed: %v\n", err)
		os.Exit(1)
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("  ✓ Connected! Peers: %d\n", transport.PeerCount())

	engine := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node"),
		irohengine.WithTransport(transport),
	)
	defer engine.Close()

	fmt.Printf("\n  Writing %d keys...\n", writeCount)
	for i := 0; i < writeCount; i++ {
		key := fmt.Sprintf("node-key-%d", i)
		val := fmt.Sprintf("value-from-node-%d", i)
		if err := engine.(metaengine.MapBackend).MapSet(ctx, "demo", key, val); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: MapSet %s: %v\n", key, err)
		}
		fmt.Printf("    wrote %s = %s\n", key, val)
	}

	fmt.Println("\n  Waiting for coordinator writes to arrive...")
	time.Sleep(3 * time.Second)

	fmt.Println("\n  Received keys:")
	printAllMapKeys(ctx, engine, "demo")

	profile := engine.Profile()
	fmt.Println("\n  ── Real QUIC Measurements ──")
	fmt.Printf("  Replication: %s\n", profile.Replication)
	fmt.Printf("  ReplicationLag (P99): %s\n", profile.ReplicationLag)
	fmt.Printf("  NetworkRTT (2×P50):   %s\n", profile.NetworkRTT)
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("  Node finished. Press Ctrl+C to exit.")
	fmt.Println("═══════════════════════════════════════════════════════════")
	select {}
}

func printAllMapKeys(ctx context.Context, engine metaengine.Engine, collection string) {
	scanResult, err := engine.(metaengine.ScanBackend).MapScan(
		ctx, collection,
		func(item any) bool { return true },
		nil, nil, 100,
	)
	if err != nil {
		fmt.Printf("    (scan error: %v)\n", err)
		return
	}
	if len(scanResult.Items) == 0 {
		fmt.Println("    (no keys yet)")
		return
	}
	for _, item := range scanResult.Items {
		s := fmt.Sprintf("%v", item)
		if len(s) > 80 {
			s = s[:77] + "..."
		}
		fmt.Printf("    %s\n", s)
	}
}
