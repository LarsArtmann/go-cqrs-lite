package event

type upcaster interface {
	SourceType() Type
	SourceVersion() SchemaVersion
	Upcast(evt Event) (*Core, error)
}

type upcasterFunc struct {
	sourceType    Type
	sourceVersion SchemaVersion
	upcast        func(evt Event) (*Core, error)
}

func newUpcaster(
	sourceType Type,
	sourceVersion SchemaVersion,
	upcast func(evt Event) (*Core, error),
) *upcasterFunc {
	return &upcasterFunc{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		upcast:        upcast,
	}
}

func (u *upcasterFunc) SourceType() Type { return u.sourceType }

func (u *upcasterFunc) SourceVersion() SchemaVersion { return u.sourceVersion }

func (u *upcasterFunc) Upcast(evt Event) (*Core, error) {
	return u.upcast(evt)
}

var _ upcaster = (*upcasterFunc)(nil)
