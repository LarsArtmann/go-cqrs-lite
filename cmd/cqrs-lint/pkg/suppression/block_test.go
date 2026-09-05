package suppression

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-finding"
)

func TestBlockSuppression_AllRules(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")

	content := `package main

//cqrs-lint:ignore-start
func badFunc() {
}
//cqrs-lint:ignore-end

func goodFunc() {
}
`

	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := newLineCache()
	f := finding.Finding{
		Rule: "A001",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 5, // inside the block
		},
	}

	if !checkBlockSuppressionInFile(cache, f) {
		t.Error("expected finding inside ignore-start/ignore-end block to be suppressed")
	}
}

func TestBlockSuppression_OutsideBlock(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")

	content := `package main

//cqrs-lint:ignore-start
func badFunc() {
}
//cqrs-lint:ignore-end

func goodFunc() {
}
`

	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := newLineCache()
	f := finding.Finding{
		Rule: "A001",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 9, // outside the block (after ignore-end)
		},
	}

	if checkBlockSuppressionInFile(cache, f) {
		t.Error("expected finding outside ignore-start/ignore-end block to NOT be suppressed")
	}
}

func TestBlockSuppression_SpecificRules(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")

	content := `package main

//cqrs-lint:ignore-start(A001,A002)
func badFunc() {
}
//cqrs-lint:ignore-end

func goodFunc() {
}
`

	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := newLineCache()

	// A001 should be suppressed
	f1 := finding.Finding{
		Rule: "A001",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 5,
		},
	}

	if !checkBlockSuppressionInFile(cache, f1) {
		t.Error("expected A001 to be suppressed inside ignore-start(A001,A002) block")
	}

	// A003 should NOT be suppressed
	f2 := finding.Finding{
		Rule: "A003",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 5,
		},
	}

	if checkBlockSuppressionInFile(cache, f2) {
		t.Error("expected A003 to NOT be suppressed inside ignore-start(A001,A002) block")
	}
}

func TestBlockSuppression_NestedStartEnd(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")

	content := `package main

//cqrs-lint:ignore-start
func f1() {
}

//cqrs-lint:ignore-end
func f2() {
}

func f3() {
}
`

	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := newLineCache()

	// Line 4 (inside block) → suppressed
	f1 := finding.Finding{
		Rule: "A001",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 4,
		},
	}

	if !checkBlockSuppressionInFile(cache, f1) {
		t.Error("expected line 4 to be suppressed")
	}

	// Line 8 (after ignore-end) → NOT suppressed
	f2 := finding.Finding{
		Rule: "A001",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 8,
		},
	}

	if checkBlockSuppressionInFile(cache, f2) {
		t.Error("expected line 8 to NOT be suppressed (after ignore-end)")
	}
}

func TestParseBlockStart(t *testing.T) {
	tests := []struct {
		input    string
		wantNil  bool
		wantLen  int
		wantFail bool // malformed input — suppresses nothing (fail closed)
	}{
		{"//cqrs-lint:ignore-start", true, 0, false},
		{"//cqrs-lint:ignore-start(A001)", false, 1, false},
		{"//cqrs-lint:ignore-start(A001,A002)", false, 2, false},
		{"//cqrs-lint:ignore-start( A001 , A002 )", false, 2, false},
		// Malformed directives must suppress NOTHING (fail closed). The old
		// behavior treated "()" and unclosed parens as "suppress all", so a
		// typo silently disabled every rule inside the block.
		{"//cqrs-lint:ignore-start()", true, 0, true},
		{"//cqrs-lint:ignore-start(   )", true, 0, true},
		{"//cqrs-lint:ignore-start(A001", true, 0, true},
		{"//cqrs-lint:ignore-start(A001,A002", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, valid := parseBlockStart(tt.input)
			if tt.wantFail {
				if valid {
					t.Errorf("expected malformed directive to fail closed, got valid")
				}

				return
			}

			if !valid {
				t.Errorf("expected valid directive, got fail-closed")
			}

			if tt.wantNil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}

			if !tt.wantNil && result == nil {
				t.Errorf("expected non-nil result")
			}

			if !tt.wantNil && len(result) != tt.wantLen {
				t.Errorf("expected %d rules, got %d", tt.wantLen, len(result))
			}
		})
	}
}

func TestBlockSuppression_MalformedDirectiveFailsClosed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")

	content := `package main

//cqrs-lint:ignore-start(A001
func badFunc() {
}
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := newLineCache()
	f := finding.Finding{
		Rule: "A003",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 4, // inside the malformed block
		},
	}

	if checkBlockSuppressionInFile(cache, f) {
		t.Error("malformed ignore-start(A001 must suppress NOTHING, not all rules")
	}
}

func TestBlockSuppression_EmptyIDListFailsClosed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")

	content := `package main

//cqrs-lint:ignore-start()
func badFunc() {
}
`
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := newLineCache()
	f := finding.Finding{
		Rule: "A001",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 4,
		},
	}

	if checkBlockSuppressionInFile(cache, f) {
		t.Error("ignore-start() with empty ID list must suppress NOTHING")
	}
}

func TestBlockSuppression_FindingLineBeyondEOFDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")

	content := "package main\n\nfunc f() {}\n" // 3 lines
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := newLineCache()
	f := finding.Finding{
		Rule: "A001",
		Position: finding.Position{
			File: finding.FilePath(file),
			Line: 8, // beyond EOF — must not panic
		},
	}

	if checkSuppressionInFile(cache, f) {
		t.Error("expected no suppression for finding beyond EOF")
	}
}
