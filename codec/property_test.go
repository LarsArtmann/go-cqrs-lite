package codec_test

import (
	"testing"

	"github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

type roundtripPayload struct {
	Name      string
	Count     int64
	Active    bool
	Tags      []string
	Score     float64
	Internal  nestedData
	CreatedNs int64
}

type nestedData struct {
	ID    string
	Value int64
}

func genPayload(t *rapid.T) *roundtripPayload {
	return &roundtripPayload{
		Name:      rapid.StringN(0, 50, 100).Draw(t, "Name"),
		Count:     rapid.Int64Min(0).Draw(t, "Count"),
		Active:    rapid.Bool().Draw(t, "Active"),
		Tags:      rapid.SliceOfN(rapid.StringN(0, 20, 50), 0, 10).Draw(t, "Tags"),
		Score:     rapid.Float64().Draw(t, "Score"),
		CreatedNs: rapid.Int64Min(0).Draw(t, "CreatedNs"),
		Internal: nestedData{
			ID:    rapid.StringN(1, 30, 50).Draw(t, "nestedID"),
			Value: rapid.Int64Min(0).Draw(t, "nestedValue"),
		},
	}
}

func TestProperty_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		original := genPayload(t)

		c := codec.JSONCodec{}
		data, err := c.Encode(original)
		g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

		var decoded roundtripPayload
		g.Expect(c.Decode(data, &decoded)).To(gomega.Succeed())

		g.Expect(decoded).To(gomega.Equal(*original))
	})
}

func TestProperty_CBORRoundtrip(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		original := genPayload(t)

		c := codec.CBORCodec{}
		data, err := c.Encode(original)
		g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

		var decoded roundtripPayload
		g.Expect(c.Decode(data, &decoded)).To(gomega.Succeed())

		g.Expect(decoded).To(gomega.Equal(*original))
	})
}

func TestProperty_CBORDeterministic(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		original := genPayload(t)

		c := codec.CBORCodec{}
		first, err := c.Encode(original)
		g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

		second, err := c.Encode(original)
		g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

		g.Expect(second).To(gomega.Equal(first), "CBOR encoding must be deterministic")
	})
}

func TestProperty_CBORCompactRoundtrip(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		g := gomega.NewWithT(t)
		original := genPayload(t)

		c := codec.CBORCompactCodec{}
		data, err := c.Encode(original)
		g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

		var decoded roundtripPayload
		g.Expect(c.Decode(data, &decoded)).To(gomega.Succeed())

		g.Expect(decoded).To(gomega.Equal(*original))
	})
}
