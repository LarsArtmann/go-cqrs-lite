package codec_test

import (
	"testing"

	"github.com/fxamacker/cbor/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

func TestCOSEConstants(t *testing.T) {
	g := NewWithT(t)

	g.Expect(codec.COSEHeaderAlg).To(Equal(int64(1)))
	g.Expect(codec.COSEHeaderKid).To(Equal(int64(4)))
	g.Expect(codec.COSEHeaderIV).To(Equal(int64(5)))
	g.Expect(codec.COSEAlgHMACSHA256).To(Equal(int64(5)))
	g.Expect(codec.COSEAlgEd25519).To(Equal(int64(-19)))
	g.Expect(codec.COSEAlgAES256GCM).To(Equal(int64(3)))
	g.Expect(codec.COSEAlgChaCha20Poly1305).To(Equal(int64(24)))
}

func TestMarshalCOSEProtectedHeader(t *testing.T) {
	g := NewWithT(t)

	headers := map[int64]any{
		codec.COSEHeaderAlg: codec.COSEAlgEd25519,
		codec.COSEHeaderKid: []byte("key-1"),
	}

	data, err := codec.MarshalCOSEProtectedHeader(headers)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).ToNot(BeEmpty())

	decoded, err := codec.UnmarshalCOSEProtectedHeader(data)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decoded).To(Equal(headers))
}

func TestMarshalUnmarshalCOSESign1(t *testing.T) {
	g := NewWithT(t)

	protected, err := codec.MarshalCOSEProtectedHeader(map[int64]any{
		codec.COSEHeaderAlg: codec.COSEAlgEd25519,
	})
	g.Expect(err).ToNot(HaveOccurred())

	msg := codec.COSESign1{
		Protected: protected,
		Unprotected: map[int64]any{
			codec.COSEHeaderKid: []byte("key-1"),
		},
		Payload:   []byte("hello cose"),
		Signature: []byte{1, 2, 3, 4},
	}

	data, err := codec.MarshalCOSESign1(msg)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).ToNot(BeEmpty())

	decoded, err := codec.UnmarshalCOSESign1(data)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decoded.Protected).To(Equal(msg.Protected))
	g.Expect(decoded.Unprotected).To(Equal(msg.Unprotected))
	g.Expect(decoded.Payload).To(Equal(msg.Payload))
	g.Expect(decoded.Signature).To(Equal(msg.Signature))
}

func TestMarshalUnmarshalCOSESign1DetachedPayload(t *testing.T) {
	g := NewWithT(t)

	msg := codec.COSESign1{
		Protected:   nil,
		Unprotected: nil,
		Payload:     nil,
		Signature:   []byte{5, 6, 7, 8},
	}

	data, err := codec.MarshalCOSESign1(msg)
	g.Expect(err).ToNot(HaveOccurred())

	decoded, err := codec.UnmarshalCOSESign1(data)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decoded.Payload).To(BeNil())
	g.Expect(decoded.Signature).To(Equal(msg.Signature))
}

func TestMarshalUnmarshalCOSEEncrypt0(t *testing.T) {
	g := NewWithT(t)

	protected, err := codec.MarshalCOSEProtectedHeader(map[int64]any{
		codec.COSEHeaderAlg: codec.COSEAlgChaCha20Poly1305,
	})
	g.Expect(err).ToNot(HaveOccurred())

	msg := codec.COSEEncrypt0{
		Protected: protected,
		Unprotected: map[int64]any{
			codec.COSEHeaderIV: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		Ciphertext: []byte{13, 14, 15, 16},
	}

	data, err := codec.MarshalCOSEEncrypt0(msg)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).ToNot(BeEmpty())

	decoded, err := codec.UnmarshalCOSEEncrypt0(data)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(decoded.Protected).To(Equal(msg.Protected))
	g.Expect(decoded.Unprotected).To(Equal(msg.Unprotected))
	g.Expect(decoded.Ciphertext).To(Equal(msg.Ciphertext))
}

