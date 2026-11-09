package metaengine_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adttest_delegation_meta_test.go pins the dependency direction agreed after
// the 2026-08-16 review: adttest DELEGATES to metaengine.CapabilityAudit; the
// audit rules (the declared-vs-implemented table, the verdict strings, the
// violation logic) live in the root package so Doctor and EXPLAIN render the
// same findings. If adttest re-grows its own audit logic, Doctor and the test
// gate silently diverge — this test fails first.

func TestAdttestStaysDelegatingOnly(t *testing.T) {
	t.Parallel()

	dir := "adttest"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read adttest dir: %v", err)
	}

	// Verdict strings are owned by metaengine/capability_audit.go. If adttest
	// needs one of these literals, it has re-implemented verdict logic instead
	// of delegating. (Checked against raw source minus comments below, so doc
	// prose mentioning the rules does not trip it.)
	forbidden := []string{
		"OVER-DECLARED",
		"UNDER-DECLARED",
		"conformance violation",
	}

	sawAuditCall := false

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		// codeOnly is src with all comment spans removed, so doc prose
		// mentioning the rules does not trip the forbidden-literal scan.
		codeOnly := string(src)
		for _, cg := range file.Comments {
			for _, c := range cg.List {
				codeOnly = strings.ReplaceAll(codeOnly, c.Text, "")
			}
		}

		for _, content := range forbidden {
			if strings.Contains(codeOnly, content) {
				t.Errorf(
					"%s contains %q — audit verdict logic belongs in metaengine, "+
						"adttest must delegate via CapabilityAudit",
					name, content)
			}
		}

		// Every CapabilityAudit invocation must go through the metaengine
		// package (delegation), not a local re-implementation.
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "CapabilityAudit" {
				return true
			}

			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "metaengine" {
				sawAuditCall = true
			} else {
				t.Errorf("%s calls CapabilityAudit on non-metaengine receiver %v", name, sel.X)
			}

			return true
		})
	}

	if !sawAuditCall {
		t.Error("adttest no longer delegates to metaengine.CapabilityAudit — " +
			"the conformance gate and Doctor/EXPLAIN would drift apart")
	}
}
