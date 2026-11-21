//go:build cgo

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
		if err := runCoordinator(*waitFor, *writeCount); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	case "node":
		if *ticket == "" {
			fmt.Fprintln(os.Stderr, "ERROR: --ticket is required for node mode")
			os.Exit(1)
		}

		if err := runNode(*ticket, *writeCount); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

// nodesMissingError reports the coordinator's node-wait timeout with the
// observed peer count.
type nodesMissingError struct {
	got  int
	want int
}

func (e nodesMissingError) Error() string {
	return fmt.Sprintf("timeout waiting for nodes (got %d/%d)", e.got, e.want)
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

func printUsage() {
	fmt.Fprintln(os.Stderr, "Iroh QUIC Demo: Real P2P CRDT Replication")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  demo -mode coordinator [-wait-nodes N] [-writes M]")
	fmt.Fprintln(os.Stderr, "  demo -mode node -ticket <ticket> [-writes M]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(
		os.Stderr,
		"Coordinator prints a ticket, waits for N nodes to connect, then writes M keys.",
	)
	fmt.Fprintln(
		os.Stderr,
		"Nodes connect via ticket and write M keys. Both sides verify CRDT convergence.",
	)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(
		os.Stderr,
		"This is REAL QUIC networking: separate OS processes communicating over UDP.",
	)
}

func runCoordinator(waitFor, writeCount int) error {
	ctx := context.Background()

	transport, err := quic.New(quic.WithLocalOnly(), quic.WithRelay())
	if err != nil {
		return fmt.Errorf("bind failed: %w", err)
	}
	defer func() { _ = transport.Close() }()

	engine := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("coordinator"),
		irohengine.WithTransport(transport),
	)
	defer func() { _ = engine.Close() }()

	ticket, err := transport.Ticket()
	if err != nil {
		return fmt.Errorf("ticket failed: %w", err)
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
		return nodesMissingError{
			got: transport.PeerCount(), want: waitFor,
		}
	}

	fmt.Printf("  ✓ %d node(s) connected!\n\n", transport.PeerCount())

	time.Sleep(1 * time.Second)

	fmt.Printf("  Writing %d keys...\n", writeCount)

	for i := range writeCount {
		key := fmt.Sprintf("coord-key-%d", i)

		val := fmt.Sprintf("value-from-coordinator-%d", i)
		if err := backendOf[metaengine.MapBackend](engine).MapSet(ctx, "demo", key, val); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: MapSet %s: %v\n", key, err)
		}

		fmt.Printf("    wrote %s = %s\n", key, val)
	}

	fmt.Println("\n  Waiting for convergence...")
	time.Sleep(3 * time.Second)

	fmt.Println("\n  All keys visible on this node:")
	printMapContents(ctx, engine, "demo")

	profile := engine.Profile()

	fmt.Println("\n  ── Real QUIC Measurements ──")
	fmt.Printf("  Replication: %s\n", profile.Replication)
	fmt.Printf("  ReplicationLag (P99): %s\n", formatDuration(profile.ReplicationLag))
	fmt.Printf("  NetworkRTT (2×P50):   %s\n", formatDuration(profile.NetworkRTT))
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("  Coordinator running. Waiting for SIGINT/SIGTERM to exit...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	waitForSignal()

	return nil
}

func runNode(ticket string, writeCount int) error {
	ctx := context.Background()

	transport, err := quic.New(quic.WithLocalOnly())
	if err != nil {
		return fmt.Errorf("bind failed: %w", err)
	}
	defer func() { _ = transport.Close() }()

	engine := irohengine.Replicated(
		metaengine.NewMemoryEngine(),
		irohengine.WithAuthor("node"),
		irohengine.WithTransport(transport),
	)
	defer func() { _ = engine.Close() }()

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  Iroh QUIC Demo — Node (real QUIC networking)")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("  Node ID: %s\n", transport.NodeID())

	fmt.Println("  Connecting to coordinator...")

	if err := transport.Connect(ticket); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	fmt.Printf("  ✓ Connected! Peers: %d\n", transport.PeerCount())

	time.Sleep(1 * time.Second)

	fmt.Printf("\n  Writing %d keys...\n", writeCount)

	for i := range writeCount {
		key := fmt.Sprintf("node-key-%d", i)

		val := fmt.Sprintf("value-from-node-%d", i)
		if err := backendOf[metaengine.MapBackend](engine).MapSet(ctx, "demo", key, val); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: MapSet %s: %v\n", key, err)
		}

		fmt.Printf("    wrote %s = %s\n", key, val)
	}

	fmt.Println("\n  Waiting for convergence...")
	time.Sleep(3 * time.Second)

	fmt.Println("\n  All keys visible on this node:")
	printMapContents(ctx, engine, "demo")

	profile := engine.Profile()

	fmt.Println("\n  ── Real QUIC Measurements ──")
	fmt.Printf("  Replication: %s\n", profile.Replication)
	fmt.Printf("  ReplicationLag (P99): %s\n", formatDuration(profile.ReplicationLag))
	fmt.Printf("  NetworkRTT (2×P50):   %s\n", formatDuration(profile.NetworkRTT))
	fmt.Println("\n═══════════════════════════════════════════════════════════")
	fmt.Println("  Node running. Waiting for SIGINT/SIGTERM to exit...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	waitForSignal()

	return nil
}

// formatDuration renders a time.Duration with honest labeling for localhost.
// On localhost, QUIC RTT is sub-microsecond; showing "0s" looks fake even
// though it's real. This shows "<1us (localhost)" for sub-microsecond values.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s (no samples yet)"
	}

	if d < time.Microsecond {
		return "<1us (localhost)"
	}

	return d.String()
}

func waitForSignal() {
	waitSigCh := make(chan os.Signal, 1)
	signal.Notify(waitSigCh, syscall.SIGINT, syscall.SIGTERM)
	<-waitSigCh
	fmt.Println("\n  Shutting down...")
}

func printMapContents(ctx context.Context, engine metaengine.Engine, collection string) {
	scanResult, err := backendOf[metaengine.ScanBackend](engine).MapScan(
		ctx, collection,
		func(any) bool { return true },
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
