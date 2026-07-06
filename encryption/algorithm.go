package encryption

import (
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

type Algorithm string

const (
	AES256GCM         Algorithm = "aes-256-gcm"
	XChaCha20Poly1305 Algorithm = "xchacha20-poly1305"
)

func (a Algorithm) String() string { return string(a) }

func (a Algorithm) IsZero() bool { return a == "" }

type KeyID string

func (k KeyID) String() string { return string(k) }

func (k KeyID) IsZero() bool { return k == "" }

const (
	AlgorithmKey event.MetadataKey = "event.encryption.algorithm"
	KeyIDKey     event.MetadataKey = "event.encryption.key-id"
)

func ExtractAlgorithm(evt event.Event) (Algorithm, error) {
	if evt == nil {
		return "", ErrNilEvent
	}

	md := evt.Metadata()
	if md.Custom == nil {
		return "", nil
	}

	v, ok := md.Custom[AlgorithmKey]
	if !ok || v == "" {
		return "", nil
	}

	alg := Algorithm(v)
	if alg != AES256GCM && alg != XChaCha20Poly1305 {
		return "", errorfamily.NewRejection(
			"encryption.unknown_algorithm",
			"unknown encryption algorithm: "+v,
		)
	}

	return alg, nil
}

func ExtractKeyID(evt event.Event) (KeyID, error) {
	if evt == nil {
		return "", ErrNilEvent
	}

	md := evt.Metadata()
	if md.Custom == nil {
		return "", nil
	}

	v, ok := md.Custom[KeyIDKey]
	if !ok {
		return "", nil
	}

	return KeyID(v), nil
}

// KeyResolver selects a Decrypter based on the key ID embedded in an encrypted event.
// Implementations typically look up keys from a map, vault, or KMS.
//
// Usage:
//
//	resolver := encryption.KeyResolverFunc(func(id encryption.KeyID) (encryption.Decrypter, error) {
//	    dec, ok := keys[id]
//	    if !ok {
//	        return nil, fmt.Errorf("unknown key: %s", id)
//	    }
//	    return dec, nil
//	})
type KeyResolver interface {
	Resolve(keyID KeyID) (Decrypter, error)
}

type KeyResolverFunc func(KeyID) (Decrypter, error)

func (f KeyResolverFunc) Resolve(keyID KeyID) (Decrypter, error) { return f(keyID) }
