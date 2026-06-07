package caseutil

import "strings"

// ToSeparated converts CamelCase to separated format using the given separator.
// Upper-case letters and digits following letters get a separator prefix.
// Spaces and underscores are replaced with the separator.
func ToSeparated(s string, sep byte) string {
	var result []byte

	runes := []rune(s)

	for i, c := range runes {
		switch {
		case c >= 'A' && c <= 'Z':
			if i > 0 {
				prev := runes[i-1]
				prevIsUpper := prev >= 'A' && prev <= 'Z'
				prevIsLower := prev >= 'a' && prev <= 'z'
				nextIsLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'

				if prevIsLower || (prevIsUpper && nextIsLower) {
					result = append(result, sep)
				}
			}

			result = append(result, byte(c+'a'-'A'))
		case c >= '0' && c <= '9':
			if i > 0 {
				prev := runes[i-1]
				isLetter := (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z')

				if isLetter {
					result = append(result, sep)
				}
			}

			result = append(result, byte(c))
		case c == ' ' || c == '_':
			result = append(result, sep)
		case c >= 0 && c <= 127:
			result = append(result, byte(c))
		}
	}

	return string(result)
}

// DotSeparated converts CamelCase to dot.separated format.
func DotSeparated(s string) string {
	return ToSeparated(s, '.')
}

// ToKebab converts CamelCase to kebab-case format.
func ToKebab(s string) string {
	return ToSeparated(s, '-')
}

// ToPascal converts a space/underscore/hyphen-separated string to PascalCase.
func ToPascal(s string) string {
	if s == "" {
		return ""
	}

	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-'
	})

	var result strings.Builder

	for _, w := range words {
		if len(w) > 0 {
			first := w[0]
			if first >= 'a' && first <= 'z' {
				result.WriteRune(rune(first - 'a' + 'A'))
			} else {
				result.WriteRune(rune(first))
			}

			if len(w) > 1 {
				result.WriteString(strings.ToLower(w[1:]))
			}
		}
	}

	return result.String()
}
