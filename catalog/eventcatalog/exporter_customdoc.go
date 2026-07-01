package eventcatalog

import (
	"fmt"
	"os"
	"path/filepath"

	errorfamily "github.com/larsartmann/go-error-family"
	yaml "github.com/go-faster/yaml"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

func (e *Exporter) writeCustomDoc(doc catalog.CustomDoc) error {
	slug := doc.Slug
	if slug == "" {
		slug = string(doc.ID)
	}

	dir := filepath.Join(e.outputDir, "docs", slug)

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_customdoc.1",
			"create custom doc dir: %v", err)
	}

	fm := customDocFM{
		ID:      string(doc.ID),
		Title:   string(doc.Title),
		Summary: string(doc.Summary),
		Slug:    slug,
		Owners:  doc.Owners,
		Badges:  toBadges(doc.Badges),
	}

	data, err := yaml.Marshal(fm)
	if err != nil {
		return errorfamily.Newf(errorfamily.Infrastructure, "catalog.exporter_customdoc.2",
			"marshal custom doc %s: %v", doc.ID, err)
	}

	body := doc.Content
	if body == "" {
		body = string(doc.Summary)
	}

	content := fmt.Sprintf("---\n%s---\n\n# %s\n\n%s\n", string(data), string(doc.Title), body)

	return e.writeMDXFile(filepath.Join(dir, "index.mdx"), content)
}
