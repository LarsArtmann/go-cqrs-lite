package tursoengine

import (
	"net/url"
	"strings"
)

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
