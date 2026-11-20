package adoption

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// versionTag matches a module release tag: <path>/v<major>.<minor>.<patch>
// with an optional prerelease suffix. Tags without a slash (v4.0.0) version
// the repository itself and are not module paths.
var versionTag = regexp.MustCompile(`^(.+)/v[0-9]+\.[0-9]+\.[0-9]+[^/]*$`)

// modulePathFromVersionTag maps a release tag to its repo-relative module
// path ("transport/http/v4.2.1" -> "transport/http"), reporting false for
// root-release and non-version tags.
func modulePathFromVersionTag(tag string) (string, bool) {
	m := versionTag.FindStringSubmatch(tag)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// tagDerivedDeletedModules is the frozen-pin universe: every module path with
// at least one release tag but no go.mod at HEAD (root or /vN). ADR-0152
// keeps deleted tags proxy-served forever, so these paths still resolve for
// consumers pinned to the old tags — exactly the class F030 coaches off.
func tagDerivedDeletedModules(t *testing.T) map[string]bool {
	t.Helper()

	root := repoRootForF030(t)
	out, err := exec.Command("git", "-C", root, "tag", "-l").Output()
	if err != nil {
		t.Skipf("git tags unavailable (hermetic/nix source) — tag-derivation gate requires a git checkout: %v", err)
	}

	tags := strings.Fields(string(out))
	if len(tags) == 0 {
		t.Skip("no tags in checkout (shallow clone) — tag-derivation gate requires the tag universe")
	}

	deleted := make(map[string]bool)
	for _, tag := range tags {
		rel, ok := modulePathFromVersionTag(tag)
		if !ok {
			continue
		}
		if len(moduleGoMods(root, rel)) == 0 {
			deleted["go-cqrs-lite/"+rel] = true
		}
	}

	return deleted
}

// acknowledgedTagOnlyModules lists tag-derived deleted paths F030 deliberately
// does not track: pre-v4-layout relics whose imports would require a pin older
// than the 2026 v4 reorganization. Each names its successor (with the move's
// commit where known) so a future session can promote it to a full registry
// row if evidence ever warrants. Anything else that becomes tag-only fails
// the completeness gate until it is registered.
var acknowledgedTagOnlyModules = map[string]string{
	"go-cqrs-lite/cqrs-lite":          "pre-rename v0.1 layout; became the repository root module",
	"go-cqrs-lite/event/v3/eventtest": "v3-era nested path; successor event/v4/eventtest",
	"go-cqrs-lite/memory":             "pre-storage layout; successor storage/memory/v4",
	"go-cqrs-lite/pebble":             "pre-storage layout; successor storage/pebble/v4",
	"go-cqrs-lite/saga":               "removed in 3d3802deb; sagas live in composition, not a module",
	"go-cqrs-lite/sync":               "extracted to github.com/larsartmann/go-localsync in 153ed5a79",
	"go-cqrs-lite/testhelpers":        "consolidated into event/eventtest in 157dde320; successor event/v4/eventtest",
	"go-cqrs-lite/testing":            "renamed to scenario in 293f6c631; successor scenario/v4",
	"go-cqrs-lite/turso":              "pre-storage layout; successor storage/turso/v4",
}

// TestTagDerivedDeletedModules_RegistryIsComplete is the C6b completeness
// gate over the tag universe, in three directions:
//
//   - TRACKED: every registry moduleDeleted row was actually released — a row
//     whose path never had a tag cannot be imported by anyone and is dead
//     weight in the table.
//   - ACKNOWLEDGED: every other tag-derived deletion carries a successor note
//     here. A future deletion that is neither registered nor acknowledged
//     fails — the drift class this gate exists to kill.
//   - FRESH: no acknowledged entry is stale (already registered, or back at
//     HEAD, or never tagged).
func TestTagDerivedDeletedModules_RegistryIsComplete(t *testing.T) {
	t.Parallel()

	derived := tagDerivedDeletedModules(t)

	registered := make(map[string]bool, len(modulePresenceRegistry))
	for _, entry := range modulePresenceRegistry {
		if entry.state == moduleDeleted {
			registered[entry.fragment] = true
			if !derived[entry.fragment] {
				t.Errorf("%s is registered moduleDeleted but no release tag ever carried it", entry.fragment)
			}
		}
	}

	for fragment := range derived {
		if registered[fragment] {
			continue
		}
		if _, ok := acknowledgedTagOnlyModules[fragment]; !ok {
			t.Errorf("%s has release tags but no go.mod at HEAD: "+
				"register it moduleDeleted or acknowledge it with a successor note", fragment)
		}
	}

	for fragment := range acknowledgedTagOnlyModules {
		switch {
		case registered[fragment]:
			t.Errorf("%s is both acknowledged and registered; drop the acknowledgment", fragment)
		case !derived[fragment]:
			t.Errorf("%s is acknowledged but not tag-derived (go.mod at HEAD or never tagged); "+
				"drop the acknowledgment", fragment)
		}
	}
}

func TestModulePathFromVersionTag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tag    string
		want   string
		wantOK bool
	}{
		{"transport/http/v4.2.1", "transport/http", true},
		{"event/v3/eventtest/v3.7.4", "event/v3/eventtest", true},
		{"cmd/cqrs-lint/v4.22.2-rc1", "cmd/cqrs-lint", true},
		{"cqrs-lite/v0.1.1", "cqrs-lite", true},
		{"v4.0.0", "", false},
		{"docs/some-note", "", false},
		{"weird/vX.Y.Z", "", false},
	}

	for _, tc := range cases {
		got, ok := modulePathFromVersionTag(tc.tag)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("modulePathFromVersionTag(%q) = (%q, %v), want (%q, %v)",
				tc.tag, got, ok, tc.want, tc.wantOK)
		}
	}
}
