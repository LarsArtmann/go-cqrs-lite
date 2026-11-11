package suppression

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-finding"
)

// WouldSuppress is the prune guard's matcher (issue #126): it must agree
// with the stale detector's position window — own line, line below, and
// the first non-blank line above the finding.
func TestWouldSuppress_PositionWindow(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "bus.go")
	source := "package sub\n\nfunc f() {\n\t_ = 1\n}\n"
	if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	at := func(line int) finding.Finding {
		return finding.NewFinding(
			"B029",
			"cqrs-lint",
			"bus var without breaker",
			finding.SeverityWarning,
			finding.Pos(finding.FilePath(file), line, 1),
			finding.ConfidenceHigh,
		)
	}

	findings := []finding.Finding{at(5)}

	cases := map[string]struct {
		line int
		want bool
	}{
		"directive on the finding line":      {5, true},
		"directive on the line above":        {4, true},
		"directive on the line below":        {6, true}, // trailing style leniency
		"directive two lines above (in code)": {3, false},
		"directive on a distant line":         {1, false},
	}

	for name, tc := range cases {
		if got := WouldSuppress(findings, file, tc.line, "B029"); got != tc.want {
			t.Errorf("%s: WouldSuppress(line %d) = %v, want %v", name, tc.line, got, tc.want)
		}
	}

	if WouldSuppress(findings, file, 5, "B030") {
		t.Error("a different rule must not match")
	}

	other := filepath.Join(dir, "other.go")
	if err := os.WriteFile(other, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	if WouldSuppress(findings, other, 5, "B029") {
		t.Error("a different file must not match")
	}
}
