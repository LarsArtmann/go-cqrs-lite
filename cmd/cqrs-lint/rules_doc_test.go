package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
)

// rulesMDAnchors extracts every explicit anchor id from RULES.md.
func rulesMDAnchors(t *testing.T) map[string]bool {
	t.Helper()

	regen := "cd cmd/cqrs-lint && GOWORK=off go run . rules --markdown > RULES.md"

	data, err := os.ReadFile("RULES.md")
	if err != nil {
		t.Fatalf("read RULES.md: %v (regenerate: %s)", err, regen)
	}

	re := regexp.MustCompile(`<a id="([a-z0-9]+)"></a>`)
	matches := re.FindAllStringSubmatch(string(data), -1)

	anchors := make(map[string]bool, len(matches))
	for _, m := range matches {
		anchors[m[1]] = true
	}

	return anchors
}

// TestRULESMD_CoversEveryCatalogID fails when a rule is added or renamed
// without regenerating RULES.md.
func TestRULESMD_CoversEveryCatalogID(t *testing.T) {
	anchors := rulesMDAnchors(t)
	regen := "cd cmd/cqrs-lint && GOWORK=off go run . rules --markdown > RULES.md"

	for _, r := range rules.AllRules() {
		if !anchors[strings.ToLower(r.ID)] {
			t.Errorf("RULES.md has no anchor for %s — regenerate: %s", r.ID, regen)
		}
	}
}

// TestRULESMD_Fresh fails when RULES.md is stale relative to the in-code
// catalog (description/severity edits count too, not just new rules).
func TestRULESMD_Fresh(t *testing.T) {
	want := renderRulesMarkdown()

	got, err := os.ReadFile("RULES.md")
	if err != nil {
		t.Fatalf("read RULES.md: %v", err)
	}

	if string(got) != want {
		t.Fatal(
			"RULES.md is stale — regenerate: cd cmd/cqrs-lint && GOWORK=off go run . rules --markdown > RULES.md",
		)
	}
}

// TestCatalogDocURLsResolve audits every catalog DocURL: RULES.md fragments
// must have anchors, other URLs must point at files that exist in the repo.
func TestCatalogDocURLsResolve(t *testing.T) {
	anchors := rulesMDAnchors(t)
	const base = "https://github.com/larsartmann/go-cqrs-lite/blob/main/"

	for _, r := range rules.AllRules() {
		if r.DocURL == "" {
			continue
		}

		if !strings.HasPrefix(r.DocURL, base) {
			t.Errorf("%s DocURL %q does not start with %q", r.ID, r.DocURL, base)
			continue
		}

		rel := strings.TrimPrefix(r.DocURL, base)
		path, frag, hasFrag := strings.Cut(rel, "#")

		if path == "cmd/cqrs-lint/RULES.md" {
			if !hasFrag {
				t.Errorf("%s DocURL points at RULES.md without a fragment", r.ID)
				continue
			}
			if !anchors[frag] {
				t.Errorf("%s DocURL fragment #%s has no anchor in RULES.md", r.ID, frag)
			}
			continue
		}

		if _, err := os.Stat("../../" + path); err != nil {
			t.Errorf("%s DocURL target docs/%s does not exist: %v", r.ID, path, err)
		}
	}
}
