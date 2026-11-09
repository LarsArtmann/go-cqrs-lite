package docserver

import (
	"sort"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// View models + builders for the data-product pages of the embedded event
// catalog: the data-mesh half of the inventory (owned datasets with
// declared input/output ports and contracts).

// eventCatalogDataProductRow is one data-product entry on the overview page.
type eventCatalogDataProductRow struct {
	Name    string
	Href    string
	Version string
	Summary string
	Owners  string
}

// eventCatalogPortRow is one input or output port row on the detail page.
type eventCatalogPortRow struct {
	RefLabel  string
	RefHref   string
	Version   string
	Contract  string
	Direction string
}

// eventCatalogDataProductDetail is the view model of the data-product page.
type eventCatalogDataProductDetail struct {
	Brand      string
	DocsPath   string
	CatalogRef string
	Name       string
	ID         string
	Version    string
	Summary    string
	Owners     []string
	Hidden     bool
	Badges     []catalog.Badge
	Inputs     []eventCatalogPortRow
	Outputs    []eventCatalogPortRow
}

// eventCatalogDataProductHref is the detail-page path for one data product.
func eventCatalogDataProductHref(docsPath, id string) string {
	return docsPath + "/eventcatalog/data-products/" + id
}

// findDataProduct locates a data product by ID.
func findDataProduct(cat *catalog.Catalog, id string) (catalog.DataProduct, bool) {
	for _, product := range cat.DataProducts {
		if string(product.ID) == id {
			return product, true
		}
	}

	return catalog.DataProduct{}, false
}

// dataProductOverviewRows builds the overview table rows for every VISIBLE
// data product (Hidden products stay reachable by direct link but are not
// listed), sorted by name for stable ordering.
func dataProductOverviewRows(cfg Config, cat *catalog.Catalog) []eventCatalogDataProductRow {
	rows := make([]eventCatalogDataProductRow, 0, len(cat.DataProducts))
	for _, product := range cat.DataProducts {
		if product.Hidden {
			continue
		}

		rows = append(rows, eventCatalogDataProductRow{
			Name:    cmpOr(string(product.Name), string(product.ID)),
			Href:    eventCatalogDataProductHref(cfg.DocsPath, string(product.ID)),
			Version: string(product.Version),
			Summary: string(product.Summary),
			Owners:  joinOr(product.Owners),
		})
	}

	sortDataProductRows(rows)

	return rows
}

func sortDataProductRows(rows []eventCatalogDataProductRow) {
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
}

// newEventCatalogDataProductDetail builds the data-product detail view
// model. The second return value is false when no product with the ID
// exists. Input and output ports link to their message pages; output
// contracts render as path (name) when declared.
func newEventCatalogDataProductDetail(
	cfg Config,
	cat *catalog.Catalog,
	id string,
) (eventCatalogDataProductDetail, bool) {
	product, ok := findDataProduct(cat, id)
	if !ok {
		return eventCatalogDataProductDetail{}, false
	}

	detail := eventCatalogDataProductDetail{
		Brand:      cmpOr(cfg.ServiceName, string(cat.Title)),
		DocsPath:   cfg.DocsPath,
		CatalogRef: cfg.DocsPath + "/catalog.json",
		Name:       cmpOr(string(product.Name), string(product.ID)),
		ID:         string(product.ID),
		Version:    string(product.Version),
		Summary:    string(product.Summary),
		Owners:     product.Owners,
		Hidden:     product.Hidden,
		Badges:     product.Badges,
		Inputs:     portViewRows(cfg.DocsPath, product.Inputs, nil),
	}

	outputRefs := make([]catalog.Ref, len(product.Outputs))
	for i, out := range product.Outputs {
		outputRefs[i] = out.Ref
	}

	detail.Outputs = portViewRows(cfg.DocsPath, outputRefs, product.Outputs)

	return detail, true
}

// portRows renders input/output refs as link rows; for outputs the
// matching DataProductOutput carries the contract when declared.
func portViewRows(
	docsPath string,
	refs []catalog.Ref,
	outputs []catalog.DataProductOutput,
) []eventCatalogPortRow {
	rows := make([]eventCatalogPortRow, 0, len(refs))
	for i, ref := range refs {
		row := eventCatalogPortRow{
			RefLabel: string(ref.ID),
			RefHref:  eventCatalogMessageHref(docsPath, ref.ID),
			Version:  string(ref.Version),
		}

		if outputs == nil {
			row.Direction = "input"
		} else {
			row.Direction = "output"

			if i < len(outputs) && outputs[i].Contract != nil {
				row.Contract = outputs[i].Contract.Path
			}
		}

		rows = append(rows, row)
	}

	return rows
}
