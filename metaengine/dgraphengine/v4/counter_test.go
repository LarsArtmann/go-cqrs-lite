package dgraphengine

import (
	"strings"
	"testing"
)

// TestKeyVarDeclsDQLSyntax locks the DQL variable-declaration syntax used in
// the CounterIncrement upsert query header. DQL requires typed args as
// "$name: type"; a missing colon makes the whole query fail to parse, so
// CounterIncrement breaks for every multi-key batch (regression guard).
func TestKeyVarDeclsDQLSyntax(t *testing.T) {
	t.Parallel()

	decls := keyVarDecls(3)

	want := []string{"$key0: string", "$key1: string", "$key2: string"}
	if strings.Join(decls, ", ") != strings.Join(want, ", ") {
		t.Fatalf("keyVarDecls(3) = %v, want %v", decls, want)
	}

	for _, d := range decls {
		if !strings.Contains(d, ": string") {
			t.Errorf("declaration %q is missing the DQL colon separator", d)
		}
	}
}
