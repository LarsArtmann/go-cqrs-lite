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
