package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC037_CodecMismatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
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
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 1)
}

func TestC037_GenericWithCodecMismatch(t *testing.T) {
	t.Parallel()

	// decider.WithCodec[State](...) explicit type arg variant.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 1)
}

func TestC037_SameCodecNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.CBORCodec{}))
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 0)
}

func TestC037_NoRepoCodecNoFinding(t *testing.T) {
	t.Parallel()

	// Snapshot store without any repository codec — cannot determine mismatch.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"all.go": `package main

func setup(store Store) {
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 0)
}

func TestC037_NonSnapshotNewTypedStoreNoFinding(t *testing.T) {
	t.Parallel()

	// A different package's NewTypedStore should not be confused with snapshot.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	// otherpkg is NOT snapshot — must not fire.
	_, _ = otherpkg.NewTypedStore(store, codec.CBORCodec{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 0)
}

func TestC037_CommandStoreCodecMismatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_, _ = command.NewTypedCommandStore[Cmd](store, codec.CBORCodec{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 1)
}

func TestC037_QueryStoreCodecMismatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_, _ = query.NewTypedQueryStore[Q](store, codec.CBORCodec{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 1)
}

func TestC037_KVStoreCodecMismatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"all.go": `package main

func setup(backend Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_ = kv.NewTypedStore[View, string](backend, kv.WithTypedCodec[View, string](codec.CBORCodec{}))
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 1)
}

func TestC037_AllStoresSameCodecNoFinding(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
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
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 0)
}

func TestC037_MultipleMismatches(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"all.go": `package main

func setup(store Store, bus Bus) {
	_, _ = decider.NewRepository(store, bus, decider.Decider[State]{},
		decider.WithCodec[State](codec.JSONCodec{}))
	_, _ = snapshot.NewTypedStore[State, string](store, codec.CBORCodec{})
	_, _ = command.NewTypedCommandStore[Cmd](store, codec.CBORCodec{})
}
`,
	})

	findings := ruletest.RunDetector(t, correctness.NewC037Detector(ctx))
	ruletest.AssertRule(t, findings, "C037", 2)
}
