package eventcatalog

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"slices"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Defaults for [StaticServer].
const (
	indexFileName         = "index.html"
	notFoundFileName      = "404.html"
	htmlContentType       = "text/html; charset=utf-8"
	htmlCacheControl      = "no-cache"
	immutableCacheControl = "public, max-age=31536000, immutable"
	astroAssetPrefix      = "_astro/"
)

// StaticServer serves a built EventCatalog site from an [fs.FS].
//
// EventCatalog's `npx eventcatalog build` emits a static site: one
// `index.html` per route, content-addressed bundles under `_astro/`, and JSON
// under `api/`. A consumer can embed that directory with [embed] and serve the
// full EventCatalog UI from their own Go binary — no Node, npm, or preview
// server at deploy time:
//
//	//go:embed eventcatalog/dist
//	var dist embed.FS
//
//	sub, _ := fs.Sub(dist, "eventcatalog/dist")
//	srv, _ := eventcatalog.NewStaticServer(sub)
//	http.Handle("/", srv.Handler())
//
// The handler performs static-site routing: directory indexes
// (`/docs/foo/` -> `docs/foo/index.html`), the canonical-trailing-slash
// redirect for `/docs/foo` -> `/docs/foo/`, extensionless HTML
// (`/docs/foo` -> `docs/foo.html`), and a custom [WithNotFoundFile] page.
// Directory listings are never generated.
//
// Astro (EventCatalog's framework) emits root-absolute asset URLs
// (`/_astro/...`), so mount a StaticServer at a root path.
type StaticServer struct {
	fsys              fs.FS
	notFoundFile      string
	immutablePrefixes []string
}

// StaticOption configures a [StaticServer].
type StaticOption func(*StaticServer)

// WithNotFoundFile overrides the file served with an HTTP 404 when no route
// matches. Defaults to "404.html"; when the file is absent the server falls
// back to a plain-text not-found response.
func WithNotFoundFile(name string) StaticOption {
	return func(s *StaticServer) { s.notFoundFile = name }
}

// WithImmutableAssetPrefixes sets the path prefixes whose responses carry a
// long-lived `Cache-Control: public, max-age=31536000, immutable` header — safe
// because Astro content-hashes those file names. Defaults to ["_astro/"].
func WithImmutableAssetPrefixes(prefixes ...string) StaticOption {
	return func(s *StaticServer) {
		s.immutablePrefixes = append([]string(nil), prefixes...)
	}
}

// NewStaticServer wraps a built EventCatalog tree (an [fs.FS], typically a
// //go:embed directory) in a static [StaticServer]. It returns a Rejection
// error when fsys is nil.
func NewStaticServer(fsys fs.FS, opts ...StaticOption) (*StaticServer, error) {
	if fsys == nil {
		return nil, errorfamily.Newf(
			errorfamily.Rejection,
			"catalog.eventcatalog.server.1",
			"eventcatalog: nil filesystem",
		)
	}

	s := &StaticServer{
		fsys:              fsys,
		notFoundFile:      notFoundFileName,
		immutablePrefixes: []string{astroAssetPrefix},
	}
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Handler returns the [http.Handler] serving the embedded site.
func (s *StaticServer) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *StaticServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	rel, ok := requestKey(r.URL.Path)
	if !ok {
		s.serveNotFound(w, r)

		return
	}

	if dest, ok := s.redirectTarget(rel); ok {
		//nolint:gosec // dest is a validated path within the embedded tree
		http.Redirect(w, r, dest+r.URL.RawQuery, http.StatusMovedPermanently)

		return
	}

	file, ok := s.resolveFile(rel)
	if !ok {
		s.serveNotFound(w, r)

		return
	}

	s.serveFile(w, r, file)
}

// requestKey maps a URL path to an fs.FS-relative key, rejecting traversal.
func requestKey(urlPath string) (string, bool) {
	rel := strings.TrimPrefix(urlPath, "/")
	if rel == "" {
		return indexFileName, true
	}
	if hasDotDotSegment(rel) {
		return "", false
	}

	return rel, true
}

func hasDotDotSegment(rel string) bool {
	return slices.Contains(strings.Split(rel, "/"), "..")
}

// redirectTarget yields the canonical trailing-slash URL when a directory
// index exists for a slash-less request path.
func (s *StaticServer) redirectTarget(rel string) (string, bool) {
	if strings.HasSuffix(rel, "/") || rel == indexFileName {
		return "", false
	}
	if _, ok := s.file(rel); ok {
		return "", false
	}
	if _, ok := s.file(rel + "/" + indexFileName); ok {
		return "/" + rel + "/", true
	}

	return "", false
}

func (s *StaticServer) resolveFile(rel string) (string, bool) {
	if strings.HasSuffix(rel, "/") {
		return s.file(rel + indexFileName)
	}
	if file, ok := s.file(rel); ok {
		return file, true
	}
	if file, ok := s.file(rel + "/" + indexFileName); ok {
		return file, true
	}

	return s.file(rel + ".html")
}

// file reports whether rel names a regular file (never a directory) in the FS.
func (s *StaticServer) file(rel string) (string, bool) {
	info, err := fs.Stat(s.fsys, rel)
	if err != nil || info.IsDir() {
		return "", false
	}

	return rel, true
}

func (s *StaticServer) serveFile(w http.ResponseWriter, r *http.Request, file string) {
	f, err := s.fsys.Open(file)
	if err != nil {
		s.serveNotFound(w, r)

		return
	}
	defer func() { _ = f.Close() }()

	s.setHeaders(w, file)
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)

		return
	}

	_, _ = io.Copy(w, f)
}

func (s *StaticServer) setHeaders(w http.ResponseWriter, file string) {
	if ct := contentTypeFor(file); ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	switch {
	case strings.HasSuffix(file, ".html"):
		w.Header().Set("Cache-Control", htmlCacheControl)
	case s.isImmutable(file):
		w.Header().Set("Cache-Control", immutableCacheControl)
	}
}

func (s *StaticServer) isImmutable(file string) bool {
	for _, prefix := range s.immutablePrefixes {
		if strings.HasPrefix(file, prefix) {
			return true
		}
	}

	return false
}

func (s *StaticServer) serveNotFound(w http.ResponseWriter, r *http.Request) {
	f, err := s.fsys.Open(s.notFoundFile)
	if err != nil {
		http.Error(w, "404 page not found", http.StatusNotFound)

		return
	}
	defer func() { _ = f.Close() }()

	w.Header().Set("Content-Type", htmlContentType)
	w.Header().Set("Cache-Control", htmlCacheControl)
	w.WriteHeader(http.StatusNotFound)

	if r.Method == http.MethodHead {
		return
	}

	_, _ = io.Copy(w, f)
}

func contentTypeFor(file string) string {
	if strings.HasSuffix(file, ".html") {
		return htmlContentType
	}

	return mime.TypeByExtension(path.Ext(file))
}
