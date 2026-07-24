package encryption

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type encTestEnv struct {
	store    *encryptedStore
	inner    *eventtest.FakeStore
	streamID id.StreamID
	ref      id.StreamRef
}

func newEncTestEnv(t *testing.T, opts ...MiddlewareOption) encTestEnv {
	t.Helper()

	ed, err := NewAES256GCM(aes256Key())
	if err != nil {
		t.Fatalf("NewAES256GCM: %v", err)
	}

	inner := eventtest.NewFakeStore()

	store, err := NewEncryptedStore(inner, ed, opts...)
	if err != nil {
		t.Fatalf("NewEncryptedStore: %v", err)
	}

	streamID := id.NewStreamID()

	return encTestEnv{
		store:    store,
		inner:    inner,
		streamID: streamID,
		ref:      id.NewStreamRef("User", streamID),
	}
}
