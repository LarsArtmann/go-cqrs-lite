package consistency

import (
	"regexp"
	"slices"
	"strings"
)

//	   v4.6.0") — the token is fresh iff it is compatible with THAT module's
//	   pin.
//	2. Otherwise the token is fresh iff it is compatible with ANY pinned
//	   module version (several different current versions are legitimate).
//
// A nil pin set falls back to the legacy single-version comparison.
func versionTokenIsFresh(
	token string,
	fields []string, i int,
	modVersion string,
	pinned map[string]string,
) bool {
	if len(pinned) == 0 {
		return isVersionCompatible(token, modVersion)
	}

	if i > 0 {
		prev := strings.ToLower(strings.Trim(fields[i-1], ".,;:!?"))

		if pin, ok := pinned[prev]; ok {
			return isVersionCompatible(token, pin)
		}
	}

	for _, pin := range pinned {
		if isVersionCompatible(token, pin) {
			return true
		}
	}

	return false
}

// isVersionCompatible checks whether a doc version reference is compatible
// with the go.mod version. This handles:
//   - Wildcards: "v4.0.x" matches "v4.0.0", "v4.0.1", etc.
//   - Major.minor only: "v4.0" matches "v4.0.0"
func isVersionCompatible(docVersion, modVersion string) bool {
	docParts := parseVersionParts(docVersion)
	modParts := parseVersionParts(modVersion)

	if len(docParts) == 0 || len(modParts) == 0 {
		return false
	}

	for i := range docParts {
		if i >= len(modParts) {
			break
		}

		// Wildcard "x" matches any number
		if docParts[i] == "x" || docParts[i] == "X" {
			continue
		}

		if docParts[i] != modParts[i] {
			return false
		}
	}

	return true
}

// parseVersionParts splits a version string like "v4.0.1" into ["4", "0", "1"].
// Returns nil if the input doesn't look like a semantic version.
func parseVersionParts(v string) []string {
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return nil
	}

	// Strip trailing punctuation (e.g., "v4.2.0." from prose like "uses v4.2.0.")
	v = strings.TrimRight(v, ".,;:!?")

	parts := strings.Split(v, ".")

	// Trailing empty parts (from trailing dots) are stripped, not fatal.
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}

	if len(parts) == 0 || slices.Contains(parts, "") {
		return nil
	}

	return parts
}

// looksLikeVersionToken reports whether a whitespace-delimited token has the
// shape of a Go module version reference: a leading "v" + digit(s) + "." +
// digit(s), e.g. v4.0, v4.0.1, v4.0.x.
//
// This rejects prose words that merely start with "v" — "via", "version",
// "very", "vectors" — AND bare major versions like "v3"/"v4" that are
// ambiguous in prose ("v3 Migration", "v4 release"). A real version reference
// always includes at least major.minor. See feedback:
// docs/feedback/archived/2026-07-16_DiscordSync (D005 false positive on "via go-cqrs-lite").
func looksLikeVersionToken(field string) bool {
	return versionTokenRe.MatchString(field)
}

var versionTokenRe = regexp.MustCompile(`^v\d+\.\d+`)
