package event_test

import (
	"encoding/hex"
	"testing"

	"github.com/fxamacker/cbor/v2"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestProbeActorCBOR(t *testing.T) {
	actor := id.NewServiceActor("order-api")

	b, err := cbor.Marshal(actor)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("top-level ActorID: %s err=%v", hex.EncodeToString(b), err)

	var back id.ActorID
	if err := cbor.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	t.Logf("decoded: %q equal=%v", back.PrefixedString(), back.Equal(actor))

	uid := id.NewUserID()
	bu, err := cbor.Marshal(uid)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("top-level UserID: %s", hex.EncodeToString(bu))
}
