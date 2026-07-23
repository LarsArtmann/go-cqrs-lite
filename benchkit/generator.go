package benchkit

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
)

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

// Payload returns a JSON-encoded byte slice of approximately the configured
// size. The payload is a realistic domain event with typed fields.
func (g *Generator) Payload() []byte {
	// Generate a base payload with realistic fields
	name := fmt.Sprintf("Order-%d", g.rng.IntN(100000))
	value := g.rng.Float64()*990 + 10 // 10.00 - 1000.00
	items := g.rng.IntN(20) + 1

	tags := generateTags(g.rng)
	meta := generateMeta(g.rng)

	payload := map[string]any{
		"id":       fmt.Sprintf("01HX%012d", g.rng.IntN(1000000000000)),
		"name":     name,
		"value":    value,
		"items":    items,
		"tags":     tags,
		"metadata": meta,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		// json.Marshal on a map[string]any with basic types never fails
		return []byte(`{}`)
	}

	// Pad or trim to target size
	if len(data) >= g.size {
		return data[:g.size]
	}

	padded := make([]byte, g.size)
	copy(padded, data)

	// Fill remainder with deterministic padding
	for i := len(data); i < g.size; i++ {
		padded[i] = byte('a' + g.rng.IntN(26))
	}

	return padded
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
