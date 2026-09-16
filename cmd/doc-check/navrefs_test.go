package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGithubSlug_GitHubExactVectors(t *testing.T) {
	t.Parallel()

	// Vectors pinned against this repo's real hand-written TOCs and GitHub's
	// slug rules: underscores kept, punctuation stripped, inline-code content
	// kept, links contribute their text, spaces all become hyphens.
	cases := []struct{ in, want string }{
		{
			"2.30 Operator Priority Routing — global / perEngine / perQuery (system-verified v4.6.0)",
			"230-operator-priority-routing--global--perengine--perquery-system-verified-v460",
		},
		{"0. Mental Model (read this first)", "0-mental-model-read-this-first"},
		{"my_section stays", "my_section-stays"},
		{"Call `Infer(samples...)` here", "call-infersamples-here"},
		{
			"Shared Terms (defined in [Domain Language](DOMAIN_LANGUAGE.md))",
			"shared-terms-defined-in-domain-language",
		},
		{"Fold DSL (Event to Projection Mapping)", "fold-dsl-event-to-projection-mapping"},
		{"hyphens-stay_hyphens", "hyphens-stay_hyphens"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := githubSlug(tc.in); got != tc.want {
			t.Errorf("githubSlug(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseHeadings_SkipsFencesAndClosesHashes(t *testing.T) {
	t.Parallel()

	md := "# Top\n" +
		"```go\n" +
		"# not a heading\n" +
		"```\n" +
		"## 2.13b Real Section ##\n" +
		"#### Production options (SQLite / Turso)\n"

	hs := parseHeadings(md)
	if len(hs) != 3 {
		t.Fatalf("got %d headings, want 3: %+v", len(hs), hs)
	}

	if hs[1].number != "2.13b" || hs[1].slug != "213b-real-section" {
		t.Errorf("2.13b heading mis-parsed: %+v", hs[1])
	}

	if hs[2].number != "" || hs[2].slug != "production-options-sqlite--turso" {
		t.Errorf("unnumbered heading mis-parsed: %+v", hs[2])
	}
}

// navTestRepo builds a minimal skill-scope doc tree: SKILL.md under
// .agents/skills/g/, references/{recipes,core}.md, plus an out-of-scope
// README.md, all under one repo root.
func navTestRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	write := func(rel, content string) {
		path := filepath.Join(root, rel)

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	write(".agents/skills/g/references/recipes.md",
		"# Recipes\n\n## 2.1 Minimal ES\n\n## 2.2 Persistence\n\n## 2.1b Retry\n")
	write(".agents/skills/g/references/core.md",
		"# Core\n\n## 3. Conventions\n\nSee [recipes](recipes.md) §2.1 and `recipes.md` §2.2.\n"+
			"ADR-0124 §7 is skipped. The former recipes §2.1 pointer is skipped.\n"+
			"Bare §2.1 resolves via the pool; bare §9.9 is broken.\n"+
			"> - §2.0 Old Home → moved to [recipes.md](recipes.md)\n"+
			"Jump to [§3](#3-conventions) and [broken](#nope).\n")
	write(".agents/skills/g/SKILL.md", "# Skill\n\nSee `references/core.md` §3.\n")
	write("README.md", "# Readme\n\nFree-form §2.1 §9.9 are not validated here.\n")
	write("AGENTS.md", "# Agents\n\nrecipes §2.1 from repo root works.\n")

	return root
}

func findIssue(issues []navIssue, substr string) *navIssue {
	for i := range issues {
		if strings.Contains(issues[i].Msg, substr) {
			return &issues[i]
		}
	}

	return nil
}

func TestCheckFiles_NavGrammar(t *testing.T) {
	t.Parallel()

	root := navTestRepo(t)

	files := []string{
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, ".agents/skills/g/SKILL.md"),
		filepath.Join(root, ".agents/skills/g/references/core.md"),
		filepath.Join(root, ".agents/skills/g/references/recipes.md"),
		filepath.Join(root, "README.md"),
	}

	issues := checkFiles(files, root)

	if n := findIssue(issues, "§9.9"); n == nil {
		t.Errorf("broken bare §9.9 not flagged: %+v", issues)
	}

	if n := findIssue(issues, "broken anchor"); n == nil {
		t.Errorf("broken anchor #nope not flagged: %+v", issues)
	}

	for _, iss := range issues {
		switch {
		case strings.Contains(iss.Msg, "§2.1"):
			t.Errorf("valid §2.1 (prefixed/pooled/moved forms) flagged: %+v", iss)
		case strings.Contains(iss.Msg, "§2.2"),
			strings.Contains(iss.Msg, "§7"),
			strings.Contains(iss.Msg, "§3"):
			t.Errorf("valid/filtered § ref flagged: %+v", iss)
		case strings.Contains(iss.File, "README"):
			t.Errorf("out-of-scope § ref validated: %+v", iss)
		}
	}
}

func TestCheckFiles_DuplicateSectionNumbers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	doc := filepath.Join(root, "dup.md")

	body := "# D\n\n## 2.13 First\n\n## 2.13 Second\n"

	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	issues := checkFiles([]string{doc}, root)
	if len(issues) != 1 || !strings.Contains(issues[0].Msg, "duplicate section number §2.13") {
		t.Fatalf("want one duplicate-number issue, got %+v", issues)
	}
}

func TestCheckFiles_CrossFileAnchorAndDedupSuffix(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	target := filepath.Join(root, "target.md")
	linking := filepath.Join(root, "links.md")

	if err := os.WriteFile(
		target,
		[]byte("# T\n\n## Only One\n\n## Only One\n"),
		0o644,
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := os.WriteFile(linking,
		[]byte("[ok](target.md#only-one) [ok2](target.md#only-one-1) [bad](target.md#only-one-2)"),
		0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	issues := checkFiles([]string{linking, target}, root)
	if len(issues) != 1 || !strings.Contains(issues[0].Msg, "#only-one-2") {
		t.Fatalf("want only the -2 anchor flagged, got %+v", issues)
	}
}

func TestCheckFiles_InlineCodeAndFencesExcluded(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	doc := filepath.Join(root, "doc.md")

	body := "# D\n\nGeneric [T](x, y) stays.\n\n`§9.9` inside code is skipped.\n\n" +
		"```markdown\n[broken](#nope-anchor)\n```\n"

	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if issues := checkFiles([]string{doc}, root); len(issues) != 0 {
		t.Fatalf("inline-code/fenced shapes must not be flagged, got %+v", issues)
	}
}
