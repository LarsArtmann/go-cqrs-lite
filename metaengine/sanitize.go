package metaengine

import "strings"

// SanitizeIdent joins parts with sep into a safe identifier, replacing
// every character outside [a-zA-Z0-9] plus extraAllowed with '_'. The SQL
// engines use it to derive stable schema objects (index and constraint
// names) from collection names; Dgraph uses it for predicate names.
// Behavior matches the engines' former per-module sanitizers byte for byte.
func SanitizeIdent(sep, extraAllowed string, parts ...string) string {
	var b strings.Builder

	for i, p := range parts {
		if i > 0 {
			b.WriteString(sep)
		}

		for _, r := range p {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
				strings.ContainsRune(extraAllowed, r) {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
	}

	return b.String()
}
