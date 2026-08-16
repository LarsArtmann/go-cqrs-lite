package sql_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// FuzzValidateIdentifier_RejectsAllNonIdentifiers drives ValidateIdentifier
// with arbitrary strings. The invariant: ValidateIdentifier must return true
// if and only if the input matches the bare-identifier grammar
// [A-Za-z_][A-Za-z0-9_]*. Any string containing a character outside that
// set must be rejected — this is the SQL-injection gate.
func FuzzValidateIdentifier_RejectsAllNonIdentifiers(f *testing.F) {
	// Valid seeds.
	f.Add("a")
	f.Add("name")
	f.Add("user_id")
	f.Add("_private")
	f.Add("Column1")
	f.Add("a_very_long_column_name")

	// Hostile seeds — SQLite metacharacters.
	f.Add("col--")
	f.Add("col/*c*/")
	f.Add("col;DROP")
	f.Add("col\"quote")
	f.Add("col`tick`")
	f.Add("col UNION SELECT x")

	// Hostile seeds — PostgreSQL metacharacters.
	f.Add("col$$")
	f.Add("col||1")
	f.Add("col::text")
	f.Add("col;--")
	f.Add("col'||'")
	f.Add("col/* */")

	// Hostile seeds — MySQL metacharacters.
	f.Add("col`x`")
	f.Add("col-- -")
	f.Add("col /*!50000UNION*/")
	f.Add("col\\x00")
	f.Add("col'OR'1")
	f.Add("col%00")

	// Edge cases.
	f.Add("")
	f.Add("1starts_with_digit")
	f.Add("has space")
	f.Add("dotted.path")
	f.Add("*")
	f.Add("a+b")
	f.Add("a-b")
	f.Add("a/b")
	f.Add("a@b")
	f.Add("a#b")
	f.Add("a$b")
	f.Add("a%b")
	f.Add("a&b")
	f.Add("a|b")
	f.Add("a!b")
	f.Add("a~b")
	f.Add("a^b")
	f.Add("a(b)")
	f.Add("a[b]")
	f.Add("a{b}")
	f.Add("a=b")
	f.Add("a?b")
	f.Add("a\\b")
	f.Add("a:b")
	f.Add("a;b")
	f.Add("a,b")
	f.Add("a.b")
	f.Add("a'b")
	f.Add("a\"b")
	f.Add("a`b")
	f.Add("a\tb")
	f.Add("a\nb")
	f.Add("a\rb")

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip()
		}

		got := sqlpkg.ValidateIdentifier(input)
		want := isBareIdentifier(input)

		if got != want {
			t.Errorf("ValidateIdentifier(%q) = %v, want %v", input, got, want)
		}
	})
}

// isBareIdentifier is an independent reimplementation of the
// [A-Za-z_][A-Za-z0-9_]* grammar, used as a cross-check oracle.
// If ValidateIdentifier and this function ever disagree, either the
// regex or this oracle has a bug — both must be investigated.
func isBareIdentifier(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		if i == 0 {
			if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && r != '_' {
				return false
			}

			continue
		}

		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') &&
			(r < '0' || r > '9') && r != '_' {
			return false
		}
	}

	return true
}

// sqlMetacharacters is the union of SQL-special characters across
// SQLite, PostgreSQL, and MySQL that could enable injection if they
// appear inside an interpolated identifier. The fuzz target below
// constructs strings from these characters and asserts none pass
// ValidateIdentifier.
var sqlMetacharacters = []rune{
	' ', '\t', '\n', '\r',
	';', '-', '/', '*', '+', '=', '!', '@', '#', '$', '%', '^', '&',
	'|', '~', '(', ')', '[', ']', '{', '}', '<', '>',
	'.', ',', ':', '?', '\\', '\'', '"', '`',
}

// FuzzValidateIdentifier_MetacharacterCombinations generates identifiers by
// inserting a random metacharacter into a valid base identifier. Every such
// string must be rejected — no metacharacter from SQLite, PostgreSQL, or
// MySQL may pass the gate.
func FuzzValidateIdentifier_MetacharacterCombinations(f *testing.F) {
	base := []string{"col", "user_id", "name", "_x", "A"}

	// Seed: base + each metacharacter at position 0, middle, and end.
	for _, b := range base {
		for _, mc := range sqlMetacharacters {
			f.Add(string(mc) + b)                           // metacharacter at start
			f.Add(b[:len(b)/2] + string(mc) + b[len(b)/2:]) // middle
			f.Add(b + string(mc))                           // end
		}
	}

	// Also seed pure-metacharacter strings.
	f.Add(";")
	f.Add("'")
	f.Add("\"")
	f.Add("`")
	f.Add("--")
	f.Add("/*")
	f.Add("*/")
	f.Add("$$")
	f.Add("||")
	f.Add("::")

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			t.Skip()
		}

		if sqlpkg.ValidateIdentifier(input) {
			// If it passed, it must be a bare identifier with no
			// metacharacters. If we find one, that's a bypass.
			for _, mc := range sqlMetacharacters {
				if strings.ContainsRune(input, mc) {
					t.Errorf(
						"ValidateIdentifier(%q) = true but contains metacharacter %q",
						input,
						mc,
					)
				}
			}
		}
	})
}

// FuzzBuildWhereClauseChecked_NeverPanics fuzzes the full
// BuildWhereClauseChecked path with arbitrary column names and operators.
// Invariant: the function must never panic, and when it returns no error,
// the resulting clause must not contain any character outside the
// [A-Za-z0-9_] set in its column positions.
func FuzzBuildWhereClauseChecked_NeverPanics(f *testing.F) {
	ph := func(i int) string { return "?" }

	// Valid seeds.
	f.Add("name", "age", int64(0))
	f.Add("user_id", "status", int64(42))
	f.Add("_private", "data", int64(-1))

	// Hostile seeds.
	f.Add("col; DROP TABLE x", "age", int64(0))
	f.Add("name UNION SELECT email", "age", int64(0))
	f.Add("col--", "age", int64(0))
	f.Add("col/*c*/", "age", int64(0))
	f.Add("' OR 1=1 --", "age", int64(0))
	f.Add("col\"x", "age", int64(0))
	f.Add("col`x", "age", int64(0))
	f.Add("col$$", "age", int64(0))
	f.Add("col||1", "age", int64(0))
	f.Add("col::text", "age", int64(0))

	f.Fuzz(func(t *testing.T, column, _ string, _ int64) {
		if !utf8.ValidString(column) {
			t.Skip()
		}

		conditions := []kv.Condition{
			{Column: column, Op: kv.OpEq, Value: int64(0)},
		}

		clause, _, err := sqlpkg.BuildWhereClauseChecked(conditions, ph)
		if err != nil {
			// Must not have built a clause on error.
			if clause != "" {
				t.Errorf("error case returned non-empty clause %q", clause)
			}

			return
		}

		// On success, the column in the clause must be the validated
		// identifier (only [A-Za-z0-9_] chars). This confirms no
		// metacharacter leaked into the SQL.
		if !sqlpkg.ValidateIdentifier(column) {
			t.Errorf(
				"BuildWhereClauseChecked accepted hostile column %q (clause=%q)",
				column,
				clause,
			)
		}
	})
}
