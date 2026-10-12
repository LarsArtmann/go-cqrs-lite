package systemscenario

import (
	"strings"
	"testing"
)

func TestTrailDiff_EqualTrailsRenderEmpty(t *testing.T) {
	t.Parallel()

	a := []normalizedStream{
		{ordinal: 0, entries: []trailEntry{{eventType: "task.created", version: 1}}},
	}

	if got := trailDiff(a, a); got != "" {
		t.Fatalf("trailDiff of equal trails:\nwant \"\"\ngot  %q", got)
	}
}

func TestTrailDiff_PairwiseRendersDivergentStream(t *testing.T) {
	t.Parallel()

	a := []normalizedStream{
		{ordinal: 0, entries: []trailEntry{{eventType: "task.created", version: 1}}},
		{ordinal: 1, entries: []trailEntry{
			{eventType: "task.created", version: 1},
			{eventType: "task.updated", version: 2},
		}},
	}
	b := []normalizedStream{
		{ordinal: 0, entries: []trailEntry{{eventType: "task.created", version: 1}}},
		{ordinal: 1, entries: []trailEntry{{eventType: "task.created", version: 1}}},
	}

	got := trailDiff(a, b)

	for _, want := range []string{"stream #1 differs", "system A:", "system B:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("trailDiff missing %q:\n%s", want, got)
		}
	}

	head, body, _ := strings.Cut(got, "system B:")
	if strings.Contains(body, "task.updated") {
		t.Fatalf("system B section must NOT carry system A's extra event:\n%s", got)
	}

	if !strings.Contains(head, "task.updated") {
		t.Fatalf("system A section must carry the extra event:\n%s", got)
	}
}

func TestTrailDiff_LengthMismatchReportsMissingStream(t *testing.T) {
	t.Parallel()

	a := []normalizedStream{
		{ordinal: 0, entries: []trailEntry{{eventType: "task.created", version: 1}}},
	}

	if got := trailDiff(a, nil); !strings.Contains(got, "stream #0 exists only in system A") {
		t.Fatalf("trailDiff missing-only-in-A:\n%s", got)
	}

	if got := trailDiff(nil, a); !strings.Contains(got, "stream #0 exists only in system B") {
		t.Fatalf("trailDiff missing-only-in-B:\n%s", got)
	}
}

func TestOrdinalCaveatMentionsFirstAppearance(t *testing.T) {
	t.Parallel()

	if !strings.Contains(ordinalCaveat, "first-appearance ordinal") {
		t.Fatalf("caveat must name the ordinal scheme: %s", ordinalCaveat)
	}
}
