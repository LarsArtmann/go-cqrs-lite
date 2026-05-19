// Package docserver provides framework-agnostic HTTP handlers for serving
// auto-generated API documentation from a catalog.Catalog.
//
// It serves both raw specification documents (OpenAPI JSON/YAML, AsyncAPI JSON/YAML)
// and rendered HTML pages (Scalar for OpenAPI, AsyncAPI React for AsyncAPI).
//
// All handlers are stdlib net/http compatible — no framework dependency.
// Use with any router: stdlib mux, Gin, Chi, Echo, etc.
package docserver

import (
	"encoding/json"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/adapters"
)

// CatalogProvider returns a fresh catalog on each call.
// Typically wraps adapters.CatalogBuilder.Build().
type CatalogProvider func() *catalog.Catalog

// Config holds all configuration for the docs server.
type Config struct {
	// ServiceName is used as the title in generated specs.
	ServiceName string

	// Version is the API version.
	Version string

	// Description is an optional description for the specs.
	Description string

	// BasePath is the URL prefix for OpenAPI endpoint paths (default "/api").
	BasePath string

	// AsyncAPIServer configures the AsyncAPI server entry.
	AsyncAPIServer AsyncAPIServerConfig

	// DocsPath is the URL prefix where docs are served (default "/docs").
	DocsPath string
}

// AsyncAPIServerConfig configures the AsyncAPI server entry.
type AsyncAPIServerConfig struct {
	Name     string
	Host     string
	Protocol string
}

// DocsServer serves auto-generated API documentation from a catalog.
type DocsServer struct {
	config   Config
	provider CatalogProvider
}

// NewDocsServer creates a new docs server.
func NewDocsServer(provider CatalogProvider, cfg Config) *DocsServer {
	if cfg.BasePath == "" {
		cfg.BasePath = "/api"
	}

	if cfg.DocsPath == "" {
		cfg.DocsPath = "/docs"
	}

	if cfg.AsyncAPIServer.Protocol == "" {
		cfg.AsyncAPIServer = AsyncAPIServerConfig{
			Name:     "development",
			Host:     "localhost:8080",
			Protocol: "http",
		}
	}

	return &DocsServer{
		config:   cfg,
		provider: provider,
	}
}

// OpenAPISpec returns the raw OpenAPI JSON handler.
func (ds *DocsServer) OpenAPISpec() http.HandlerFunc {
	return ds.serveOpenAPIJSON
}

// OpenAPISpecYAML returns the raw OpenAPI YAML handler.
func (ds *DocsServer) OpenAPISpecYAML() http.HandlerFunc {
	return ds.serveOpenAPIYAML
}

// OpenAPIUI returns the rendered OpenAPI documentation (Scalar) handler.
func (ds *DocsServer) OpenAPIUI() http.HandlerFunc {
	return ds.serveOpenAPIHTML
}

// AsyncAPISpec returns the raw AsyncAPI JSON handler.
func (ds *DocsServer) AsyncAPISpec() http.HandlerFunc {
	return ds.serveAsyncAPIJSON
}

// AsyncAPISpecYAML returns the raw AsyncAPI YAML handler.
func (ds *DocsServer) AsyncAPISpecYAML() http.HandlerFunc {
	return ds.serveAsyncAPIYAML
}

// AsyncAPIUI returns the rendered AsyncAPI documentation handler.
func (ds *DocsServer) AsyncAPIUI() http.HandlerFunc {
	return ds.serveAsyncAPIHTML
}

// CatalogJSON returns the raw catalog JSON handler.
func (ds *DocsServer) CatalogJSON() http.HandlerFunc {
	return ds.serveCatalogJSON
}

// StaticFS returns an http.FileSystem for the embedded static assets
// (Scalar JS, AsyncAPI React JS/CSS) from the binary.
func (ds *DocsServer) StaticFS() http.FileSystem {
	return http.FS(staticAssets)
}

// RegisterRoutes registers all documentation routes on the given mux.
// Uses the DocsPath prefix from config.
func (ds *DocsServer) RegisterRoutes(mux *http.ServeMux) {
	prefix := ds.config.DocsPath

	mux.HandleFunc("GET "+prefix+"/openapi", ds.serveOpenAPIHTML)
	mux.HandleFunc("GET "+prefix+"/openapi.json", ds.serveOpenAPIJSON)
	mux.HandleFunc("GET "+prefix+"/openapi.yaml", ds.serveOpenAPIYAML)
	mux.HandleFunc("GET "+prefix+"/asyncapi", ds.serveAsyncAPIHTML)
	mux.HandleFunc("GET "+prefix+"/asyncapi.json", ds.serveAsyncAPIJSON)
	mux.HandleFunc("GET "+prefix+"/asyncapi.yaml", ds.serveAsyncAPIYAML)
	mux.HandleFunc("GET "+prefix+"/catalog.json", ds.serveCatalogJSON)

	// Serve embedded static assets (Scalar JS, AsyncAPI React JS/CSS)
	mux.Handle(
		"GET "+prefix+"/static/",
		http.StripPrefix(prefix+"/static/", http.FileServer(ds.StaticFS())),
	)
}

func (ds *DocsServer) serveOpenAPIJSON(writer http.ResponseWriter, _ *http.Request) {
	doc := ds.buildOpenAPI()

	writer.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")

	//nolint:errchkjson
	_ = enc.Encode(doc)
}

func (ds *DocsServer) serveOpenAPIYAML(writer http.ResponseWriter, _ *http.Request) {
	doc := ds.buildOpenAPI()

	b, err := json.Marshal(doc)
	if err != nil {
		http.Error(writer, "failed to marshal OpenAPI spec", http.StatusInternalServerError)

		return
	}

	yamlStr, err := adapters.JSONToYAML(b)
	if err != nil {
		http.Error(writer, "failed to convert to YAML", http.StatusInternalServerError)

		return
	}

	writer.Header().Set("Content-Type", "text/yaml; charset=utf-8")

	_, _ = writer.Write(yamlStr)
}

func (ds *DocsServer) serveOpenAPIHTML(writer http.ResponseWriter, _ *http.Request) {
	specURL := ds.config.DocsPath + "/openapi.json"

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, _ = writer.Write([]byte(scalarHTML(specURL, ds.config.ServiceName)))
}

func (ds *DocsServer) serveAsyncAPIJSON(writer http.ResponseWriter, _ *http.Request) {
	doc := ds.buildAsyncAPI()

	writer.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")

	//nolint:errchkjson
	_ = enc.Encode(doc)
}

func (ds *DocsServer) serveAsyncAPIYAML(writer http.ResponseWriter, _ *http.Request) {
	doc := ds.buildAsyncAPI()

	b, err := doc.MarshalYAML()
	if err != nil {
		http.Error(writer, "failed to marshal AsyncAPI YAML", http.StatusInternalServerError)

		return
	}

	writer.Header().Set("Content-Type", "text/yaml; charset=utf-8")

	_, _ = writer.Write(b)
}

func (ds *DocsServer) serveAsyncAPIHTML(writer http.ResponseWriter, _ *http.Request) {
	specURL := ds.config.DocsPath + "/asyncapi.json"

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, _ = writer.Write([]byte(asyncAPIHTML(specURL, ds.config.ServiceName)))
}

func (ds *DocsServer) serveCatalogJSON(writer http.ResponseWriter, _ *http.Request) {
	cat := ds.buildCatalog()

	writer.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")

	//nolint:errchkjson
	_ = enc.Encode(cat)
}
