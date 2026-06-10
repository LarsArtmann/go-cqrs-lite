package encryption

import "github.com/larsartmann/go-cqrs-lite/event/v2"

type Algorithm string

const (
	AES256GCM         Algorithm = "aes-256-gcm"
	XChaCha20Poly1305 Algorithm = "xchacha20-poly1305"
)

func (a Algorithm) String() string { return string(a) }

func (a Algorithm) IsZero() bool { return a == "" }

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
		return "", event.NewRejection(
			"encryption.unknown_algorithm",
			"unknown encryption algorithm: "+v,
		)
	}

	return alg, nil
}

func ExtractKeyID(evt event.Event) (string, error) {
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

	return v, nil
}
