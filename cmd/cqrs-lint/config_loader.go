package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
)

// JSONCLoader implements cmdguard.ConfigFileLoader with support for JSON with
// Comments (JSONC). Standard JSON parsers reject // and /* */ comments, which
// makes bare .json files unusable as hand-edited configuration. JSONCLoader
// strips comments before parsing, so users can document their config inline:
//
//	{
//	  // Suppress rules that are false positives for our architecture
//	  "preset": "production",
//
//	  // Raise the bar: only show warnings and above
//	  "min-severity": "warning",
//
//	  "rules": {
//	    /* D002 is N/A: our API structs mirror Discord's snake_case */
//	    "disable": ["D002"]
//	  }
//	}
//
// The loader is a drop-in replacement for cmdguard's KoanfLoader: it receives
// raw bytes (cmdguard reads the file), strips comments, then uses the same
// case-insensitive JSON parsing path to populate the config struct and track
// which fields were explicitly set.
type JSONCLoader struct{}

// Load strips comments from data, parses the resulting JSON into cfg, and
// returns the struct field names that were explicitly present in the config.
func (JSONCLoader) Load(data []byte, cfg any) ([]string, error) {
	cleaned := stripJSONComments(data)

	var raw map[string]jsontext.Value
	if err := json.Unmarshal(cleaned, &raw); err != nil {
		return nil, fmt.Errorf("parse JSONC config: %w", err)
	}

	tags, err := cmdguard.ParseFlagTags(cfg)
	if err != nil {
		return nil, fmt.Errorf("parse flag tags: %w", err)
	}

	present := make(map[string]bool, len(raw))
	collectConfigKeys(raw, present)
	setFields := cmdguard.FilterSetFields(tags, present)

	if err := json.Unmarshal(cleaned, cfg, json.MatchCaseInsensitiveNames(true)); err != nil {
		return nil, fmt.Errorf("parse JSONC config into struct: %w", err)
	}

	return setFields, nil
}

// collectConfigKeys walks a JSON raw-message map at every nesting level,
// recording every key. This lets FilterSetFields detect leaf-level flag names
// that appear inside nested config objects (e.g. {"rules":{"disable":[...]}}
// → "disable").
func collectConfigKeys(raw map[string]jsontext.Value, keys map[string]bool) {
	for k, v := range raw {
		keys[k] = true

		var nested map[string]jsontext.Value
		if json.Unmarshal(v, &nested) == nil {
			collectConfigKeys(nested, keys)
		}
	}
}

// stripJSONComments removes // line comments and /* block */ comments from JSON
// data while respecting string literals. A // or /* inside a "..." string is
// preserved as-is. It also strips trailing commas (allowed by the JSONC spec
// but rejected by strict JSON parsers). This makes JSONC-compatible config
// files parseable by any strict JSON parser.
func stripJSONComments(data []byte) []byte {
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

		switch {
		case c == '"':
			inString = true
			result = append(result, c)
			pendingComma = -1
		case c == ',':
			result = append(result, c)
			pendingComma = len(result) - 1
		case c == '}' || c == ']':
			if pendingComma >= 0 {
				result = result[:pendingComma]
			}
			result = append(result, c)
			pendingComma = -1
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			result = append(result, c)
		default:
			result = append(result, c)
			pendingComma = -1
		}
	}

	return result
}
