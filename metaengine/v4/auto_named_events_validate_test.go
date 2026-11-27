package metaengine

import (
	"strings"
	"testing"
)

type emptyCreated struct{}

type namedStringCreated string

// A sample whose type cannot carry the key field is a caller mistake the
// error-returning generator must report, not a panic.
func TestAutoCRUDByNamedEvents_UnusableSampleReturnsError(t *testing.T) {
	t.Parallel()

	type view struct {
		ID string
	}

	for name, sample := range map[string]any{
		"empty struct value": struct{}{},
		"named empty struct": emptyCreated{},
		"non-struct":         namedStringCreated("x"),
	} {
		_, err := AutoCRUDByNamedEvents[view]("ID", NamedEvent("thing.created", sample))
		if err == nil {
			t.Errorf("%s: expected an error, got none", name)
			continue
		}
		if !strings.Contains(err.Error(), `"thing.created"`) {
			t.Errorf("%s: error %q does not name the event type", name, err)
		}
	}
}
