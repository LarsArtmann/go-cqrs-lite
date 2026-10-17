package architecture_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// idTargets collects every id="..." in the document.
var idTargets = regexp.MustCompile(`id="([^"]+)"`)

// hrefTargets collects every in-page link target.
var hrefTargets = regexp.MustCompile(`href="#([^"]*)"`)

// searchAnchors collects every anchor in the embedded search index.
var searchAnchors = regexp.MustCompile(`"h":"([^"]+)"`)

// navAnchors collects the sidebar entry links (distinct from resource links
// and group headings by their class attribute).
var navAnchors = regexp.MustCompile(`<a class="item" href="#([^"]+)"`)

// sectionAnchors collects the rendered body sections.
var sectionAnchors = regexp.MustCompile(`<section class="section" id="([^"]+)"`)

// anchorsOf runs re over html and returns the distinct capture groups.
func anchorsOf(t *testing.T, re *regexp.Regexp, html string) []string {
	t.Helper()

	found := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		found[m[1]] = true
	}

	out := make([]string, 0, len(found))
	for anchor := range found {
		out = append(out, anchor)
	}

	slices.Sort(out)

	return out
}

// TestExportLinksResolveToSections walks every clickable anchor in the
// document — sidebar entries, resource references, group headings, markdown
// links and the search index — and asserts each one lands on an id the
// document actually rendered. A failure here is a broken promise: the
// reference advertises itself as fully navigable.
func TestExportLinksResolveToSections(t *testing.T) {
	out := renderFixture(t)

	ids := map[string]bool{}
	for _, anchor := range anchorsOf(t, idTargets, out) {
		ids[anchor] = true
	}

	if len(ids) == 0 {
		t.Fatal("document renders no ids at all")
	}

	for _, source := range []struct {
		name  string
		re    *regexp.Regexp
	 what string
	}{
		{"href", hrefTargets, "link"},
		{"search index", searchAnchors, "search hit"},
	} {
		for _, anchor := range anchorsOf(t, source.re, out) {
			if anchor == "" {
				continue
			}

			if !ids[anchor] {
				t.Errorf("%s points at missing anchor %q", source.what, anchor)
			}
		}
	}
}

// TestExportNavMatchesSections pins the sidebar↔body bijection in both
// directions: every section is reachable from the sidebar and every sidebar
// entry has a section. A navigation document that hides a resource, or
// advertises one it did not render, fails here.
func TestExportNavMatchesSections(t *testing.T) {
	out := renderFixture(t)

	nav := anchorsOf(t, navAnchors, out)
	sections := anchorsOf(t, sectionAnchors, out)

	if len(nav) == 0 {
		t.Fatal("sidebar renders no entries")
	}

	if len(sections) == 0 {
		t.Fatal("body renders no sections")
	}

	for _, anchor := range nav {
		if !slices.Contains(sections, anchor) {
			t.Errorf("sidebar entry %q has no section", anchor)
		}
	}

	for _, anchor := range sections {
		if !slices.Contains(nav, anchor) {
			t.Errorf("section %q is unreachable from the sidebar", anchor)
		}
	}

	if len(nav) != len(sections) {
		t.Errorf("sidebar has %d entries but body renders %d sections", len(nav), len(sections))
	}
}

// TestExportAnchorSlugsAreUnique guards the collision class the slug scheme
// exists to prevent: two resources whose ids slug to the same anchor would
// silently overwrite each other's target.
func TestExportAnchorSlugsAreUnique(t *testing.T) {
	out := renderFixture(t)

	seen := map[string]string{}
	for _, anchor := range anchorsOf(t, idTargets, out) {
		if previous, clash := seen[strings.ToLower(anchor)]; clash {
			t.Errorf("anchor %q and %q collide case-insensitively", previous, anchor)
		}

		seen[strings.ToLower(anchor)] = anchor
	}
}
