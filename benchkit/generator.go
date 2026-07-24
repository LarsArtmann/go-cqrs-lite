package benchkit

import (
	"fmt"
	"math/rand/v2"
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
type Generator struct {
	rng   *rand.Rand
	size  int
	codec codec.Codec
	mu    sync.Mutex
}

// NewGenerator creates a Generator with the given seed, target payload size,
// and codec for size measurement. If size <= 0, defaults to 256 bytes.
// If codec is nil, defaults to JSONCodec.
func NewGenerator(seed int64, size int, c codec.Codec) *Generator {
	if size <= 0 {
		size = 256
	}

	if c == nil {
		c = codec.JSONCodec{}
	}

	return &Generator{
		rng:   rand.New(rand.NewPCG(uint64(seed), 0)),
		size:  size,
		codec: c,
	}
}

// Payload returns a BenchPayload populated with deterministic random data.
// The Padding field is sized so the codec-encoded payload matches the target
// byte size as closely as possible (within a few bytes).
func (g *Generator) Payload() BenchPayload {
	g.mu.Lock()
	defer g.mu.Unlock()

	p := BenchPayload{
		ID:       fmt.Sprintf("01HX%012d", g.rng.IntN(1000000000000)),
		Name:     fmt.Sprintf("Order-%d", g.rng.IntN(100000)),
		Value:    g.rng.Float64()*990 + 10,
		Items:    g.rng.IntN(20) + 1,
		Tags:     generateTags(g.rng),
		Metadata: generateMeta(g.rng),
	}

	p.Padding = g.computePadding(p)

	return p
}

// computePadding calculates how many padding characters are needed so the
// codec-encoded payload matches the target size. It uses the configured codec
// (not hardcoded JSON) so CBOR payloads are sized correctly.
//
// The algorithm: encode without padding to get the base size, then encode with
// a 1-char padding to measure the per-field overhead (key + type header).
// Each additional padding char adds exactly 1 byte.
func (g *Generator) computePadding(p BenchPayload) string {
	base, err := g.codec.Encode(p)
	if err != nil || g.size <= len(base) {
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
	needed := g.size - len(withOne) + 1
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
