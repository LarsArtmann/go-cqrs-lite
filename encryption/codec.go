package encryption

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
)

const EncryptionEncoding codec.Encoding = "encrypted"

type encryptingCodec struct {
	inner     codec.Codec
	encrypter Encrypter
	decrypter Decrypter
}

var _ codec.Codec = (*encryptingCodec)(nil)

func NewCodec(inner codec.Codec, enc EncrypterDecrypter) *encryptingCodec {
	return &encryptingCodec{inner: inner, encrypter: enc, decrypter: enc}
}

func (c *encryptingCodec) Encoding() codec.Encoding { return EncryptionEncoding }

func (c *encryptingCodec) Encode(v any) ([]byte, error) {
	plaintext, err := c.inner.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("encryption codec: inner encode: %w", err)
	}

	if len(plaintext) == 0 {
		return nil, nil
	}

	ct, err := c.encrypter.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encryption codec: encrypt: %w", err)
	}

	return ct.Bytes(), nil
}

func (c *encryptingCodec) Decode(data []byte, v any) error {
	if len(data) == 0 {
		return c.inner.Decode(data, v) //nolint:wrapcheck
	}

	plaintext, err := c.decrypter.Decrypt(Ciphertext(data))
	if err != nil {
		return fmt.Errorf("encryption codec: decrypt: %w", err)
	}

	return c.inner.Decode(plaintext, v)
}
