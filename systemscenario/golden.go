package systemscenario

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Trail renders the deterministic one-line-per-event trail of a scenario:
// type, version, and actor when stamped. Minted IDs and timestamps are
// deliberately excluded — the trail of a fixed scenario is byte-stable
// across runs, which is what golden files pin.
func Trail(events []event.Event) string {
	if len(events) == 0 {
		return "(no events)\n"
	}

	lines := make([]string, len(events))
	for i, evt := range events {
		line := fmt.Sprintf("- %s v%d", evt.Type(), evt.Version())
		if actor := evt.Metadata().ActorID.String(); actor != "" {
			line += " actor=" + actor
		}

		lines[i] = line
	}

	return strings.Join(lines, "\n") + "\n"
}

// ThenGolden asserts the act events' [Trail] against a go-snaps golden
// snapshot at path (filename relative to the test's snapshot directory —
// the file is placed next to it under __snapshots__). Update all goldens
// with UPDATE_SNAPS=true go test ./...; clean obsolete ones with
// UPDATE_SNAPS=clean (go-snaps' standard workflow, same as eventtest).
//
//	sc.Given(...).When(cmd).ThenGolden(t, "task_lifecycle")
func (p *WhenPhase) ThenGolden(t *testing.T, path string) *WhenPhase {
	s := p.sc

	t.Helper()
	s.requireAct("ThenGolden")

	snaps.WithConfig(
		snaps.Dir(filepath.Join("__snapshots__")),
		snaps.Filename(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))),
	).MatchSnapshot(t, Trail(s.actEvents()))

	return p
}
