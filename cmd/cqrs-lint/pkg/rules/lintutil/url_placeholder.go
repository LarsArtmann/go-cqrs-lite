package lintutil

import "strings"

// IsURLOrPlaceholder reports whether a string literal value is a URL or an
// unfilled placeholder template rather than a real credential. Documentation
// links (apiKeyDocsURL = "https://…") and env-var/insertion templates
// ("${API_KEY}", "<your-token>") trip secret-name heuristics without
// embedding a secret. Tradeoff: credential-bearing DSNs (postgres://…)
// are also skipped, but those live under dsn/connectionString-style names,
// not the secret keywords those rules match on.
//
// Single shared implementation (extracted from S001 so a second rule that
// needs the same value-classifier cannot fork the semantics).
func IsURLOrPlaceholder(val string) bool {
	if strings.Contains(val, "://") {
		return true
	}

	trimmed := strings.TrimSpace(val)

	return strings.HasPrefix(trimmed, "${") ||
		(strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">"))
}

// IsNameShapedValue reports whether a string literal is plain schema
// vocabulary (pure lowercase snake_case, e.g. "enroll_tokens",
// "user_sessions") rather than credential material. Secret material carries
// entropy: mixed case, digits mixed with symbols, or length; a name-shaped
// value is an identifier, not a key.
//
// This is a value-side allowlist for substring-based secret heuristics: an
// identifier like enrollTokensCollection is not a secret just because it
// contains "token".
func IsNameShapedValue(val string) bool {
	if val == "" {
		return false
	}

	for i, r := range val {
		switch {
		case r >= 'a' && r <= 'z':
		case r == '_':
			if i == 0 {
				return false
			}
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}

	return true
}
