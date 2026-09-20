package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	cmdguard "github.com/larsartmann/cmdguard/v4/pkg/cmdguard/v4"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
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
// Delegates to the shared analyzer implementation so the CLI loader and the
// embedded (toolspec) loader decode JSONC identically.
func stripJSONComments(data []byte) []byte {
	return analyzer.StripJSONC(data)
}
