package consistency

import (
	"os"
	"regexp"
	"slices"
	"strings"
)

// readGoModCQRSVersion reads the go-cqrs-lite version from go.mod.
//
// In multi-module repos (go-cqrs-lite itself is one), a consumer's go.mod may
// list both direct imports (command/v4, query/v4, id/v4) and transitive ones
// (dispatcher/v4, event/v4, etc. marked `// indirect`). We prefer the DIRECT
// import version — the `// indirect` marker reflects import topology, not a
// version mismatch. If only indirect entries exist, we fall back to those.
func readGoModCQRSVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var indirectVersion string

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		trimmed := strings.TrimSpace(line)

		// The module directive of this very repo (or a fork) contains
		// "go-cqrs-lite" without a version field — without this skip the
		// module path itself was parsed as the "version", making every
		// doc version reference read as stale.
		if strings.HasPrefix(trimmed, "module ") || strings.Contains(line, "replace") {
			continue
		}

		if !strings.Contains(line, "go-cqrs-lite") {
			continue
		}

		isIndirect := strings.Contains(line, "// indirect")

		// Strip trailing comments before field-splitting so the last field
		// is the version, not "indirect".
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		version := parts[len(parts)-1]
		if !strings.HasPrefix(version, "v") {
			continue
		}

		if !isIndirect {
			return version
		}

		if indirectVersion == "" {
			indirectVersion = version
		}
	}

	return indirectVersion
}

func extractCQRSVersion(content, modVersion string) string {
	versions := []string{}
	inCodeBlock := false

	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimLeft(line, " \t")

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		if !strings.Contains(strings.ToLower(line), "go-cqrs-lite") {
			continue
		}

		// Skip markdown headings (ADR titles, section headers) — these contain
		// historical version references like "ADR-0044: Migrate from v3 to v4"
		// that describe past migrations, not current version claims.
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.Fields(line)

		for i, field := range fields {
			if !looksLikeVersionToken(field) {
				continue
			}

			// Skip migration arrows like "v2→v3" — these describe historical
			// migrations, not current version claims.
			if strings.Contains(field, "→") || strings.Contains(field, "->") {
				continue
			}

			// Skip import paths: version preceded by / (e.g. cqrs-htmx/v4.2.0)
			if idx := strings.Index(line, field); idx > 0 && line[idx-1] == '/' {
				continue
			}

			// Skip Go pseudo-versions (e.g. v4.2.1-0.20260808200723-546259830b28)
			if strings.Contains(field, "-0.") {
				continue
			}

			// Skip inline code fragments (backtick-wrapped)
			if strings.Contains(field, "`") {
				continue
			}

			// Positional attachment (#43): only tokens textually attached to a
			// go-cqrs-lite mention count — the first token on the line may belong
			// to another module ("pins go-finding v1.12.0 and go-cqrs-lite
			// v4.12.1") or describe a past state ("upgraded from go-cqrs-lite
			// v4.11.1").
			if !attachedToCQRSMention(fields, i) {
				continue
			}

			versions = append(versions, field)
		}
	}

	if len(versions) == 0 {
		return modVersion
	}

	docVersion := versions[0]

	// Wildcard compatibility: "v4.0.x" matches any "v4.0.N" in go.mod.
	if isVersionCompatible(docVersion, modVersion) {
		return modVersion
	}

	return docVersion
}

// connectorWords may sit between a module mention and its version token
// without breaking textual attachment ("go-cqrs-lite at version v4.2.0").
//
//nolint:gochecknoglobals // read-only lookup table
var connectorWords = map[string]struct{}{
	"to": {}, "and": {}, "the": {}, "a": {}, "an": {}, "at": {}, "of": {},
	"is": {}, "as": {}, "or": {}, "for": {}, "in": {}, "with": {},
	"uses": {}, "on": {}, "version": {}, "v": {}, "module": {}, "package": {},
}

// historicalCues in a mention's lead-in mark the attached version token as a
// past state, not the current claim.
//
//nolint:gochecknoglobals // read-only lookup table
var historicalCues = map[string]struct{}{
	"from": {}, "upgraded": {}, "upgrades": {}, "migrated": {},
	"migration": {}, "previously": {}, "prior": {}, "before": {},
	"was": {}, "were": {}, "older": {}, "earlier": {},
}

// attachedToCQRSMention reports whether the version token at fields[i] is
// textually attached to a go-cqrs-lite mention: the nearest preceding
// non-connector token contains "go-cqrs-lite", and the phrase introducing
// that mention carries no historical cue.
func attachedToCQRSMention(fields []string, i int) bool {
	mentionIdx := -1

	for j := i - 1; j >= 0 && i-j <= 4; j-- {
		lower := strings.ToLower(fields[j])

		if _, ok := connectorWords[lower]; ok {
			continue
		}

		if strings.Contains(lower, "go-cqrs-lite") {
			mentionIdx = j
		}

		break
	}

	if mentionIdx < 0 {
		return false
	}

	for j := mentionIdx - 1; j >= 0 && mentionIdx-j <= 2; j-- {
		if _, ok := historicalCues[strings.ToLower(fields[j])]; ok {
			return false
		}
	}

	return true
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
