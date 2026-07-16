package correctness_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

func runDetector(t *testing.T, det finding.Detector) []finding.Finding {
	t.Helper()
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("detector %s: %v", det.Name(), err)
	}

	return findings
}

func assertRule(t *testing.T, findings []finding.Finding, ruleID string, wantCount int) {
	t.Helper()
	count := 0
	for _, f := range findings {
		if string(f.Rule) == ruleID {
			count++
		}
	}
	if count != wantCount {
		t.Errorf("rule %s: got %d findings, want %d", ruleID, count, wantCount)
	}
}

// --- C006: Manual Version Arithmetic ---

func TestC006_DetectsManualArithmetic(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decide.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func decide(version event.Version) {
	_ = event.Version(version.Int() + 1)
}
`,
	})
	findings := runDetector(t, correctness.NewC006Detector(ctx))
	assertRule(t, findings, "C006", 1)
}

func TestC006_NoFindingForIncrement(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decide.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func decide(version event.Version) {
	_ = version.Increment()
}
`,
	})
	findings := runDetector(t, correctness.NewC006Detector(ctx))
	assertRule(t, findings, "C006", 0)
}

// --- C003: Silent Unknown Event Fold ---

func TestC003_DetectsSilentDefault(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	next := s
	switch evt.Type() {
	case "incremented":
		next.Count++
	default:
		return s, nil
	}
	return next, nil
}
`,
	})
	findings := runDetector(t, correctness.NewC003Detector(ctx))
	assertRule(t, findings, "C003", 1)
}

func TestC003_NoFindingForErrorInDefault(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"fold.go": `package main

import (
	"fmt"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

type State struct{ Count int }

func fold(s State, evt event.Event) (State, error) {
	next := s
	switch evt.Type() {
	case "incremented":
		next.Count++
	default:
		return s, fmt.Errorf("unknown event type: %s", evt.Type())
	}
	return next, nil
}
`,
	})
	findings := runDetector(t, correctness.NewC003Detector(ctx))
	assertRule(t, findings, "C003", 0)
}

// --- C001: Missing Transaction Commit ---

func TestC001_DetectsMissingCommit(t *testing.T) {
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
	return nil
}
`,
	})
	findings := runDetector(t, correctness.NewC001Detector(ctx))
	assertRule(t, findings, "C001", 1)
}

func TestC001_NoFindingForProperCommit(t *testing.T) {
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
	findings := runDetector(t, correctness.NewC001Detector(ctx))
	assertRule(t, findings, "C001", 0)
}

// --- C005: Raw json.Unmarshal on Event Payload ---

func TestC005_DetectsRawJSONUnmarshal(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "encoding/json"

type Payload struct{ Name string }

func handle(payloadBytes []byte) {
	var p Payload
	json.Unmarshal(payloadBytes, &p)
}
`,
	})
	findings := runDetector(t, correctness.NewC005Detector(ctx))
	// Note: without type info, this will match any json.Unmarshal where first arg is a call.
	// In the test fixture, payloadBytes is not a .Payload() call, so it should be 0.
	// We need a fixture with evt.Payload().
	assertRule(t, findings, "C005", 0)
}

func TestC005_DetectsPayloadCall(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

import "encoding/json"

type Payload struct{ Name string }

type evt struct{}

func (e *evt) Payload() []byte { return nil }

func handle(e *evt) {
	var p Payload
	json.Unmarshal(e.Payload(), &p)
}
`,
	})
	findings := runDetector(t, correctness.NewC005Detector(ctx))
	assertRule(t, findings, "C005", 1)
}

// --- C009: panic in production ---

func TestC009_DetectsPanic(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": `package main

func doSomething() {
	panic("should not be here")
}
`,
	})
	findings := runDetector(t, correctness.NewC009Detector(ctx))
	assertRule(t, findings, "C009", 1)
}

func TestC009_NoFindingInTestFiles(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler_test.go": `package main

func doSomething() {
	panic("test panic")
}
`,
	})
	findings := runDetector(t, correctness.NewC009Detector(ctx))
	assertRule(t, findings, "C009", 0)
}

// --- C009: panic inside must* functions is NOT flagged (established Go convention)

func TestC009_NoFindingInMustFunc(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"commands.go": `package main

func mustCommand(cmdType string, aggID string) *Command {
	cmd, err := newCommand(cmdType, aggID)
	if err != nil {
		panic(err)
	}
	return cmd
}
`,
	})
	findings := runDetector(t, correctness.NewC009Detector(ctx))
	assertRule(t, findings, "C009", 0)
}

// --- C008: float64 for money ---

func TestC008_DetectsFloat64ForMoney(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type Order struct {
	TotalAmount float64
}
`,
	})
	findings := runDetector(t, correctness.NewC008Detector(ctx))
	assertRule(t, findings, "C008", 1)
}

func TestC008_NoFindingForNonMoneyFloat(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type Config struct {
	Ratio float64
}
`,
	})
	findings := runDetector(t, correctness.NewC008Detector(ctx))
	assertRule(t, findings, "C008", 0)
}

// --- C002: Broken Command ID ---

func TestC002_DetectsZeroCommandID(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"command.go": `package main

type CommandID struct{ id string }

type CreateCmd struct{}

func (c CreateCmd) ID() CommandID {
	return CommandID{}
}
`,
	})
	findings := runDetector(t, correctness.NewC002Detector(ctx))
	assertRule(t, findings, "C002", 1)
}
