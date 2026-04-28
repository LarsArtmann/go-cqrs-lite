package query

import "fmt"

// CatalogMeta contains documentation metadata for auto-catalog generation.
type CatalogMeta struct {
	Name    string
	Version string
	Summary string
}

// Catalogable is implemented by queries that want to be auto-documented
// by the catalog/adapters package.
type Catalogable interface {
	Query
	CatalogInfo() CatalogMeta
}

// CatalogCore combines query.Core with catalog metadata.
// Embed this struct in your query to make it auto-catalogable.
//
// Example:
//
//	type GetUser struct {
//	    *CatalogCore
//	    UserID string `json:"userId"`
//	}
type CatalogCore struct {
	*Core

	Meta CatalogMeta
}

// NewCatalogCore creates a CatalogCore with query metadata.
func NewCatalogCore(qtype Type, meta CatalogMeta) (*CatalogCore, error) {
	core, err := New(qtype)
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
func MustNewCatalogCore(qtype Type, meta CatalogMeta) *CatalogCore {
	cc, err := NewCatalogCore(qtype, meta)
	if err != nil {
		panic(fmt.Sprintf("query.MustNewCatalogCore: %v", err))
	}

	return cc
}

// CatalogInfo returns the catalog metadata for this query.
func (c *CatalogCore) CatalogInfo() CatalogMeta {
	return c.Meta
}
