package command

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

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
func NewCatalogCore(cmdType Type, aggregateID id.AggregateID, meta CatalogMeta) (*CatalogCore, error) {
	core, err := New(cmdType, aggregateID)
	if err != nil {
		return nil, err
	}

	return &CatalogCore{
		Core: core,
		Meta: meta,
	}, nil
}

// MustNewCatalogCore creates a CatalogCore or panics on validation failure.
// Use only in tests where inputs are guaranteed valid.
func MustNewCatalogCore(cmdType Type, aggregateID id.AggregateID, meta CatalogMeta) *CatalogCore {
	cc, err := NewCatalogCore(cmdType, aggregateID, meta)
	if err != nil {
		panic(fmt.Sprintf("command.MustNewCatalogCore: %v", err))
	}

	return cc
}

// CatalogInfo returns the catalog metadata for this command.
func (c *CatalogCore) CatalogInfo() CatalogMeta {
	return c.Meta
}
