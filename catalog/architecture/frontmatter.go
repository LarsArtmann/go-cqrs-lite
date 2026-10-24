package architecture

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-faster/yaml"
	errorfamily "github.com/larsartmann/go-error-family"
)

// ExportSpec is a custom export declared in a markdown file's frontmatter
// (ADR-0160): the title and selectors of a redacted slice plus a markdown
// introduction taken from the file body.
//
//	---
//	title: Orders slice
//	include:
//	  - kind: domain
//	    id: ordering
//	hide: [owners]
//	---
//
//	Read this file with the code.
type ExportSpec struct {
	// Title is the slice's document title. The sentinel "catalog" uses the
	// catalog's own title, so one spec file can travel across catalogs.
	Title       string
	Description string
	Include     []Selector
	Exclude     []Selector
	NeverExport []Selector
	// References is how references to resources outside the kept set
	// render: "" or "omit" (the default), "name-only" or "include".
	// LoadExport rejects any other spelling; the CLI's -references flag
	// stays lenient and treats unknown values as omit.
	References   ReferencesMode
	Hide         []HideSection
	Introduction string
}

// specSelector is one frontmatter selector entry.
type specSelector struct {
	Kind string `yaml:"kind"`
	ID   string `yaml:"id"`
}

// specFrontmatter is the raw frontmatter shape.
type specFrontmatter struct {
	Title       string         `yaml:"title"`
	Description string         `yaml:"description"`
	Include     []specSelector `yaml:"include"`
	Exclude     []specSelector `yaml:"exclude"`
	NeverExport []specSelector `yaml:"neverExport"`
	References  string         `yaml:"references"`
	Hide        []string       `yaml:"hide"`
}

// LoadExport parses a custom-export spec from a .md/.mdx file: YAML
// frontmatter between --- fences, then the body as the markdown
// introduction. A missing title or include list is a Rejection, and so is
// any "x-*" selector key — ADR-0160 rules custom-property selectors out
// until the model has a typed custom-property concept — and any unknown
// "references" spelling (want omit, name-only or include).
func LoadExport(path string) (*ExportSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "catalog.architecture.5",
			"architecture: read export spec "+path)
	}

	text := string(raw)

	fence := "---"

	if !strings.HasPrefix(text, fence+"\n") && text != fence+"\n" {
		return nil, errorfamily.NewRejection("catalog.architecture.6",
			fmt.Sprintf("architecture: %s has no frontmatter fence (--- on the first line)", path))
	}

	rest := text[len(fence)+1:]

	end := strings.Index(rest, "\n"+fence)
	if end < 0 {
		return nil, errorfamily.NewRejection("catalog.architecture.6",
			fmt.Sprintf("architecture: %s frontmatter is never closed", path))
	}

	front, body := rest[:end], strings.TrimPrefix(rest[end+1+len(fence):], "\n")

	var parsed specFrontmatter

	if err := yaml.Unmarshal([]byte(front), &parsed); err != nil {
		return nil, errorfamily.WrapRejection(err, "catalog.architecture.6",
			fmt.Sprintf("architecture: %s frontmatter is not valid YAML", path))
	}

	return parsed.toSpec(body)
}

// toSpec validates the parsed frontmatter and attaches the body.
func (f specFrontmatter) toSpec(body string) (*ExportSpec, error) {
	spec := &ExportSpec{
		Title:        f.Title,
		Description:  f.Description,
		Include:      nil,
		Exclude:      nil,
		NeverExport:  nil,
		References:   "",
		Hide:         nil,
		Introduction: body,
	}

	if strings.TrimSpace(spec.Title) == "" {
		return nil, errorfamily.NewRejection("catalog.architecture.7",
			"architecture: export spec needs a title")
	}

	if len(f.Include) == 0 {
		return nil, errorfamily.NewRejection(
			"catalog.architecture.7",
			fmt.Sprintf(
				"architecture: export spec %q needs at least one include selector",
				spec.Title,
			),
		)
	}

	selectors := func(entries []specSelector) ([]Selector, error) {
		out := make([]Selector, 0, len(entries))

		for _, entry := range entries {
			if strings.HasPrefix(entry.Kind, "x-") || strings.HasPrefix(entry.ID, "x-") {
				return nil, errorfamily.NewRejection(
					"catalog.architecture.7",
					fmt.Sprintf(
						"architecture: custom-property selector %q/%q is not supported (ADR-0160)",
						entry.Kind,
						entry.ID,
					),
				)
			}

			out = append(out, Selector(entry))
		}

		return out, nil
	}

	include, err := selectors(f.Include)
	if err != nil {
		return nil, err
	}

	exclude, err := selectors(f.Exclude)
	if err != nil {
		return nil, err
	}

	never, err := selectors(f.NeverExport)
	if err != nil {
		return nil, err
	}

	spec.Include = include
	spec.Exclude = exclude
	spec.NeverExport = never

	switch f.References {
	case "":
		// The zero value is the default omit mode.
	case "omit":
		spec.References = ReferencesOmit
	case string(ReferencesNameOnly):
		spec.References = ReferencesNameOnly
	case string(ReferencesInclude):
		spec.References = ReferencesInclude
	default:
		return nil, errorfamily.NewRejection(
			"catalog.architecture.7",
			fmt.Sprintf(
				"architecture: export spec %q has unknown references mode %q (want omit, name-only or include)",
				spec.Title,
				f.References,
			),
		)
	}

	spec.Hide = make([]HideSection, 0, len(f.Hide))
	for _, hidden := range f.Hide {
		spec.Hide = append(spec.Hide, HideSection(hidden))
	}

	return spec, nil
}

// Options maps the spec onto exporter options. The introduction rides with
// the title option set separately, because a spec file may want the
// catalog's own title — spelled with the "catalog" sentinel, which simply
// sets no title option so the exporter falls back to the catalog's.
func (s *ExportSpec) Options() []Option {
	options := []Option{
		WithInclude(s.Include...),
	}

	if s.Title != "catalog" {
		options = append(options, WithTitle(s.Title))
	}

	if s.Description != "" {
		options = append(options, WithDescription(s.Description))
	}

	if len(s.Exclude) > 0 {
		options = append(options, WithExclude(s.Exclude...))
	}

	if len(s.NeverExport) > 0 {
		options = append(options, WithNeverExport(s.NeverExport...))
	}

	if s.References != "" {
		options = append(options, WithReferences(s.References))
	}

	if len(s.Hide) > 0 {
		options = append(options, WithHide(s.Hide...))
	}

	if s.Introduction != "" {
		options = append(options, WithIntroduction(s.Introduction))
	}

	return options
}
