package correctness_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
)

// When C001 fires on the tx-used-without-bare-return-nil path, it must be
// suggest-only: no BeforeCode/AfterCode. A direct auto-fix with
// BeforeCode("return nil") would match an UNRELATED "return nil" elsewhere in
// the file and silently corrupt it. Regression test for the wrong-auto-fix bug.
func TestC001_TxUsedWithoutReturnNilIsSuggestOnly(t *testing.T) {
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
	findings, err := correctness.NewC001Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.BeforeCode != "" {
		t.Errorf("txUsed-without-returnNil finding must be suggest-only, but BeforeCode=%q "+
			"(would let the fixer rewrite an unrelated return nil)", f.BeforeCode)
	}
	if f.AfterCode != "" {
		t.Errorf(
			"txUsed-without-returnNil finding must be suggest-only, but AfterCode=%q",
			f.AfterCode,
		)
	}
}

// Read-only bbolt Begin(false) transactions must not trigger C001 — read-only
// tx cannot be committed and the rollback is handled elsewhere (iterator Close).
func TestC001_ReadOnlyBeginFalse_NoFinding(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"iter.go": `package main

type bboltDB interface {
	Begin(writable bool) (*bboltTx, error)
}

type bboltTx struct{}

func (tx *bboltTx) Bucket(name []byte) *bboltBucket { return nil }
func (tx *bboltTx) Rollback() error                 { return nil }

type bboltBucket struct{}

type iterator struct {
	tx     *bboltTx
	prefix []byte
}

func newIterator(db bboltDB, prefix []byte) (*iterator, error) {
	tx, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	bucket := tx.Bucket([]byte("events"))
	_ = bucket
	return &iterator{tx: tx, prefix: prefix}, nil
}
`,
	})
	findings, err := correctness.NewC001Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for read-only Begin(false), got %d: %v", len(findings), findings)
	}
}

// tx stored in a composite-literal struct (e.g. &iter{tx: tx}) must be treated
// as an escape — the struct owns the tx lifecycle, so C001 must not fire.
func TestC001_CompositeLiteralEscape_NoFinding(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"iter.go": `package main

import (
	"context"
	"database/sql"
)

type iterator struct {
	tx   *sql.Tx
	rows *sql.Rows
}

func newIterator(ctx context.Context, db *sql.DB) (*iterator, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT * FROM events")
	if err != nil {
		return nil, err
	}
	return &iterator{tx: tx, rows: rows}, nil
}
`,
	})
	findings, err := correctness.NewC001Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for composite-literal escape, got %d: %v", len(findings), findings)
	}
}

// The returnsNil path keeps the direct auto-fix (BeforeCode="return nil").
func TestC001_ReturnsNilPathKeepsDirectFix(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"tx.go": `package main

import (
	"context"
	"database/sql"
)

func writeAndReturnNil(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	_, _ = tx.Exec("INSERT INTO t VALUES (1)")
	return nil
}
`,
	})
	findings, err := correctness.NewC001Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.BeforeCode != "return nil" {
		t.Errorf("returnsNil finding should keep BeforeCode=%q, got %q", "return nil", f.BeforeCode)
	}
}
