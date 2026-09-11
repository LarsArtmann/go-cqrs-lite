package tursoengine

import (
	"net/url"
	"strings"
)

// normalizeEmbeddedDSN accepts the SQLite-conventional `file:` scheme for
// embedded databases (README-documented) by stripping it down to the plain
// path the turso driver expects: `file:/data/app.db` → `/data/app.db`
// (`file://` host forms included). A `file:` DSN carrying QUERY PARAMETERS
// (libsql-style `?mode=memory`) passes through UNCHANGED — the turso driver
// has no such parameters, and silently dropping them would turn a
// misconfigured DSN into a different database; the driver's own error is the
// honest answer there. Remote DSNs pass through untouched.
func normalizeEmbeddedDSN(dsn string) string {
	if isRemoteDSN(dsn) {
		return dsn
	}

	for _, prefix := range []string{"file://", "file:"} {
		if !strings.HasPrefix(dsn, prefix) {
			continue
		}

		rest := strings.TrimPrefix(dsn, prefix)

		if strings.Contains(rest, "?") {
			return dsn
		}

		return rest
	}

	return dsn
}

// withExperimentalToken merges an experimental-feature flag into the
// "experimental" comma-separated DSN parameter of an embedded DSN, leaving
// every other query parameter byte-identical. Remote DSNs pass through
// unchanged (the driver ignores DSN params there). A DSN whose query cannot
// be parsed passes through unchanged too — construction then surfaces the
// driver's own error instead of a mangled DSN.
func withExperimentalToken(dsn, token string) string {
	if isRemoteDSN(dsn) {
		return dsn
	}

	base, query, hasQuery := strings.Cut(dsn, "?")
	if !hasQuery {
		return dsn + "?experimental=" + token
	}

	parts := strings.Split(query, "&")
	for i, part := range parts {
		key, value, hasValue := strings.Cut(part, "=")
		if key != "experimental" {
			continue
		}

		newValue, ok := mergeExperimentalParam(value, hasValue, token)
		if !ok {
			return dsn
		}

		parts[i] = "experimental=" + newValue

		return base + "?" + strings.Join(parts, "&")
	}

	return dsn + "&experimental=" + token
}

// mergeExperimentalParam returns the new raw value for the "experimental"
// parameter (ok=true), or ok=false when the DSN should pass through
// unchanged (value already contains token, or is not valid query escape).
func mergeExperimentalParam(value string, hasValue bool, token string) (string, bool) {
	if !hasValue || value == "" {
		return token, true
	}

	decoded, err := url.QueryUnescape(value)
	if err != nil || hasExperimentalToken(decoded, token) {
		return "", false
	}

	return url.QueryEscape(decoded + "," + token), true
}

// hasExperimentalToken reports whether a comma-separated feature list
// contains token (case-insensitive, whitespace-tolerant).
func hasExperimentalToken(list, token string) bool {
	for _, t := range strings.Split(list, ",") {
		if strings.EqualFold(strings.TrimSpace(t), token) {
			return true
		}
	}

	return false
}
