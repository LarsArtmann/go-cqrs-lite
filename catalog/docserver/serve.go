package docserver

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// scalarFontCDN is the origin the Scalar bundle hardcodes for its Inter and
// mono @font-face files. Fully self-hosted deployments rewrite it to
// <DocsPath>/static/fonts (the vendored woff2 subsets), so no font request
// leaves this server and the browser console stays free of denied origins.
const scalarFontCDN = "https://fonts.scalar.com/"

// scalarJSHandler serves the embedded scalar.js with its @font-face URLs
// rewritten from Scalar's CDN to this server's vendored fonts. The rewrite is
// deterministic, so the rewritten body is computed once per DocsPath.
func (ds *DocsServer) scalarJSHandler(prefix string) http.HandlerFunc {
	selfPath := prefix + "/static/fonts/"
	rewritten := sync.OnceValues(func() ([]byte, error) {
		f, err := staticFilesystem.Open("scalar.js")
		if err != nil {
			return nil, fmt.Errorf("open embedded scalar.js: %w", err)
		}
		defer f.Close()

		raw, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("read embedded scalar.js: %w", err)
		}

		return bytes.ReplaceAll(raw, []byte(scalarFontCDN), []byte(selfPath)), nil
	})

	return func(w http.ResponseWriter, _ *http.Request) {
		body, err := rewritten()
		if err != nil {
			http.Error(w, "scalar.js asset unavailable", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write(body)
	}
}

func (ds *DocsServer) serveOpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	ds.serveJSON(w, ds.buildOpenAPIForRequest(r))
}

func (ds *DocsServer) serveOpenAPIYAML(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(ds.buildOpenAPIForRequest(r))
	if err != nil {
		http.Error(w, "failed to marshal OpenAPI spec", http.StatusInternalServerError)

		return
	}

	ds.serveYAML(w, b, "failed to convert to YAML")
}

func (ds *DocsServer) serveOpenAPIHTML(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)
	ds.renderComponent(
		w, r,
		ScalarPage(ds.config.ServiceName, ds.config.DocsPath, ds.config.DocsPath+"/openapi.json"),
	)
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

func (ds *DocsServer) serveAsyncAPIHTML(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSPAllowingEval(w, r)
	ds.renderComponent(
		w,
		r,
		AsyncAPIPage(
			ds.config.ServiceName,
			ds.config.DocsPath,
			ds.config.DocsPath+"/asyncapi.json",
		),
	)
}

func (ds *DocsServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	r = ds.applyCSP(w, r)
	data := newIndexPageData(ds.config, ds.provider())
	ds.renderComponent(w, r, IndexPage(data))
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
