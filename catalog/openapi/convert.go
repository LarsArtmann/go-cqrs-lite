package openapi

import "strings"

func toKebab(s string) string {
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
					result = append(result, '-')
				}
			}

			result = append(result, byte(c+'a'-'A'))
		case c >= '0' && c <= '9':
			if i > 0 {
				prev := runes[i-1]
				isLetter := (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z')

				if isLetter {
					result = append(result, '-')
				}
			}

			result = append(result, byte(c))
		case c == ' ' || c == '_':
			result = append(result, '-')
		case c >= 0 && c <= 127:
			result = append(result, byte(c))
		}
	}

	return string(result)
}

func toPascal(s string) string {
	if s == "" {
		return ""
	}

	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-'
	})

	var result strings.Builder

	for _, w := range words {
		if len(w) > 0 {
			result.WriteRune(rune(w[0] - 'a' + 'A'))

			if len(w) > 1 {
				result.WriteString(strings.ToLower(w[1:]))
			}
		}
	}

	return result.String()
}
