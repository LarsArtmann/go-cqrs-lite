package catalog

import (
	"strings"
	"unicode"
)

// knownSuffixes are stripped from type names before converting to human-readable
// names. This prevents catalog entries from showing technical suffixes like
// "Cmd" or "Event" in their display names.
var knownSuffixes = []string{
	"Command", "Cmd",
	"Event", "Evt",
	"Query", "Qry",
}

// camelCaseToHuman converts a CamelCase Go type name to a human-readable
// title. It strips known suffixes and inserts spaces before uppercase letters.
//
// Examples:
//   - "CreateUserCmd" → "Create User"
//   - "UserCreatedEvent" → "User Created"
//   - "GetUserQuery" → "Get User"
//   - "ChangeUserName" → "Change User Name"
func camelCaseToHuman(s string) string {
	for _, suffix := range knownSuffixes {
		if strings.HasSuffix(s, suffix) {
			stripped := strings.TrimSuffix(s, suffix)
			if stripped != "" {
				s = stripped

				break
			}
		}
	}

	var result strings.Builder

	for i, r := range s {
		if i == 0 {
			result.WriteRune(unicode.ToUpper(r))
		} else if unicode.IsUpper(r) {
			result.WriteRune(' ')
			result.WriteRune(r)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
