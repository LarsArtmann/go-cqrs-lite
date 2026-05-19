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
	"github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"
	"github.com/larsartmann/go-cqrs-lite/catalog/openapi"
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
}

// registerRoutesPrefix registers all documentation routes with a pattern prefix.
// Compatible with Go 1.22+ ServeMux patterns.
func (ds *DocsServer) registerRoutesPrefix(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"/openapi", ds.serveOpenAPIHTML)
	mux.HandleFunc("GET "+prefix+"/openapi.json", ds.serveOpenAPIJSON)
	mux.HandleFunc("GET "+prefix+"/openapi.yaml", ds.serveOpenAPIYAML)
	mux.HandleFunc("GET "+prefix+"/asyncapi", ds.serveAsyncAPIHTML)
	mux.HandleFunc("GET "+prefix+"/asyncapi.json", ds.serveAsyncAPIJSON)
	mux.HandleFunc("GET "+prefix+"/asyncapi.yaml", ds.serveAsyncAPIYAML)
	mux.HandleFunc("GET "+prefix+"/catalog.json", ds.serveCatalogJSON)
}

func (ds *DocsServer) buildOpenAPI() *openapi.Document {
	cat := ds.provider()
	opts := []openapi.Option{}
	if ds.config.Description != "" {
		opts = append(opts, openapi.WithDescription(ds.config.Description))
	}
	if ds.config.BasePath != "" {
		opts = append(opts, openapi.WithBasePath(ds.config.BasePath))
	}

	return openapi.NewExporter(ds.config.ServiceName, ds.config.Version, opts...).Export(cat)
}

func (ds *DocsServer) buildAsyncAPI() *asyncapi.Document {
	cat := ds.provider()
	opts := []asyncapi.Option{
		asyncapi.WithServer(
			ds.config.AsyncAPIServer.Name,
			ds.config.AsyncAPIServer.Host,
			ds.config.AsyncAPIServer.Protocol,
		),
	}
	if ds.config.Description != "" {
		opts = append(opts, asyncapi.WithDescription(ds.config.Description))
	}

	return asyncapi.NewExporter(ds.config.ServiceName, ds.config.Version, opts...).Export(cat)
}

func (ds *DocsServer) serveOpenAPIJSON(w http.ResponseWriter, _ *http.Request) {
	doc := ds.buildOpenAPI()
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(doc)
}

func (ds *DocsServer) serveOpenAPIYAML(w http.ResponseWriter, _ *http.Request) {
	doc := ds.buildOpenAPI()
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")

	b, err := json.Marshal(doc)
	if err != nil {
		http.Error(w, "failed to marshal OpenAPI spec", http.StatusInternalServerError)
		return
	}

	yamlStr, err := adapters.JSONToYAML(b)
	if err != nil {
		http.Error(w, "failed to convert to YAML", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(yamlStr)
}

func (ds *DocsServer) serveOpenAPIHTML(w http.ResponseWriter, _ *http.Request) {
	specURL := ds.config.DocsPath + "/openapi.json"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(scalarHTML(specURL, ds.config.ServiceName)))
}

func (ds *DocsServer) serveAsyncAPIJSON(w http.ResponseWriter, _ *http.Request) {
	doc := ds.buildAsyncAPI()
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(doc)
}

func (ds *DocsServer) serveAsyncAPIYAML(w http.ResponseWriter, _ *http.Request) {
	doc := ds.buildAsyncAPI()

	b, err := doc.MarshalYAML()
	if err != nil {
		http.Error(w, "failed to marshal AsyncAPI YAML", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	_, _ = w.Write(b)
}

func (ds *DocsServer) serveAsyncAPIHTML(w http.ResponseWriter, _ *http.Request) {
	specURL := ds.config.DocsPath + "/asyncapi.json"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(asyncAPIHTML(specURL, ds.config.ServiceName)))
}

func (ds *DocsServer) serveCatalogJSON(w http.ResponseWriter, _ *http.Request) {
	cat := ds.provider()
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(cat)
}
