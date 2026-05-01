package event

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// CatalogMeta contains documentation metadata for auto-catalog generation.
type CatalogMeta struct {
	Name          string
	Version       string
	Summary       string
	AggregateType AggregateType
}

// Catalogable is implemented by events that want to be auto-documented
// by the catalog/adapters package.
type Catalogable interface {
	Event
	CatalogInfo() CatalogMeta
}

// CatalogCore combines event.Core with catalog metadata.
// Embed this struct in your event to make it auto-catalogable.
type CatalogCore struct {
	*Core

	Meta CatalogMeta
}

// NewCatalogCore creates a CatalogCore with event metadata.
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

// MustNewCatalogCore creates a CatalogCore or panics on validation failure.
// Use only in tests where inputs are guaranteed valid.
func MustNewCatalogCore(
	eventType Type,
	aggregateID id.AggregateID,
	aggregateType AggregateType,
	version int,
	payload []byte,
	meta CatalogMeta,
	opts ...Option,
) *CatalogCore {
	cc, err := NewCatalogCore(eventType, aggregateID, aggregateType, version, payload, meta, opts...)
	if err != nil {
		panic(fmt.Sprintf("event.MustNewCatalogCore: %v", err))
	}

	return cc
}

// CatalogInfo returns the catalog metadata for this event.
func (c *CatalogCore) CatalogInfo() CatalogMeta {
	return c.Meta
}
