package encryption

import (
	"fmt"
	"sort"
	"strings"
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
	for k, v := range keys {
		copied[k] = v
	}

	return &StaticKeyResolver{keys: copied}
}

func (r *StaticKeyResolver) Resolve(keyID KeyID) (Decrypter, error) {
	dec, ok := r.keys[keyID]
	if !ok {
		available := r.availableKeys()
		return nil, fmt.Errorf("encryption: unknown key %q (available: %s)", keyID, available)
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
