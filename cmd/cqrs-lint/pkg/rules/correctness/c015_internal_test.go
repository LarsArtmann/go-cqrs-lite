package correctness

import (
	"go/ast"
	"testing"
)

// c015_internal_test.go exercises the unexported suppression helpers that
// decide whether a discarded Close() error is an accepted cleanup pattern.
// These predicates gate the C015 detector, so regressions here either hide
// real leaks (over-suppression) or flood findings (under-suppression).

func TestIsInDefer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   []ast.Node
		want bool
	}{
		{"defer ancestor suppresses", []ast.Node{&ast.DeferStmt{}}, true},
		{"no defer does not suppress", []ast.Node{&ast.AssignStmt{}}, false},
		{"empty stack", nil, false},
		{
			"defer nested in other nodes",
			[]ast.Node{&ast.FuncDecl{}, &ast.DeferStmt{}, &ast.BlockStmt{}},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isInDefer(tc.in); got != tc.want {
				t.Fatalf("isInDefer(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsInCleanupCallback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   []ast.Node
		want bool
	}{
		{"func lit ancestor suppresses", []ast.Node{&ast.FuncLit{}}, true},
		{"no func lit does not suppress", []ast.Node{&ast.AssignStmt{}}, false},
		{"empty stack", nil, false},
		{
			"func lit nested in other nodes",
			[]ast.Node{&ast.CallExpr{}, &ast.FuncLit{}, &ast.BlockStmt{}},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isInCleanupCallback(tc.in); got != tc.want {
				t.Fatalf("isInCleanupCallback(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsInErrorCleanup(t *testing.T) {
	t.Parallel()

	retInIfBody := &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}}
	ifWithReturn := &ast.IfStmt{Body: retInIfBody}

	blockNoReturn := &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{}}}
	ifWithoutReturn := &ast.IfStmt{Body: blockNoReturn}

	blockWithReturnInFor := &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}}
	forWithReturn := &ast.ForStmt{Body: blockWithReturnInFor}

	cases := []struct {
		name string
		in   []ast.Node
		want bool
	}{
		{
			name: "if body with return suppresses",
			in:   []ast.Node{ifWithReturn, retInIfBody},
			want: true,
		},
		{
			name: "if body without return does not suppress",
			in:   []ast.Node{ifWithoutReturn, blockNoReturn},
			want: false,
		},
		{
			name: "for body with return does not suppress (parent not If)",
			in:   []ast.Node{forWithReturn, blockWithReturnInFor},
			want: false,
		},
		{name: "empty stack", in: nil, want: false},
		{name: "no enclosing block", in: []ast.Node{&ast.IfStmt{}}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isInErrorCleanup(tc.in); got != tc.want {
				t.Fatalf("isInErrorCleanup(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsSuppressedClose(t *testing.T) {
	t.Parallel()

	deferAncestors := []ast.Node{&ast.DeferStmt{}}
	callbackAncestors := []ast.Node{&ast.FuncLit{}}

	errCleanupBody := &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}}
	errCleanupAncestors := []ast.Node{&ast.IfStmt{Body: errCleanupBody}, errCleanupBody}

	plainAncestors := []ast.Node{&ast.AssignStmt{}, &ast.BlockStmt{}}

	cases := []struct {
		name string
		in   []ast.Node
		want bool
	}{
		{"defer suppresses", deferAncestors, true},
		{"cleanup callback suppresses", callbackAncestors, true},
		{"error cleanup suppresses", errCleanupAncestors, true},
		{"plain statement does not suppress", plainAncestors, false},
		{"empty does not suppress", nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isSuppressedClose(tc.in); got != tc.want {
				t.Fatalf("isSuppressedClose(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