func TestCOSESigStructure(t *testing.T) {
	g := NewWithT(t)

	protected, err := codec.MarshalCOSEProtectedHeader(map[int64]any{
		codec.COSEHeaderAlg: codec.COSEAlgEd25519,
	})
	g.Expect(err).ToNot(HaveOccurred())

	data, err := codec.SigStructure(protected, []byte("aad"), []byte("payload"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).ToNot(BeEmpty())

	var arr []cbor.RawMessage
	if err := cbor.Unmarshal(data, &arr); err != nil {
		g.Expect(err).ToNot(HaveOccurred())
	}
	g.Expect(len(arr)).To(Equal(4))

	var context string
	g.Expect(cbor.Unmarshal(arr[0], &context)).To(Succeed())
	g.Expect(context).To(Equal("Signature1"))

	var bodyProtected []byte
	g.Expect(cbor.Unmarshal(arr[1], &bodyProtected)).To(Succeed())
	g.Expect(bodyProtected).To(Equal(protected))

	var externalAAD []byte
	g.Expect(cbor.Unmarshal(arr[2], &externalAAD)).To(Succeed())
	g.Expect(externalAAD).To(Equal([]byte("aad")))

	var payload []byte
	g.Expect(cbor.Unmarshal(arr[3], &payload)).To(Succeed())
	g.Expect(payload).To(Equal([]byte("payload")))
}

func TestCOSEEncStructure0(t *testing.T) {
	g := NewWithT(t)

	protected, err := codec.MarshalCOSEProtectedHeader(map[int64]any{
		codec.COSEHeaderAlg: codec.COSEAlgChaCha20Poly1305,
	})
	g.Expect(err).ToNot(HaveOccurred())

	data, err := codec.EncStructure0(protected, []byte("aad"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(data).ToNot(BeEmpty())

	var arr []cbor.RawMessage
	g.Expect(cbor.Unmarshal(data, &arr)).To(Succeed())
	g.Expect(len(arr)).To(Equal(3))

	var context string
	g.Expect(cbor.Unmarshal(arr[0], &context)).To(Succeed())
	g.Expect(context).To(Equal("Encrypt0"))

	var bodyProtected []byte
	g.Expect(cbor.Unmarshal(arr[1], &bodyProtected)).To(Succeed())
	g.Expect(bodyProtected).To(Equal(protected))

	var externalAAD []byte
	g.Expect(cbor.Unmarshal(arr[2], &externalAAD)).To(Succeed())
	g.Expect(externalAAD).To(Equal([]byte("aad")))
}

func TestUnmarshalCOSESign1InvalidLength(t *testing.T) {
	g := NewWithT(t)

	// A 3-element array is not a valid COSE_Sign1.
	data, err := codec.CBOREncMode().Marshal([]any{[]byte{}, map[int64]any{}, []byte{}})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = codec.UnmarshalCOSESign1(data)
	g.Expect(err).To(MatchError(ContainSubstring("COSE_Sign1 has 3 elements")))
}

func TestUnmarshalCOSEEncrypt0InvalidLength(t *testing.T) {
	g := NewWithT(t)

	data, err := codec.CBOREncMode().Marshal([]any{[]byte{}, map[int64]any{}, []byte{}, []byte{}})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = codec.UnmarshalCOSEEncrypt0(data)
	g.Expect(err).To(MatchError(ContainSubstring("COSE_Encrypt0 has 4 elements")))
}

func TestCOSESign1String(t *testing.T) {
	g := NewWithT(t)

	msg := codec.COSESign1{
		Protected: []byte{},
		Signature: []byte{1},
	}

	data, err := codec.MarshalCOSESign1(msg)
	g.Expect(err).ToNot(HaveOccurred())

	diag := codec.COSESign1String(data)
	g.Expect(diag).To(ContainSubstring("h'"))
}

func TestCOSEEncrypt0String(t *testing.T) {
	g := NewWithT(t)

	msg := codec.COSEEncrypt0{
		Protected:  []byte{},
		Ciphertext: []byte{1},
	}

	data, err := codec.MarshalCOSEEncrypt0(msg)
	g.Expect(err).ToNot(HaveOccurred())

	diag := codec.COSEEncrypt0String(data)
	g.Expect(diag).To(ContainSubstring("h'"))
}
