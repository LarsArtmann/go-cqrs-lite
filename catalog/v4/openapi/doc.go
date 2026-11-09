// Package openapi generates OpenAPI 3.0.3 documents from a [catalog.Catalog].
//
// Commands become POST endpoints, queries become GET endpoints (with an {id}
// path parameter auto-extracted when the schema has an ID-like field), and
// events become POST /events/ endpoints. Each service becomes a tag.
//
//	exp := openapi.NewExporter("My API", "1.0.0", openapi.WithBasePath("/api"))
//	doc := exp.Export(cat)
//	yaml, _ := doc.MarshalYAML()
package openapi
