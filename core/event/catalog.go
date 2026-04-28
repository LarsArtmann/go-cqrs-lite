package event

import "github.com/larsartmann/go-cqrs-lite/core/pkg/id"

type CatalogMeta struct {
	Name          string
	Version       string
	Summary       string
	AggregateType AggregateType
}

type Catalogable interface {
	Event
	CatalogInfo() CatalogMeta
}

type CatalogCore struct {
	*Core

	Meta CatalogMeta
}

func NewCatalogCore(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version int,
	payload []byte,
	meta CatalogMeta,
	opts ...Option,
) (*CatalogCore, error) {
	base, err := NewEvent(eventType, aggregateID, aggregateType, version, payload, opts...)
	if err != nil {
		return nil, err
	}

	return &CatalogCore{
		Core: base,
		Meta: meta,
	}, nil
}

func (c *CatalogCore) CatalogInfo() CatalogMeta {
	return c.Meta
}
