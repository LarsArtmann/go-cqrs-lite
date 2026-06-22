package encryption

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
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
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"encryption.codec_inner_encode",
			"inner encode",
		)
	}

	if len(plaintext) == 0 {
		return nil, nil
	}

	ciphertext, err := c.encrypter.Encrypt(plaintext)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"encryption.codec_encrypt",
			"encrypt",
		)
	}

	return ciphertext.Bytes(), nil
}

func (c *encryptingCodec) Decode(data []byte, v any) error {
	if len(data) == 0 {
		return c.inner.Decode(data, v) //nolint:wrapcheck // transparent delegation
	}

	plaintext, err := c.decrypter.Decrypt(Ciphertext(data))
	if err != nil {
		return errorfamily.Wrapf(err, errorfamily.Corruption, "encryption.codec_decrypt", "decrypt")
	}

	return c.inner.Decode(plaintext, v) //nolint:wrapcheck // transparent delegation
}
