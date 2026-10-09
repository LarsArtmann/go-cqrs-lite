package analyzer

import "strings"

// stackPresetFromImport maps a go-cqrs-lite/stack import to its preset short
// name: engine presets ("sqlite", "postgres", "mysql", "pebble", "memory",
// "turso", "duckdb", "bbolt", "metaengine") and the root module ("bundle" —
// its subpackages like sqlopt share the module path). All of them are deleted
// at the v5 cut (ADR-0123). Returns "" for non-stack imports.
func stackPresetFromImport(path string) string {
	const prefix = "go-cqrs-lite/stack/"
	i := strings.Index(path, prefix)
	if i < 0 {
		return ""
	}
	name, _, _ := strings.Cut(path[i+len(prefix):], "/")
	if name == "" || name == "v4" {
		return "bundle"
	}
	return name
}
