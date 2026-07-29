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
		t.Errorf("txUsed-without-returnNil finding must be suggest-only, but AfterCode=%q", f.AfterCode)
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
