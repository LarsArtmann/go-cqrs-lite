package event

// Upcaster transforms an event from one schema version to the next.
// Chain multiple upcasters to migrate events across several versions.
//
// Example:
//
//	v1ToV2 := event.UpcasterFunc(func(e event.Core) (event.Core, error) {
//	    payload, _ := json.Marshal(map[string]any{
//	        "name":  string(e.Payload()),
//	        "email": "",
//	    })
//	    return event.Core{...}, nil
//	})
type Upcaster interface {
	// SourceType returns the event type this upcaster applies to.
	SourceType() Type

	// SourceVersion returns the schema version this upcaster transforms from.
	SourceVersion() int

	// Upcast transforms the event to the next schema version.
	Upcast(evt Event) (*Core, error)
}

// UpcasterFunc is a convenience type for creating upcasters.
type UpcasterFunc struct {
	sourceType    Type
	sourceVersion int
	upcast        func(evt Event) (*Core, error)
}

// NewUpcaster creates an UpcasterFunc.
func NewUpcaster(
	sourceType Type,
	sourceVersion int,
	upcast func(evt Event) (*Core, error),
) *UpcasterFunc {
	return &UpcasterFunc{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		upcast:        upcast,
	}
}

// SourceType returns the event type.
func (u *UpcasterFunc) SourceType() Type { return u.sourceType }

// SourceVersion returns the source schema version.
func (u *UpcasterFunc) SourceVersion() int { return u.sourceVersion }

// Upcast delegates to the upcast function.
func (u *UpcasterFunc) Upcast(evt Event) (*Core, error) {
	return u.upcast(evt)
}

var _ Upcaster = (*UpcasterFunc)(nil)
