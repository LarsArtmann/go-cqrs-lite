package irohengine_test

import (
	"context"
	"testing"

	"github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// engine_capability_forwarding_test.go pins the optional-capability forwarding
// policy documented in engine_passthrough.go. `Replicated` was caught dropping
// graph dispatch once (fixed 2026-08-16); these tests fail if a future refactor
// silently changes which optional capabilities the wrapper exposes.

func newReplicatedForPolicyTest(t *testing.T) metaengine.Engine {
	t.Helper()

	eng := irohengine.Replicated(metaengine.NewMemoryEngine())
	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

func TestReplicatedImplementsCloser(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	eng := newReplicatedForPolicyTest(t)

	_, isCloser := eng.(metaengine.Closer)
	g.Expect(isCloser).To(gomega.BeTrue())
	g.Expect(eng.Close()).To(gomega.Succeed())
}

// TestReplicatedDoesNotExposeWritePathCapabilities pins the DELIBERATE
// non-forwarding of capabilities whose forwarding would silently diverge state
// across peers: Transactional transactions would replicate per-write (or never,
// if the callback captures the local engine), and StreamAppend/AtomicAppender
// writes have no WriteOp wire kind. System adapters fall back to the replicated
// LogBackend path instead — the degraded route is the converging one.
func TestReplicatedDoesNotExposeWritePathCapabilities(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	eng := newReplicatedForPolicyTest(t)

	_, isTransactional := eng.(metaengine.Transactional)
	g.Expect(isTransactional).
		To(gomega.BeFalse(), "RunInTx must not be exposed: see engine_passthrough.go policy")

	_, isStreamLog := eng.(metaengine.StreamLogBackend)
	g.Expect(isStreamLog).
		To(gomega.BeFalse(), "StreamAppend must not be exposed: see engine_passthrough.go policy")

	_, isSeqSeek := eng.(metaengine.SeqSeekableStreamLog)
	g.Expect(isSeqSeek).To(gomega.BeFalse())

	_, isAtomic := eng.(metaengine.AtomicAppender)
	g.Expect(isAtomic).To(gomega.BeFalse())
}

// TestReplicatedDoesNotExposeProbers pins the DELIBERATE non-forwarding of
// Prober/TransactMeasurer: a forwarded probe measures local-engine RTT (~0) and
// live calibration would override the honest replication-derived NetworkRTT
// maintained by the wrapper's own latency tracker.
func TestReplicatedDoesNotExposeProbers(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	eng := newReplicatedForPolicyTest(t)

	_, isProber := eng.(metaengine.Prober)
	g.Expect(isProber).
		To(gomega.BeFalse(), "probes must not be forwarded: see engine_passthrough.go policy")

	_, isMeasurer := eng.(metaengine.TransactMeasurer)
	g.Expect(isMeasurer).To(gomega.BeFalse())
}

// TestReplicatedVectorPassthrough verifies the VectorBackend local passthrough
// end-to-end (the "every engine" CHANGELOG claim includes iroh): inserts and
// k-NN searches through the wrapper execute against the local engine, and
// VectorSearchPath forwards the local engine's reported path. Filtered k-NN
// and VectorCounter are deliberately NOT promoted (engine_passthrough.go
// forwarding policy — no wire kinds for those writes, no size introspection
// through the wrapper).
func TestReplicatedVectorPassthrough(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	eng := newReplicatedForPolicyTest(t)
	ctx := context.Background()

	vb, isVB := eng.(metaengine.VectorBackend)
	g.Expect(isVB).To(gomega.BeTrue())

	g.Expect(vb.VectorInsert(ctx, "docs",
		metaengine.Embedding{ID: "a", Values: []float32{1, 0}})).To(gomega.Succeed())
	g.Expect(vb.VectorInsert(ctx, "docs",
		metaengine.Embedding{ID: "b", Values: []float32{0, 1}})).To(gomega.Succeed())

	results, err := vb.VectorSearch(ctx, "docs", []float32{1, 0}, 2, "cosine")
	g.Expect(err).To(gomega.Succeed())
	g.Expect(results).To(gomega.HaveLen(2))
	g.Expect(results[0].ID).To(gomega.Equal("a"), "nearest-first ordering must hold through the wrapper")

	vp, isVP := eng.(metaengine.VectorPathReporter)
	g.Expect(isVP).To(gomega.BeTrue())
	g.Expect(vp.VectorSearchPath()).To(gomega.Equal(metaengine.VectorPathScan),
		"memory local engine reports go-scan; the wrapper must forward it")

	_, isFiltered := eng.(metaengine.VectorFilterBackend)
	g.Expect(isFiltered).
		To(gomega.BeFalse(), "VectorSearchFiltered must not be promoted: see engine_passthrough.go policy")

	_, isCounter := eng.(metaengine.VectorCounter)
	g.Expect(isCounter).
		To(gomega.BeFalse(), "VectorCounter must not be promoted: see engine_passthrough.go policy")
}
