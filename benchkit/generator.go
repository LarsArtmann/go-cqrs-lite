package benchkit

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// BenchPayload is the synthetic event payload used by the benchmark generator.
// It mimics a realistic e-commerce order event with typed fields.
// The Padding field ensures payloads reach approximately the target byte size
// while remaining valid JSON/CBOR at any size.
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
type Generator struct {
	rng  *rand.Rand
	size int
}

// NewGenerator creates a Generator with the given seed and target payload size.
// If size <= 0, defaults to 256 bytes.
func NewGenerator(seed int64, size int) *Generator {
	if size <= 0 {
		size = 256
	}

	return &Generator{
		rng:  rand.New(rand.NewPCG(uint64(seed), 0)),
		size: size,
	}
}

// Payload returns a BenchPayload populated with deterministic random data.
// The Padding field is sized so the JSON encoding is approximately the
// configured target size. The payload is always valid JSON at any size.
func (g *Generator) Payload() BenchPayload {
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

// computePadding estimates how many padding characters are needed so the
// JSON encoding of the payload is approximately the target size.
func (g *Generator) computePadding(p BenchPayload) string {
	const paddingKeyOverhead = len(`,"_padding":""`)

	base := estimateJSONSize(p)
	if g.size <= base+paddingKeyOverhead {
		return ""
	}

	needed := g.size - base - paddingKeyOverhead
	if needed <= 0 {
		return ""
	}

	return strings.Repeat("x", needed)
}

// estimateJSONSize returns a rough byte count of the payload without Padding.
func estimateJSONSize(p BenchPayload) int {
	const baseTemplate = `{"id":"01HX000000000000","name":"Order-00000","value":000.00,"items":0,"tags":["a"],"metadata":{"source":"web","session":"sess-000000"}}`

	size := len(baseTemplate)

	for _, tag := range p.Tags {
		size += len(tag) - 1
	}

	for k, v := range p.Metadata {
		size += len(k) + len(v)
	}

	return size
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
