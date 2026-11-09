package encryption

import (
	"maps"
	"sort"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// StaticKeyResolver resolves keys from a static map. It implements KeyResolver
// for the common case where keys are known at startup and rarely change.
//
// Usage:
//
//	resolver := encryption.NewStaticKeyResolver(map[encryption.KeyID]encryption.Decrypter{
//	    "key-v1": decV1,
//	    "key-v2": decV2,
//	})
type StaticKeyResolver struct {
	keys map[KeyID]Decrypter
}

// NewStaticKeyResolver creates a StaticKeyResolver from a map of key IDs to Decrypters.
// The map is copied internally so mutations to the original do not affect the resolver.
func NewStaticKeyResolver(keys map[KeyID]Decrypter) *StaticKeyResolver {
	copied := make(map[KeyID]Decrypter, len(keys))
	maps.Copy(copied, keys)

	return &StaticKeyResolver{keys: copied}
}

func (r *StaticKeyResolver) Resolve(keyID KeyID) (Decrypter, error) {
	dec, ok := r.keys[keyID]
	if !ok {
		available := r.availableKeys()

		return nil, errorfamily.Wrapf(
			ErrUnknownKeyID,
			errorfamily.Rejection,
			"encryption.unknown_key_id",
			"%q (available: %s)",
			keyID,
			available,
		)
	}

	return dec, nil
}

func (r *StaticKeyResolver) availableKeys() string {
	ids := make([]string, 0, len(r.keys))
	for k := range r.keys {
		ids = append(ids, string(k))
	}

	sort.Strings(ids)

	return strings.Join(ids, ", ")
}
