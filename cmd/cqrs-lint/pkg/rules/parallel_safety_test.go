package rules_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

// TestParallelDetectorSafety runs ALL registered detectors concurrently
// against a shared AnalysisContext to verify thread safety under -race.
//
// In production, the linter runs detectors in parallel via a worker pool.
// This test catches data races where two detectors concurrently read/write
// shared AnalysisContext state (sync.Map lineCache, GoFiles slice, etc.).
func TestParallelDetectorSafety(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
)

type UserCreated struct {
	Name  string
	Email string
}

var globalCache = map[string]string{}

func process(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	_ = tx.Rollback()

	var m sync.Mutex
	m.Lock()
	defer m.Unlock()

	data, _ := json.Marshal(UserCreated{Name: "Alice"})
	_ = data

	return fmt.Errorf("something failed")
}
`,
	})

	detectors := rules.RegisterAll(ctx)
	if len(detectors) == 0 {
		t.Fatal("expected at least one detector")
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(detectors))

	for _, det := range detectors {
		wg.Add(1)
		go func(d finding.Detector) {
			defer wg.Done()
			_, err := d.Detect(context.Background())
			if err != nil {
				errCh <- err
			}
		}(det)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("detector error under concurrent execution: %v", err)
	}
}
