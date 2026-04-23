package command

import "github.com/larsartmann/go-cqrs-lite/core/pkg/id"

// CatalogMeta contains documentation metadata for auto-catalog generation.
type CatalogMeta struct {
	Name    string
	Version string
	Summary string
}

// Catalogable is implemented by commands that want to be auto-documented
// by the catalog/adapters package.
type Catalogable interface {
	Command
	CatalogInfo() CatalogMeta
}

// CatalogCore combines command.Core with catalog metadata.
// Embed this struct in your command to make it auto-catalogable.
//
// Example:
//
//	type CreateUser struct {
//	    *CatalogCore
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
type CatalogCore struct {
	*Core

	Meta CatalogMeta
}

// NewCatalogCore creates a CatalogCore with command metadata.
func NewCatalogCore(cmdType Type, aggregateID id.AggregateID, meta CatalogMeta) *CatalogCore {
	return &CatalogCore{
		Core: New(cmdType, aggregateID),
		Meta: meta,
	}
}

// CatalogInfo returns the catalog metadata for this command.
func (c *CatalogCore) CatalogInfo() CatalogMeta {
	return c.Meta
}
