package correctness_test

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/fix"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/correctness"
)

const c001BareReturnSource = `package main

import (
	"context"
	"database/sql"
)

func write(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	_, _ = tx.Exec("INSERT INTO t VALUES (1)")
	return nil
}
`

// sourceLine returns the 1-based line n of src ("" when out of range).
func sourceLine(src string, n int) string {
	lines := strings.Split(src, "\n")
	if n < 1 || n > len(lines) {
		return ""
	}

	return lines[n-1]
}

// TestC001_DirectFixIsAnchoredAtReturnStmt runs the real fix provider over
// the C001 finding: the finding must sit on the `return nil` statement so
// the provider's position match finds its BeforeCode and emits the
// `return tx.Commit()` edit. Anchored at the func declaration, the Direct
// fix was advertised but never applied (a silent no-op).
func TestC001_DirectFixIsAnchoredAtReturnStmt(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{"tx.go": c001BareReturnSource})
	findings, err := correctness.NewC001Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if line := sourceLine(c001BareReturnSource, f.Position.Line); !strings.Contains(line, "return nil") {
		t.Fatalf("finding must be anchored on the `return nil` line, got line %d: %q",
			f.Position.Line, line)
	}

	edits, err := fix.NewCQRSFixProvider().Edits([]byte(c001BareReturnSource), f)
	if err != nil {
		t.Fatalf("edits: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("fix provider must produce exactly one edit for a Direct finding, got %d", len(edits))
	}

	if string(edits[0].Replacement) != "return tx.Commit()" {
		t.Errorf("replacement = %q, want %q", edits[0].Replacement, "return tx.Commit()")
	}
}

// TestC001_MultiResultReturnNilIsSuggestOnly: `return nil, err` must stay a
// finding (the tx is still never committed) but suggest-only — a Direct fix
// would rewrite it into the invalid `return tx.Commit(), err`.
func TestC001_MultiResultReturnNilIsSuggestOnly(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"tx.go": `package main

import (
	"context"
	"database/sql"
)

func write(ctx context.Context, db *sql.DB) (int, error) {
	tx, _ := db.BeginTx(ctx, nil)
	_, _ = tx.Exec("INSERT INTO t VALUES (1)")
	return 0, nil
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

	if findings[0].BeforeCode != "" {
		t.Errorf("multi-result return must be suggest-only, BeforeCode=%q", findings[0].BeforeCode)
	}
}
