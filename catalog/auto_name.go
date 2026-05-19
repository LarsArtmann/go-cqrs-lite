package catalog

import (
	"strings"
	"unicode"
)

// knownSuffixes are stripped from type names before converting to human-readable
// names. This prevents catalog entries from showing technical suffixes like
// "Cmd" or "Event" in their display names.
func camelCaseToHuman(s string) string {
	knownSuffixes := []string{"Command", "Cmd", "Event", "Evt", "Query", "Qry"} //nolint:goconst

	for _, suffix := range knownSuffixes {
		if stripped, ok := strings.CutSuffix(s, suffix); ok && stripped != "" {
			s = stripped

			break
		}
	}

	var result strings.Builder

	for i, r := range s {
		switch {
		case i == 0:
			result.WriteRune(unicode.ToUpper(r))
		case unicode.IsUpper(r):
			result.WriteRune(' ')
			result.WriteRune(r)
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}
