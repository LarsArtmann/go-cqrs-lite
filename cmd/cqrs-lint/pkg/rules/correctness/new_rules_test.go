package correctness_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

// --- C004: Checkpoint before async complete ---

func TestC004_NoFindingWithoutProjections(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main
`,
	})
	findings := runDetector(t, correctness.NewC004Detector(ctx))
	assertRule(t, findings, "C004", 0)
}

// --- C007: time.Now in decider ---

func TestC007_DetectsTimeNowInDecider(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decide.go": `package main

func decide(state int, cmd Command) (int, error) {
	now := time.Now()
	return state + int(now.Unix()), nil
}
`,
	})
	findings := runDetector(t, correctness.NewC007Detector(ctx))
	assertRule(t, findings, "C007", 1)
}

func TestC007_NoFindingOutsideDecider(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func handleRequest() {
	now := time.Now()
	_ = now
}
`,
	})
	findings := runDetector(t, correctness.NewC007Detector(ctx))
	assertRule(t, findings, "C007", 0)
}

func TestC004_DetectsAsyncProjection(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

import "projection"

func setup() {
	p := projection.NewProjection("users", func(evt Event) error {
		go processAsync(evt)
		return nil
	}, nil)
	_ = p
}
`,
	})
	findings := runDetector(t, correctness.NewC004Detector(ctx))
	assertRule(t, findings, "C004", 1)
}

func TestC004_NoFindingOnSyncProjection(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

import "projection"

func setup() {
	p := projection.NewProjection("users", func(evt Event) error {
		processSync(evt)
		return nil
	}, nil)
	_ = p
}
`,
	})
	findings := runDetector(t, correctness.NewC004Detector(ctx))
	assertRule(t, findings, "C004", 0)
}

// --- C011: Nondeterministic decider ---

func TestC011_DetectsRandInDecider(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decide.go": `package main

func decide(state int) (int, error) {
	x := rand.Intn(100)
	return state + x, nil
}
`,
	})
	findings := runDetector(t, correctness.NewC011Detector(ctx))
	assertRule(t, findings, "C011", 1)
}

func TestC011_NoFindingInTestFiles(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decide_test.go": `package main

func testDecide() {
	x := rand.Intn(100)
	_ = x
}
`,
	})
	findings := runDetector(t, correctness.NewC011Detector(ctx))
	assertRule(t, findings, "C011", 0)
}

// --- C010: Swallowed error in fold ---

func TestC010_DetectsSwallowedError(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

type State struct{ Count int }
type Payload struct{ N int }

func fold(s State, evt event.Event) (State, error) {
	next := s
	var p Payload
	_ = json.Unmarshal(evt.Payload(), &p)
	next.Count += p.N
	return next, nil
}
`,
	})
	findings := runDetector(t, correctness.NewC010Detector(ctx))
	assertRule(t, findings, "C010", 1)
}

func TestC010_NoCrashOnEmptyInput(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"empty.go": `package main
`,
	})
	findings := runDetector(t, correctness.NewC010Detector(ctx))
	assertRule(t, findings, "C010", 0)
}

// --- C012: Missing error return in withTx ---

func TestC012_DetectsMissingErrorReturn(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"tx.go": `package main

import (
	"context"
	"database/sql"
)

func withTx(ctx context.Context, db *sql.DB, body func(*sql.Tx) error) error {
	tx, _ := db.BeginTx(ctx, nil)
	_ = body(tx)
	return tx.Commit()
}
`,
	})
	findings := runDetector(t, correctness.NewC012Detector(ctx))
	assertRule(t, findings, "C012", 1)
}

func TestC012_NoFindingForProperErrorReturn(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"tx.go": `package main

import (
	"context"
	"database/sql"
)

func withTx(ctx context.Context, db *sql.DB, body func(*sql.Tx) error) error {
	tx, _ := db.BeginTx(ctx, nil)
	if err := body(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
`,
	})
	findings := runDetector(t, correctness.NewC012Detector(ctx))
	assertRule(t, findings, "C012", 0)
}

// --- C002: Broken command ID ---

func TestC002_DetectsZeroID(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"cmd.go": `package main

type CommandID struct{ Value string }

type MyCmd struct{}

func (c *MyCmd) ID() CommandID {
	return CommandID{}
}
`,
	})
	findings := runDetector(t, correctness.NewC002Detector(ctx))
	assertRule(t, findings, "C002", 1)
}

func TestC002_NoCrashOnEmptyInput(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"empty.go": `package main
`,
	})
	det := correctness.NewC002Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("C002 on empty: %v", err)
	}
	_ = findings
}
