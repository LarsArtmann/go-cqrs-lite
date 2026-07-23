package metaengine

import (
	"fmt"
	"sort"
	"strings"
)

// ReadModel owns the write side of a projection: how events update its state.
// Multiple queries can read from the same ReadModel with different read patterns.
// This eliminates fold duplication and write amplification when several queries
// read from the same underlying projection.
type ReadModel struct {
	Name  string
	Folds []Fold
	ADT   ADT
}

// Model creates a ReadModel from a name and fold declarations.
// Returns an error if fold classification fails (e.g., only OnSkip folds).
func Model(name string, folds ...Fold) (ReadModel, error) {
	adt, err := classifyADT(folds)
	if err != nil {
		return ReadModel{}, fmt.Errorf("metaengine.Model(%q): %w", name, err)
	}

	return ReadModel{Name: name, Folds: folds, ADT: adt}, nil
}

// MustModel creates a ReadModel, panicking on error.
// Intended for package-level var declarations where failure is a programming error.
func MustModel(name string, folds ...Fold) ReadModel {
	rm, err := Model(name, folds...)
	if err != nil {
		panic(err)
	}

	return rm
}

// EventTypes returns the set of event types this model reacts to.
// Used for projection.Projection compatibility.
func (rm ReadModel) EventTypes() []string {
	seen := make(map[string]struct{}, len(rm.Folds))

	for _, f := range rm.Folds {
		if f.Kind != FoldSkip {
			seen[f.EventType] = struct{}{}
		}
	}

	types := make([]string, 0, len(seen))
	for t := range seen {
		types = append(types, t)
	}

	sort.Strings(types)

	return types
}

func (rm ReadModel) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s (%d folds)", rm.Name, rm.ADT, len(rm.Folds))

	return b.String()
}
