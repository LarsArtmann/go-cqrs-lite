package ruletest

import (
	"strings"
	"testing"
)

func TestAliasedImportSource_RendersAliasImport(t *testing.T) {
	t.Parallel()

	got := AliasedImportSource("ev", "github.com/larsartmann/go-cqrs-lite/event/v4", `func create() {
	ev.NewEvent("user.created", "id", "User", 1, nil)
}`)

	for _, want := range []string{
		"package main",
		`import ev "github.com/larsartmann/go-cqrs-lite/event/v4"`,
		"ev.NewEvent(",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("fixture missing %q:\n%s", want, got)
		}
	}
}
