package tursoengine

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
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
	for t := range strings.SplitSeq(list, ",") {
		if strings.EqualFold(strings.TrimSpace(t), token) {
			return true
		}
	}

	return false
}

// ErrUnknownEncryptionParam is returned by New when the DSN carries an
// encryption- or key-shaped query parameter the turso driver does not
// recognize. The driver silently ignores unknown parameters, so a mistyped
// name (encryption_hexkkey) would otherwise open the database UNENCRYPTED
// with no error. Match it with errors.Is; the wrapping error is
// Rejection-family (strict DSN policy, ruled 2026-10-08).
var ErrUnknownEncryptionParam error = errors.New(
	"unrecognized encryption/key DSN parameter",
)

// knownDSNParams lists the query parameters this engine and the turso driver
// understand. Anything else whose name carries encrypt/key material is
// rejected at construction (see ErrUnknownEncryptionParam).
var knownDSNParams = []string{
	"experimental",
	"encryption_cipher",
	"encryption_hexkey",
}

// strictDSNParams enforces the strict DSN policy on the caller-owned DSN: a
// query parameter whose name contains "encrypt" or "key" (either matched
// case-insensitively) but is not one of knownDSNParams spelled EXACTLY is
// rejected, because the turso driver silently ignores it and a mistyped
// encryption parameter would open the database unencrypted. The driver reads
// parameters case-sensitively, so a mixed-case spelling (Encryption_HexKey)
// is as good as unknown to it. The error names the parameter and its closest
// known spelling.
func strictDSNParams(dsn string) error {
	_, query, hasQuery := strings.Cut(dsn, "?")
	if !hasQuery {
		return nil
	}

	for part := range strings.SplitSeq(query, "&") {
		key, _, _ := strings.Cut(part, "=")

		lower := strings.ToLower(key)
		if !strings.Contains(lower, "encrypt") && !strings.Contains(lower, "key") {
			continue
		}

		if slices.Contains(knownDSNParams, key) {
			continue
		}

		return errorfamily.WrapRejectionf(
			ErrUnknownEncryptionParam,
			"tursoengine.unknown_dsn_param",
			"DSN parameter %q is not recognized; the turso driver silently ignores unknown "+
				"parameters, and a mistyped encryption parameter opens the database "+
				"unencrypted%s (pass WithEncryption(cipher, hexKey) instead of hand-writing "+
				"these parameters)",
			key, formatDSNSuggestions(suggestDSNParams(lower)),
		)
	}

	return nil
}

// suggestDSNParams returns the known encryption parameters close to name for
// error messages: an edit distance within the scaled threshold, or one being
// a prefix of the other.
func suggestDSNParams(name string) []string {
	var suggestions []string

	for _, known := range knownDSNParams[1:] { // "experimental" is never a key misspelling
		if strings.HasPrefix(known, name) || strings.HasPrefix(name, known) ||
			editDistance(name, known) <= dsnSuggestionThreshold(name, known) {
			suggestions = append(suggestions, known)
		}
	}

	return suggestions
}

// Suggestion-distance tuning: long names tolerate one extra edit.
const (
	longParamName        = 12
	longNameMaxDistance  = 3
	shortNameMaxDistance = 2
)

// dsnSuggestionThreshold scales the allowed edit distance with name length so
// long param names tolerate one extra character class of typos.
func dsnSuggestionThreshold(nameA, nameB string) int {
	if min(len(nameA), len(nameB)) >= longParamName {
		return longNameMaxDistance
	}

	return shortNameMaxDistance
}

// formatDSNSuggestions renders the close-match suffix ("; did you mean ...?")
// for the rejection message; empty when there is nothing close to suggest.
func formatDSNSuggestions(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}

	quoted := make([]string, len(suggestions))
	for i, s := range suggestions {
		quoted[i] = fmt.Sprintf("%q", s)
	}

	return fmt.Sprintf("; did you mean %s?", strings.Join(quoted, " or "))
}

// editDistance is the Levenshtein distance between two ASCII strings.
func editDistance(source, target string) int {
	prev := make([]int, len(target)+1)
	curr := make([]int, len(target)+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(source); i++ {
		curr[0] = i

		for j := 1; j <= len(target); j++ {
			cost := 1
			if source[i-1] == target[j-1] {
				cost = 0
			}

			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}

		prev, curr = curr, prev
	}

	return prev[len(target)]
}
