package sql_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func TestValidateIdentifier(t *testing.T) {
	t.Parallel()

	valid := []string{
		"a", "name", "user_id", "_private", "Column1", "a_very_long_column_name",
	}
	for _, id := range valid {
		if !sqlpkg.ValidateIdentifier(id) {
			t.Errorf("ValidateIdentifier(%q) = false, want true", id)
		}
	}

	hostile := []string{
		"", "1starts_with_digit", "has space", "has;semicolon", "has--comment",
		"quoted\"ident", "dotted.path", "col)--", "col UNION SELECT x",
		"col/*c*/", "'quoted'", "back`tick`", "name ASC, email", "*",
	}
	for _, id := range hostile {
		if sqlpkg.ValidateIdentifier(id) {
			t.Errorf("ValidateIdentifier(%q) = true, want false", id)
		}
	}
}

func TestValidateOperator(t *testing.T) {
	t.Parallel()

	supported := []kv.Operator{
		kv.OpEq, kv.OpNeq, kv.OpLt, kv.OpLte, kv.OpGt, kv.OpGte,
		kv.OpLike, kv.OpIn, kv.OpIsNull, kv.OpIsNotNull,
	}
	for _, op := range supported {
		if !sqlpkg.ValidateOperator(op) {
			t.Errorf("ValidateOperator(%q) = false, want true", op)
		}
	}

	hostile := []kv.Operator{
		"= 1 OR 1=1",
		"IS NULL; DROP TABLE x",
		"IN (SELECT password FROM users)",
		"",
		"LIKE '%' --",
	}
	for _, op := range hostile {
		if sqlpkg.ValidateOperator(op) {
			t.Errorf("ValidateOperator(%q) = true, want false", op)
		}
	}
}

func TestBuildWhereClauseChecked_RejectsInjection(t *testing.T) {
	t.Parallel()

	ph := func(i int) string { return "$1" }

	cases := []struct {
		name       string
		conditions []kv.Condition
		wantInErr  string
	}{
		{
			name: "column smuggling comment",
			conditions: []kv.Condition{
				{Column: "age; DROP TABLE users", Op: kv.OpEq, Value: 1},
			},
			wantInErr: "age; DROP TABLE users",
		},
		{
			name: "column smuggling union",
			conditions: []kv.Condition{
				{Column: "name UNION SELECT email FROM users", Op: kv.OpEq, Value: "x"},
			},
			wantInErr: "condition 0",
		},
		{
			name: "hostile operator",
			conditions: []kv.Condition{
				{Column: "name", Op: "= 'x' OR 1=1 --", Value: "x"},
			},
			wantInErr: "unsupported operator",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clause, _, err := sqlpkg.BuildWhereClauseChecked(tt.conditions, ph)
			if err == nil {
				t.Fatalf("BuildWhereClauseChecked: want error, got clause %q", clause)
			}

			if !strings.Contains(err.Error(), tt.wantInErr) {
				t.Errorf("error %q should contain %q", err, tt.wantInErr)
			}
		})
	}
}

func TestBuildWhereClauseChecked_BuildsValidClauses(t *testing.T) {
	t.Parallel()

	ph := func(i int) string {
		if i == 1 {
			return "?"
		}

		return "?"
	}

	clause, args, err := sqlpkg.BuildWhereClauseChecked([]kv.Condition{
		{Column: "age", Op: kv.OpGte, Value: 18},
		{Column: "status", Op: kv.OpIn, Values: []any{"active", "pending"}},
		{Column: "deleted", Op: kv.OpIsNull},
	}, ph)
	if err != nil {
		t.Fatalf("BuildWhereClauseChecked: %v", err)
	}

	want := "age >= ? AND status IN (?, ?) AND deleted IS NULL"
	if clause != want {
		t.Errorf("clause = %q, want %q", clause, want)
	}

	if len(args) != 3 {
		t.Errorf("args: got %d, want 3 (18, active, pending)", len(args))
	}

	empty, args, err := sqlpkg.BuildWhereClauseChecked(nil, ph)
	if err != nil || empty != "" || args != nil {
		t.Errorf("empty conditions: got (%q, %v, %v), want (\"\", nil, nil)", empty, args, err)
	}
}
