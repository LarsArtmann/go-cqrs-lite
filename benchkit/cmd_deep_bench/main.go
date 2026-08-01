package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     DEEP BENCHMARK: memory vs sqlite vs pebble               ║")
	fmt.Println("║     Profile: Small (1K streams × 10 events = 10K events)    ║")
	fmt.Println("║     Repeat: 5 (for statistical reliability / CoV)           ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// ── Phase 1: Full comparison with repeat ──
	config := benchkit.Config{
		Profile:     benchkit.ProfileSmall,
		PayloadSize: 256,
		Repeat:      5,
	}

	fmt.Println("Phase 1: Multi-backend comparison (Repeat=5 for CoV)")
	fmt.Println("─────────────────────────────────────────────────────────")

	results, err := benchkit.Compare(ctx, config, map[string]benchkit.Factory{
		"memory": func() (*stack.Bundle, error) {
			return memory.New()
		},
		"sqlite": func() (*stack.Bundle, error) {
			dir, _ := os.MkdirTemp("", "bench-sqlite-*")
			return sqlite.New(filepath.Join(dir, "bench.db"))
		},
		"pebble": func() (*stack.Bundle, error) {
			dir, _ := os.MkdirTemp("", "bench-pebble-*")
			bundle, err := pebble.New(dir)
			if err != nil {
				return nil, err
			}
			return bundle.Bundle, nil
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compare failed: %v\n", err)
		os.Exit(1)
	}

	benchkit.PrintComparison(os.Stdout, results)

	// ── Phase 2: Detailed single reports ──
	fmt.Println("\nPhase 2: Detailed reports per backend")
	fmt.Println("─────────────────────────────────────────────────────────")

	for _, name := range []string{"memory", "sqlite", "pebble"} {
		r := results[name]
		if r == nil || r.Error != "" {
			continue
		}

		fmt.Printf("\n%s%s%s\n", "\x1b[1m", name, "\x1b[0m")
		benchkit.PrintReport(os.Stdout, r)
	}

	// ── Phase 3: Markdown table ──
	fmt.Println("\nPhase 3: Markdown table (for docs/PRs)")
	fmt.Println("─────────────────────────────────────────────────────────")
	benchkit.PrintMarkdown(os.Stdout, results)

	// ── Phase 4: Reliability assessment ──
	fmt.Println("\nPhase 4: Statistical reliability assessment")
	fmt.Println("─────────────────────────────────────────────────────────")

	for _, name := range []string{"memory", "sqlite", "pebble"} {
		r := results[name]
		if r == nil || r.RepeatCount < 2 {
			continue
		}

		verdict := "✓ TRUSTWORTHY"
		if !r.RepeatIsReliable {
			verdict = "⚠ TOO NOISY — increase Repeat"
		}

		fmt.Printf("  %-8s CoV=%4.1f%%  mean=%.0f/s  stddev=%.0f/s  %s\n",
			name,
			r.RepeatCoV*100,
			r.RepeatMean,
			r.RepeatStdDev,
			verdict,
		)
	}

	fmt.Println("\nDone.")
}
