package systemscenario

import (
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// recordingSink captures Logf output so the advisory warning is assertable
// without a real testing.TB.
type recordingSink struct {
	lines []string
}

func (r *recordingSink) Logf(format string, args ...any) {
	r.lines = append(r.lines, fmt.Sprintf(format, args...))
}

func TestWarnOnEmptyDomain(t *testing.T) {
	t.Parallel()

	t.Run("empty config warns", func(t *testing.T) {
		t.Parallel()

		sink := &recordingSink{}
		warnOnEmptyDomain(sink, system.DomainConfig{})

		if len(sink.lines) != 1 {
			t.Fatalf("got %d warnings, want 1: %v", len(sink.lines), sink.lines)
		}
		for _, want := range []string{"DomainConfig is empty", "Adopt"} {
			if !strings.Contains(sink.lines[0], want) {
				t.Errorf("warning %q missing %q", sink.lines[0], want)
			}
		}
	})

	t.Run("wired config stays silent", func(t *testing.T) {
		t.Parallel()

		for name, cfg := range map[string]system.DomainConfig{
			"commands":    {Commands: func(*system.System) {}},
			"queries":     {Queries: func(*system.System) {}},
			"timers":      {Timers: func(*system.System) {}},
			"projections": {Projections: []system.ProjectionDeclaration{nil}},
			"evolutions":  {Evolutions: []system.EvolutionSpec{nil}},
			"events":      {Events: []event.Type{event.Type("x")}},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				sink := &recordingSink{}
				warnOnEmptyDomain(sink, cfg)
				if len(sink.lines) != 0 {
					t.Errorf("got %d warnings, want 0: %v", len(sink.lines), sink.lines)
				}
			})
		}
	})
}
