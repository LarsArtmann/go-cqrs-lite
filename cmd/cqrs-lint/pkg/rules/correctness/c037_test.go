package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC037(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sources   map[string]string
		wantCount int
	}{
		{
			"CodecMismatch",
			map[string]string{
				"repo.go": `package main

import "github.com/larsartmann/go-cqrs-lite/decider/v4"

func newRepo(store Store, bus Bus) *decider.Repository[State] {
	repo, _ := decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	return repo
}
`,
				"snap.go": `package main

func newSnap(store Store) {
	snap, _ := snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
	_ = snap
}
`,
			},
			1,
		},
		{
			"GenericWithCodecMismatch",
			map[string]string{
				"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
}
`,
			},
			1,
		},
		{
			"SameCodecNoFinding",
			map[string]string{
				"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.CBORCodec{}))
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
}
`,
			},
			0,
		},
		{
			"NoRepoCodecNoFinding",
			map[string]string{
				"all.go": `package main

func setup(store Store) {
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
}
`,
			},
			0,
		},
		{
			"NonSnapshotNewTypedStoreNoFinding",
			map[string]string{
				"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	// otherpkg is NOT snapshot — must not fire.
	_, _ = otherpkg.NewTypedStore(store, codec.CBORCodec{})
}
`,
			},
			0,
		},
		{
			"CommandStoreCodecMismatch",
			map[string]string{
				"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_, _ = command.NewTypedCommandStore[Cmd](store, codec.CBORCodec{})
}
`,
			},
			1,
		},
		{
			"QueryStoreCodecMismatch",
			map[string]string{
				"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_, _ = query.NewTypedQueryStore[Q](store, codec.CBORCodec{})
}
`,
			},
			1,
		},
		{
			"KVStoreCodecMismatch",
			map[string]string{
				"all.go": `package main

func setup(backend Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_ = kv.NewTypedStore[View, string](backend, kv.WithTypedCodec[View, string](codec.CBORCodec{}))
}
`,
			},
			1,
		},
		{
			"AllStoresSameCodecNoFinding",
			map[string]string{
				"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.CBORCodec{}))
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
	_, _ = command.NewTypedCommandStore[Cmd](store, codec.CBORCodec{})
	_, _ = query.NewTypedQueryStore[Q](store, codec.CBORCodec{})
	_ = kv.NewTypedStore[View, string](store, kv.WithTypedCodec[View, string](codec.CBORCodec{}))
}
`,
			},
			0,
		},
		{
			"MultipleMismatches",
			map[string]string{
				"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
	_, _ = command.NewTypedCommandStore[Cmd](store, codec.CBORCodec{})
}
`,
			},
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := analyzer.BuildContextFromSource(t, tt.sources)
			findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
			ruletest.AssertRule(t, findings, "C037", tt.wantCount)
		})
	}
}
