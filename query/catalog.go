package query

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
func NewCatalogCore(qtype Type, meta CatalogMeta) *CatalogCore {
	return &CatalogCore{
		Core: New(qtype),
		Meta: meta,
	}
}

// CatalogInfo returns the catalog metadata for this query.
func (c *CatalogCore) CatalogInfo() CatalogMeta {
	return c.Meta
}
