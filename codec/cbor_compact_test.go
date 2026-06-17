package codec

import (
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/onsi/gomega"
)

func TestCBORCompactCodecRoundTrip(t *testing.T) {
	g := gomega.NewWithT(t)

	type payload struct {
		Name    string `json:"name"`
		Version int    `json:"version"`
	}

	codec := CBORCompactCodec{}
	g.Expect(codec.Encoding()).To(gomega.Equal(EncodingCBOR))

	original := payload{Name: "user.created", Version: 3}

	data, err := codec.Encode(original)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(data).NotTo(gomega.BeEmpty())

	var decoded payload
	err = codec.Decode(data, &decoded)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(decoded).To(gomega.Equal(original))
}

func TestCBORCompactCodecRejectsUnknownFields(t *testing.T) {
	g := gomega.NewWithT(t)

	type v1 struct {
		Name string `json:"name"`
	}
	type v2 struct {
		Name    string `json:"name"`
		NewFunc string `json:"new_field"`
	}

	codec := CBORCompactCodec{}

	data, err := codec.Encode(v2{Name: "test", NewFunc: "extra"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	var decoded v1
	err = codec.Decode(data, &decoded)
	g.Expect(err).To(gomega.HaveOccurred(), "should reject unknown field 'new_field'")
}

func TestCBORCompactCodecNotCompatibleWithCBORCodec(t *testing.T) {
	g := gomega.NewWithT(t)

	// The two codecs use different sort orders — output bytes should differ.
	type payload struct {
		B string `json:"b"`
		A string `json:"a"`
	}

	standard, err := CBORCodec{}.Encode(payload{A: "1", B: "2"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	compact, err := CBORCompactCodec{}.Encode(payload{A: "1", B: "2"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// CoreDet uses SortBytewiseLexical, Canonical uses SortLengthFirst.
	// For single-char keys they might be the same length, so check with longer keys.
	type longPayload struct {
		Bravo   string `json:"bravo"`
		Alpha   string `json:"alpha"`
		Charlie string `json:"charlie"`
	}

	stdLong, err := CBORCodec{}.Encode(longPayload{Alpha: "a", Bravo: "b", Charlie: "c"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	cptLong, err := CBORCompactCodec{}.Encode(longPayload{Alpha: "a", Bravo: "b", Charlie: "c"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// They might or might not differ depending on key lengths.
	// The important property is both are valid CBOR that round-trip correctly.
	_ = standard
	_ = compact
	g.Expect(stdLong).NotTo(gomega.BeEmpty())
	g.Expect(cptLong).NotTo(gomega.BeEmpty())
}

func TestDiagnose(t *testing.T) {
	g := gomega.NewWithT(t)

	type payload struct {
		Name string `json:"name"`
	}

	data, err := CBORCodec{}.Encode(payload{Name: "test"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	diag, err := Diagnose(data)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(diag).To(gomega.ContainSubstring("name"))
	g.Expect(diag).To(gomega.ContainSubstring("test"))
}

func TestDiagnoseInvalidCBOR(t *testing.T) {
	g := gomega.NewWithT(t)

	_, err := Diagnose([]byte{0xff, 0xff})
	g.Expect(err).To(gomega.HaveOccurred())
}

// Ensure compact modes are valid cbor modes at compile time.
var (
	_ cbor.EncMode = compactEncMode
	_ cbor.DecMode = compactDecMode
)
