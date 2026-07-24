package benchkit

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

// BenchPayload is the synthetic event payload used by the benchmark generator.
// It mimics a realistic e-commerce order event with typed fields.
// The Padding field ensures the encoded payload matches the target byte size
// as closely as possible (within a few bytes for any codec).
type BenchPayload struct {
	ID       string            `cbor:"1,keyasint"           json:"id"`
	Name     string            `cbor:"2,keyasint"           json:"name"`
	Value    float64           `cbor:"3,keyasint"           json:"value"`
	Items    int               `cbor:"4,keyasint"           json:"items"`
	Tags     []string          `cbor:"5,keyasint"           json:"tags"`
	Metadata map[string]string `cbor:"6,keyasint"           json:"metadata"`
	Padding  string            `cbor:"7,keyasint,omitempty" json:"_padding,omitempty"`
}

// Generator produces deterministic synthetic payloads for benchmarking.
// Same seed always produces the same data, enabling reproducible runs.
// The codec determines how payload sizing is calculated — use the same
// codec that the benchmark will encode events with.
//
// A Generator may hold a single target size (uniform payloads) or a
// distribution of sizes (mixed payloads). When mixed, each Payload() call
// picks a size uniformly at random from the distribution, modelling real
// workloads where event sizes vary from tiny status changes to large events
// carrying embedded collections.
type Generator struct {
	rng   *rand.Rand
	sizes []int
	codec codec.Codec
	mu    sync.Mutex
}

// NewGenerator creates a Generator with the given seed, target payload size,
// and codec for size measurement. If size <= 0, defaults to 256 bytes.
// If codec is nil, defaults to JSONCodec.
func NewGenerator(seed int64, size int, c codec.Codec) *Generator {
	return newGenerator(seed, []int{size}, c)
}

// NewMixedGenerator creates a Generator that produces payloads of varying
// sizes, picking uniformly at random from sizes on each Payload() call. This
// models real workloads where event payloads vary widely (small status
// changes, medium domain events, large events with embedded collections). At
// least one size must be provided; sizes <= 0 default to 256. A
// single-element slice behaves identically to NewGenerator.
func NewMixedGenerator(seed int64, sizes []int, c codec.Codec) *Generator {
	return newGenerator(seed, sizes, c)
}

func newGenerator(seed int64, sizes []int, c codec.Codec) *Generator {
	cleaned := slices.Clone(sizes)
	if len(cleaned) == 0 {
		cleaned = []int{256}
	}

	for i, s := range cleaned {
		if s <= 0 {
			cleaned[i] = 256
		}
	}

	if c == nil {
		c = codec.JSONCodec{}
	}

	return &Generator{
		rng:   rand.New(rand.NewPCG(uint64(seed), 0)),
		sizes: cleaned,
		codec: c,
	}
}

// MeanSize returns the arithmetic mean of the payload-size distribution. For
// a single-size Generator this equals that size; for a mixed Generator it is
// the expected per-event byte count reported in Result.PayloadBytes.
func (g *Generator) MeanSize() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	sum := 0

	for _, s := range g.sizes {
		sum += s
	}

	return sum / len(g.sizes)
}

// SizeDistribution returns a copy of the configured size distribution.
// A single-element slice means uniform payloads.
func (g *Generator) SizeDistribution() []int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return slices.Clone(g.sizes)
}

// Payload returns a BenchPayload populated with deterministic random data.
// The Padding field is sized so the codec-encoded payload matches the target
// byte size as closely as possible (within a few bytes). When the Generator
// holds a size distribution, the target is picked uniformly at random each
// call.
func (g *Generator) Payload() BenchPayload {
	g.mu.Lock()
	defer g.mu.Unlock()

	target := g.sizes[0]
	if len(g.sizes) > 1 {
		target = g.sizes[g.rng.IntN(len(g.sizes))]
	}

	p := BenchPayload{
		ID:       fmt.Sprintf("01HX%012d", g.rng.IntN(1000000000000)),
		Name:     fmt.Sprintf("Order-%d", g.rng.IntN(100000)),
		Value:    g.rng.Float64()*990 + 10,
		Items:    g.rng.IntN(20) + 1,
		Tags:     generateTags(g.rng),
		Metadata: generateMeta(g.rng),
	}

	p.Padding = g.computePadding(p, target)

	return p
}

// computePadding calculates how many padding characters are needed so the
// codec-encoded payload matches the target size. It uses the configured codec
// (not hardcoded JSON) so CBOR payloads are sized correctly.
//
// The algorithm: encode without padding to get the base size, then encode with
// a 1-char padding to measure the per-field overhead (key + type header).
// Each additional padding char adds exactly 1 byte.
func (g *Generator) computePadding(p BenchPayload, target int) string {
	base, err := g.codec.Encode(p)
	if err != nil || target <= len(base) {
		return ""
	}

	// Probe with 1-char padding to measure field key + type header overhead.
	probe := p
	probe.Padding = "x"

	withOne, err := g.codec.Encode(probe)
	if err != nil {
		return ""
	}

	// withOne = base + overhead + 1 char. Each extra char = +1 byte.
	// Solve: target = withOne + (N - 1), so N = target - withOne + 1.
	needed := target - len(withOne) + 1
	if needed <= 0 {
		return ""
	}

	return strings.Repeat("x", needed)
}

func generateTags(rng *rand.Rand) []string {
	allTags := []string{"priority", "express", "fragile", "gift", "international", "refundable"}
	count := rng.IntN(3) + 1

	tags := make([]string, 0, count)
	used := make(map[int]bool)

	for len(tags) < count {
		idx := rng.IntN(len(allTags))
		if !used[idx] {
			tags = append(tags, allTags[idx])
			used[idx] = true
		}
	}

	return tags
}

func generateMeta(rng *rand.Rand) map[string]string {
	sources := []string{"web", "mobile", "api", "marketplace"}

	return map[string]string{
		"source":  sources[rng.IntN(len(sources))],
		"session": fmt.Sprintf("sess-%d", rng.IntN(1000000)),
	}
}
