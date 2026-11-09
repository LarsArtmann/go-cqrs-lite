package schema

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

type Upcaster interface {
	SourceType() event.Type
	SourceVersion() event.SchemaVersion
	Upcast(evt event.Event) (event.Event, error)
}

func NewUpcaster(
	sourceType event.Type,
	sourceVersion event.SchemaVersion,
	upcast func(evt event.Event) (event.Event, error),
) Upcaster {
	return &upcasterFunc{
		sourceType:    sourceType,
		sourceVersion: sourceVersion,
		upcast:        upcast,
	}
}

type upcasterFunc struct {
	sourceType    event.Type
	sourceVersion event.SchemaVersion
	upcast        func(evt event.Event) (event.Event, error)
}

func (u *upcasterFunc) SourceType() event.Type { return u.sourceType }

func (u *upcasterFunc) SourceVersion() event.SchemaVersion { return u.sourceVersion }

func (u *upcasterFunc) Upcast(evt event.Event) (event.Event, error) {
	if u.upcast == nil {
		return nil, ErrNilUpcaster
	}

	return u.upcast(evt)
}

var _ Upcaster = (*upcasterFunc)(nil)
