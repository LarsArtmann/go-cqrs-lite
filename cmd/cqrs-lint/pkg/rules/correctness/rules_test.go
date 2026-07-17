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

func writeNoCommit(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	_, _ = tx.Exec("INSERT INTO t VALUES (1)")
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

func writeWithCommit(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	_, _ = tx.Exec("INSERT INTO t VALUES (1)")
	return tx.Commit()
}
`,
	})
	findings := runDetector(t, correctness.NewC001Detector(ctx))
	assertRule(t, findings, "C001", 0)
}

// C001 must NOT flag closure-based transaction helpers where the tx variable
// escapes to a callback that contractually owns the commit. Suggesting
// `return tx.Commit()` here would double-commit (sql.ErrTxDone).
// Regression test for the DiscordSync false positive.
func TestC001_NoFindingWhenTxEscapesToCallback(t *testing.T) {
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
	assertRule(t, findings, "C001", 0)
}

// C001 must flag a function that uses the tx (tx.Exec) and never commits,
// even when there is no bare `return nil` success path. tx usage is a stronger
// bug signal than the return shape: the Exec ran, the tx is abandoned, the work
// is silently rolled back. Covers item f-7 in the DiscordSync feedback triage.
func TestC001_DetectsTxUsedWithoutBareReturnNil(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"tx.go": `package main

import (
	"context"
	"database/sql"
	"errors"
)

func writeAndReturnSentinel(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	_, _ = tx.Exec("INSERT INTO t VALUES (1)")
	return errors.New("sentinel")
}
`,
	})
	findings := runDetector(t, correctness.NewC001Detector(ctx))
	assertRule(t, findings, "C001", 1)
}

// C001 must NOT flag a function that begins a tx but neither uses it nor
// returns nil — there's no work to lose and no clean fix to suggest. Guards
// against txIsUsed widening the rule into noise on degenerate stubs.
func TestC001_NoFindingWhenTxUnusedAndNoReturnNil(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"tx.go": `package main

import (
	"context"
	"database/sql"
	"errors"
)

func stub(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	_ = tx
	return errors.New("not implemented")
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

// C008 must NOT flag a generic "value" field in an observability struct —
// the weak "value" signal needs a money-related struct/package name.
// Regression test for the DiscordSync false positives (rateTracker.value,
// SparklineSample.LagValue).
func TestC008_NoFindingForValueInObservabilityStruct(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"obs.go": `package main

type SparklineSample struct {
	LagValue float64
}

type rateTracker struct {
	value float64
}
`,
	})
	findings := runDetector(t, correctness.NewC008Detector(ctx))
	assertRule(t, findings, "C008", 0)
}

// C008 flags a weak "value" field when the enclosing struct name corroborates
// a monetary context.
func TestC008_FindingForValueInMonetaryStruct(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"model.go": `package main

type Wallet struct {
	Value float64
}
`,
	})
	findings := runDetector(t, correctness.NewC008Detector(ctx))
	assertRule(t, findings, "C008", 1)
}

// C008 downgrades strong-field findings (amount, balance) to Info/Low when the
// project has no monetary signal anywhere — no money-named package or struct.
// A lone "amount" in a non-payments codebase is suspicious but uncertain, so it
// shouldn't cost the same health-score points as a confirmed money field.
// Covers item f-8 in the DiscordSync feedback triage.
func TestC008_DowngradesStrongFieldWhenProjectNotMonetary(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"buffer.go": `package main

type Buffer struct {
	Amount float64
}
`,
	})
	findings := runDetector(t, correctness.NewC008Detector(ctx))
	if len(findings) != 1 {
		t.Fatalf("expected 1 C008 finding, got %d", len(findings))
	}

	if findings[0].Severity != finding.SeverityInfo {
		t.Errorf("expected Info severity in non-monetary project, got %s", findings[0].Severity)
	}

	if findings[0].Confidence != finding.ConfidenceLow {
		t.Errorf("expected Low confidence in non-monetary project, got %s", findings[0].Confidence)
	}
}

// C008 keeps strong-field findings at Warning/Medium when the project DOES have
// a monetary signal (here: a money-named struct in the same project). Guards
// against the project-downgrade suppressing real money fields in payments apps.
func TestC008_KeepsWarningInMonetaryProject(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"buffer.go": `package main

type Buffer struct {
	Amount float64
}

type Wallet struct {
	Balance float64
}
`,
	})
	findings := runDetector(t, correctness.NewC008Detector(ctx))
	// Wallet/Balance is unambiguously monetary → full severity. Buffer/Amount
	// is corroborated by the project signal too, so both stay Warning/Medium.
	count := 0
	for _, f := range findings {
		if string(f.Rule) != "C008" {
			continue
		}
		count++
		if f.Severity != finding.SeverityWarning {
			t.Errorf("expected Warning in monetary project, got %s", f.Severity)
		}
		if f.Confidence != finding.ConfidenceMedium {
			t.Errorf("expected Medium confidence in monetary project, got %s", f.Confidence)
		}
	}
	if count == 0 {
		t.Fatal("expected at least one C008 finding in a monetary project")
	}
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
