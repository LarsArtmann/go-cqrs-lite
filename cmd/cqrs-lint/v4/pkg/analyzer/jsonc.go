package analyzer

// StripJSONC removes // line comments and /* block */ comments from JSON data
// while respecting string literals, then strips trailing commas (allowed by
// the JSONC spec but rejected by strict JSON parsers). This is the single
// JSONC decoder for both config paths (CLI loader and embedded loader).
func StripJSONC(data []byte) []byte {
	var result []byte
	result = make([]byte, 0, len(data))

	i := 0
	inString := false

	for i < len(data) {
		c := data[i]

		if inString {
			if c == '\\' && i+1 < len(data) {
				result = append(result, c, data[i+1])
				i += 2
				continue
			}
			if c == '"' {
				inString = false
			}
			result = append(result, c)
			i++
			continue
		}

		switch {
		case c == '"':
			inString = true
			result = append(result, c)
			i++

		case c == '/' && i+1 < len(data) && data[i+1] == '/':
			for i < len(data) && data[i] != '\n' {
				i++
			}

		case c == '/' && i+1 < len(data) && data[i+1] == '*':
			i += 2
			for i+1 < len(data) && (data[i] != '*' || data[i+1] != '/') {
				i++
			}
			if i+1 < len(data) {
				i += 2
			} else {
				i = len(data)
			}

		default:
			result = append(result, c)
			i++
		}
	}

	return stripTrailingCommas(result)
}

// stripTrailingCommas removes commas that are followed only by whitespace
// until a closing } or ]. This handles the JSONC trailing comma convention
// while respecting string literals (a comma inside a string followed by } is
// NOT removed).
func stripTrailingCommas(data []byte) []byte {
	result := make([]byte, 0, len(data))
	inString := false
	pendingComma := -1 // index in result where a comma may be trailing

	for i := 0; i < len(data); i++ {
		c := data[i]

		if inString {
			if c == '\\' && i+1 < len(data) {
				result = append(result, c, data[i+1])
				i++
				continue
			}
			if c == '"' {
				inString = false
			}
			result = append(result, c)
			pendingComma = -1
			continue
		}

		switch c {
		case '"':
			inString = true
			result = append(result, c)
			pendingComma = -1
		case ',':
			result = append(result, c)
			pendingComma = len(result) - 1
		case '}', ']':
			if pendingComma >= 0 {
				result = result[:pendingComma]
			}
			result = append(result, c)
			pendingComma = -1
		case ' ', '\t', '\n', '\r':
			result = append(result, c)
		default:
			result = append(result, c)
			pendingComma = -1
		}
	}

	return result
}
