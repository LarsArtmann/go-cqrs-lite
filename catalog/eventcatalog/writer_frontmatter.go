package eventcatalog

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
)

type frontmatterWriter struct {
	*strings.Builder
}

func newFrontmatterWriter() *frontmatterWriter {
	f := &frontmatterWriter{Builder: new(strings.Builder)}
	_, _ = f.WriteString("---\n")

	return f
}

func (f *frontmatterWriter) addField(key, value string) {
	_, _ = fmt.Fprintf(f, "%s: %s\n", key, value)
}

func (f *frontmatterWriter) addQuotedField(key, value string) {
	_, _ = fmt.Fprintf(f, "%s: %q\n", key, value)
}

func (f *frontmatterWriter) addListField(key string, values []string) {
	if len(values) == 0 {
		return
	}

	_, _ = fmt.Fprintf(f, "%s:\n", key)

	for _, v := range values {
		_, _ = fmt.Fprintf(f, "  - %s\n", v)
	}
}

func addObjectIDsListField[S ~string](f *frontmatterWriter, key string, ids []S) {
	if len(ids) == 0 {
		return
	}

	_, _ = fmt.Fprintf(f, "%s:\n", key)

	for _, id := range ids {
		_, _ = fmt.Fprintf(f, "  - id: %s\n", id)
	}
}

func (f *frontmatterWriter) finish(title, summary string) {
	_, _ = f.WriteString("---\n\n")
	_, _ = fmt.Fprintf(f, "# %s\n\n%s\n", title, summary)
}

func (f *frontmatterWriter) finishWithGraph(title, summary string) {
	_, _ = f.WriteString("---\n\n")
	_, _ = fmt.Fprintf(f, "# %s\n\n%s\n\n<NodeGraph />\n", title, summary)
}

func writeIDListField[S ~string](md *frontmatterWriter, key string, ids []S) {
	if len(ids) == 0 {
		return
	}

	_, _ = fmt.Fprintf(md, "%s:\n", key)

	for _, id := range ids {
		_, _ = fmt.Fprintf(md, "  - id: %s\n", id)
	}
}

func writeBadges(md *frontmatterWriter, badges []catalog.Badge) {
	if len(badges) == 0 {
		return
	}

	_, _ = md.WriteString("badges:\n")

	for _, b := range badges {
		_, _ = fmt.Fprintf(md, "  - content: %q\n", b.Content)

		if b.BackgroundColor != "" {
			_, _ = fmt.Fprintf(md, "    backgroundColor: %s\n", b.BackgroundColor)
		}

		if b.TextColor != "" {
			_, _ = fmt.Fprintf(md, "    textColor: %s\n", b.TextColor)
		}

		if b.Icon != "" {
			_, _ = fmt.Fprintf(md, "    icon: %s\n", b.Icon)
		}
	}
}

func writeRepository(md *frontmatterWriter, repo *catalog.Repository) {
	if repo == nil {
		return
	}

	_, _ = md.WriteString("repository:\n")

	if repo.Language != "" {
		_, _ = fmt.Fprintf(md, "  language: %q\n", repo.Language)
	}

	if repo.URL != "" {
		_, _ = fmt.Fprintf(md, "  url: %q\n", repo.URL)
	}
}

func writeOperation(md *frontmatterWriter, op *catalog.Operation) {
	if op == nil {
		return
	}

	_, _ = md.WriteString("operation:\n")
	_, _ = fmt.Fprintf(md, "  method: %s\n", op.Method)
	_, _ = fmt.Fprintf(md, "  path: %q\n", op.Path)

	if len(op.StatusCodes) > 0 {
		_, _ = md.WriteString("  statusCodes:\n")

		for _, sc := range op.StatusCodes {
			_, _ = fmt.Fprintf(md, "    - %q\n", sc)
		}
	}
}

func writeSpecifications(md *frontmatterWriter, specs []catalog.Specification) {
	if len(specs) == 0 {
		return
	}

	_, _ = md.WriteString("specifications:\n")

	for _, s := range specs {
		_, _ = fmt.Fprintf(md, "  - type: %s\n", s.Type)
		_, _ = fmt.Fprintf(md, "    path: %q\n", s.Path)

		if s.Name != "" {
			_, _ = fmt.Fprintf(md, "    name: %q\n", s.Name)
		}
	}
}

func writeAttachments(md *frontmatterWriter, attachments []catalog.Attachment) {
	if len(attachments) == 0 {
		return
	}

	_, _ = md.WriteString("attachments:\n")

	for _, a := range attachments {
		_, _ = fmt.Fprintf(md, "  - url: %q\n", a.URL)

		if a.Title != "" {
			_, _ = fmt.Fprintf(md, "    title: %q\n", a.Title)
		}

		if a.Type != "" {
			_, _ = fmt.Fprintf(md, "    type: %q\n", a.Type)
		}
	}
}

func writeMessagePointers(md *frontmatterWriter, key string, ptrs []catalog.Ref) {
	if len(ptrs) == 0 {
		return
	}

	_, _ = fmt.Fprintf(md, "%s:\n", key)

	for _, p := range ptrs {
		_, _ = fmt.Fprintf(md, "  - id: %s\n", p.ID)

		if p.Version != "" {
			_, _ = fmt.Fprintf(md, "    version: %q\n", p.Version)
		}
	}
}
