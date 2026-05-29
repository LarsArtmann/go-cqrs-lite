package event

// Upcaster transforms an event from one schema version to the next.
// Register upcasters with NewVersionedStore to apply them automatically
// when loading events from the store.
type Upcaster interface {
	SourceType() Type
	SourceVersion() SchemaVersion
	Upcast(evt Event) (*ImmutableEvent, error)
}

// NewUpcaster creates an Upcaster from a function.
// The sourceType and sourceVersion identify which events to transform;
// the upcast function produces the new version of the event.
func NewUpcaster(
	sourceType Type,
	sourceVersion SchemaVersion,
	upcast func(evt Event) (*ImmutableEvent, error),
) Upcaster {
	return &upcasterFunc{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		upcast:        upcast,
	}
}

type upcasterFunc struct {
	sourceType    Type
	sourceVersion SchemaVersion
	upcast        func(evt Event) (*ImmutableEvent, error)
}

func (u *upcasterFunc) SourceType() Type { return u.sourceType }

func (u *upcasterFunc) SourceVersion() SchemaVersion { return u.sourceVersion }

func (u *upcasterFunc) Upcast(evt Event) (*ImmutableEvent, error) {
	return u.upcast(evt)
}

var _ Upcaster = (*upcasterFunc)(nil)
