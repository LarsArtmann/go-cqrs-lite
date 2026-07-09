// Package docserver provides framework-agnostic HTTP handlers for serving
// auto-generated API documentation from a catalog.Catalog.
//
// It serves both raw specification documents (OpenAPI JSON/YAML, AsyncAPI JSON/YAML)
// and rendered HTML pages (Scalar for OpenAPI, AsyncAPI React for AsyncAPI).
//
// # Full Docs Server
//
// Use [NewDocsServer] for the complete documentation suite — OpenAPI/AsyncAPI
// JSON+YAML+HTML UI, catalog JSON, and embedded static assets (Scalar, AsyncAPI React).
//
// # Standalone Handlers
//
// For lighter use cases, these package-level functions work without a DocsServer:
//
//   - [D2Handler] — D2 architecture diagram (text/plain)
//   - [HealthCheckHandler] — liveness probe that verifies the catalog has services
//   - [GenerateEventCatalog] — write EventCatalog MDX files at startup (not an HTTP handler)
//
// All handlers are stdlib net/http compatible — no framework dependency.
// Use with any router: stdlib mux, Gin, Chi, Echo, etc.
package docserver

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io/fs"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/schema"
)

const yamlContentType = "text/yaml; charset=utf-8"

// CatalogProvider returns a fresh catalog on each call.
// Typically wraps catalog.Builder.Build().
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
	cfg.BasePath = cmp.Or(cfg.BasePath, "/api")
	cfg.DocsPath = cmp.Or(cfg.DocsPath, "/docs")

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
// The FS is rooted at the "static" subdirectory, so files are accessible
// as "asyncapi-react.js", "scalar.js", etc. (no "static/" prefix needed).
// staticFilesystem caches the sub-filesystem for embedded static assets.
// Computed once at init; falls back to root embed if the "static" subdir
// is missing (which only happens if the //go:embed directive is wrong).
var staticFilesystem = func() http.FileSystem { //nolint:gochecknoglobals // cached once at init
	sub, err := fs.Sub(staticAssets, "static")
	if err != nil {
		return http.FS(staticAssets)
	}

	return http.FS(sub)
}()

func (ds *DocsServer) StaticFS() http.FileSystem {
	return staticFilesystem
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

func (ds *DocsServer) serveOpenAPIJSON(w http.ResponseWriter, _ *http.Request) {
	ds.serveJSON(w, ds.buildOpenAPI())
}

func (ds *DocsServer) serveOpenAPIYAML(w http.ResponseWriter, _ *http.Request) {
	b, err := json.Marshal(ds.buildOpenAPI())
	if err != nil {
		http.Error(w, "failed to marshal OpenAPI spec", http.StatusInternalServerError)

		return
	}

	ds.serveYAML(w, b, "failed to convert to YAML")
}

func (ds *DocsServer) serveOpenAPIHTML(w http.ResponseWriter, _ *http.Request) {
	ds.serveHTML(w, ds.config.DocsPath+"/openapi.json", scalarHTML)
}

func (ds *DocsServer) serveAsyncAPIJSON(w http.ResponseWriter, _ *http.Request) {
	ds.serveJSON(w, ds.buildAsyncAPI())
}

func (ds *DocsServer) serveAsyncAPIYAML(w http.ResponseWriter, _ *http.Request) {
	b, err := ds.buildAsyncAPI().MarshalYAML()
	if err != nil {
		http.Error(w, "failed to marshal AsyncAPI YAML", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", yamlContentType)

	_, _ = w.Write(b)
}

func (ds *DocsServer) serveAsyncAPIHTML(w http.ResponseWriter, _ *http.Request) {
	ds.serveHTML(w, ds.config.DocsPath+"/asyncapi.json", asyncAPIHTML)
}

func (ds *DocsServer) serveCatalogJSON(w http.ResponseWriter, _ *http.Request) {
	ds.serveJSON(w, ds.buildCatalog())
}

func (ds *DocsServer) serveJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")

	enc := jsontext.NewEncoder(w, jsontext.WithIndent("  "))

	//nolint:errchkjson // known-safe JSON template string
	_ = json.MarshalEncode(enc, v)
}

func (ds *DocsServer) serveYAML(w http.ResponseWriter, jsonBytes []byte, errMsg string) {
	yamlStr, err := schema.JSONToYAML(jsonBytes)
	if err != nil {
		http.Error(w, errMsg, http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", yamlContentType)

	_, _ = w.Write(yamlStr)
}

type htmlRenderer func(specURL, serviceName string) string

func (ds *DocsServer) serveHTML(w http.ResponseWriter, specURL string, render htmlRenderer) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, _ = w.Write([]byte(render(specURL, ds.config.ServiceName)))
}
