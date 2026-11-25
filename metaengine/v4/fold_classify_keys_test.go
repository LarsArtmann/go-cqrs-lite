package metaengine_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// An update or remove fold without an insert fold has no key type to derive
// its key from; registration must say so instead of panicking on the first
// applied event (issue #133).
func TestQuery_UpdateFoldWithoutKeySourceFailsAtRegistration(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic at registration: an update fold with no insert fold has no key source")
		}

		msg, _ := r.(string)
		if !strings.Contains(msg, "pet.renamed") || !strings.Contains(msg, "no key source") {
			t.Fatalf("panic must name the fold and the missing key source, got: %v", r)
		}
	}()

	metaengine.Query[getTypedPet, typedPetView](
		"unkeyed_update",
		metaengine.OnTyped("pet.renamed", typedPetRenamed{}, func(e typedPetRenamed, prev typedPetView) typedPetView {
			prev.Name = e.Name

			return prev
		}),
	)
}

func TestQuery_RemoveFoldWithoutKeySourceFailsAtRegistration(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic at registration: a remove fold with no insert fold has no key source")
		}

		msg, _ := r.(string)
		if !strings.Contains(msg, "pet.released") || !strings.Contains(msg, "no key source") {
			t.Fatalf("panic must name the fold and the missing key source, got: %v", r)
		}
	}()

	metaengine.Query[getTypedPet, typedPetView](
		"unkeyed_remove",
		metaengine.OnTyped("pet.released", typedPetReleased{}, metaengine.Remove[typedPetView]()),
	)
}
