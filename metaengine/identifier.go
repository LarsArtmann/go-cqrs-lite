package metaengine

import "fmt"

// ValidateIdentifier reports whether s is a safe SQL/JSON-path identifier
// fragment: ASCII letters, digits, '_', '.', '-', space, or any non-ASCII
// rune (unicode collection/field names). Quotes, parentheses, semicolons,
// and every other ASCII punctuation are rejected.
//
// It is the shared allowlist behind materialized-view identifiers and the
// column names that engine scan paths splice into json_extract path
// expressions — apply it before any string built from operator input
// reaches a SQL builder.
func ValidateIdentifier(s string) error {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '_', r == '.', r == '-', r == ' ', r >= 0x80:
			// allowed
		default:
			return fmt.Errorf(
				"metaengine: invalid identifier %q (use letters, digits, '_', '.', '-')",
				s,
			)
		}
	}

	return nil
}
